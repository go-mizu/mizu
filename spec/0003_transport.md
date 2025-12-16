# Transport Layer Refactoring Specification

## Overview

This document specifies the refactoring of the contract package's transport layer into a modular, well-structured design following Go standard library conventions.

## Goals

1. **Separation of Concerns**: Transport implementations isolated in dedicated packages
2. **Shared Abstractions**: Common interfaces and types in `contract/transport.go`
3. **Go Idioms**: Follow patterns from `net/http`, `encoding/json`, and other stdlib packages
4. **Testability**: Each transport independently testable
5. **Extensibility**: Easy to add new transports (gRPC, WebSocket, etc.)

## Architecture

### Package Structure

```
contract/
├── contract.go           # Core: Service, Method, Register
├── schema.go             # TypeRegistry, schema generation
├── errors.go             # Error types and codes
├── transport.go          # Transport interfaces and shared types
├── transport/
│   ├── jsonrpc/
│   │   ├── jsonrpc.go    # JSON-RPC 2.0 codec and types
│   │   ├── handler.go    # HTTP handler implementation
│   │   └── jsonrpc_test.go
│   └── openapi/
│       ├── spec.go       # OpenAPI document generation
│       ├── handler.go    # HTTP handler for serving spec
│       └── openapi_test.go
├── transport_rest.go     # REST transport (stays in main package)
├── transport_mcp.go      # MCP transport (stays in main package)
├── transport_trpc.go     # tRPC transport (stays in main package)
└── ...
```

## Core Interfaces (`contract/transport.go`)

### Transport Interface

```go
// Transport represents a protocol-agnostic transport layer.
type Transport interface {
    // Name returns the transport identifier (e.g., "jsonrpc", "rest").
    Name() string
}

// Handler is an http.Handler that serves a transport.
type Handler interface {
    http.Handler
    Transport
}
```

### Codec Interface

```go
// Codec handles request/response encoding for a transport.
type Codec interface {
    // ContentType returns the MIME type for this codec.
    ContentType() string

    // DecodeRequest decodes a request from the reader.
    DecodeRequest(r io.Reader) (*Request, error)

    // EncodeResponse writes a response to the writer.
    EncodeResponse(w io.Writer, resp *Response) error

    // EncodeError writes an error response to the writer.
    EncodeError(w io.Writer, err error) error
}
```

### Request and Response Types

```go
// Request represents a decoded transport request.
type Request struct {
    // ID is the request identifier (for request-response correlation).
    ID any

    // Method is the service method name.
    Method string

    // Params contains the raw input parameters.
    Params []byte

    // Metadata contains transport-specific metadata.
    Metadata map[string]any
}

// Response represents a transport response.
type Response struct {
    // ID matches the request ID.
    ID any

    // Result contains the method output (nil for void methods).
    Result any

    // Error contains any error that occurred.
    Error error
}
```

### Method Resolver

```go
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
```

### Invoker Interface

```go
// TransportInvoker invokes methods with transport context.
type TransportInvoker interface {
    // Invoke calls a method with the given input.
    Invoke(ctx context.Context, method *Method, input []byte) (any, error)
}

// DefaultInvoker is the standard invoker implementation.
type DefaultInvoker struct{}

func (d *DefaultInvoker) Invoke(ctx context.Context, m *Method, input []byte) (any, error) {
    var in any
    if m.HasInput() {
        in = m.NewInput()
        if len(input) > 0 && !isJSONNull(input) {
            if err := json.Unmarshal(input, in); err != nil {
                return nil, &Error{
                    Code:    ErrCodeInvalidArgument,
                    Message: "invalid input: " + err.Error(),
                }
            }
        }
    }
    return m.Invoker.Call(ctx, in)
}
```

### Error Types

```go
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

func (e *TransportError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return fmt.Sprintf("transport error: code=%d", e.Code)
}

func (e *TransportError) Unwrap() error {
    return e.Cause
}
```

### Options Pattern

```go
// Option configures a transport handler.
type Option func(*Options)

// Options contains transport configuration.
type Options struct {
    // Resolver finds methods by name.
    Resolver Resolver

    // Invoker calls methods.
    Invoker TransportInvoker

    // Middleware wraps method invocations.
    Middleware []MethodMiddleware

    // Logger logs transport events.
    Logger Logger

    // ErrorHandler customizes error responses.
    ErrorHandler func(error) *TransportError
}

// WithResolver sets a custom resolver.
func WithResolver(r Resolver) Option {
    return func(o *Options) { o.Resolver = r }
}

// WithInvoker sets a custom invoker.
func WithInvoker(i TransportInvoker) Option {
    return func(o *Options) { o.Invoker = i }
}

// WithMiddleware adds middleware.
func WithMiddleware(mw ...MethodMiddleware) Option {
    return func(o *Options) { o.Middleware = append(o.Middleware, mw...) }
}
```

