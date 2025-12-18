// Package contract defines transport-neutral API contracts.
//
// This package contains pure data models only.
// All types are fully serializable to JSON/YAML.
//
// Design principles
//
//   - Transport-neutral: no REST / RPC / WS baked into the core model
//   - SDK-first: shaped to generate elegant client SDKs
//   - Resource-oriented: client.posts.list(), client.responses.create()
//   - Streaming is explicit and first-class
//   - No pointer syntax: optionality and nullability live on fields
//   - Map keys are always string
//
// Runtime bindings (reflection, HTTP, JSON-RPC, SSE, WS, async, etc.)
// live in separate packages.
package contract

import "context"

//
// ────────────────────────────────────────────────────────────
// Core service descriptor
// ────────────────────────────────────────────────────────────
//

// Service is a transport-neutral API descriptor.
type Service struct {
	// Name is the service name, for example "OpenAI", "Petstore".
	Name string `json:"name" yaml:"name"`

	// Description is optional human documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Defaults provides optional global hints for transports and SDK generators.
	Defaults *Defaults `json:"defaults,omitempty" yaml:"defaults,omitempty"`

	// Resources group methods into namespaces for SDK generation.
	Resources []*Resource `json:"resources" yaml:"resources"`

	// Types is the schema registry.
	// Primitives and external types (string, int64, time.Time) do not need entries.
	Types []*Type `json:"types,omitempty" yaml:"types,omitempty"`
}

// Defaults are global hints shared across transports and SDKs.
type Defaults struct {
	// BaseURL is the default API endpoint.
	BaseURL string `json:"base_url,omitempty" yaml:"base_url,omitempty"`

	// Auth is an auth hint such as "bearer", "basic", "none".
	Auth string `json:"auth,omitempty" yaml:"auth,omitempty"`

	// Headers are default headers to send with requests.
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

//
// ────────────────────────────────────────────────────────────
// Resources and methods
// ────────────────────────────────────────────────────────────
//

// Resource is a logical namespace for related methods.
//
// Example client usage:
//   client.models.list()
//   client.responses.create()
type Resource struct {
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Methods     []*Method `json:"methods" yaml:"methods"`
}

// Method describes a single operation.
type Method struct {
	// Name is the method name within the resource.
	// Examples: list, retrieve, create, update, delete.
	Name string `json:"name" yaml:"name"`

	// Description is optional documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Input is the request type.
	// Empty means no input.
	Input TypeRef `json:"input,omitempty" yaml:"input,omitempty"`

	// Output is the response type.
	// Empty means no output (error-only).
	Output TypeRef `json:"output,omitempty" yaml:"output,omitempty"`

	// Stream describes streaming semantics, if this method is streaming.
	Stream *MethodStream `json:"stream,omitempty" yaml:"stream,omitempty"`

	// HTTP is an optional HTTP binding (REST, SSE, WS entrypoint).
	HTTP *MethodHTTP `json:"http,omitempty" yaml:"http,omitempty"`
}

//
// ────────────────────────────────────────────────────────────
// Streaming model
// ────────────────────────────────────────────────────────────
//

// MethodStream defines streaming semantics for a method.
//
// This is transport-neutral. SSE, WebSocket, gRPC, and async brokers
// are all carriers for the same logical stream.
type MethodStream struct {
	// Mode is a hint for SDKs and docs: "sse", "ws", "grpc", "async".
	// Optional. Semantics are defined by Item/InputItem.
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Item is the type emitted by the server.
	// Required for streaming methods.
	Item TypeRef `json:"item" yaml:"item"`

	// Done is an optional terminal message type.
	// If empty, end-of-stream is implied by connection close.
	Done TypeRef `json:"done,omitempty" yaml:"done,omitempty"`

	// Error is an optional typed error message for streams.
	Error TypeRef `json:"error,omitempty" yaml:"error,omitempty"`

	// InputItem enables bidirectional streaming (WebSocket).
	// If set, the client may send messages after connect.
	InputItem TypeRef `json:"input_item,omitempty" yaml:"input_item,omitempty"`
}

//
// ────────────────────────────────────────────────────────────
// Transport bindings
// ────────────────────────────────────────────────────────────
//

// MethodHTTP binds a method to an HTTP endpoint.
//
// This is intentionally minimal and extensible.
type MethodHTTP struct {
	Method string `json:"method" yaml:"method"`
	Path   string `json:"path" yaml:"path"`
}

//
// ────────────────────────────────────────────────────────────
// Type system
// ────────────────────────────────────────────────────────────
//

// TypeRef references a type by name.
//
// If the name matches a declared Type, it refers to that schema.
// Otherwise it is treated as a primitive or external type.
type TypeRef string

// TypeKind describes the shape of a declared type.
type TypeKind string

const (
	KindStruct TypeKind = "struct"
	KindSlice  TypeKind = "slice"
	KindMap    TypeKind = "map"
)

// Type is a schema definition.
type Type struct {
	// Name is the canonical type name.
	Name string `json:"name" yaml:"name"`

	// Description is optional documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Kind determines which fields are valid.
	Kind TypeKind `json:"kind" yaml:"kind"`

	// Fields are used when Kind is struct.
	Fields []Field `json:"fields,omitempty" yaml:"fields,omitempty"`

	// Elem is used for slice and map.
	//
	//   slice: elem is the element type
	//   map:   elem is the value type (key is always string)
	Elem TypeRef `json:"elem,omitempty" yaml:"elem,omitempty"`
}

// Field describes a struct field.
type Field struct {
	// Name is the wire name (JSON/YAML).
	Name string `json:"name" yaml:"name"`

	// Description is optional documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Type is the field type.
	Type TypeRef `json:"type" yaml:"type"`

	// Optional indicates the field may be omitted.
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	// Nullable indicates the field may be present with null value.
	Nullable bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`
}

//
// ────────────────────────────────────────────────────────────
// Descriptor helpers
// ────────────────────────────────────────────────────────────
//

// Method finds a method by resource and method name.
func (s *Service) Method(resourceName, methodName string) *Method {
	for _, r := range s.Resources {
		if r != nil && r.Name == resourceName {
			for _, m := range r.Methods {
				if m != nil && m.Name == methodName {
					return m
				}
			}
			return nil
		}
	}
	return nil
}

//
// ────────────────────────────────────────────────────────────
// Runtime interfaces (transport-agnostic)
// ────────────────────────────────────────────────────────────
//

// Descriptor exposes the service descriptor.
type Descriptor interface {
	Descriptor() *Service
}

// Invoker is the unary-call runtime surface.
type Invoker interface {
	Descriptor

	Call(ctx context.Context, resource, method string, in any) (any, error)
	NewInput(resource, method string) (any, error)
}

// StreamInvoker is the streaming runtime surface.
type StreamInvoker interface {
	Descriptor

	// Stream starts a stream and returns a channel-like iterator.
	Stream(ctx context.Context, resource, method string, in any) (Stream, error)
}

// Stream is a generic streaming interface.
type Stream interface {
	// Recv blocks until the next item, or returns error on end/failure.
	Recv() (any, error)

	// Send sends a client-to-server message (bidirectional streams).
	// Returns ErrUnsupported if not allowed.
	Send(any) error

	// Close terminates the stream.
	Close() error
}
