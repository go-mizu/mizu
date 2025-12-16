// Package contract provides transport abstractions for protocol-neutral service contracts.
//
// Transport layers handle the encoding and decoding of requests and responses
// over different protocols (HTTP, JSON-RPC, etc.) while the core contract
// remains unchanged.
package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Transport represents a protocol-agnostic transport layer.
type Transport interface {
	// Name returns the transport identifier (e.g., "jsonrpc", "rest", "mcp").
	Name() string
}

// Handler is an http.Handler that serves a transport.
type Handler interface {
	http.Handler
	Transport
}

// Codec handles request/response encoding for a transport.
type Codec interface {
	// ContentType returns the MIME type for this codec.
	ContentType() string

	// DecodeRequest decodes a request from the reader.
	DecodeRequest(r io.Reader) (*TransportRequest, error)

	// EncodeResponse writes a response to the writer.
	EncodeResponse(w io.Writer, resp *TransportResponse) error

	// EncodeError writes an error response to the writer.
	EncodeError(w io.Writer, err error) error
}

// TransportRequest represents a decoded transport request.
type TransportRequest struct {
	// ID is the request identifier (for request-response correlation).
	ID any

	// Method is the service method name.
	Method string

	// Params contains the raw input parameters.
	Params []byte

	// Metadata contains transport-specific metadata.
	Metadata map[string]any

	// IsNotification indicates the request expects no response.
	IsNotification bool
}

// TransportResponse represents a transport response.
type TransportResponse struct {
	// ID matches the request ID.
	ID any

	// Result contains the method output (nil for void methods).
	Result any

	// Error contains any error that occurred.
	Error error
}

// Resolver finds methods from a service.
type Resolver interface {
	// Resolve returns the method for the given name.
	// Name may be "Method" or "Service.Method".
	Resolve(name string) *Method
}

// ServiceResolver is the default resolver implementation.
type ServiceResolver struct {
	Service *Service
}

// Resolve finds a method by name in the service.
// Supports both "Method" and "Service.Method" formats.
func (r *ServiceResolver) Resolve(name string) *Method {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	// Support "service.Method" and "Method"
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 || parts[0] != r.Service.Name {
			return nil
		}
		return r.Service.Method(parts[1])
	}

	return r.Service.Method(name)
}

// TransportInvoker invokes methods with transport context.
type TransportInvoker interface {
	// Invoke calls a method with the given raw JSON input.
	Invoke(ctx context.Context, method *Method, input []byte) (any, error)
}

// DefaultInvoker is the standard invoker implementation.
type DefaultInvoker struct{}

// Invoke unmarshals the input and calls the method.
func (d *DefaultInvoker) Invoke(ctx context.Context, m *Method, input []byte) (any, error) {
	var in any
	if m.HasInput() {
		in = m.NewInput()
		input = TrimJSONSpace(input)
		if len(input) > 0 && !IsJSONNull(input) {
			if len(input) > 0 && input[0] == '{' {
				if err := json.Unmarshal(input, in); err != nil {
					return nil, &Error{
						Code:    ErrCodeInvalidArgument,
						Message: "invalid input: " + err.Error(),
					}
				}
			} else {
				return nil, &Error{
					Code:    ErrCodeInvalidArgument,
					Message: "input must be a JSON object",
				}
			}
		}
	} else {
		input = TrimJSONSpace(input)
		if len(input) > 0 && !IsJSONNull(input) {
			return nil, &Error{
				Code:    ErrCodeInvalidArgument,
				Message: "method does not accept parameters",
			}
		}
	}
	return m.Invoker.Call(ctx, in)
}

// TransportError represents a transport-level error.
type TransportError struct {
	// Code is the transport-specific error code.
	Code int

	// Message is the error message.
	Message string

	// Data contains additional error context.
	Data any

	// Cause is the underlying error.
	Cause error
}

// Error implements the error interface.
func (e *TransportError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("transport error: code=%d", e.Code)
}

// Unwrap returns the underlying cause.
func (e *TransportError) Unwrap() error {
	return e.Cause
}

// TransportOptions contains transport configuration.
type TransportOptions struct {
	// Resolver finds methods by name.
	Resolver Resolver

	// Invoker calls methods.
	Invoker TransportInvoker

	// Middleware wraps method invocations.
	Middleware []MethodMiddleware

	// ErrorMapper maps contract errors to transport errors.
	ErrorMapper func(error) *TransportError
}

// TransportOption configures transport options.
type TransportOption func(*TransportOptions)

// WithResolver sets a custom resolver.
func WithResolver(r Resolver) TransportOption {
	return func(o *TransportOptions) { o.Resolver = r }
}

// WithTransportInvoker sets a custom invoker.
func WithTransportInvoker(i TransportInvoker) TransportOption {
	return func(o *TransportOptions) { o.Invoker = i }
}

// WithTransportMiddleware adds middleware.
func WithTransportMiddleware(mw ...MethodMiddleware) TransportOption {
	return func(o *TransportOptions) { o.Middleware = append(o.Middleware, mw...) }
}

// WithErrorMapper sets an error mapper.
func WithErrorMapper(m func(error) *TransportError) TransportOption {
	return func(o *TransportOptions) { o.ErrorMapper = m }
}

// ApplyTransportOptions applies options and returns configured options.
func ApplyTransportOptions(svc *Service, opts ...TransportOption) *TransportOptions {
	o := &TransportOptions{
		Resolver: &ServiceResolver{Service: svc},
		Invoker:  &DefaultInvoker{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// TrimJSONSpace removes leading/trailing whitespace from JSON bytes.
// This is an allocation-free operation.
func TrimJSONSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && isJSONWhitespace(b[i]) {
		i++
	}
	for j > i && isJSONWhitespace(b[j-1]) {
		j--
	}
	return b[i:j]
}

func isJSONWhitespace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

// IsJSONNull checks if bytes represent JSON null (case-insensitive).
func IsJSONNull(b []byte) bool {
	b = TrimJSONSpace(b)
	if len(b) != 4 {
		return false
	}
	return (b[0] == 'n' || b[0] == 'N') &&
		(b[1] == 'u' || b[1] == 'U') &&
		(b[2] == 'l' || b[2] == 'L') &&
		(b[3] == 'l' || b[3] == 'L')
}

// SafeErrorString returns a safe string representation of an error.
func SafeErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
