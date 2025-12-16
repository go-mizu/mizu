// Package trpc implements a tRPC-like HTTP transport for contract services.
//
// tRPC provides a simple HTTP-based RPC protocol with typed response envelopes,
// similar to the TypeScript tRPC library but adapted for Go.
//
// Usage:
//
//	svc, _ := contract.Register("todo", &TodoService{})
//	trpc.Mount(mux, "/trpc", svc)
//
// Endpoint layout:
//   - POST /trpc/<procedure> - Call a procedure
//   - GET  /trpc.meta        - Introspection (methods + schemas)
//
// Response envelope:
//   - Success: {"result": {"data": <output>}}
//   - Error:   {"error": {"code": "...", "message": "..."}}
package trpc

import (
	"encoding/json"
)

// Envelope wraps all tRPC responses.
type Envelope struct {
	Result *Result `json:"result,omitempty"`
	Error  *Error  `json:"error,omitempty"`
}

// Result contains successful response data.
type Result struct {
	Data any `json:"data"`
}

// Error contains error information.
type Error struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Data    any    `json:"data,omitempty"`
}

// Error codes.
const (
	CodeBadRequest    = "BAD_REQUEST"
	CodeNotFound      = "NOT_FOUND"
	CodeInternalError = "INTERNAL_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
)

// SuccessEnvelope creates a success response.
func SuccessEnvelope(data any) Envelope {
	return Envelope{
		Result: &Result{Data: data},
	}
}

// ErrorEnvelope creates an error response.
func ErrorEnvelope(code, message string) Envelope {
	return Envelope{
		Error: &Error{Code: code, Message: message},
	}
}

// ErrorEnvelopeWithData creates an error response with additional data.
func ErrorEnvelopeWithData(code, message string, data any) Envelope {
	return Envelope{
		Error: &Error{Code: code, Message: message, Data: data},
	}
}

// ProcedureMeta contains metadata about a procedure.
type ProcedureMeta struct {
	Name     string   `json:"name"`
	FullName string   `json:"fullName"`
	Proc     string   `json:"proc"`
	Input    *TypeRef `json:"input,omitempty"`
	Output   *TypeRef `json:"output,omitempty"`
}

// TypeRef is a reference to a type.
type TypeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ServiceMeta contains service introspection data.
type ServiceMeta struct {
	Service string          `json:"service"`
	Methods []ProcedureMeta `json:"methods"`
	Schemas []Schema        `json:"schemas"`
}

// Schema holds a JSON schema for a type.
type Schema struct {
	ID   string         `json:"id"`
	JSON map[string]any `json:"json"`
}

// MarshalJSON implements json.Marshaler for Schema.
func (s Schema) MarshalJSON() ([]byte, error) {
	type alias Schema
	return json.Marshal(alias(s))
}
