// Package async provides an HTTP-based async transport using Server-Sent Events (SSE).
//
// It exposes contract methods via two HTTP endpoints:
//   - POST {path} - Submit async request, returns immediately with accepted status
//   - GET  {path} - SSE connection for receiving responses
//
// Example usage:
//
//	svc := contract.Register[TodoAPI](impl)
//	r := mizu.NewRouter()
//	async.Mount(r, "/async", svc)
package async

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
)

// Mount registers async endpoints on a mizu router.
// POST {path} - Submit async request
// GET  {path} - SSE connection for receiving responses
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error {
	if r == nil {
		return errors.New("async: nil router")
	}
	h, err := Handler(inv, opts...)
	if err != nil {
		return err
	}

	// Normalize path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	r.Post(path, h)
	r.Get(path, h)
	return nil
}

// Handler returns a mizu.Handler for the async endpoint.
// The handler routes based on HTTP method (POST=submit, GET=stream).
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
	if inv == nil {
		return nil, errors.New("async: nil invoker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("async: nil descriptor")
	}

	o := applyOptions(opts)
	h := newHub(o.bufferSize)

	return func(c *mizu.Ctx) error {
		switch c.Request().Method {
		case http.MethodPost:
			return handleSubmit(c, inv, svc, h, o)
		case http.MethodGet:
			return handleStream(c, h, o)
		default:
			c.Header().Set("Allow", "GET, POST")
			return c.Text(http.StatusMethodNotAllowed, "method not allowed")
		}
	}, nil
}

func handleSubmit(c *mizu.Ctx, inv contract.Invoker, svc *contract.Service, h *hub, o *options) error {
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, o.maxBodySize+1))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "read error"})
	}
	if int64(len(body)) > o.maxBodySize {
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{"error": "body too large"})
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}
	if strings.TrimSpace(req.ID) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing id"})
	}
	if strings.TrimSpace(req.Method) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing method"})
	}

	// Parse method: resource.method
	resource, method, ok := parseMethod(req.Method)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid method format, expected resource.method"})
	}

	// Check method exists
	methodDesc := svc.Method(resource, method)
	if methodDesc == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "unknown method"})
	}

	// Accept immediately
	if err := c.JSON(http.StatusAccepted, acceptedResponse{ID: req.ID, Status: "accepted"}); err != nil {
		return err
	}

	// Process async
	go func() {
		ctx := context.Background()
		resp := processRequest(ctx, inv, svc, resource, method, req.ID, req.Params, o)
		data, _ := json.Marshal(resp)

		eventType := "result"
		if resp.Error != nil {
			eventType = "error"
		}

		// Format SSE event
		event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
		h.broadcast([]byte(event))
	}()

	return nil
}

func handleStream(c *mizu.Ctx, h *hub, o *options) error {
	// Set SSE headers
	c.Header().Set("Content-Type", "text/event-stream")
	c.Header().Set("Cache-Control", "no-cache")
	c.Header().Set("Connection", "keep-alive")
	c.Header().Set("X-Accel-Buffering", "no")

	// Generate client ID
	clientID := generateID()

	client := &client{
		id:     clientID,
		events: make(chan []byte, o.bufferSize),
	}
	h.register(client)
	defer h.unregister(clientID)

	if o.onConnect != nil {
		o.onConnect(clientID)
	}
	defer func() {
		if o.onDisconnect != nil {
			o.onDisconnect(clientID)
		}
	}()

	w := c.Writer()

	// Flush headers using Ctx.Flush() which properly handles wrapped writers
	_ = c.Flush()

	ctx := c.Request().Context()
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-client.events:
			if !ok {
				return nil
			}
			if _, err := w.Write(event); err != nil {
				return nil
			}
			_ = c.Flush()
		case <-ping.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return nil
			}
			_ = c.Flush()
		}
	}
}

func processRequest(ctx context.Context, inv contract.Invoker, svc *contract.Service, resource, method, reqID string, params json.RawMessage, o *options) Response {
	methodDesc := svc.Method(resource, method)
	if methodDesc == nil {
		return Response{
			ID:    reqID,
			Error: &Error{Code: "method_not_found", Message: "unknown method"},
		}
	}

	// Allocate and decode input
	var in any
	if methodDesc.Input != "" {
		var err error
		in, err = inv.NewInput(resource, method)
		if err != nil || in == nil {
			return Response{
				ID:    reqID,
				Error: &Error{Code: "internal_error", Message: "failed to allocate input"},
			}
		}
		if len(params) != 0 && string(params) != "null" {
			if err := json.Unmarshal(params, in); err != nil {
				return Response{
					ID:    reqID,
					Error: &Error{Code: "invalid_params", Message: err.Error()},
				}
			}
		}
	}

	// Invoke method
	out, err := inv.Call(ctx, resource, method, in)
	if err != nil {
		code, msg := o.errorMapper(err)
		return Response{
			ID:    reqID,
			Error: &Error{Code: code, Message: msg},
		}
	}

	// Marshal result
	var result json.RawMessage
	if out != nil {
		b, err := json.Marshal(out)
		if err != nil {
			return Response{
				ID:    reqID,
				Error: &Error{Code: "marshal_error", Message: err.Error()},
			}
		}
		result = b
	} else {
		result = json.RawMessage("null")
	}

	return Response{
		ID:     reqID,
		Result: result,
	}
}

func parseMethod(s string) (resource, method string, ok bool) {
	idx := strings.LastIndex(s, ".")
	if idx <= 0 || idx >= len(s)-1 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

// AsyncAPI generates an AsyncAPI 2.6 specification document.
func AsyncAPI(svc *contract.Service) ([]byte, error) {
	return AsyncAPIDocument(svc)
}
