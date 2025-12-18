# Contract v2 Documentation Specification

**Status:** In Progress
**Package:** `contract/v2`

## Overview

This specification defines the complete rewrite of contract documentation for `contract/v2`. The v2 package introduces an interface-first approach where services are defined as Go interfaces, providing compile-time safety and clearer API contracts.

## Key Changes from v1

| Aspect | v1 | v2 |
|--------|----|----|
| **Registration** | `contract.Register("name", &struct{})` | `contract.Register[Interface](impl, opts...)` |
| **Service Definition** | Struct with methods | Interface + implementation |
| **Method Discovery** | Reflect on struct methods | Reflect on interface methods |
| **Resource Organization** | Flat methods | Methods grouped into resources |
| **Method Naming** | `method.Name` | `resource.method` (e.g., `todos.create`) |
| **Transport Integration** | Uses mizu.Router | Uses mizu.Router with Mount functions |

## Documentation Structure

### Core Pages

1. **overview.mdx** - What is Contract and why use it
2. **quick-start.mdx** - Build first API in 5 minutes
3. **service.mdx** - Defining interfaces and implementations
4. **register.mdx** - Registration with options
5. **types.mdx** - Type system and schema generation
6. **errors.mdx** - Error handling (needs update for v2)

### Transport Pages

7. **transports-overview.mdx** - Comparison of all transports
8. **rest.mdx** - REST transport with Mount functions
9. **jsonrpc.mdx** - JSON-RPC 2.0 transport
10. **mcp.mdx** - MCP (Model Context Protocol) transport
11. **openapi.mdx** - OpenAPI spec generation

### Advanced Pages

12. **testing.mdx** - Testing strategies
13. **architecture.mdx** - Deep dive internals
14. **api-reference.mdx** - Complete API reference

## API Reference

### Registration

```go
// Register creates a contract service from a Go interface and implementation.
// T must be an interface type. impl must implement T.
func Register[T any](impl T, opts ...Option) *RegisteredService
```

### Options

```go
// Service configuration
WithName(name string) Option              // Set service name (default: interface name)
WithDescription(desc string) Option       // Set service description
WithDefaults(defaults Defaults) Option    // Set global defaults (base URL, auth, headers)

// Resource organization
WithResource(name string, methods ...string) Option  // Group methods into resource
WithDefaultResource(name string) Option              // Default resource for ungrouped methods

// HTTP bindings
WithHTTP(bindings map[string]HTTPBinding) Option     // Explicit HTTP bindings
WithMethodHTTP(method, httpMethod, path string) Option // Single method HTTP binding

// Streaming
WithStreaming(method string, mode StreamMode) Option // Mark method as streaming
```

### HTTPBinding

```go
type HTTPBinding struct {
    Method string // GET, POST, PUT, DELETE, PATCH
    Path   string // /todos, /todos/{id}
}
```

### StreamMode

```go
const (
    StreamSSE   StreamMode = "sse"   // Server-Sent Events
    StreamWS    StreamMode = "ws"    // WebSocket
    StreamGRPC  StreamMode = "grpc"  // gRPC streaming
    StreamAsync StreamMode = "async" // Async message broker
)
```

### Service Descriptor

```go
type Service struct {
    Name        string      // Service name
    Description string      // Service description
    Defaults    *Defaults   // Global defaults
    Resources   []*Resource // Resource definitions
    Types       []*Type     // Type definitions
}
```

### Resource

```go
type Resource struct {
    Name        string    // Resource name (e.g., "todos")
    Description string    // Resource description
    Methods     []*Method // Methods in this resource
}
```

### Method

```go
type Method struct {
    Name        string     // Method name (camelCase)
    Description string     // Method description
    Input       TypeRef    // Input type reference
    Output      TypeRef    // Output type reference
    Stream      *StreamDef // Streaming config (optional)
    HTTP        *MethodHTTP // HTTP binding
}
```

### Invoker Interface

