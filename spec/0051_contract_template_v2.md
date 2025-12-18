# Contract Template v2 Specification

**Status:** Implementation Ready
**Depends on:** 0049_contract_v2.md, 0050_contract_cli_v2.md

## Overview

This spec defines the upgrade of the `contract` template in `cmd/cli/templates/contract/` to use the new contract/v2 definition-first approach. The template will include:

1. A YAML-based contract definition (`api.yaml`)
2. Server implementation using contract/v2 transports
3. Comprehensive test suite for the generated service
4. Makefile with standard commands

## Current State (v1)

The current contract template uses contract/v1's reflection-based approach:

- `service/todo/todo.go` - Plain Go service with method signatures
- `app/server/server.go` - Uses `contract.Register()` to extract metadata at runtime
- `cmd/api/main.go` - Entry point

The v1 approach has limitations:
- No explicit contract definition
- Type information extracted via reflection
- No support for streaming, unions, or advanced type features
- Less portable (Go-only)

## Target State (v2)

The v2 template will use a definition-first approach:

```
contract/
├── api.yaml                    # Contract definition (NEW)
├── Makefile                    # Build, test, run commands (NEW)
├── readme.md                   # Documentation
├── .gitignore
├── go.mod.tmpl
├── app/
│   └── server/
│       ├── config.go
│       └── server.go           # Uses contract/v2 transports
├── cmd/
│   └── api/
│       └── main.go
└── service/
    └── todo/
        ├── todo.go             # Service implementation
        └── todo_test.go        # Service tests (NEW)
```

## Contract Definition (api.yaml)

The template will include a complete contract definition:

```yaml
name: Todo
description: A simple todo list API demonstrating contract/v2 features.

defaults:
  base_url: http://localhost:8080
  auth: bearer
  headers:
    content-type: application/json

resources:
  - name: todos
    description: Manage todo items.
    methods:
      - name: list
        description: List all todos.
        output: TodoList
        http:
          method: GET
          path: /todos

      - name: create
        description: Create a new todo.
        input: CreateTodoRequest
        output: Todo
        http:
          method: POST
          path: /todos

      - name: get
        description: Get a todo by ID.
        input: GetTodoRequest
        output: Todo
        http:
          method: GET
          path: /todos/{id}

      - name: update
        description: Update an existing todo.
        input: UpdateTodoRequest
        output: Todo
        http:
          method: PUT
          path: /todos/{id}

      - name: delete
        description: Delete a todo.
        input: DeleteTodoRequest
        http:
          method: DELETE
          path: /todos/{id}

  - name: health
    description: Health check endpoints.
    methods:
      - name: check
        description: Check service health.
        output: HealthStatus
        http:
          method: GET
          path: /health

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
      - name: priority
        type: string
        enum: ["low", "medium", "high"]
        optional: true

  - name: TodoList
    kind: struct
    fields:
      - name: items
        type: Todos
      - name: total
        type: int

  - name: Todos
    kind: slice
    elem: Todo

  - name: CreateTodoRequest
    kind: struct
    fields:
      - name: title
        type: string
      - name: priority
        type: string
        enum: ["low", "medium", "high"]
        optional: true

  - name: GetTodoRequest
    kind: struct
    fields:
      - name: id
        type: string

  - name: UpdateTodoRequest
    kind: struct
    fields:
      - name: id
        type: string
      - name: title
        type: string
        optional: true
      - name: completed
        type: bool
        optional: true
      - name: priority
        type: string
        enum: ["low", "medium", "high"]
        optional: true

  - name: DeleteTodoRequest
    kind: struct
    fields:
      - name: id
        type: string

  - name: HealthStatus
    kind: struct
    fields:
      - name: status
        type: string
        enum: ["ok", "degraded", "unhealthy"]
      - name: version
        type: string
        optional: true
```

## Server Implementation Changes

### server.go

