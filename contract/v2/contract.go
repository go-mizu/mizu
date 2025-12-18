// Package contract defines transport-neutral service contracts.
//
// Design goals
//
//   - Service is a pure descriptor: fully serializable (JSON/YAML).
//   - The descriptor is transport-neutral. It does not assume REST, JSON-RPC, gRPC, etc.
//   - The descriptor is SDK-friendly. It is shaped to generate elegant client SDKs by grouping
//     operations into resources (for example client.posts.list(), client.models.retrieve()).
//   - Types are intentionally minimal in v1: struct, slice, map.
//     Primitives and external types (for example string, int64, bool, time.Time) are allowed
//     without explicit declaration.
//
// Notes
//
//   - No pointer syntax. Optionality and nullability are expressed on fields, not through "*T".
//   - Map keys are always string. This matches JSON object semantics and simplifies codegen.
//   - Arrays with fixed length are not modeled. Use slice instead.
//
// Runtime
//
// This file contains data models only. Runtime binding (reflection, registration, invocation)
// should live in separate files so the descriptor remains pure data.
package contract

import "context"

// Service is a transport-neutral API descriptor.
//
// Service is pure data and safe to serialize to JSON/YAML.
// It contains no reflection types, no runtime bindings, and no hidden lookup maps.
type Service struct {
	// Name is the service name, typically PascalCase (for example "OpenAI", "Petstore").
	Name string `json:"name" yaml:"name"`

	// Description is optional human documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Defaults describes optional shared settings used by transports and client generators.
	// These are hints. Transports may ignore them.
	Defaults *Defaults `json:"defaults,omitempty" yaml:"defaults,omitempty"`

	// Resources groups methods into resource namespaces for SDK generation.
	// Example:
	//   resources: [{ name: "models", methods: [{ name: "list" }, { name: "retrieve" }] }]
	Resources []*Resource `json:"resources" yaml:"resources"`

	// Types is the schema registry for inputs/outputs.
	// Primitives and external types do not need to appear here.
	Types []*Type `json:"types,omitempty" yaml:"types,omitempty"`
}

