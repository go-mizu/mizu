# Contract Layer Enhancement Specification

## Overview

This document specifies enhancements to Mizu's contract layer to achieve a fully transport-neutral service contract that enables:

- Pure Go service definitions (structs + methods, zero dependencies)
- Consistent exposure across REST, RPC, JSON-RPC, gRPC, MCP, and client generators
- Complete decoupling of business logic from transport concerns

## Current State

The contract package currently provides:
- Service registration via `Register(name, svc)`
- Four canonical method signatures
- TypeRegistry with basic JSON schema generation
- Compiled invokers (reflection at startup only)
- REST transport with convention-based routing
- JSON-RPC 2.0 transport
- Basic OpenAPI generation

## Gap Analysis

| Feature | Status | Notes |
|---------|--------|-------|
| Plain Go service definition | Done | Zero dependencies |
| Single canonical method signature | Done | 4 variants supported |
| One-time registration | Done | Stable descriptor |
| Deterministic naming | Done | Predictable |
| Reflection at startup only | Done | Compiled invokers |
| Transport-agnostic schemas | Partial | Missing arrays, maps, enums |
| Shared error contract | Partial | Minimal ErrorContract |
| Convention-based mapping | Done | REST verbs from method names |
| REST surface generation | Done | No annotations needed |
| JSON-RPC surface | Done | 2.0 compliant |
| OpenAPI generation | Partial | Not using openapi package |
| gRPC adapter | Missing | - |
| MCP exposure | Missing | - |
| Introspection endpoint | Missing | - |
| Client generation inputs | Missing | - |
| Testing utilities | Missing | - |
| Middleware support | Missing | - |

## Implementation Plan

### Phase 1: Core Enhancements

#### 1.1 Enhanced Error Contract

Implement a portable error system that maps consistently across transports.

```go
// Error codes aligned with common conventions
type ErrorCode string

const (
    ErrCodeInvalidArgument   ErrorCode = "INVALID_ARGUMENT"
    ErrCodeNotFound          ErrorCode = "NOT_FOUND"
    ErrCodeAlreadyExists     ErrorCode = "ALREADY_EXISTS"
    ErrCodePermissionDenied  ErrorCode = "PERMISSION_DENIED"
    ErrCodeUnauthenticated   ErrorCode = "UNAUTHENTICATED"
    ErrCodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
    ErrCodeFailedPrecondition ErrorCode = "FAILED_PRECONDITION"
    ErrCodeAborted           ErrorCode = "ABORTED"
    ErrCodeOutOfRange        ErrorCode = "OUT_OF_RANGE"
    ErrCodeUnimplemented     ErrorCode = "UNIMPLEMENTED"
    ErrCodeInternal          ErrorCode = "INTERNAL"
    ErrCodeUnavailable       ErrorCode = "UNAVAILABLE"
    ErrCodeDataLoss          ErrorCode = "DATA_LOSS"
    ErrCodeUnknown           ErrorCode = "UNKNOWN"
)

// Error is the portable error type
type Error struct {
    Code    ErrorCode      `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

// HTTP status mapping
func (e *Error) HTTPStatus() int

// JSON-RPC error code mapping
func (e *Error) JSONRPCCode() int

// gRPC status code mapping
func (e *Error) GRPCCode() int
```

#### 1.2 Extended Type Support

Enhance schema generation to support:

```go
// Supported types:
// - Primitives: string, bool, int, int8-64, uint, uint8-64, float32, float64
// - Time: time.Time (date-time format)
// - Arrays: []T, [N]T
// - Maps: map[string]T
// - Pointers: *T (nullable)
// - Structs: embedded, nested
// - Enums: via struct tags or custom types

