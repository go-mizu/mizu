// contract/transport/jsonrpc/server.go
package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Server exposes a contract.Invoker via JSON-RPC 2.0 over HTTP.
//
// Method naming convention:
//   - JSON-RPC method name is "<resource>.<method>"
//     Example: "models.list", "pets.retrieve", "responses.create"
//
// Params convention:
//   - params is an object and is decoded into the input struct (by JSON field names)
//   - params omitted or null for methods without input
//   - array params are rejected (keep the contract DX simple)
//
// Output convention:
//   - result is the method output (marshaled as JSON)
//   - if method has no output, result is omitted and the response is "null" result
//
// Notifications:
//   - if id is missing or null, the call is treated as a notification (no response written)
type Server struct {
	inv contract.Invoker
	svc *contract.Service
}

// NewServer constructs a JSON-RPC server for the given invoker.
func NewServer(inv contract.Invoker) (*Server, error) {
	if inv == nil {
		return nil, errors.New("jsonrpc: nil invoker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("jsonrpc: nil descriptor")
	}
	return &Server{inv: inv, svc: svc}, nil
}

// Handler returns an http.Handler for JSON-RPC requests.
func (s *Server) Handler() http.Handler { return http.HandlerFunc(s.handleHTTP) }

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)

	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		s.writeRPC(w, rpcErrorResponse(nil, errParse, "parse error", err.Error()))
		return
	}

	// Batch if raw starts with '['
	rawTrim := strings.TrimSpace(string(raw))
	if rawTrim == "" {
		s.writeRPC(w, rpcErrorResponse(nil, errInvalidRequest, "invalid request", "empty body"))
		return
	}

	if rawTrim[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(raw, &batch); err != nil {
			s.writeRPC(w, rpcErrorResponse(nil, errParse, "parse error", err.Error()))
			return
		}
		if len(batch) == 0 {
			s.writeRPC(w, rpcErrorResponse(nil, errInvalidRequest, "invalid request", "empty batch"))
			return
		}

		var responses []any
		for _, item := range batch {
			resp, hasResp := s.handleOne(r.Context(), item)
			if hasResp {
				responses = append(responses, resp)
			}
		}

		// If all were notifications, no response.
		if len(responses) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		writeJSON(w, http.StatusOK, responses)
		return
	}

	resp, hasResp := s.handleOne(r.Context(), raw)
	if !hasResp {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.writeRPC(w, resp)
}

func (s *Server) handleOne(ctx context.Context, raw json.RawMessage) (resp any, hasResp bool) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		// No id available for malformed JSON.
		return rpcErrorResponse(nil, errInvalidRequest, "invalid request", err.Error()), true
	}
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		return rpcErrorResponse(req.ID, errInvalidRequest, "invalid request", "missing jsonrpc=2.0 or method"), req.hasID()
	}

	resName, mName, ok := splitMethod(req.Method)
	if !ok {
		return rpcErrorResponse(req.ID, errMethodNotFound, "method not found", "method must be <resource>.<method>"), req.hasID()
	}

	// Allocate input if needed.
	var in any
	methodDesc := s.svc.Method(resName, mName)
	if methodDesc == nil {
		return rpcErrorResponse(req.ID, errMethodNotFound, "method not found", "unknown method"), req.hasID()
	}
	if methodDesc.Input != "" {
		var err error
		in, err = s.inv.NewInput(resName, mName)
		if err != nil || in == nil {
			return rpcErrorResponse(req.ID, errInternal, "internal error", "failed to allocate input"), req.hasID()
		}
	}

	// Decode params into input.
	if in != nil && len(req.Params) != 0 && string(req.Params) != "null" {
		// Reject array params to keep DX predictable.
		if isJSONArray(req.Params) {
			return rpcErrorResponse(req.ID, errInvalidParams, "invalid params", "array params are not supported"), req.hasID()
		}
		if err := json.Unmarshal(req.Params, in); err != nil {
			return rpcErrorResponse(req.ID, errInvalidParams, "invalid params", err.Error()), req.hasID()
		}
	}

	out, err := s.inv.Call(ctx, resName, mName, in)
	if err != nil {
		return rpcErrorResponse(req.ID, errServer, "server error", err.Error()), req.hasID()
	}

	// Notification: no response if id is absent or null.
	if !req.hasID() {
		return nil, false
	}

	// If no output type, result is null.
	if methodDesc.Output == "" {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage("null")}, true
	}

	b, mErr := json.Marshal(out)
	if mErr != nil {
		return rpcErrorResponse(req.ID, errInternal, "internal error", mErr.Error()), true
	}
	return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(b)}, true
}

func (s *Server) writeRPC(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusOK, v)
}

func splitMethod(full string) (resource string, method string, ok bool) {
	full = strings.TrimSpace(full)
	i := strings.IndexByte(full, '.')
	if i <= 0 || i == len(full)-1 {
		return "", "", false
	}
	return full[:i], full[i+1:], true
}

func isJSONArray(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return len(s) > 0 && s[0] == '['
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeHTTPError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

// ---- JSON-RPC 2.0 wire types ----

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

func (r rpcRequest) hasID() bool {
	if r.ID == nil {
		return false
	}
	// JSON-RPC: id may be null => notification-like (no response)
	if s, ok := r.ID.(string); ok && s == "" {
		return true
	}
	return r.ID != nil
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func rpcErrorResponse(id any, code int, message string, data any) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
}

// Standard JSON-RPC codes and one generic server range.
const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603

	// Generic server error range is -32000..-32099.
	errServer = -32000
)

func (s *Server) String() string { return fmt.Sprintf("jsonrpc.Server(%s)", s.svc.Name) }
