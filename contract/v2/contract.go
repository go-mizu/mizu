// Package contract defines transport-neutral API contracts.
//
// This package contains pure data models and minimal runtime interfaces.
// All types are fully serializable to JSON/YAML.
//
// Design principles
//
//   - Transport-neutral (REST, JSON-RPC, SSE, WS, async, gRPC)
//   - SDK-first (OpenAI / Stripe style clients)
//   - Resource-oriented (client.responses.create())
//   - Streaming is explicit and first-class
//   - Minimal surface area
//   - No pointer syntax in schema
//   - Map keys are always string
//
// Runtime bindings live in transport-specific packages.
package contract

import (
	"context"
	"errors"
)

//
// ────────────────────────────────────────────────────────────
// Core service descriptor
// ────────────────────────────────────────────────────────────
//

// Service is a transport-neutral API descriptor.
type Service struct {
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Defaults    *Defaults  `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Resources   []*Resource `json:"resources" yaml:"resources"`
	Types       []*Type    `json:"types,omitempty" yaml:"types,omitempty"`
}

// Defaults are global hints for transports and SDK generators.
type Defaults struct {
	BaseURL string            `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Auth    string            `json:"auth,omitempty" yaml:"auth,omitempty"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

//
// ────────────────────────────────────────────────────────────
// Resources and methods
// ────────────────────────────────────────────────────────────
//

// Resource groups related methods into a namespace.
type Resource struct {
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Methods     []*Method `json:"methods" yaml:"methods"`
}

// Method describes a single operation.
type Method struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`

	// Unary input/output
	Input  TypeRef `json:"input,omitempty" yaml:"input,omitempty"`
	Output TypeRef `json:"output,omitempty" yaml:"output,omitempty"`

	// Streaming semantics (optional)
	Stream *struct {
		// Mode is a hint: "sse", "ws", "grpc", "async"
		Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

		// Item is the server -> client message type (required for streaming)
		Item TypeRef `json:"item" yaml:"item"`

		// Done is an optional terminal message type
		Done TypeRef `json:"done,omitempty" yaml:"done,omitempty"`

		// Error is an optional typed stream error
		Error TypeRef `json:"error,omitempty" yaml:"error,omitempty"`

		// InputItem enables bidirectional streams (WebSocket)
		InputItem TypeRef `json:"input_item,omitempty" yaml:"input_item,omitempty"`
	} `json:"stream,omitempty" yaml:"stream,omitempty"`

	// Optional HTTP binding
	HTTP *MethodHTTP `json:"http,omitempty" yaml:"http,omitempty"`
}

// MethodHTTP binds a method to an HTTP endpoint.
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
// If it matches a declared Type.Name, it refers to that schema.
// Otherwise it is treated as a primitive or external type
// (string, int64, bool, time.Time, json.RawMessage, etc).
type TypeRef string

// TypeKind describes the shape of a declared type.
type TypeKind string

const (
	KindStruct TypeKind = "struct"
	KindSlice  TypeKind = "slice"
	KindMap    TypeKind = "map"
	KindUnion  TypeKind = "union"
)

// Type is a schema definition.
type Type struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Kind        TypeKind `json:"kind" yaml:"kind"`

	// Struct only
	Fields []Field `json:"fields,omitempty" yaml:"fields,omitempty"`

	// Slice and map
	//   slice: elem is the element type
	//   map:   elem is the value type (key is always string)
	Elem TypeRef `json:"elem,omitempty" yaml:"elem,omitempty"`

	// Union only (discriminated union)
	//
	// Tag is the discriminator field name (for example "type").
	// Variants define the allowed shapes.
	Tag      string    `json:"tag,omitempty" yaml:"tag,omitempty"`
	Variants []Variant `json:"variants,omitempty" yaml:"variants,omitempty"`
}

// Variant is a single union alternative.
type Variant struct {
	// Value is the discriminator value (string literal).
	// Example: "text", "image", "response.output_text".
	Value string `json:"value" yaml:"value"`

	// Type is the referenced struct type.
	Type TypeRef `json:"type" yaml:"type"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Field describes a struct field.
type Field struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Type        TypeRef `json:"type" yaml:"type"`

	// Optional indicates the field may be omitted.
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	// Nullable indicates the field may be present with null value.
	Nullable bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`

	// Enum restricts allowed values (usually for string).
	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Const fixes the value to a literal (used for discriminators).
	Const string `json:"const,omitempty" yaml:"const,omitempty"`
}

//
// ────────────────────────────────────────────────────────────
// Descriptor helpers
// ────────────────────────────────────────────────────────────
//

// Method finds a method by resource and method name.
func (s *Service) Method(resource, method string) *Method {
	for _, r := range s.Resources {
		if r != nil && r.Name == resource {
			for _, m := range r.Methods {
				if m != nil && m.Name == method {
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

// Invoker is the unified runtime surface.
//
// Unary transports implement Call.
// Streaming transports implement Stream.
// Unsupported operations must return ErrUnsupported.
type Invoker interface {
	Descriptor

	// Unary
	Call(ctx context.Context, resource, method string, in any) (any, error)
	NewInput(resource, method string) (any, error)

	// Streaming
	Stream(ctx context.Context, resource, method string, in any) (Stream, error)
}

// Stream represents a live stream.
//
// SSE implements Recv only.
// WebSocket implements Recv + Send.
// Async brokers may implement both.
type Stream interface {
	Recv() (any, error)
	Send(any) error
	Close() error
}

// ErrUnsupported indicates a transport does not support an operation.
var ErrUnsupported = errors.New("contract: unsupported")