```go
package server

import (
    "embed"
    "net/http"

    "github.com/go-mizu/mizu/contract/v2"
    "github.com/go-mizu/mizu/contract/v2/transport/rest"
    "github.com/go-mizu/mizu/contract/v2/transport/jsonrpc"
    "gopkg.in/yaml.v3"
    "{{.Module}}/service/todo"
)

//go:embed api.yaml
var apiYAML []byte

// Server wraps the HTTP server with all transports.
type Server struct {
    cfg      Config
    server   *http.Server
    contract *contract.Service
}

// New creates a server with REST and JSON-RPC transports.
func New(cfg Config, todoSvc *todo.Service) (*Server, error) {
    // Load contract from embedded YAML
    var svc contract.Service
    if err := yaml.Unmarshal(apiYAML, &svc); err != nil {
        return nil, err
    }

    // Create invoker that dispatches to the service
    invoker := todo.NewInvoker(todoSvc, &svc)

    mux := http.NewServeMux()

    // Mount REST endpoints
    rest.Mount(mux, invoker)

    // Mount JSON-RPC 2.0 endpoint
    jsonrpc.Mount(mux, "/rpc", invoker)

    // Serve OpenAPI 3.1 spec
    mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
        spec, err := rest.OpenAPIDocument(&svc)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write(spec)
    })

    return &Server{
        cfg: cfg,
        server: &http.Server{
            Addr:    cfg.Addr,
            Handler: mux,
        },
        contract: &svc,
    }, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
    return s.server.ListenAndServe()
}

// Close shuts down the server.
func (s *Server) Close() error {
    return s.server.Close()
}

// Contract returns the loaded contract service.
func (s *Server) Contract() *contract.Service {
    return s.contract
}
```

## Service Implementation

### service/todo/todo.go

The service remains a plain Go struct with method implementations:

```go
package todo

import (
    "context"
    "errors"
    "sync"
)

// Priority levels for todos.
type Priority string

const (
    PriorityLow    Priority = "low"
    PriorityMedium Priority = "medium"
    PriorityHigh   Priority = "high"
)

// Service is the todo business logic.
type Service struct {
    mu    sync.RWMutex
    todos map[string]*Todo
    seq   int
}

// Todo represents a todo item.
type Todo struct {
    ID        string   `json:"id"`
    Title     string   `json:"title"`
    Completed bool     `json:"completed"`
    Priority  Priority `json:"priority,omitempty"`
}

// ... (method implementations)
```

### service/todo/invoker.go (NEW)

Bridge between contract/v2 Invoker interface and the service:

```go
package todo

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/go-mizu/mizu/contract/v2"
)

// Invoker implements contract.Invoker for the todo service.
type Invoker struct {
    svc      *Service
    contract *contract.Service
}

// NewInvoker creates an invoker for the todo service.
func NewInvoker(svc *Service, c *contract.Service) *Invoker {
    return &Invoker{svc: svc, contract: c}
}

// Descriptor returns the service descriptor.
func (i *Invoker) Descriptor() *contract.Service {
    return i.contract
}

// Call dispatches a method call to the service.
func (i *Invoker) Call(ctx context.Context, resource, method string, in any) (any, error) {
    switch resource + "." + method {
    case "todos.list":
        return i.svc.List(ctx)
    case "todos.create":
        req, err := decodeInput[CreateIn](in)
        if err != nil {
            return nil, err
        }
        return i.svc.Create(ctx, req)
    case "todos.get":
        req, err := decodeInput[GetIn](in)
        if err != nil {
            return nil, err
        }
        return i.svc.Get(ctx, req)
    case "todos.update":
        req, err := decodeInput[UpdateIn](in)
        if err != nil {
            return nil, err
        }
        return i.svc.Update(ctx, req)
    case "todos.delete":
        req, err := decodeInput[DeleteIn](in)
        if err != nil {
            return nil, err
        }
        return i.svc.Delete(ctx, req)
    case "health.check":
        if err := i.svc.Health(ctx); err != nil {
            return &HealthStatus{Status: "unhealthy"}, nil
        }
        return &HealthStatus{Status: "ok"}, nil
    default:
        return nil, fmt.Errorf("unknown method: %s.%s", resource, method)
    }
}

// NewInput creates a new input instance for a method.
func (i *Invoker) NewInput(resource, method string) (any, error) {
    switch resource + "." + method {
    case "todos.create":
        return &CreateIn{}, nil
    case "todos.get":
        return &GetIn{}, nil
    case "todos.update":
        return &UpdateIn{}, nil
    case "todos.delete":
        return &DeleteIn{}, nil
    default:
        return nil, nil
    }
}

// Stream is not supported for this service.
func (i *Invoker) Stream(ctx context.Context, resource, method string, in any) (contract.Stream, error) {
    return nil, contract.ErrUnsupported
}

func decodeInput[T any](in any) (*T, error) {
    if in == nil {
        return new(T), nil
    }
    if v, ok := in.(*T); ok {
        return v, nil
    }
    // Handle map[string]any from JSON decode
    data, err := json.Marshal(in)
    if err != nil {
        return nil, err
    }
    var v T
    if err := json.Unmarshal(data, &v); err != nil {
        return nil, err
    }
    return &v, nil
}
```