// Schema tags for customization
type Example struct {
    ID          string    `json:"id" contract:"required"`
    Name        string    `json:"name" contract:"required,minLength=1,maxLength=100"`
    Age         int       `json:"age" contract:"minimum=0,maximum=150"`
    Tags        []string  `json:"tags" contract:"maxItems=10"`
    Status      Status    `json:"status" contract:"enum"`
    Metadata    map[string]any `json:"metadata,omitempty"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

// Enum type via custom type
type Status string

const (
    StatusPending  Status = "pending"
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
)

func (Status) ContractEnum() []any {
    return []any{"pending", "active", "inactive"}
}
```

#### 1.3 Method and Service Metadata

Add support for descriptions, tags, and deprecation:

```go
// Method options via interface implementation
type MethodMeta interface {
    ContractMeta() map[string]MethodOptions
}

type MethodOptions struct {
    Description string
    Summary     string
    Tags        []string
    Deprecated  bool
    // REST-specific overrides (optional)
    HTTPMethod  string // Override verb
    HTTPPath    string // Override path
}

// Service metadata via interface
type ServiceMeta interface {
    ContractServiceMeta() ServiceOptions
}

type ServiceOptions struct {
    Description string
    Version     string
    Tags        []string
}

// Example implementation
type TodoService struct{}

func (TodoService) ContractServiceMeta() ServiceOptions {
    return ServiceOptions{
        Description: "Manages todo items",
        Version:     "1.0.0",
        Tags:        []string{"todos"},
    }
}

func (TodoService) ContractMeta() map[string]MethodOptions {
    return map[string]MethodOptions{
        "Create": {Description: "Creates a new todo item"},
        "Get":    {Description: "Retrieves a todo by ID"},
        "List":   {Description: "Lists all todos with pagination"},
        "Update": {Description: "Updates an existing todo"},
        "Delete": {Description: "Deletes a todo by ID"},
    }
}
```

### Phase 2: Transports and Adapters

#### 2.1 Introspection Endpoint

Provide a machine-readable contract description:

```go
// ServeIntrospect mounts an introspection endpoint
func ServeIntrospect(mux *http.ServeMux, path string, services ...*Service)

// Introspection response format
type IntrospectionResponse struct {
    Services []ServiceDescriptor `json:"services"`
    Types    []TypeDescriptor    `json:"types"`
}

type ServiceDescriptor struct {
    Name        string             `json:"name"`
    Description string             `json:"description,omitempty"`
    Version     string             `json:"version,omitempty"`
    Methods     []MethodDescriptor `json:"methods"`
}

type MethodDescriptor struct {
    Name        string      `json:"name"`
    FullName    string      `json:"fullName"`
    Description string      `json:"description,omitempty"`
    Input       *TypeRef    `json:"input,omitempty"`
    Output      *TypeRef    `json:"output,omitempty"`
    Errors      []ErrorCode `json:"errors,omitempty"`
    Deprecated  bool        `json:"deprecated,omitempty"`
    Tags        []string    `json:"tags,omitempty"`
}

type TypeDescriptor struct {
    ID     string         `json:"id"`
    Name   string         `json:"name"`
    Schema map[string]any `json:"schema"`
}
```

#### 2.2 MCP (Model Context Protocol) Exposure

Expose services as MCP tools:

```go
// MountMCP mounts services as MCP tools
func MountMCP(mux *http.ServeMux, path string, services ...*Service)

// MCPTool represents a tool in MCP format
type MCPTool struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
}

// MCP endpoints:
// POST /mcp/tools/list     - List available tools
// POST /mcp/tools/call     - Call a tool
```

#### 2.3 OpenAPI Integration

Use the openapi package for proper OpenAPI generation:

```go
// GenerateOpenAPI creates a full OpenAPI document from services
func GenerateOpenAPI(services ...*Service) *openapi.Document

// ServeOpenAPI serves OpenAPI JSON using the openapi package
func ServeOpenAPI(mux *http.ServeMux, path string, services ...*Service)
```

### Phase 3: Client Generation

#### 3.1 Client Descriptor

Provide structured data for client generators:

```go
// ClientDescriptor contains all information needed to generate clients
type ClientDescriptor struct {
    Package     string              `json:"package"`
    Services    []ClientService     `json:"services"`
    Types       []ClientType        `json:"types"`
    Errors      []ClientError       `json:"errors"`
}

type ClientService struct {
    Name        string         `json:"name"`
    Description string         `json:"description,omitempty"`
    Methods     []ClientMethod `json:"methods"`
}

type ClientMethod struct {
    Name        string     `json:"name"`
    Description string     `json:"description,omitempty"`
    Input       *TypeRef   `json:"input,omitempty"`
    Output      *TypeRef   `json:"output,omitempty"`
    // Transport hints
    REST        *RESTHint  `json:"rest,omitempty"`
    RPC         *RPCHint   `json:"rpc,omitempty"`
}

type RESTHint struct {
    Method string `json:"method"`
    Path   string `json:"path"`
}

type RPCHint struct {
    Method string `json:"method"` // Full method name
}

type ClientType struct {
    ID         string            `json:"id"`
    Name       string            `json:"name"`
    GoType     string            `json:"goType"`
    TSType     string            `json:"tsType"`
    Fields     []ClientField     `json:"fields,omitempty"`
    EnumValues []any             `json:"enumValues,omitempty"`
}

type ClientField struct {
    Name       string `json:"name"`
    Type       string `json:"type"`
    Required   bool   `json:"required"`
    Nullable   bool   `json:"nullable"`
}

// GenerateClientDescriptor creates a client descriptor
func GenerateClientDescriptor(services ...*Service) *ClientDescriptor

// ServeClientDescriptor serves the descriptor as JSON
func ServeClientDescriptor(mux *http.ServeMux, path string, services ...*Service)
```

### Phase 4: Testing Support

#### 4.1 Testing Utilities

Provide utilities for testing services without transport:

```go
// TestClient provides a transport-free way to call service methods
type TestClient struct {
    service *Service
}

// NewTestClient creates a test client for a service
func NewTestClient(svc *Service) *TestClient

// Call invokes a method by name
func (c *TestClient) Call(ctx context.Context, method string, in any) (any, error)

// MustCall panics on error (for tests)
func (c *TestClient) MustCall(ctx context.Context, method string, in any) any

// Example test
func TestTodoService(t *testing.T) {
    svc, _ := contract.Register("todo", &TodoService{})
    client := contract.NewTestClient(svc)

    // Create
    out, err := client.Call(ctx, "Create", &CreateTodoInput{Title: "Test"})
    if err != nil {
        t.Fatal(err)
    }

    // Get
    todo := out.(*Todo)
    got, err := client.Call(ctx, "Get", &GetTodoInput{ID: todo.ID})
    // ...
}
```

#### 4.2 Mock Service Support

Generate mocks for testing consumers:

```go
// MockService creates a mock implementation
type MockService struct {
    handlers map[string]func(ctx context.Context, in any) (any, error)
}

// On registers a mock handler for a method
func (m *MockService) On(method string, fn func(ctx context.Context, in any) (any, error))

// OnReturn registers a simple return value
func (m *MockService) OnReturn(method string, out any, err error)
```

### Phase 5: Middleware and Hooks

#### 5.1 Method Middleware

Add middleware support at the contract level:

```go
// MethodMiddleware wraps method invocations
type MethodMiddleware func(MethodInvoker) MethodInvoker

type MethodInvoker func(ctx context.Context, method *Method, in any) (any, error)

// WithMiddleware adds middleware to a service
func (s *Service) WithMiddleware(mw ...MethodMiddleware) *Service

// Built-in middleware
func LoggingMiddleware(logger Logger) MethodMiddleware
func ValidationMiddleware() MethodMiddleware
func RecoveryMiddleware() MethodMiddleware
func MetricsMiddleware(metrics Metrics) MethodMiddleware
```

### Phase 6: gRPC Support (Future)

#### 6.1 Protocol Buffer Generation

```go
// GenerateProto generates .proto files from services
func GenerateProto(services ...*Service) string

// Example output:
// syntax = "proto3";
// package todo;
//
// service TodoService {
//   rpc Create(CreateTodoInput) returns (Todo);
//   rpc Get(GetTodoInput) returns (Todo);
//   rpc List(ListTodosInput) returns (ListTodosOutput);
// }
```

#### 6.2 gRPC Server Adapter

```go
// GRPCServer adapts a contract service to gRPC
func GRPCServer(svc *Service) grpc.ServiceRegistrar
```

## File Structure

```
contract/
├── contract.go          # Core: Service, Method, Register
├── errors.go            # Enhanced error contract
├── schema.go            # Type registry, schema generation
├── metadata.go          # Service/method metadata interfaces
├── invoker.go           # Invoker implementation
├── transport_rest.go    # REST transport
├── transport_jsonrpc.go # JSON-RPC transport
├── transport_mcp.go     # MCP transport
├── introspect.go        # Introspection endpoint
├── openapi.go           # OpenAPI generation (uses openapi pkg)
├── client.go            # Client descriptor generation
├── middleware.go        # Middleware support
├── testing.go           # Test utilities
└── *_test.go            # Tests for each file
```

## API Surface

### Public Types

```go
// Core
type Service struct { ... }
type Method struct { ... }
type TypeRegistry struct { ... }
type TypeRef struct { ... }
type Schema struct { ... }

// Errors
type Error struct { ... }
type ErrorCode string

// Metadata
type MethodOptions struct { ... }
type ServiceOptions struct { ... }

// Introspection
type IntrospectionResponse struct { ... }
type ServiceDescriptor struct { ... }
type MethodDescriptor struct { ... }

// Client
type ClientDescriptor struct { ... }

// Testing
type TestClient struct { ... }
type MockService struct { ... }

// Middleware
type MethodMiddleware func(MethodInvoker) MethodInvoker
```

### Public Functions

```go
// Registration
func Register(name string, svc any) (*Service, error)

// Transports
func MountREST(mux *http.ServeMux, svc *Service)
func MountJSONRPC(mux *http.ServeMux, path string, svc *Service)
func MountMCP(mux *http.ServeMux, path string, services ...*Service)

// Documentation
func ServeOpenAPI(mux *http.ServeMux, path string, services ...*Service)
func ServeIntrospect(mux *http.ServeMux, path string, services ...*Service)
func ServeClientDescriptor(mux *http.ServeMux, path string, services ...*Service)

// Generation
func GenerateOpenAPI(services ...*Service) *openapi.Document
func GenerateClientDescriptor(services ...*Service) *ClientDescriptor

// Testing
func NewTestClient(svc *Service) *TestClient
```

## Backwards Compatibility

All existing APIs remain unchanged. New features are additive:
- `Register()` gains optional metadata detection
- `Service` gains new methods for middleware
- New transport functions added
- New generation functions added

## Implementation Order

1. Enhanced error contract (`errors.go`)
2. Extended type support (`schema.go`)
3. Metadata interfaces (`metadata.go`)
4. Introspection endpoint (`introspect.go`)
5. MCP transport (`transport_mcp.go`)
6. OpenAPI integration (`openapi.go`)
7. Client descriptor (`client.go`)
8. Testing utilities (`testing.go`)
9. Middleware support (`middleware.go`)
10. Comprehensive tests

## Success Criteria

- [ ] All existing tests pass
- [ ] New features have >90% test coverage
- [ ] Services can be exposed on REST, JSON-RPC, and MCP from single definition
- [ ] OpenAPI spec validates correctly
- [ ] Client descriptor enables TypeScript/Go client generation
- [ ] TestClient enables service testing without HTTP
- [ ] Error codes map consistently across all transports
