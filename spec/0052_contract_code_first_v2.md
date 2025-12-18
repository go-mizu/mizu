# Contract v2: Code-First API Design

## Overview

This specification describes a **code-first** approach for contract/v2 that allows developers to define APIs using Go interfaces instead of YAML files. The YAML contract can then be generated from code, enabling a workflow where Go is the source of truth.

## Motivation

Contract v2 currently uses a **definition-first** approach where YAML files are the source of truth. While powerful for multi-language code generation, many Go developers prefer:

1. **Type safety at definition time** - IDE autocomplete, compiler checks
2. **Single language workflow** - No YAML learning curve
3. **Refactoring support** - Rename methods/types with IDE tooling
4. **Co-location** - Interface near implementation
5. **Documentation in code** - Godoc comments as descriptions

The code-first approach provides these benefits while maintaining full compatibility with v2's transport system.

## Design Goals

1. **Zero-friction registration** - Single function call to register a service
2. **Interface-centric** - Go interfaces define the API contract
3. **Automatic type extraction** - Reflect input/output types from method signatures
4. **Convention over configuration** - Sensible defaults, opt-in customization
5. **YAML generation** - Generate api.yaml from registered services
6. **Full v2 compatibility** - Works with all transports (REST, JSON-RPC, async)

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Code                                 │
├─────────────────────────────────────────────────────────────────┤
│  type TodoAPI interface {                                        │
│      Create(ctx, *CreateInput) (*Todo, error)                   │
│      List(ctx) (*TodoList, error)                               │
│      Get(ctx, *GetInput) (*Todo, error)                         │
│  }                                                               │
│                                                                  │
│  impl := &todoService{}                                          │
│  svc := contract.Register[TodoAPI](impl, opts)                  │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      contract.Register                           │
├─────────────────────────────────────────────────────────────────┤
│  1. Reflect interface methods                                    │
│  2. Extract input/output types                                   │
│  3. Build type schemas via reflection                            │
│  4. Apply HTTP binding conventions/overrides                     │
│  5. Create Invoker that dispatches to impl                       │
│  6. Return RegisteredService (Invoker + Service)                │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                    RegisteredService                             │
├─────────────────────────────────────────────────────────────────┤
│  - Implements Invoker interface                                  │
│  - Contains *Service descriptor                                  │
│  - Can generate YAML: svc.YAML() string                         │
│  - Works with all transports                                     │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Transports                                 │
├─────────────────────────────────────────────────────────────────┤
│  rest.NewServer(svc)     → HTTP REST server                     │
│  jsonrpc.NewServer(svc)  → JSON-RPC 2.0 server                  │
│  async.NewServer(svc, b) → Async messaging server               │
│                                                                  │
│  rest.OpenAPIDocument(svc.Descriptor())  → openapi.json         │
└─────────────────────────────────────────────────────────────────┘
```

### API Surface

#### Register Function

```go
// Register creates a contract service from a Go interface and implementation.
// T must be an interface type. impl must implement T.
func Register[T any](impl T, opts ...Option) *RegisteredService

// RegisteredService combines the Invoker interface with service metadata.
type RegisteredService struct {
    // Embedded Invoker for transport compatibility
    Invoker

    // service is the extracted contract descriptor
    service *Service
}

// Descriptor returns the contract service descriptor.
func (r *RegisteredService) Descriptor() *Service

// YAML generates the api.yaml content from the registered service.
func (r *RegisteredService) YAML() ([]byte, error)

// JSON generates the api.json content from the registered service.
func (r *RegisteredService) JSON() ([]byte, error)
```

#### Options

```go
// Option configures the registration process.
type Option func(*registerOptions)

// WithName sets the service name (default: interface name).
func WithName(name string) Option

// WithDescription sets the service description.
func WithDescription(desc string) Option

// WithResource groups methods into a named resource.
// Methods not explicitly grouped go into a default resource.
func WithResource(name string, methods ...string) Option

// WithHTTP provides explicit HTTP bindings for methods.
func WithHTTP(bindings map[string]HTTPBinding) Option

// WithDefaults sets service-level defaults.
func WithDefaults(defaults Defaults) Option

