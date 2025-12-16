// Package jsonrpc implements JSON-RPC 2.0 transport for contract services.
//
// JSON-RPC 2.0 specification: https://www.jsonrpc.org/specification
//
// This package provides a complete implementation including:
//   - Single and batch requests
//   - Notifications (no response expected)
//   - Standard error codes
//   - Method resolution with service prefix support
//
// Usage:
//
//	svc, _ := contract.Register("todo", &TodoService{})
//	jsonrpc.Mount(mux, "/rpc", svc)
package jsonrpc

import (
	"encoding/json"
	"io"
)

// Version is the JSON-RPC protocol version.
const Version = "2.0"

// Standard error codes per JSON-RPC 2.0 specification.
const (
	// Parse error: Invalid JSON was received by the server.
	CodeParseError = -32700

	// Invalid Request: The JSON sent is not a valid Request object.
	CodeInvalidRequest = -32600

	// Method not found: The method does not exist / is not available.
	CodeMethodNotFound = -32601

	// Invalid params: Invalid method parameter(s).
	CodeInvalidParams = -32602

	// Internal error: Internal JSON-RPC error.
	CodeInternalError = -32603

	// Server error range: -32000 to -32099 (reserved for implementation-defined server-errors).
	CodeServerErrorStart = -32099
	CodeServerErrorEnd   = -32000
)

// Request represents a JSON-RPC 2.0 request object.
type Request struct {
	// JSONRPC specifies the version of the JSON-RPC protocol.
	// MUST be exactly "2.0".
	JSONRPC string `json:"jsonrpc"`

	// ID is an identifier established by the Client.
	// If omitted or null, this is a notification.
	ID json.RawMessage `json:"id,omitempty"`

	// Method is the name of the method to be invoked.
	Method string `json:"method"`

	// Params are the parameter values to be used during invocation.
	Params json.RawMessage `json:"params,omitempty"`
}

// IsNotification returns true if this request is a notification.
// A notification has no id field (or null id).
func (r *Request) IsNotification() bool {
	id := trimSpace(r.ID)
	if len(id) == 0 {
		return true
	}
	return isNull(id)
}