## JSON-RPC 2.0 Package (`contract/transport/jsonrpc`)

### Types

```go
package jsonrpc

import "encoding/json"

// Version is the JSON-RPC version string.
const Version = "2.0"

// Standard error codes per JSON-RPC 2.0 specification.
const (
    CodeParseError     = -32700
    CodeInvalidRequest = -32600
    CodeMethodNotFound = -32601
    CodeInvalidParams  = -32602
    CodeInternalError  = -32603
    // Server errors: -32000 to -32099
    CodeServerError    = -32000
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      json.RawMessage `json:"id,omitempty"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      json.RawMessage `json:"id,omitempty"`
    Result  any             `json:"result,omitempty"`
    Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
    return e.Message
}
```

### Codec Implementation

```go
// Codec implements JSON-RPC 2.0 encoding.
type Codec struct{}

func (c *Codec) ContentType() string {
    return "application/json"
}

func (c *Codec) DecodeRequest(r io.Reader) (*Request, error) {
    var req Request
    if err := json.NewDecoder(r).Decode(&req); err != nil {
        return nil, &Error{Code: CodeParseError, Message: "Parse error", Data: err.Error()}
    }
    if req.JSONRPC != Version {
        return nil, &Error{Code: CodeInvalidRequest, Message: "Invalid Request"}
    }
    return &req, nil
}

func (c *Codec) EncodeResponse(w io.Writer, resp *Response) error {
    resp.JSONRPC = Version
    return json.NewEncoder(w).Encode(resp)
}
```

### Handler

```go
// Handler handles JSON-RPC 2.0 requests over HTTP.
type Handler struct {
    service  *contract.Service
    resolver contract.Resolver
    invoker  contract.TransportInvoker
    codec    *Codec
}

// NewHandler creates a JSON-RPC handler.
func NewHandler(svc *contract.Service, opts ...contract.Option) *Handler {
    options := contract.ApplyOptions(opts...)
    h := &Handler{
        service:  svc,
        codec:    &Codec{},
        resolver: options.Resolver,
        invoker:  options.Invoker,
    }
    if h.resolver == nil {
        h.resolver = &contract.ServiceResolver{Service: svc}
    }
    if h.invoker == nil {
        h.invoker = &contract.DefaultInvoker{}
    }
    return h
}

func (h *Handler) Name() string { return "jsonrpc" }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    // Parse request (single or batch)
    // Handle each request
    // Write response
}

// Mount registers the handler with a ServeMux at the given path.
func Mount(mux *http.ServeMux, path string, svc *contract.Service, opts ...contract.Option) {
    if path == "" {
        path = "/jsonrpc"
    }
    mux.Handle(path, NewHandler(svc, opts...))
}
```

### Batch Support

```go
// DecodeBatch attempts to decode a batch request.
// Returns nil if not a batch (single request).
func (c *Codec) DecodeBatch(r io.Reader) ([]*Request, error) {
    raw, err := io.ReadAll(r)
    if err != nil {
        return nil, err
    }

    raw = bytes.TrimSpace(raw)
    if len(raw) == 0 {
        return nil, &Error{Code: CodeInvalidRequest, Message: "empty body"}
    }

    if raw[0] == '[' {
        var batch []*Request
        if err := json.Unmarshal(raw, &batch); err != nil {
            return nil, &Error{Code: CodeParseError, Message: "Parse error"}
        }
        return batch, nil
    }

    var req Request
    if err := json.Unmarshal(raw, &req); err != nil {
        return nil, &Error{Code: CodeParseError, Message: "Parse error"}
    }
    return []*Request{&req}, nil
}
```

## OpenAPI Package (`contract/transport/openapi`)

### Spec Generation

```go
package openapi

// Document represents an OpenAPI 3.1 document.
type Document struct {
    OpenAPI    string                `json:"openapi"`
    Info       *Info                 `json:"info"`
    Paths      map[string]*PathItem  `json:"paths,omitempty"`
    Components *Components           `json:"components,omitempty"`
}