```go
type Invoker interface {
    Descriptor
    Call(ctx context.Context, resource, method string, in any) (any, error)
    NewInput(resource, method string) (any, error)
    Stream(ctx context.Context, resource, method string, in any) (Stream, error)
}
```

## HTTP Binding Inference

Contract v2 automatically infers HTTP bindings from method names:

| Method Name Prefix | HTTP Method | Path Pattern |
|--------------------|-------------|--------------|
| `Create`, `Add`, `New` | POST | `/{resource}` |
| `List`, `All`, `Search`, `Find*s` | GET | `/{resource}` |
| `Get`, `Find`, `Fetch`, `Read` | GET | `/{resource}/{id}` |
| `Update`, `Edit`, `Modify`, `Set` | PUT | `/{resource}/{id}` |
| `Delete`, `Remove` | DELETE | `/{resource}/{id}` |
| `Patch` | PATCH | `/{resource}/{id}` |
| Others | POST | `/{resource}/{action}` |

### Path Parameter Extraction

The `{id}` parameter is extracted from the input type by looking for:
1. Field with `path:"paramName"` tag
2. Field named `ID` or ending with `ID`
3. Default to `id`

## Transport Mount Functions

### REST

```go
// Mount all routes on router
rest.Mount(r *mizu.Router, inv contract.Invoker, opts ...Option) error

// Mount under prefix
rest.MountAt(r *mizu.Router, prefix string, inv contract.Invoker, opts ...Option) error

// Get routes for manual registration
rest.Routes(inv contract.Invoker, opts ...Option) ([]Route, error)

// Generate OpenAPI spec
rest.OpenAPI(svc *contract.Service) ([]byte, error)
```

### JSON-RPC

```go
// Mount JSON-RPC endpoint
jsonrpc.Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error

// Get handler for manual use
jsonrpc.Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error)

// Generate OpenRPC spec
jsonrpc.OpenRPC(svc *contract.Service) ([]byte, error)
```

### MCP

```go
// Mount MCP endpoint
mcp.Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error

// Get handler for manual use
mcp.Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error)
```

## Method Signature Patterns

Contract v2 supports the same method signature patterns:

```go
// Pattern 1: Input and Output
func (T) Method(ctx context.Context, in *InputType) (*OutputType, error)

// Pattern 2: Output Only
func (T) Method(ctx context.Context) (*OutputType, error)

// Pattern 3: Input Only
func (T) Method(ctx context.Context, in *InputType) error

// Pattern 4: No Input or Output
func (T) Method(ctx context.Context) error
```

## Type System

### TypeRef

Reference to a type by name. Primitives: `string`, `int`, `float`, `bool`, `any`

### TypeKind

```go
const (
    KindStruct TypeKind = "struct"
    KindSlice  TypeKind = "slice"
    KindMap    TypeKind = "map"
    KindUnion  TypeKind = "union"
)
```

### Type

```go
type Type struct {
    Name        string    // Type name
    Description string    // Type description
    Kind        TypeKind  // Type kind
    Fields      []Field   // Struct fields
    Elem        TypeRef   // Slice element or map value type
    Tag         string    // Union discriminator field
    Variants    []Variant // Union variants
}
```

### Field

```go
type Field struct {
    Name        string   // Field name (from json tag)
    Description string   // From `desc` tag
    Type        TypeRef  // Field type
    Optional    bool     // From `omitempty`
    Nullable    bool     // Pointer type or `nullable` tag
    Enum        []string // From `enum` tag
    Const       string   // Fixed value for discriminators
}
```

## Documentation Writing Guidelines

1. **No level 1 headers** - frontmatter has title
2. **Beginner friendly** - explain concepts from first principles
3. **Practical examples** - runnable code in every section
4. **Progressive complexity** - simple to advanced
5. **Cross-references** - link to related pages
6. **Code annotations** - explain each significant line
7. **Common mistakes** - what NOT to do with explanations
8. **FAQ sections** - address common questions