// WithStreaming marks methods as streaming with specified mode.
func WithStreaming(method string, mode StreamMode) Option
```

#### HTTP Binding

```go
// HTTPBinding specifies the HTTP method and path for an API method.
type HTTPBinding struct {
    Method string // GET, POST, PUT, DELETE, PATCH
    Path   string // /todos, /todos/{id}
}

// StreamMode specifies the streaming protocol.
type StreamMode string

const (
    StreamSSE  StreamMode = "sse"
    StreamWS   StreamMode = "ws"
    StreamGRPC StreamMode = "grpc"
)
```

## Method Signature Conventions

### Supported Method Signatures

```go
// Standard request-response
Method(ctx context.Context, in *InputType) (*OutputType, error)

// No input (list operations)
Method(ctx context.Context) (*OutputType, error)

// No output (fire-and-forget)
Method(ctx context.Context, in *InputType) error

// Streaming response (SSE/WebSocket)
Method(ctx context.Context, in *InputType) (<-chan *ItemType, error)

// Bidirectional streaming
Method(ctx context.Context, in <-chan *InputItem) (<-chan *OutputItem, error)
```

### HTTP Method Inference

When no explicit HTTP binding is provided, methods are mapped by convention:

| Method Name Pattern | HTTP Method | Path Pattern |
|---------------------|-------------|--------------|
| `Create*`, `Add*`, `New*` | POST | `/{resource}` |
| `Get*`, `Find*`, `Fetch*` | GET | `/{resource}/{id}` |
| `List*`, `All*`, `Search*` | GET | `/{resource}` |
| `Update*`, `Edit*`, `Modify*` | PUT | `/{resource}/{id}` |
| `Delete*`, `Remove*` | DELETE | `/{resource}/{id}` |
| `Patch*` | PATCH | `/{resource}/{id}` |
| Other | POST | `/{resource}/{method}` |

### Path Parameter Inference

Path parameters are inferred from input struct fields:

```go
type GetInput struct {
    ID string `json:"id"` // Becomes {id} in path
}

// Method: Get(ctx, *GetInput) → GET /todos/{id}
```

Fields can explicitly declare path binding:

```go
type GetInput struct {
    TodoID string `json:"todo_id" path:"id"` // Maps to {id}
}
```

## Type Extraction

### Struct Types

```go
type Todo struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Completed bool      `json:"completed"`
    CreatedAt time.Time `json:"created_at"`
}
```

Becomes:

```yaml
types:
  - name: Todo
    kind: struct
    fields:
      - name: id
        type: string
      - name: title
        type: string
      - name: completed
        type: bool
      - name: created_at
        type: string  # time.Time → string (ISO 8601)
```

### Field Annotations

```go
type CreateInput struct {
    // Title is required and must be non-empty
    Title string `json:"title" required:"true" desc:"Todo title"`

    // Body is optional
    Body string `json:"body,omitempty"`

    // Priority must be one of: low, medium, high
    Priority string `json:"priority" enum:"low,medium,high"`

    // Tags is nullable (can be null vs empty)
    Tags *[]string `json:"tags" nullable:"true"`
}
```

### Slice and Map Types

```go
type TodoList struct {
    Items []*Todo           `json:"items"`
    Meta  map[string]string `json:"meta"`
}
```

### Union Types (Discriminated)

```go
// Event is a discriminated union of event types.
// The Type field determines which concrete type is present.
type Event struct {
    Type string `json:"type" discriminator:"true"`

    // Exactly one of these will be non-nil based on Type
    Created *CreatedEvent `json:"created,omitempty" variant:"created"`
    Updated *UpdatedEvent `json:"updated,omitempty" variant:"updated"`
    Deleted *DeletedEvent `json:"deleted,omitempty" variant:"deleted"`
}
```

## Complete Example

### Interface Definition

```go
package api

import "context"