// Generate creates an OpenAPI document from services.
func Generate(services ...*contract.Service) *Document {
    doc := &Document{
        OpenAPI: "3.1.0",
        Info:    &Info{Title: "API", Version: "1.0.0"},
        Paths:   make(map[string]*PathItem),
        Components: &Components{
            Schemas: make(map[string]*Schema),
        },
    }

    for _, svc := range services {
        addService(doc, svc)
    }

    return doc
}

func addService(doc *Document, svc *contract.Service) {
    // Add service info
    if len(doc.Info.Title) == 0 || doc.Info.Title == "API" {
        doc.Info.Title = svc.Name + " API"
        if svc.Description != "" {
            doc.Info.Description = svc.Description
        }
        if svc.Version != "" {
            doc.Info.Version = svc.Version
        }
    }

    // Add schemas
    for _, schema := range svc.Types.Schemas() {
        doc.Components.Schemas[schema.ID] = convertSchema(schema.JSON)
    }

    // Add paths
    basePath := "/" + pluralize(svc.Name)
    for _, m := range svc.Methods {
        addMethod(doc, svc, m, basePath)
    }
}
```

### Handler

```go
// Handler serves OpenAPI documents over HTTP.
type Handler struct {
    document *Document
    json     []byte
}

// NewHandler creates an OpenAPI handler.
func NewHandler(services ...*contract.Service) (*Handler, error) {
    doc := Generate(services...)
    data, err := json.MarshalIndent(doc, "", "  ")
    if err != nil {
        return nil, err
    }
    return &Handler{document: doc, json: data}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(h.json)
}

// Mount registers the handler at the given path.
func Mount(mux *http.ServeMux, path string, services ...*contract.Service) error {
    if path == "" {
        path = "/openapi.json"
    }
    h, err := NewHandler(services...)
    if err != nil {
        return err
    }
    mux.Handle(path, h)
    return nil
}
```

### OpenAPI Types

```go
// Info provides metadata about the API.
type Info struct {
    Title       string `json:"title"`
    Description string `json:"description,omitempty"`
    Version     string `json:"version"`
}

// PathItem describes operations on a path.
type PathItem struct {
    Get    *Operation `json:"get,omitempty"`
    Post   *Operation `json:"post,omitempty"`
    Put    *Operation `json:"put,omitempty"`
    Delete *Operation `json:"delete,omitempty"`
    Patch  *Operation `json:"patch,omitempty"`
}