## Test Suite

### service/todo/todo_test.go (NEW)

```go
package todo

import (
    "context"
    "testing"
)

func TestService_Create(t *testing.T) {
    svc := &Service{}
    ctx := context.Background()

    t.Run("creates todo with title", func(t *testing.T) {
        todo, err := svc.Create(ctx, &CreateIn{Title: "Test todo"})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if todo.Title != "Test todo" {
            t.Errorf("got title %q, want %q", todo.Title, "Test todo")
        }
        if todo.ID == "" {
            t.Error("expected non-empty ID")
        }
        if todo.Completed {
            t.Error("new todo should not be completed")
        }
    })

    t.Run("rejects empty title", func(t *testing.T) {
        _, err := svc.Create(ctx, &CreateIn{Title: ""})
        if err != ErrTitleEmpty {
            t.Errorf("got error %v, want ErrTitleEmpty", err)
        }
    })

    t.Run("creates todo with priority", func(t *testing.T) {
        todo, err := svc.Create(ctx, &CreateIn{
            Title:    "Priority todo",
            Priority: PriorityHigh,
        })
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if todo.Priority != PriorityHigh {
            t.Errorf("got priority %q, want %q", todo.Priority, PriorityHigh)
        }
    })
}

func TestService_Get(t *testing.T) {
    svc := &Service{}
    ctx := context.Background()

    // Create a todo first
    created, _ := svc.Create(ctx, &CreateIn{Title: "Get test"})

    t.Run("retrieves existing todo", func(t *testing.T) {
        todo, err := svc.Get(ctx, &GetIn{ID: created.ID})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if todo.ID != created.ID {
            t.Errorf("got ID %q, want %q", todo.ID, created.ID)
        }
    })

    t.Run("returns error for non-existent todo", func(t *testing.T) {
        _, err := svc.Get(ctx, &GetIn{ID: "nonexistent"})
        if err != ErrNotFound {
            t.Errorf("got error %v, want ErrNotFound", err)
        }
    })
}

func TestService_List(t *testing.T) {
    svc := &Service{}
    ctx := context.Background()

    t.Run("returns empty list initially", func(t *testing.T) {
        list, err := svc.List(ctx)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(list.Items) != 0 {
            t.Errorf("got %d items, want 0", len(list.Items))
        }
    })

    t.Run("returns all todos", func(t *testing.T) {
        svc.Create(ctx, &CreateIn{Title: "Todo 1"})
        svc.Create(ctx, &CreateIn{Title: "Todo 2"})

        list, err := svc.List(ctx)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(list.Items) != 2 {
            t.Errorf("got %d items, want 2", len(list.Items))
        }
    })
}

func TestService_Update(t *testing.T) {
    svc := &Service{}
    ctx := context.Background()

    created, _ := svc.Create(ctx, &CreateIn{Title: "Update test"})

    t.Run("updates title", func(t *testing.T) {
        updated, err := svc.Update(ctx, &UpdateIn{
            ID:    created.ID,
            Title: "Updated title",
        })
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if updated.Title != "Updated title" {
            t.Errorf("got title %q, want %q", updated.Title, "Updated title")
        }
    })

    t.Run("updates completed status", func(t *testing.T) {
        updated, err := svc.Update(ctx, &UpdateIn{
            ID:        created.ID,
            Completed: true,
        })
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if !updated.Completed {
            t.Error("expected completed to be true")
        }
    })

    t.Run("returns error for non-existent todo", func(t *testing.T) {
        _, err := svc.Update(ctx, &UpdateIn{ID: "nonexistent"})
        if err != ErrNotFound {
            t.Errorf("got error %v, want ErrNotFound", err)
        }
    })
}

func TestService_Delete(t *testing.T) {
    svc := &Service{}
    ctx := context.Background()

    created, _ := svc.Create(ctx, &CreateIn{Title: "Delete test"})

    t.Run("deletes existing todo", func(t *testing.T) {
        err := svc.Delete(ctx, &DeleteIn{ID: created.ID})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }

        // Verify deletion
        _, err = svc.Get(ctx, &GetIn{ID: created.ID})
        if err != ErrNotFound {
            t.Error("expected todo to be deleted")
        }
    })

    t.Run("returns error for non-existent todo", func(t *testing.T) {
        err := svc.Delete(ctx, &DeleteIn{ID: "nonexistent"})
        if err != ErrNotFound {
            t.Errorf("got error %v, want ErrNotFound", err)
        }
    })
}

func TestService_Health(t *testing.T) {
    svc := &Service{}
    ctx := context.Background()

    err := svc.Health(ctx)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### service/todo/invoker_test.go (NEW)

```go
package todo