// TodoAPI defines the Todo service contract.
type TodoAPI interface {
    // Create creates a new todo item.
    Create(ctx context.Context, in *CreateInput) (*Todo, error)

    // List returns all todo items.
    List(ctx context.Context) (*TodoList, error)

    // Get retrieves a todo by ID.
    Get(ctx context.Context, in *GetInput) (*Todo, error)

    // Update modifies an existing todo.
    Update(ctx context.Context, in *UpdateInput) (*Todo, error)

    // Delete removes a todo.
    Delete(ctx context.Context, in *DeleteInput) error
}

// CreateInput is the input for creating a todo.
type CreateInput struct {
    Title string `json:"title" required:"true"`
    Body  string `json:"body"`
}

// GetInput identifies a todo by ID.
type GetInput struct {
    ID string `json:"id"`
}

// UpdateInput modifies a todo.
type UpdateInput struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    Body      string `json:"body"`
    Completed bool   `json:"completed"`
}

// DeleteInput identifies a todo to delete.
type DeleteInput struct {
    ID string `json:"id"`
}

// Todo is a todo item.
type Todo struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    Body      string `json:"body"`
    Completed bool   `json:"completed"`
}

// TodoList is a list of todos.
type TodoList struct {
    Items []*Todo `json:"items"`
    Total int     `json:"total"`
}
```

### Implementation

```go
package main

import (
    "context"
    "net/http"

    "github.com/mizu-framework/mizu"
    contract "github.com/mizu-framework/mizu/contract/v2"
    "github.com/mizu-framework/mizu/contract/v2/transport/rest"
)

// todoService implements TodoAPI.
type todoService struct {
    todos map[string]*api.Todo
}

func (s *todoService) Create(ctx context.Context, in *api.CreateInput) (*api.Todo, error) {
    todo := &api.Todo{
        ID:    generateID(),
        Title: in.Title,
        Body:  in.Body,
    }
    s.todos[todo.ID] = todo
    return todo, nil
}

func (s *todoService) List(ctx context.Context) (*api.TodoList, error) {
    items := make([]*api.Todo, 0, len(s.todos))
    for _, t := range s.todos {
        items = append(items, t)
    }
    return &api.TodoList{Items: items, Total: len(items)}, nil
}

// ... other methods ...

func main() {
    impl := &todoService{todos: make(map[string]*api.Todo)}

    // Register the service (code-first)
    svc := contract.Register[api.TodoAPI](impl,
        contract.WithName("Todo"),
        contract.WithDescription("Todo management service"),
        contract.WithResource("todos", "Create", "List", "Get", "Update", "Delete"),
        contract.WithDefaults(contract.Defaults{
            BaseURL: "https://api.example.com",
        }),
    )

    // Mount REST transport
    restServer, _ := rest.NewServer(svc)

    // Optionally generate YAML
    yaml, _ := svc.YAML()
    os.WriteFile("api.yaml", yaml, 0644)

    // Start server
    app := mizu.New()
    app.Mount("/api", restServer.Handler())
    app.Listen(":8080")
}
```

### Generated YAML

```yaml
name: Todo
description: Todo management service

defaults:
  base_url: https://api.example.com

resources:
  - name: todos
    methods:
      - name: create
        description: Create creates a new todo item.
        input: CreateInput
        output: Todo
        http:
          method: POST
          path: /todos

      - name: list
        description: List returns all todo items.
        output: TodoList
        http:
          method: GET
          path: /todos

      - name: get
        description: Get retrieves a todo by ID.
        input: GetInput
        output: Todo
        http:
          method: GET
          path: /todos/{id}

      - name: update
        description: Update modifies an existing todo.
        input: UpdateInput
        output: Todo
        http:
          method: PUT
          path: /todos/{id}

      - name: delete
        description: Delete removes a todo.
        input: DeleteInput
        http:
          method: DELETE
          path: /todos/{id}

types:
  - name: CreateInput
    kind: struct
    fields:
      - name: title
        type: string
      - name: body
        type: string
        optional: true

  - name: GetInput
    kind: struct
    fields:
      - name: id
        type: string

  - name: UpdateInput
    kind: struct
    fields:
      - name: id
        type: string
      - name: title
        type: string
        optional: true
      - name: body
        type: string
        optional: true
      - name: completed
        type: bool
        optional: true

  - name: DeleteInput
    kind: struct
    fields:
      - name: id
        type: string

  - name: Todo
    kind: struct
    fields:
      - name: id
        type: string
      - name: title
        type: string
      - name: body
        type: string
      - name: completed
        type: bool

  - name: TodoList
    kind: struct
    fields:
      - name: items
        type: "[]Todo"
      - name: total
        type: int
