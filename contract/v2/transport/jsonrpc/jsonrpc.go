// Package jsonrpc provides JSON-RPC 2.0 transport for contract services.
//
// Method naming convention:
//   - JSON-RPC method name is "<resource>.<method>"
//     Example: "todos.list", "todos.get", "todos.create"
//
// Params convention:
//   - params is an object decoded into the input struct (by JSON field names)
//   - params omitted or null for methods without input
//   - array params are rejected (keeps the contract DX simple)
//
// Output convention:
//   - result is the method output (marshaled as JSON)
//   - if method has no output, result is null
//
// Notifications:
//   - if id is missing or null, the call is treated as a notification (no response)
package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
)

// Mount registers a JSON-RPC endpoint on a mizu router.
// The endpoint accepts POST requests with JSON-RPC 2.0 payloads.
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error {
	if r == nil {
		return errors.New("jsonrpc: nil router")
	}
	handler, err := Handler(inv, opts...)
	if err != nil {
		return err
	}
	if path == "" {
		path = "/"
	}
	r.Post(path, handler)
	return nil
}

// Handler returns a mizu.Handler for JSON-RPC 2.0 requests.
// This is the primary API when you need direct control.
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
	if inv == nil {
		return nil, errors.New("jsonrpc: nil invoker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("jsonrpc: nil descriptor")
	}

	o := applyOptions(opts)

	return func(c *mizu.Ctx) error {
		if c.Request().Method != http.MethodPost {
			c.Header().Set("Allow", "POST")
			c.Header().Set("Content-Type", "text/plain; charset=utf-8")
			c.Status(http.StatusMethodNotAllowed)
			_, _ = c.Write([]byte("method not allowed"))
			return nil
		}

		// Read request body with size limit
		body, err := io.ReadAll(io.LimitReader(c.Request().Body, o.maxBodySize+1))
		if err != nil {
			return writeError(c, nil, errParse, "parse error", err.Error())
		}
		if int64(len(body)) > o.maxBodySize {
			return writeError(c, nil, errInvalidRequest, "invalid request", "body too large")
		}

		raw := strings.TrimSpace(string(body))
		if raw == "" {
			return writeError(c, nil, errInvalidRequest, "invalid request", "empty body")
		}

		// Handle batch or single request
		if raw[0] == '[' {
			return handleBatch(c, inv, svc, body, o)
		}
		return handleSingle(c, inv, svc, body, o)
	}, nil
}

// OpenRPC generates an OpenRPC 1.2 specification from a contract descriptor.
func OpenRPC(svc *contract.Service) ([]byte, error) {
	return OpenRPCDocument(svc)
}

// handleSingle processes a single JSON-RPC request.
func handleSingle(c *mizu.Ctx, inv contract.Invoker, svc *contract.Service, body []byte, o *options) error {
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return writeError(c, nil, errParse, "parse error", err.Error())
	}

	resp := processRequest(c.Context(), inv, svc, &req, o)
	if resp == nil {
		// Notification (no id) - no response
		return c.NoContent()
	}

	return c.JSON(http.StatusOK, resp)
}

// handleBatch processes a batch of JSON-RPC requests.
func handleBatch(c *mizu.Ctx, inv contract.Invoker, svc *contract.Service, body []byte, o *options) error {
	var batch []json.RawMessage
	if err := json.Unmarshal(body, &batch); err != nil {
		return writeError(c, nil, errParse, "parse error", err.Error())
	}
	if len(batch) == 0 {
		return writeError(c, nil, errInvalidRequest, "invalid request", "empty batch")
	}

	var responses []any
	for _, raw := range batch {
		var req request
		if err := json.Unmarshal(raw, &req); err != nil {
			responses = append(responses, errorResponse(nil, errInvalidRequest, "invalid request", err.Error()))
			continue
		}
		if resp := processRequest(c.Context(), inv, svc, &req, o); resp != nil {
			responses = append(responses, resp)
		}
	}

	if len(responses) == 0 {
		// All notifications - no response
		return c.NoContent()
	}

	return c.JSON(http.StatusOK, responses)
}

// processRequest handles a single parsed request.
// Returns nil for notifications (requests without id).
func processRequest(ctx context.Context, inv contract.Invoker, svc *contract.Service, req *request, o *options) *response {
	// Validate JSON-RPC version and method
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		if !req.hasID() {
			return nil
		}
		resp := errorResponse(req.ID, errInvalidRequest, "invalid request", "missing jsonrpc=2.0 or method")
		return &resp
	}

	// Parse method name: "resource.method"
	resource, method, ok := splitMethod(req.Method)
	if !ok {
		if !req.hasID() {
			return nil
		}
		resp := errorResponse(req.ID, errMethodNotFound, "method not found", "method must be <resource>.<method>")
		return &resp
	}

	// Find method in descriptor
	methodDesc := svc.Method(resource, method)
	if methodDesc == nil {
		if !req.hasID() {
			return nil
		}
		resp := errorResponse(req.ID, errMethodNotFound, "method not found", "unknown method")
		return &resp
	}

	// Allocate input if method has input type
	var in any
	if methodDesc.Input != "" {
		var err error
		in, err = inv.NewInput(resource, method)
		if err != nil || in == nil {
			if !req.hasID() {
				return nil
			}
			resp := errorResponse(req.ID, errInternal, "internal error", "failed to allocate input")
			return &resp
		}
	}

	// Decode params into input
	if in != nil && len(req.Params) != 0 && string(req.Params) != "null" {
		// Reject array params
		if isJSONArray(req.Params) {
			if !req.hasID() {
				return nil
			}
			resp := errorResponse(req.ID, errInvalidParams, "invalid params", "array params not supported")
			return &resp
		}
		if err := json.Unmarshal(req.Params, in); err != nil {
			if !req.hasID() {
				return nil
			}
			resp := errorResponse(req.ID, errInvalidParams, "invalid params", err.Error())
			return &resp
		}
	}

	// Invoke the contract method
	out, err := inv.Call(ctx, resource, method, in)
	if err != nil {
		if !req.hasID() {
			return nil
		}
		code, msg, data := o.errorMapper(err)
		resp := errorResponse(req.ID, code, msg, data)
		return &resp
	}

	// Notification: no response
	if !req.hasID() {
		return nil
	}

	// No output type: result is null
	if methodDesc.Output == "" {
		resp := successResponse(req.ID, json.RawMessage("null"))
		return &resp
	}

	// Marshal output
	b, err := json.Marshal(out)
	if err != nil {
		resp := errorResponse(req.ID, errInternal, "internal error", err.Error())
		return &resp
	}
	resp := successResponse(req.ID, json.RawMessage(b))
	return &resp
}

// writeError writes a JSON-RPC error response.
func writeError(c *mizu.Ctx, id any, code int, message string, data any) error {
	return c.JSON(http.StatusOK, errorResponse(id, code, message, data))
}

// splitMethod splits "resource.method" into parts.
func splitMethod(full string) (resource string, method string, ok bool) {
	full = strings.TrimSpace(full)
	i := strings.IndexByte(full, '.')
	if i <= 0 || i == len(full)-1 {
		return "", "", false
	}
	return full[:i], full[i+1:], true
}

// isJSONArray checks if raw JSON starts with '['.
func isJSONArray(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return len(s) > 0 && s[0] == '['
}