// Defaults contains global hints for transports and generated clients.
//
// This is optional and intentionally small.
// Keep it stable and transport-agnostic.
// If you need transport-specific knobs, add them inside the transport binding (for example MethodHTTP).
type Defaults struct {
	// BaseURL is an optional default API base URL (for example "https://api.openai.com").
	BaseURL string `json:"base_url,omitempty" yaml:"base_url,omitempty"`

	// Auth is an optional auth hint, commonly "bearer" or "basic".
	// Transports and generators may use this to shape constructors and headers.
	Auth string `json:"auth,omitempty" yaml:"auth,omitempty"`

	// Headers are optional default headers.
	// Map keys are always strings.
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Resource is a namespace that groups related methods.
//
// This enables elegant client SDK shapes such as:
//   client.responses.create(...)
//   client.models.list()
//   client.models.retrieve(...)
type Resource struct {
	// Name is the resource namespace, typically lowerCamel or lower_snake_case.
	// Examples: "responses", "models", "pets", "users".
	Name string `json:"name" yaml:"name"`

	// Description is optional human documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Methods are the operations on this resource.
	Methods []*Method `json:"methods" yaml:"methods"`
}

// Method describes a single callable operation.
//
// Method does not assume any particular transport.
// Optional transport bindings (for example HTTP) can be attached.
type Method struct {
	// Name is the method name within the resource.
	// For SDK-friendly design, use verbs like: "list", "create", "retrieve", "update", "delete".
	Name string `json:"name" yaml:"name"`

	// Description is optional human documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Input is the request type name. Empty means no input.
	Input TypeRef `json:"input,omitempty" yaml:"input,omitempty"`

	// Output is the response type name. Empty means no output (error-only in runtime terms).
	Output TypeRef `json:"output,omitempty" yaml:"output,omitempty"`

	// HTTP is an optional binding that maps this method to an HTTP endpoint.
	// If present, SDK generators can produce a concrete HTTP client.
	HTTP *MethodHTTP `json:"http,omitempty" yaml:"http,omitempty"`
}

// MethodHTTP is an optional HTTP binding.
//
// This is a small, practical subset sufficient for generating clients.
// It can be expanded later if needed (for example query/body rules, pagination hints, retries).
type MethodHTTP struct {
	// Method is the HTTP verb (GET, POST, PATCH, DELETE, etc).
	Method string `json:"method" yaml:"method"`

	// Path is the route path, optionally including template params like "{id}".
	// Example: "/v1/models/{id}".
	Path string `json:"path" yaml:"path"`
}

// TypeRef is a schema type reference.
//
// v1 rules:
//
//   - If TypeRef matches a declared type name in Service.Types, it refers to that type.
//   - Otherwise it is treated as a primitive or external type.
//     Examples: "string", "int", "int64", "bool", "float64", "time.Time".
//
// No pointer syntax is used. Optionality and nullability belong to fields.
type TypeRef string

// TypeKind is the shape kind of a declared type.
type TypeKind string

const (
	// KindStruct is a record/object type with named fields.
	KindStruct TypeKind = "struct"

	// KindSlice is a list/array type with a single element type.
	KindSlice TypeKind = "slice"

	// KindMap is a dictionary type with string keys and a single element (value) type.
	// Keys are always string.
	KindMap TypeKind = "map"
)

// Type is a schema definition in the registry.
//
// Types are stored as a list to preserve ordering and allow rich metadata.
// Use Name as the default reference key.
type Type struct {
	// Name is the type name used by TypeRef (for example "User", "CreateUserRequest").
	Name string `json:"name" yaml:"name"`

	// Description is optional human documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Kind determines which fields are valid.
	Kind TypeKind `json:"kind" yaml:"kind"`

	// Fields is only used when Kind is "struct".
	Fields []Field `json:"fields,omitempty" yaml:"fields,omitempty"`

	// Elem is used when Kind is "slice" or "map".
	//
	// For KindSlice:
	//   Elem is the element type (for example "User").
	//
	// For KindMap:
	//   Elem is the value type (for example "int").
	//   Keys are always string.
	Elem TypeRef `json:"elem,omitempty" yaml:"elem,omitempty"`
}

// Field describes a field of a struct type.
type Field struct {
	// Name is the wire name used in JSON/YAML payloads.
	// Recommended: lower_snake_case to match common REST conventions and OpenAPI schemas.
	Name string `json:"name" yaml:"name"`

	// Description is optional human documentation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Type is the field type.
	Type TypeRef `json:"type" yaml:"type"`

	// Optional indicates the field may be omitted.
	//
	// JSON Schema mapping:
	//   If Optional is false, the field belongs in the parent's "required" list.
	//   If Optional is true, it does not belong in "required".
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	// Nullable indicates the field may be present with a null value.
	//
	// JSON Schema mapping:
	//   type: [T, "null"] or anyOf(T, null)
	Nullable bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`
}

// Method finds a method by resource and method name.
// This is a convenience helper for descriptor consumers.
// It performs a linear scan and does not require hidden indexes on the descriptor.
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

// Descriptor exposes the transport-neutral descriptor.
type Descriptor interface {
	Descriptor() *Service
}

// Invoker is the callable surface for any runtime binding (reflection or generated).
//
// This is intentionally small so transports can adapt it consistently.
// Implementations typically validate input types and marshal/unmarshal as needed.
type Invoker interface {
	Descriptor

	// Call invokes a method by resource and method name.
	// If the method has no input, pass nil for in.
	Call(ctx context.Context, resource string, method string, in any) (any, error)

	// NewInput allocates a suitable input value for the method, or nil if no input.
	NewInput(resource string, method string) (any, error)
}