// Response represents a JSON-RPC 2.0 response object.
type Response struct {
	// JSONRPC specifies the version of the JSON-RPC protocol.
	JSONRPC string `json:"jsonrpc"`

	// ID is the identifier of the request this response is for.
	ID json.RawMessage `json:"id,omitempty"`

	// Result is the result of the method invocation.
	// MUST NOT exist if there was an error.
	Result any `json:"result,omitempty"`

	// Error is present when there was an error.
	// MUST NOT exist if the method was successful.
	Error *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	// Code indicates the error type that occurred.
	Code int `json:"code"`

	// Message provides a short description of the error.
	Message string `json:"message"`

	// Data contains additional information about the error.
	Data any `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return errorCodeMessage(e.Code)
}

// NewError creates a new JSON-RPC error.
func NewError(code int, message string) *Error {
	if message == "" {
		message = errorCodeMessage(code)
	}
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WithData adds data to the error.
func (e *Error) WithData(data any) *Error {
	e.Data = data
	return e
}

// ParseError creates a parse error (-32700).
func ParseError(cause error) *Error {
	e := NewError(CodeParseError, "Parse error")
	if cause != nil {
		e.Data = cause.Error()
	}
	return e
}

// InvalidRequest creates an invalid request error (-32600).
func InvalidRequest(cause error) *Error {
	e := NewError(CodeInvalidRequest, "Invalid Request")
	if cause != nil {
		e.Data = cause.Error()
	}
	return e
}

// MethodNotFound creates a method not found error (-32601).
func MethodNotFound(method string) *Error {
	return NewError(CodeMethodNotFound, "Method not found").WithData(map[string]any{
		"method": method,
	})
}

// InvalidParams creates an invalid params error (-32602).
func InvalidParams(cause error) *Error {
	e := NewError(CodeInvalidParams, "Invalid params")
	if cause != nil {
		e.Data = cause.Error()
	}
	return e
}

// InternalError creates an internal error (-32603).
func InternalError(cause error) *Error {
	e := NewError(CodeInternalError, "Internal error")
	if cause != nil {
		e.Data = cause.Error()
	}
	return e
}

// NewResponse creates a success response.
func NewResponse(id json.RawMessage, result any) *Response {
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse creates an error response.
func NewErrorResponse(id json.RawMessage, err *Error) *Response {
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Error:   err,
	}
}

// Codec implements JSON-RPC 2.0 encoding and decoding.
type Codec struct{}

// NewCodec creates a new JSON-RPC codec.
func NewCodec() *Codec {
	return &Codec{}
}

// ContentType returns the MIME type for JSON-RPC.
func (c *Codec) ContentType() string {
	return "application/json"
}

// Decode reads and decodes a JSON-RPC request or batch.
// Returns a slice with one element for single requests.
func (c *Codec) Decode(r io.Reader) ([]*Request, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, ParseError(err)
	}

	raw = trimSpace(raw)
	if len(raw) == 0 {
		return nil, InvalidRequest(nil)
	}

	// Check if batch (starts with '[')
	if raw[0] == '[' {
		var batch []*Request
		if err := json.Unmarshal(raw, &batch); err != nil {
			return nil, ParseError(err)
		}
		if len(batch) == 0 {
			return nil, InvalidRequest(nil)
		}
		return batch, nil
	}

	// Single request
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, ParseError(err)
	}

	return []*Request{&req}, nil
}

// DecodeOne reads and decodes a single JSON-RPC request.
func (c *Codec) DecodeOne(r io.Reader) (*Request, error) {
	var req Request
	dec := json.NewDecoder(r)
	if err := dec.Decode(&req); err != nil {
		return nil, ParseError(err)
	}
	return &req, nil
}

// Encode writes a JSON-RPC response.
func (c *Codec) Encode(w io.Writer, resp *Response) error {
	return json.NewEncoder(w).Encode(resp)
}

// EncodeBatch writes a batch of JSON-RPC responses.
func (c *Codec) EncodeBatch(w io.Writer, responses []*Response) error {
	return json.NewEncoder(w).Encode(responses)
}

// EncodeError writes a JSON-RPC error response.
func (c *Codec) EncodeError(w io.Writer, id json.RawMessage, err *Error) error {
	return c.Encode(w, NewErrorResponse(id, err))
}

// Validate checks if a request is valid per JSON-RPC 2.0.
func (c *Codec) Validate(req *Request) *Error {
	if req.JSONRPC != Version {
		return InvalidRequest(nil)
	}
	if trimSpace([]byte(req.Method)) == nil || len(req.Method) == 0 {
		return InvalidRequest(nil)
	}
	return nil
}

// errorCodeMessage returns the standard message for an error code.
func errorCodeMessage(code int) string {
	switch code {
	case CodeParseError:
		return "Parse error"
	case CodeInvalidRequest:
		return "Invalid Request"
	case CodeMethodNotFound:
		return "Method not found"
	case CodeInvalidParams:
		return "Invalid params"
	case CodeInternalError:
		return "Internal error"
	default:
		if code >= CodeServerErrorStart && code <= CodeServerErrorEnd {
			return "Server error"
		}
		return "Unknown error"
	}
}

// Helper functions for JSON handling

func trimSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && isSpace(b[i]) {
		i++
	}
	for j > i && isSpace(b[j-1]) {
		j--
	}
	return b[i:j]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

func isNull(b []byte) bool {
	b = trimSpace(b)
	if len(b) != 4 {
		return false
	}
	return (b[0] == 'n' || b[0] == 'N') &&
		(b[1] == 'u' || b[1] == 'U') &&
		(b[2] == 'l' || b[2] == 'L') &&
		(b[3] == 'l' || b[3] == 'L')
}
