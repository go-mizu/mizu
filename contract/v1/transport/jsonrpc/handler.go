package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract/v1"
)

// Handler handles JSON-RPC 2.0 requests over HTTP.
type Handler struct {
	service *contract.Service
	codec   *Codec
}

// Option configures the handler.
type Option func(*Handler)

// NewHandler creates a new JSON-RPC handler for the service.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service: svc,
		codec:   NewCodec(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Name returns the transport name.
func (h *Handler) Name() string {
	return "jsonrpc"
}

// ServeHTTP handles HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requests, err := h.codec.Decode(r.Body)
	if err != nil {
		var rpcErr *Error
		if errors.As(err, &rpcErr) {
			h.writeError(w, nil, rpcErr)
			return
		}
		h.writeError(w, nil, InternalError(err))
		return
	}

	ctx := r.Context()
	isBatch := len(requests) > 1

	var responses []*Response
	for _, req := range requests {
		resp, hasResponse := h.handleOne(ctx, req)
		if hasResponse {
			responses = append(responses, resp)
		}
	}

	if len(responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", h.codec.ContentType())

	if isBatch {
		_ = h.codec.EncodeBatch(w, responses)
		return
	}

	_ = h.codec.Encode(w, responses[0])
}

func (h *Handler) handleOne(ctx context.Context, req *Request) (*Response, bool) {
	if err := h.codec.Validate(req); err != nil {
		if req.IsNotification() {
			return nil, false
		}
		return NewErrorResponse(req.ID, err), true
	}

	method := h.resolve(req.Method)
	if method == nil {
		if req.IsNotification() {
			return nil, false
		}
		return NewErrorResponse(req.ID, MethodNotFound(req.Method)), true
	}

	result, err := h.invoke(ctx, method, req.Params)
	if err != nil {
		if req.IsNotification() {
			return nil, false
		}
		return NewErrorResponse(req.ID, MapError(err)), true
	}

	if req.IsNotification() {
		return nil, false
	}

	return NewResponse(req.ID, result), true
}

func (h *Handler) resolve(name string) *contract.Method {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 || parts[0] != h.service.Name {
			return nil
		}
		return h.service.Method(parts[1])
	}
	return h.service.Method(name)
}

func (h *Handler) invoke(ctx context.Context, m *contract.Method, params []byte) (any, error) {
	var in any
	if m.HasInput() {
		in = m.NewInput()
		params = trimSpace(params)
		if len(params) > 0 && !isNull(params) {
			if len(params) > 0 && params[0] == '{' {
				if err := json.Unmarshal(params, in); err != nil {
					return nil, contract.NewError(contract.InvalidArgument, "invalid input: "+err.Error())
				}
			} else {
				return nil, contract.NewError(contract.InvalidArgument, "input must be a JSON object")
			}
		}
	} else {
		params = trimSpace(params)
		if len(params) > 0 && !isNull(params) {
			return nil, contract.NewError(contract.InvalidArgument, "method does not accept parameters")
		}
	}
	return m.Call(ctx, in)
}

func (h *Handler) writeError(w http.ResponseWriter, id json.RawMessage, err *Error) {
	w.Header().Set("Content-Type", h.codec.ContentType())
	_ = h.codec.EncodeError(w, id, err)
}

// MapError converts a Go error to a JSON-RPC error.
func MapError(err error) *Error {
	if err == nil {
		return nil
	}

	var rpcErr *Error
	if errors.As(err, &rpcErr) {
		return rpcErr
	}

	var ce *contract.Error
	if errors.As(err, &ce) {
		return &Error{
			Code:    contract.CodeToJSONRPC(ce.Code),
			Message: ce.Message,
			Data:    ce.Details,
		}
	}

	return InternalError(err)
}

// Mount registers a JSON-RPC handler at the given path.
func Mount(mux *http.ServeMux, path string, svc *contract.Service, opts ...Option) {
	if mux == nil || svc == nil {
		return
	}
	if path == "" {
		path = "/jsonrpc"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	mux.Handle(path, NewHandler(svc, opts...))
}
