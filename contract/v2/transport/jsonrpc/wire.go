package jsonrpc

import "encoding/json"

// Standard JSON-RPC 2.0 error codes.
const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603

	// Server error range: -32000 to -32099
	errServer = -32000
)

// request is the JSON-RPC 2.0 request structure.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// hasID returns true if the request has an id (not a notification).
func (r *request) hasID() bool {
	return r.ID != nil
}

// response is the JSON-RPC 2.0 response structure.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError is the JSON-RPC 2.0 error structure.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// errorResponse creates an error response.
func errorResponse(id any, code int, message string, data any) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
}

// successResponse creates a success response.
func successResponse(id any, result json.RawMessage) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}