```

## Streaming Support

### Server-Sent Events (SSE)

```go
type StreamAPI interface {
    // Watch streams todo updates via SSE.
    Watch(ctx context.Context, in *WatchInput) (<-chan *TodoEvent, error)
}

// Registration with streaming
svc := contract.Register[StreamAPI](impl,
    contract.WithStreaming("Watch", contract.StreamSSE),
)
```

### WebSocket Bidirectional

```go
type ChatAPI interface {
    // Chat enables bidirectional messaging.
    Chat(ctx context.Context, in <-chan *Message) (<-chan *Message, error)
}

svc := contract.Register[ChatAPI](impl,
    contract.WithStreaming("Chat", contract.StreamWS),
)
```

## Implementation Plan

### Phase 1: Core Registration

1. **`register.go`** - Register function with generics
2. **`reflect.go`** - Interface and type reflection
3. **`invoker.go`** - Auto-generated invoker from reflection
4. **`options.go`** - Option functions

### Phase 2: Type Extraction

1. **`types.go`** - Go type to contract Type conversion
2. **`fields.go`** - Struct field extraction with tags
3. **`unions.go`** - Discriminated union detection

### Phase 3: HTTP Binding

1. **`http.go`** - Convention-based HTTP method inference
2. **`path.go`** - Path parameter extraction

### Phase 4: Streaming

1. **`stream.go`** - Channel-based streaming detection
2. Integration with existing transport stream support

## File Structure

```
contract/v2/
├── contract.go          # Existing: Core types
├── register.go          # NEW: Register function + type extraction
├── options.go           # NEW: Option functions
├── http.go              # NEW: HTTP binding inference
└── transport/           # Existing: Transport implementations

cmd/cli/
└── contract_v2.go       # YAML marshaling (using gopkg.in/yaml.v3)
```

Note: The core `contract/v2` package remains **dependency-free**. YAML generation
is handled by user code or the CLI using `gopkg.in/yaml.v3` to marshal the
`*Service` descriptor returned by `RegisteredService.Descriptor()`.

### YAML Generation Example

```go
import (
    "os"
    contract "github.com/go-mizu/mizu/contract/v2"
    "gopkg.in/yaml.v3"
)

func main() {
    svc := contract.Register[TodoAPI](impl, opts...)

    // Get the service descriptor
    desc := svc.Descriptor()

    // Marshal to YAML (requires gopkg.in/yaml.v3)
    yamlBytes, _ := yaml.Marshal(desc)
    os.WriteFile("api.yaml", yamlBytes, 0644)
}
```

## Compatibility

The code-first approach is fully compatible with definition-first:

1. **Same `Service` type** - Both produce identical `*Service` descriptors
2. **Same transports** - REST, JSON-RPC, async all work unchanged
3. **Same spec generation** - OpenAPI, OpenRPC, AsyncAPI generation works
4. **Interop** - Can mix code-first and definition-first in same app

## Migration Path

Existing definition-first users can:

1. **Continue using YAML** - No changes required
2. **Gradually adopt code-first** - For new services
3. **Generate YAML from code** - Use `svc.YAML()` to bootstrap YAML

New users can:

1. **Start with code-first** - Fastest path to working API
2. **Export YAML when needed** - For multi-language code generation
3. **Switch to definition-first** - If YAML becomes source of truth

## Summary

The code-first approach adds a `Register[T]` function that:

1. Takes a Go interface `T` and implementation
2. Reflects methods to build `*Service` descriptor
3. Extracts types from method signatures
4. Infers HTTP bindings from conventions
5. Returns `*RegisteredService` implementing `Invoker`
6. Enables YAML generation via `svc.YAML()`

This provides a zero-friction developer experience while maintaining full compatibility with contract v2's transport-neutral architecture.