import (
    "context"
    "testing"

    "github.com/go-mizu/mizu/contract/v2"
    "gopkg.in/yaml.v3"
)

func TestInvoker_Call(t *testing.T) {
    svc := &Service{}

    // Minimal contract for testing
    contractYAML := `
name: Todo
resources:
  - name: todos
    methods:
      - name: list
      - name: create
      - name: get
  - name: health
    methods:
      - name: check
`
    var c contract.Service
    if err := yaml.Unmarshal([]byte(contractYAML), &c); err != nil {
        t.Fatalf("failed to parse contract: %v", err)
    }

    invoker := NewInvoker(svc, &c)
    ctx := context.Background()

    t.Run("todos.list", func(t *testing.T) {
        result, err := invoker.Call(ctx, "todos", "list", nil)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if _, ok := result.(*TodoList); !ok {
            t.Errorf("expected *TodoList, got %T", result)
        }
    })

    t.Run("todos.create", func(t *testing.T) {
        result, err := invoker.Call(ctx, "todos", "create", map[string]any{
            "title": "Test",
        })
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        todo, ok := result.(*Todo)
        if !ok {
            t.Fatalf("expected *Todo, got %T", result)
        }
        if todo.Title != "Test" {
            t.Errorf("got title %q, want %q", todo.Title, "Test")
        }
    })

    t.Run("health.check", func(t *testing.T) {
        result, err := invoker.Call(ctx, "health", "check", nil)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        status, ok := result.(*HealthStatus)
        if !ok {
            t.Fatalf("expected *HealthStatus, got %T", result)
        }
        if status.Status != "ok" {
            t.Errorf("got status %q, want %q", status.Status, "ok")
        }
    })

    t.Run("unknown method returns error", func(t *testing.T) {
        _, err := invoker.Call(ctx, "unknown", "method", nil)
        if err == nil {
            t.Error("expected error for unknown method")
        }
    })
}

