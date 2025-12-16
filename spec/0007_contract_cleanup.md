# Spec 0007: Contract Package Cleanup

## Summary

Aggressively simplify the contract package by removing duplicates, consolidating types, and following Go standard library naming conventions. The goal is **less is more** - a minimal, clear API surface.

## Current Problems

### 1. Duplicate Types (introspect.go vs client.go)
- `ServiceDescriptor` vs `ClientService` - same concept
- `MethodDescriptor` vs `ClientMethod` - same concept
- `TypeDescriptor` vs `ClientType` - same concept
- `RESTDescriptor` vs `RESTHint` - identical
- `RPCDescriptor` vs `RPCHint` - identical
- `ClientTypeRef` duplicates `TypeRef`

### 2. Unnecessary Abstractions
- `Transport` interface - only has `Name()`, not useful
- `Handler` interface - just `http.Handler` + `Name()`
- `Codec` interface - unused, overly abstract
- `TransportRequest`/`TransportResponse` - should be in transport packages
- `TransportError` - redundant with `Error`
- `TransportOptions`/`TransportOption` - overly complex
- `WrappedService` - unnecessary wrapper

### 3. Too Many Middleware Functions
- `ValidationMiddleware` - placeholder, does nothing
- `ConditionalMiddleware` - rarely needed
- `RetryMiddleware` - transport concern
- `HooksMiddleware` - can use regular middleware
- `ContextMiddleware` - trivial, users can do this

### 4. Unnecessary Helper Functions
- `SafeErrorString` - trivial
- `TrimJSONSpace`/`IsJSONNull` - transport internal
- `GoTypeString`/`TSTypeString` - client generation concern

### 5. Testing Bloat
- `MockService`, `RecordingMock`, `AssertInput` - too complex
- `TestService` - redundant with `TestClient`

## Proposed API (After Cleanup)

### Core Types (contract.go)

```go
// Register creates a Service from a Go struct.
func Register(name string, impl any) (*Service, error)

// Service represents a registered service contract.
type Service struct {
    Name        string
    Description string
    Version     string
    Tags        []string
    Methods     []*Method
    Types       *Types
}

func (s *Service) Method(name string) *Method
func (s *Service) MethodNames() []string

// Method represents a callable service method.
type Method struct {
    Service     *Service
    Name        string
    Description string
    Deprecated  bool
    Input       *Type  // renamed from TypeRef
    Output      *Type
    // HTTP hints for REST transport
    HTTPMethod  string
    HTTPPath    string
}

func (m *Method) Call(ctx context.Context, in any) (any, error)
func (m *Method) HasInput() bool
func (m *Method) HasOutput() bool
func (m *Method) NewInput() any
```

### Types (types.go) - renamed from schema.go

```go
// Types manages registered types and their schemas.
type Types struct { ... }

func (t *Types) Get(id string) *Type
func (t *Types) All() []*Type
func (t *Types) Schema(id string) map[string]any

// Type represents a registered type.
type Type struct {
    ID     string
    Name   string
    Schema map[string]any
}
```

### Errors (errors.go)

```go
// Code represents a portable error code.
type Code string

const (
    OK                 Code = "OK"
    Canceled           Code = "CANCELED"
    Unknown            Code = "UNKNOWN"
    InvalidArgument    Code = "INVALID_ARGUMENT"
    NotFound           Code = "NOT_FOUND"
    AlreadyExists      Code = "ALREADY_EXISTS"
    PermissionDenied   Code = "PERMISSION_DENIED"
    Unauthenticated    Code = "UNAUTHENTICATED"
    ResourceExhausted  Code = "RESOURCE_EXHAUSTED"
    FailedPrecondition Code = "FAILED_PRECONDITION"
    Aborted            Code = "ABORTED"
    OutOfRange         Code = "OUT_OF_RANGE"
    Unimplemented      Code = "UNIMPLEMENTED"
    Internal           Code = "INTERNAL"
    Unavailable        Code = "UNAVAILABLE"
    DataLoss           Code = "DATA_LOSS"
    DeadlineExceeded   Code = "DEADLINE_EXCEEDED"
)

// Error is the portable error type.
type Error struct {
    Code    Code
    Message string
    Details map[string]any
}

func NewError(code Code, message string) *Error
func Errorf(code Code, format string, args ...any) *Error
func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) Is(target error) bool
func (e *Error) HTTPStatus() int
```

### Middleware (middleware.go)

```go
// Middleware wraps method invocation.
type Middleware func(next func(context.Context, *Method, any) (any, error)) func(context.Context, *Method, any) (any, error)

// Chain combines middleware.
func Chain(mw ...Middleware) Middleware

// Recovery returns middleware that recovers from panics.
func Recovery() Middleware

// Timeout returns middleware that adds timeout.
func Timeout(d time.Duration) Middleware

// Logging returns middleware that logs calls.
func Logging(log func(method string, duration time.Duration, err error)) Middleware
```

### Description (describe.go) - replaces introspect.go AND client.go

