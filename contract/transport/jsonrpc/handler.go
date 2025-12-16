package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Handler handles JSON-RPC 2.0 requests over HTTP.
type Handler struct {
	service  *contract.Service
	resolver contract.Resolver
	invoker  contract.TransportInvoker
	codec    *Codec
}

// Option configures the handler.
type Option func(*Handler)

// WithResolver sets a custom method resolver.
func WithResolver(r contract.Resolver) Option {
	return func(h *Handler) { h.resolver = r }
}

// WithInvoker sets a custom method invoker.
func WithInvoker(i contract.TransportInvoker) Option {
	return func(h *Handler) { h.invoker = i }
}

// NewHandler creates a new JSON-RPC handler for the service.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service:  svc,
		codec:    NewCodec(),
		resolver: &contract.ServiceResolver{Service: svc},
		invoker:  &contract.DefaultInvoker{},
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

	// All notifications: no response
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

// handleOne processes a single JSON-RPC request.
// Returns the response and whether to include it in output.
func (h *Handler) handleOne(ctx context.Context, req *Request) (*Response, bool) {
	// Validate request
	if err := h.codec.Validate(req); err != nil {
		if req.IsNotification() {
			return nil, false
		}
		return NewErrorResponse(req.ID, err), true
	}

	// Resolve method
	method := h.resolver.Resolve(req.Method)
	if method == nil {
		if req.IsNotification() {
			return nil, false
		}
		return NewErrorResponse(req.ID, MethodNotFound(req.Method)), true
	}

	// Invoke method
	result, err := h.invoker.Invoke(ctx, method, req.Params)
	if err != nil {
		if req.IsNotification() {
			return nil, false
		}
		return NewErrorResponse(req.ID, MapError(err)), true
	}

	// Notification: no response
	if req.IsNotification() {
		return nil, false
	}

	return NewResponse(req.ID, result), true
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, id json.RawMessage, err *Error) {
	w.Header().Set("Content-Type", h.codec.ContentType())
	_ = h.codec.EncodeError(w, id, err)
}

// MapError converts a Go error to a JSON-RPC error.
func MapError(err error) *Error {
	if err == nil {
		return nil
	}

	// Check for JSON-RPC error
	var rpcErr *Error
	if errors.As(err, &rpcErr) {
		return rpcErr
	}

	// Check for contract error
	var ce *contract.Error
	if errors.As(err, &ce) {
		return &Error{
			Code:    mapErrorCode(ce.Code),
			Message: ce.Message,
			Data:    ce.Details,
		}
	}

	// Generic error
	return InternalError(err)
}

// mapErrorCode maps contract error codes to JSON-RPC codes.
func mapErrorCode(code contract.ErrorCode) int {
	switch code {
	case contract.ErrCodeInvalidArgument:
		return CodeInvalidParams
	case contract.ErrCodeNotFound:
		return CodeMethodNotFound
	case contract.ErrCodeUnimplemented:
		return CodeMethodNotFound
	case contract.ErrCodeInternal:
		return CodeInternalError
	case contract.ErrCodeCanceled:
		return -32001
	case contract.ErrCodeDeadlineExceeded:
		return -32002
	case contract.ErrCodeAlreadyExists:
		return -32003
	case contract.ErrCodePermissionDenied:
		return -32004
	case contract.ErrCodeResourceExhausted:
		return -32005
	case contract.ErrCodeFailedPrecondition:
		return -32006
	case contract.ErrCodeAborted:
		return -32007
	case contract.ErrCodeOutOfRange:
		return -32008
	case contract.ErrCodeUnavailable:
		return -32009
	case contract.ErrCodeDataLoss:
		return -32010
	case contract.ErrCodeUnauthenticated:
		return -32011
	default:
		return CodeInternalError
	}
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