func TestInvoker_Stream(t *testing.T) {
    svc := &Service{}
    var c contract.Service
    invoker := NewInvoker(svc, &c)

    _, err := invoker.Stream(context.Background(), "todos", "list", nil)
    if err != contract.ErrUnsupported {
        t.Errorf("expected ErrUnsupported, got %v", err)
    }
}
```

## Makefile

```makefile
.PHONY: build test run clean dev

# Build the binary
build:
	go build -o bin/api ./cmd/api

# Run tests
test:
	go test ./...

# Run the server
run:
	go run ./cmd/api

# Run with hot reload (requires mizu CLI)
dev:
	mizu dev

# Generate OpenAPI spec
openapi:
	go run ./cmd/api -openapi > openapi.json

# Clean build artifacts
clean:
	rm -rf bin/
```

## Template Files Summary

| File | Purpose |
|------|---------|
| `template.json` | Template metadata and variables |
| `api.yaml.tmpl` | Contract definition (new) |
| `Makefile.tmpl` | Build and test commands (new) |
| `readme.md.tmpl` | Project documentation |
| `go.mod.tmpl` | Go module file |
| `gitignore.tmpl` | Git ignore patterns |
| `app/server/config.go.tmpl` | Server configuration |
| `app/server/server.go.tmpl` | HTTP server setup (updated for v2) |
| `cmd/api/main.go.tmpl` | Entry point |
| `service/todo/todo.go.tmpl` | Service implementation (updated) |
| `service/todo/invoker.go.tmpl` | Contract invoker (new) |
| `service/todo/todo_test.go.tmpl` | Service tests (new) |
| `service/todo/invoker_test.go.tmpl` | Invoker tests (new) |

## Template Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `{{.Name}}` | Project name | Directory name |
| `{{.Module}}` | Go module path | `example.com/<name>` |
| `{{.License}}` | License identifier | `MIT` |
| `{{.Year}}` | Current year | Auto-generated |

## Implementation Plan

### Phase 1: Template Files

1. Update `templates/contract/template.json` with new description
2. Create `templates/contract/api.yaml.tmpl` with todo contract
3. Create `templates/contract/Makefile.tmpl`
4. Update `templates/contract/app/server/server.go.tmpl` for v2
5. Update `templates/contract/service/todo/todo.go.tmpl` with Priority enum
6. Create `templates/contract/service/todo/invoker.go.tmpl`
7. Create `templates/contract/service/todo/todo_test.go.tmpl`
8. Create `templates/contract/service/todo/invoker_test.go.tmpl`

### Phase 2: Regenerate Sample

1. Remove existing `cmd/cli/samples/contract/`
2. Run `mizu new cmd/cli/samples/contract --template contract`
3. Verify sample compiles: `go build ./cmd/cli/samples/contract/...`
4. Verify tests pass: `go test ./cmd/cli/samples/contract/...`

### Phase 3: Validation

1. Ensure all template files render correctly
2. Verify OpenAPI spec generation works
3. Test REST and JSON-RPC endpoints manually
4. Update documentation if needed

## Migration Notes

- The v2 template is a breaking change from v1
- Users with existing v1 projects should not regenerate without reviewing changes
- The `contract.Register()` approach is replaced by explicit YAML + Invoker pattern
- Types are now defined in YAML, not inferred from Go structs

## Testing Strategy

1. **Unit tests** (`todo_test.go`): Test service methods in isolation
2. **Integration tests** (`invoker_test.go`): Test invoker dispatch logic
3. **HTTP tests** (optional, in server_test.go): Test REST/JSON-RPC endpoints

The test suite demonstrates:
- Table-driven tests with `t.Run()`
- Error case coverage
- State management across test cases
- Clean setup for each test