// Operation describes a single API operation.
type Operation struct {
    OperationID string              `json:"operationId,omitempty"`
    Summary     string              `json:"summary,omitempty"`
    Description string              `json:"description,omitempty"`
    Tags        []string            `json:"tags,omitempty"`
    Deprecated  bool                `json:"deprecated,omitempty"`
    Parameters  []*Parameter        `json:"parameters,omitempty"`
    RequestBody *RequestBody        `json:"requestBody,omitempty"`
    Responses   map[string]*Response `json:"responses"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
    Name        string  `json:"name"`
    In          string  `json:"in"`
    Description string  `json:"description,omitempty"`
    Required    bool    `json:"required,omitempty"`
    Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes a request body.
type RequestBody struct {
    Description string                `json:"description,omitempty"`
    Required    bool                  `json:"required,omitempty"`
    Content     map[string]*MediaType `json:"content"`
}

// MediaType describes a media type.
type MediaType struct {
    Schema *SchemaRef `json:"schema,omitempty"`
}

// Response describes a single response.
type Response struct {
    Description string                `json:"description"`
    Content     map[string]*MediaType `json:"content,omitempty"`
}

// Components holds reusable objects.
type Components struct {
    Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// Schema represents a JSON Schema.
type Schema struct {
    Type                 string             `json:"type,omitempty"`
    Format               string             `json:"format,omitempty"`
    Description          string             `json:"description,omitempty"`
    Properties           map[string]*Schema `json:"properties,omitempty"`
    Required             []string           `json:"required,omitempty"`
    Items                *Schema            `json:"items,omitempty"`
    Enum                 []any              `json:"enum,omitempty"`
    Nullable             bool               `json:"nullable,omitempty"`
    AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
    Minimum              *float64           `json:"minimum,omitempty"`
    Maximum              *float64           `json:"maximum,omitempty"`
    MinLength            *int               `json:"minLength,omitempty"`
    MaxLength            *int               `json:"maxLength,omitempty"`
}

// SchemaRef is either a schema or a $ref.
type SchemaRef struct {
    Ref    string  `json:"$ref,omitempty"`
    Schema *Schema `json:"-"`
}

func (s *SchemaRef) MarshalJSON() ([]byte, error) {
    if s.Ref != "" {
        return json.Marshal(map[string]string{"$ref": s.Ref})
    }
    return json.Marshal(s.Schema)
}
```

## Helper Functions (`contract/transport.go`)

### JSON Utilities

```go
// TrimSpace removes leading/trailing whitespace from JSON bytes.
func TrimSpace(b []byte) []byte {
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

// IsNull checks if bytes represent JSON null.
func IsNull(b []byte) bool {
    b = TrimSpace(b)
    return len(b) == 4 && string(b) == "null"
}

// SafeError returns a safe string representation of an error.
func SafeError(err error) string {
    if err == nil {
        return ""
    }
    return err.Error()
}
```

## Migration Path

### Phase 1: Add New Packages

1. Create `contract/transport.go` with interfaces
2. Create `contract/transport/jsonrpc/` package
3. Create `contract/transport/openapi/` package
4. Add tests for new packages

### Phase 2: Deprecate Old Functions

1. Mark existing functions as deprecated
2. Add forwarding to new packages
3. Update documentation

### Phase 3: Remove Old Code (Future)

1. Remove deprecated transport files
2. Update all consumers

## Backwards Compatibility

Existing APIs remain functional:

```go
// These continue to work (wrappers around new packages)
contract.MountJSONRPC(mux, "/rpc", svc)
contract.ServeOpenAPI(mux, "/openapi.json", svc)
contract.GenerateOpenAPI(services...)
```

New preferred APIs:

```go
import (
    "github.com/go-mizu/mizu/contract/transport/jsonrpc"
    "github.com/go-mizu/mizu/contract/transport/openapi"
)

jsonrpc.Mount(mux, "/rpc", svc)
openapi.Mount(mux, "/openapi.json", svc)
```

## Testing Strategy

### Unit Tests

Each package has isolated unit tests:

```go
// transport/jsonrpc/jsonrpc_test.go
func TestCodec_DecodeRequest(t *testing.T)
func TestCodec_EncodeResponse(t *testing.T)
func TestHandler_ServeHTTP(t *testing.T)
func TestHandler_Batch(t *testing.T)
func TestHandler_Notification(t *testing.T)

// transport/openapi/openapi_test.go
func TestGenerate_SingleService(t *testing.T)
func TestGenerate_MultipleServices(t *testing.T)
func TestHandler_ServeHTTP(t *testing.T)
```

### Integration Tests

```go
// contract_test.go
func TestJSONRPCTransport_Integration(t *testing.T)
func TestOpenAPITransport_Integration(t *testing.T)
```

## Naming Conventions

Following Go stdlib patterns:

| Pattern | Example |
|---------|---------|
| Package names | `jsonrpc`, `openapi` (not `json_rpc`, `open_api`) |
| Handler types | `Handler` (not `JSONRPCHandler`) |
| Factory functions | `NewHandler`, `New` |
| Mount functions | `Mount` (registers with mux) |
| Generate functions | `Generate` (creates documents) |
| Interfaces | Verb-like: `Resolver`, `Invoker`, `Codec` |
| Options | `Option`, `WithX` pattern |

## Error Handling

### Transport Errors

Transport-level errors use transport-specific codes:

```go
// JSON-RPC
jsonrpc.NewError(jsonrpc.CodeInvalidParams, "missing field: name")

// Contract errors map to transport errors
contract.ErrCodeNotFound -> jsonrpc.CodeMethodNotFound (-32601)
contract.ErrCodeInvalidArgument -> jsonrpc.CodeInvalidParams (-32602)
contract.ErrCodeInternal -> jsonrpc.CodeInternalError (-32603)
```

### Error Mapping

```go
// MapError converts a contract.Error to a transport error.
func MapError(err error) *Error {
    var ce *contract.Error
    if errors.As(err, &ce) {
        return &Error{
            Code:    mapErrorCode(ce.Code),
            Message: ce.Message,
            Data:    ce.Details,
        }
    }
    return &Error{
        Code:    CodeInternalError,
        Message: err.Error(),
    }
}

func mapErrorCode(code contract.ErrorCode) int {
    switch code {
    case contract.ErrCodeInvalidArgument:
        return CodeInvalidParams
    case contract.ErrCodeNotFound:
        return CodeMethodNotFound
    default:
        return CodeInternalError
    }
}
```

## Success Criteria

- [ ] All existing tests pass
- [ ] New packages have >90% test coverage
- [ ] No breaking changes to existing APIs
- [ ] Documentation updated
- [ ] Clean package boundaries (no import cycles)
- [ ] Follows Go stdlib naming conventions