## Example: Complete v2 Service

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "sync"

    contract "github.com/go-mizu/mizu/contract/v2"
    "github.com/go-mizu/mizu/contract/v2/transport/rest"
    "github.com/go-mizu/mizu/contract/v2/transport/jsonrpc"
    "github.com/go-mizu/mizu"
)

// ─────────────────────────────────────────────────────────────
// Step 1: Define your types
// ─────────────────────────────────────────────────────────────

type Todo struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
}

type CreateInput struct {
    Title string `json:"title" required:"true"`
}

type GetInput struct {
    ID string `json:"id"`
}

type ListOutput struct {
    Items []*Todo `json:"items"`
    Total int     `json:"total"`
}

// ─────────────────────────────────────────────────────────────
// Step 2: Define your interface (the contract)
// ─────────────────────────────────────────────────────────────

type TodoAPI interface {
    Create(ctx context.Context, in *CreateInput) (*Todo, error)
    List(ctx context.Context) (*ListOutput, error)
    Get(ctx context.Context, in *GetInput) (*Todo, error)
    Delete(ctx context.Context, in *GetInput) error
}

// ─────────────────────────────────────────────────────────────
// Step 3: Implement the interface
// ─────────────────────────────────────────────────────────────

type todoService struct {
    mu    sync.RWMutex
    todos map[string]*Todo
    seq   int
}

func NewTodoService() TodoAPI {
    return &todoService{todos: make(map[string]*Todo)}
}

func (s *todoService) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.seq++
    todo := &Todo{
        ID:    fmt.Sprintf("%d", s.seq),
        Title: in.Title,
    }
    s.todos[todo.ID] = todo
    return todo, nil
}

func (s *todoService) List(ctx context.Context) (*ListOutput, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    items := make([]*Todo, 0, len(s.todos))
    for _, t := range s.todos {
        items = append(items, t)
    }
    return &ListOutput{Items: items, Total: len(items)}, nil
}

func (s *todoService) Get(ctx context.Context, in *GetInput) (*Todo, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.todos[in.ID], nil
}

func (s *todoService) Delete(ctx context.Context, in *GetInput) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.todos, in.ID)
    return nil
}

// ─────────────────────────────────────────────────────────────
// Step 4: Register and serve
// ─────────────────────────────────────────────────────────────

func main() {
    // Create implementation
    impl := NewTodoService()

    // Register with contract (interface-first!)
    svc := contract.Register[TodoAPI](impl,
        contract.WithName("Todo"),
        contract.WithDescription("Todo management API"),
        contract.WithDefaultResource("todos"),
    )

    // Create mizu app
    app := mizu.New()

    // Mount transports
    rest.Mount(app.Router, svc)                    // REST: /todos
    jsonrpc.Mount(app.Router, "/rpc", svc)         // JSON-RPC: /rpc

    // Serve OpenAPI
    spec, _ := rest.OpenAPI(svc.Descriptor())
    app.Get("/openapi.json", func(c *mizu.Ctx) error {
        return c.JSON(200, spec)
    })

    fmt.Println("Server running on http://localhost:8080")
    app.Listen(":8080")
}
```

## Test Examples

```bash
# REST
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy milk"}'

curl http://localhost:8080/todos

# JSON-RPC
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"todos.create","params":{"title":"Buy milk"}}'

curl -X POST http://localhost:8080/rpc \
  -d '{"jsonrpc":"2.0","id":2,"method":"todos.list"}'
```

## Migration from v1

```go
// v1 style (struct-first)
type TodoService struct{}
func (s *TodoService) Create(...) {}
svc, _ := contract.Register("todo", &TodoService{})

// v2 style (interface-first)
type TodoAPI interface {
    Create(...) error
}
type todoService struct{}
func (s *todoService) Create(...) {}
svc := contract.Register[TodoAPI](&todoService{})
```

Key migration steps:
1. Extract interface from struct methods
2. Change struct to unexported (lowercase)
3. Update registration call to use generic form
4. Update transport mount calls to use v2 packages
