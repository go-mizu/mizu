package contract

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// MountJSONRPC mounts a JSON-RPC 2.0 endpoint at path.
//
// It serves POST requests only.
// It supports single and batch requests.
// Notifications (id is missing or null) return no response body (204).
//
// Method resolution rules:
//   - If req.method is "Service.Method", Service must match svc.Name and Method must exist.
//   - If req.method is "Method", Method must exist.
//   - Otherwise: method not found.
func MountJSONRPC(mux *http.ServeMux, path string, svc *Service) {
	if mux == nil || svc == nil {
		return
	}
	if path == "" {
		path = "/jsonrpc"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var raw json.RawMessage
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&raw); err != nil {
			writeJSONRPCError(w, nil, jsonrpcParseError(err))
			return
		}

		// Determine if batch or single.
		raw = trimSpaceRaw(raw)
		if len(raw) == 0 {
			writeJSONRPCError(w, nil, jsonrpcInvalidRequest(errors.New("empty body")))
			return
		}

		if raw[0] == '[' {
			var batch []json.RawMessage
			if err := json.Unmarshal(raw, &batch); err != nil {
				writeJSONRPCError(w, nil, jsonrpcParseError(err))
				return
			}
			if len(batch) == 0 {
				writeJSONRPCError(w, nil, jsonrpcInvalidRequest(errors.New("empty batch")))
				return
			}

			var replies []jsonrpcResponse
			for _, item := range batch {
				resp, ok := handleJSONRPCOne(r.Context(), svc, item)
				if ok {
					replies = append(replies, resp)
				}
			}

			if len(replies) == 0 {
				// Batch of notifications only.
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(replies)
			return
		}

		resp, ok := handleJSONRPCOne(r.Context(), svc, raw)
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *jsonrpcError  `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func handleJSONRPCOne(ctx context.Context, svc *Service, raw json.RawMessage) (jsonrpcResponse, bool) {
	var req jsonrpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return jsonrpcResponseFromErr(nil, jsonrpcParseError(err)), true
	}

	// Invalid request checks.
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		return jsonrpcResponseFromErr(req.ID, jsonrpcInvalidRequest(errors.New("missing jsonrpc=2.0 or method"))), isNotNotification(req.ID)
	}

	m := resolveJSONRPCMethod(svc, req.Method)
	if m == nil {
		return jsonrpcResponseFromErr(req.ID, jsonrpcMethodNotFound(req.Method)), isNotNotification(req.ID)
	}

	var in any
	if m.Input != nil {
		in = m.NewInput()

		// If params is missing or null, treat as empty object for struct inputs.
		p := trimSpaceRaw(req.Params)
		if len(p) == 0 || isJSONNull(p) {
			// Keep zero value input.
		} else if len(p) > 0 && p[0] == '{' {
			if err := json.Unmarshal(p, in); err != nil {
				return jsonrpcResponseFromErr(req.ID, jsonrpcInvalidParams(err)), isNotNotification(req.ID)
			}
		} else {
			// We only support object params in v1 (named params).
			return jsonrpcResponseFromErr(req.ID, jsonrpcInvalidParams(errors.New("params must be an object"))), isNotNotification(req.ID)
		}
	} else {
		// Method has no input. Reject non-empty params that are not null.
		p := trimSpaceRaw(req.Params)
		if len(p) != 0 && !isJSONNull(p) {
			return jsonrpcResponseFromErr(req.ID, jsonrpcInvalidParams(errors.New("method does not accept params"))), isNotNotification(req.ID)
		}
	}

	out, err := m.Invoker.Call(ctx, in)
	if err != nil {
		return jsonrpcResponseFromErr(req.ID, jsonrpcInternalError(err)), isNotNotification(req.ID)
	}

	// Notification: no response.
	if !isNotNotification(req.ID) {
		return jsonrpcResponse{}, false
	}

	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  out,
	}, true
}

func resolveJSONRPCMethod(svc *Service, name string) *Method {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	// Support "service.Method" and "Method".
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			return nil
		}
		if parts[0] != svc.Name {
			return nil
		}
		return svc.Method(parts[1])
	}

	return svc.Method(name)
}

func isNotNotification(id json.RawMessage) bool {
	id = trimSpaceRaw(id)
	if len(id) == 0 {
		return false
	}
	return !isJSONNull(id)
}

func writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, e *jsonrpcError) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   e,
	})
}

func jsonrpcResponseFromErr(id json.RawMessage, e *jsonrpcError) jsonrpcResponse {
	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   e,
	}
}

// JSON-RPC standard errors.
func jsonrpcParseError(err error) *jsonrpcError {
	return &jsonrpcError{
		Code:    -32700,
		Message: "Parse error",
		Data:    safeErr(err),
	}
}

func jsonrpcInvalidRequest(err error) *jsonrpcError {
	return &jsonrpcError{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    safeErr(err),
	}
}

func jsonrpcMethodNotFound(method string) *jsonrpcError {
	return &jsonrpcError{
		Code:    -32601,
		Message: "Method not found",
		Data: map[string]any{
			"method": method,
		},
	}
}

func jsonrpcInvalidParams(err error) *jsonrpcError {
	return &jsonrpcError{
		Code:    -32602,
		Message: "Invalid params",
		Data:    safeErr(err),
	}
}

func jsonrpcInternalError(err error) *jsonrpcError {
	return &jsonrpcError{
		Code:    -32603,
		Message: "Internal error",
		Data:    safeErr(err),
	}
}

// Aliases to shared helpers for local use
func trimSpaceRaw(b []byte) []byte {
	return TrimJSONSpace(b)
}

func isJSONNull(b []byte) bool {
	return IsJSONNull(b)
}

func safeErr(err error) any {
	if err == nil {
		return nil
	}
	return SafeErrorString(err)
}