```go
// Describe returns a machine-readable description of services.
func Describe(services ...*Service) *Description

// Description is the unified introspection/client descriptor.
type Description struct {
    Services []ServiceDesc
    Types    []TypeDesc
    Errors   []ErrorDesc
}

type ServiceDesc struct {
    Name        string
    Description string
    Version     string
    Methods     []MethodDesc
}

type MethodDesc struct {
    Name        string
    Description string
    Deprecated  bool
    Input       *TypeRef
    Output      *TypeRef
    HTTP        *HTTPDesc  // REST hints
    RPC         string     // RPC method name
}

type HTTPDesc struct {
    Method string
    Path   string
}

type TypeRef struct {
    ID   string
    Name string
}

type TypeDesc struct {
    ID     string
    Name   string
    Schema map[string]any
}

type ErrorDesc struct {
    Code       string
    HTTPStatus int
}

// Serve mounts a description endpoint.
func ServeDescription(mux *http.ServeMux, path string, services ...*Service)
```

### Testing (testing.go)

```go
// Client is a test client for calling service methods.
type Client struct { ... }

func NewClient(svc *Service) *Client
func (c *Client) Call(ctx context.Context, method string, in any) (any, error)
func (c *Client) CallJSON(ctx context.Context, method string, in []byte) ([]byte, error)
```

## Files After Cleanup

```
contract/
  contract.go      # Service, Method, Register
  types.go         # Types, Type (renamed from schema.go)
  errors.go        # Code, Error (simplified)
  middleware.go    # Middleware, Chain, Recovery, Timeout, Logging
  describe.go      # Description, Describe (merged introspect+client)
  testing.go       # Client only
  doc.go           # Package documentation
```

## Removed Files
- `transport.go` - abstractions moved to transport packages
- `client.go` - merged into describe.go
- `introspect.go` - merged into describe.go
- `helpers.go` - inlined or moved to transport packages

## Removed Types

### Interfaces
- `Transport` - unused abstraction
- `Handler` - just use http.Handler
- `Codec` - transport-specific
- `Resolver` - inline into transports
- `TransportInvoker` - simplify to function
- `ServiceMeta` - keep, but rename method
- `MethodMeta` - keep, but rename method
- `ContractEnum` - rename to `Enum`

### Structs
- `TransportRequest` - move to transport packages
- `TransportResponse` - move to transport packages
- `TransportError` - use Error instead
- `TransportOptions` - simplify in transports
- `ServiceResolver` - inline
- `DefaultInvoker` - inline
- `WrappedService` - remove
- `LoggerFunc` - use function directly
- All `Client*` types - merge with introspect types
- `IntrospectionResponse` - rename to Description
- `ServiceDescriptor` - rename to ServiceDesc
- `MethodDescriptor` - rename to MethodDesc
- `TypeDescriptor` - rename to TypeDesc
- `RESTDescriptor`/`RESTHint` - merge to HTTPDesc
- `RPCDescriptor`/`RPCHint` - just use string
- `MockService`, `RecordingMock`, `AssertInput` - remove
- `TestService` - remove (use Client)

### Functions
- `SafeErrorString` - remove
- `TrimJSONSpace` - move to internal
- `IsJSONNull` - move to internal
- `ApplyTransportOptions` - remove
- `WithResolver`, `WithTransportInvoker`, etc. - remove
- `GoTypeString`, `TSTypeString` - remove
- All `Err*` convenience constructors except `NewError`/`Errorf`
- `ValidationMiddleware` - remove (placeholder)
- `ConditionalMiddleware` - remove
- `RetryMiddleware` - remove
- `HooksMiddleware` - remove
- `ContextMiddleware` - remove
- `MetricsMiddleware` - remove (trivial)
- `StdLoggingMiddleware` - remove
- `MethodFromContext`, `ServiceFromContext` - remove
- `ServeIntrospect` - rename to ServeDescription
- `GenerateClientDescriptor` - merge into Describe
- `ServeClientDescriptor` - remove

### Constants
- `ErrCode*` prefix - change to just code name (e.g., `InvalidArgument`)
- `ContextKey*` - remove
- Sentinel errors (`ErrInvalidService`, etc.) - keep but simplify

## Naming Conventions (Go Standard Library Style)

| Old Name | New Name | Rationale |
|----------|----------|-----------|
| `ErrorCode` | `Code` | Shorter, context is clear |
| `ErrCodeInvalidArgument` | `InvalidArgument` | No redundant prefix |
| `TypeRef` | `Type` | Simpler |
| `TypeRegistry` | `Types` | Go uses plural for collections |
| `MethodMiddleware` | `Middleware` | Context is clear |
| `MethodInvoker` | inline func | No need for type |
| `ServiceDescriptor` | `ServiceDesc` | Shorter |
| `IntrospectionResponse` | `Description` | Clearer purpose |
| `TestClient` | `Client` | In testing context |

## Transport Package Changes

Each transport package should handle its own:
- Request/Response types
- Error mapping
- Options pattern

The contract package provides only:
- `Service` and `Method` for method resolution
- `Error` for error handling
- `Types` for schema access

## Migration

Transport packages need updates:
1. Use `Method.Call()` instead of `Invoker.Call()`
2. Define own request/response types
3. Import error codes as `contract.InvalidArgument` etc.
4. Use `contract.Describe()` for introspection data

## Implementation Order

1. Rename and consolidate types in contract package
2. Remove unused interfaces and abstractions
3. Simplify middleware
4. Merge introspect.go and client.go into describe.go
5. Simplify testing.go
6. Update transport packages
7. Run tests and fix issues
