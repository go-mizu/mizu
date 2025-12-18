# Spec 0058: Enhance Contract Documentation

## Overview

Enhance all documentation files in `docs/contract/*.mdx` to improve:
1. **Coding style** - Use package-based naming conventions
2. **Writing style** - Add more detailed explanations for absolute beginners

## Coding Style Changes

### Before (Current Style)
```go
type TodoAPI interface {
    Create(ctx context.Context, in *CreateInput) (*Todo, error)
}

type todoService struct {
    // ...
}

func (s *todoService) Create(...) {...}
```

### After (Package-Based Style)
```go
package todo

// API defines the contract for todo operations.
// This is the interface that your implementation must satisfy.
type API interface {
    Create(ctx context.Context, in *CreateInput) (*Todo, error)
}

// Service implements the API interface.
// It contains the actual business logic for managing todos.
type Service struct {
    // ...
}

func (s *Service) Create(...) {...}
```

### Rationale
- **Cleaner naming**: Within a package, `todo.API` and `todo.Service` are more idiomatic than `TodoAPI` and `todoService`
- **Better organization**: Encourages developers to organize code into proper packages
- **Reduced redundancy**: Package name provides context, so type names don't need prefixes
- **Go conventions**: Follows standard Go naming conventions (exported types are PascalCase)

## Writing Style Changes

### Before (Current Style)
Brief paragraphs that assume prior knowledge:
```markdown
## Quick Start

Register your service:

\```go
svc := contract.Register[TodoAPI](impl)
\```
```

### After (Beginner-Friendly Style)
Detailed explanations that guide beginners step by step:
```markdown
## Quick Start

Before you can use your service with Contract, you need to register it. Registration
is the process where Contract analyzes your Go interface and implementation, discovers
all the methods, and prepares them to be called through different protocols like REST
or JSON-RPC.

Think of registration like telling Contract: "Here's my interface (the blueprint) and
my implementation (the actual code). Please make it available to clients."

\```go
// Register takes two things:
// 1. A type parameter [todo.API] - this is your interface that defines the API contract
// 2. An implementation (&todo.Service{}) - this is the struct that contains your business logic
svc := contract.Register[todo.API](&todo.Service{})
\```

After registration, `svc` is a `*RegisteredService` that can be mounted on any transport
(REST, JSON-RPC, MCP). The registration only happens once when your server starts.
```

### Principles for Beginner-Friendly Writing
1. **Explain the "why"**: Don't just show code, explain why we do things this way
2. **Define terms**: When introducing a concept, define what it means
3. **Use analogies**: Compare technical concepts to everyday things
4. **Step by step**: Break down complex operations into discrete steps
5. **Show cause and effect**: Explain what happens as a result of each action
6. **Anticipate questions**: Address common "but what if..." questions inline

## Files to Update

| File | Priority | Key Changes |
|------|----------|-------------|
| `overview.mdx` | High | Update intro examples, add beginner context |
| `quick-start.mdx` | High | Detailed step-by-step with package structure |
| `service.mdx` | High | Package-based naming, method signature explanations |
| `register.mdx` | High | Explain registration process in detail |
| `invoker.mdx` | Medium | Explain what invokers are and why they matter |
| `types.mdx` | Medium | Type mapping explanations for beginners |
| `errors.mdx` | High | Error handling philosophy and patterns |
| `error-codes.mdx` | Medium | When to use each error code |
| `transports-overview.mdx` | High | Protocol comparisons for beginners |
| `rest.mdx` | High | REST concepts for beginners |
| `jsonrpc.mdx` | Medium | JSON-RPC concepts explained |
| `mcp.mdx` | Medium | MCP and AI tool concepts |
| `openapi.mdx` | Medium | API documentation concepts |
| `middleware.mdx` | Medium | Middleware pattern for beginners |
| `testing.mdx` | High | Testing philosophy and patterns |
| `architecture.mdx` | Medium | System design explanations |
| `api-reference.mdx` | Low | Keep concise but add usage examples |
| `client-generation.mdx` | Low | Add beginner context |

## Example Transformations

### Example 1: Service Definition

**Before:**
```go
type TodoAPI interface {
    Create(ctx context.Context, in *CreateInput) (*Todo, error)
}

type todoService struct{}

func (s *todoService) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
    return &Todo{Title: in.Title}, nil
}
```

**After:**
```go
package todo

// API is the interface that defines all operations for managing todos.
// Any struct that implements all these methods can be used as a todo service.
// Contract uses this interface to understand what methods your API provides.
type API interface {
    // Create adds a new todo item to the system.
    // It takes a CreateInput containing the todo details and returns the created Todo.
    Create(ctx context.Context, in *CreateInput) (*Todo, error)
}

// Service implements the todo.API interface.
// This is where your actual business logic lives.
// It can have dependencies like databases, caches, or external services.
type Service struct {
    // Add your dependencies here, for example:
    // db *sql.DB
    // cache *redis.Client
}

// Create implements the todo.API interface.
// It validates the input, creates a new todo, and returns it.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
    // Validate input - always check user-provided data
    if in.Title == "" {
        return nil, contract.ErrInvalidArgument("title is required")
    }

    // Create the todo with generated ID
    todo := &Todo{
        ID:    generateID(),
        Title: in.Title,
    }

    return todo, nil
}
```

### Example 2: Registration

**Before:**
```go
svc := contract.Register[TodoAPI](impl,
    contract.WithDefaultResource("todos"),
)
```

**After:**
```go
// Create an instance of your service implementation.
// This is the struct that contains your business logic.
impl := &todo.Service{}

// Register the service with Contract.
// The type parameter [todo.API] tells Contract which interface to use.
// The implementation (impl) is the struct that implements that interface.
svc := contract.Register[todo.API](impl,
    // WithDefaultResource groups all methods under the "todos" resource.
    // This affects URL paths in REST (/todos, /todos/{id})
    // and method names in JSON-RPC (todos.create, todos.list).
    contract.WithDefaultResource("todos"),
)

// svc is now a *RegisteredService that can be mounted on any transport
```

## Implementation Plan

1. **Phase 1**: Update high-priority files (overview, quick-start, service, register, errors, testing)
2. **Phase 2**: Update medium-priority files (types, invoker, transports, middleware)
3. **Phase 3**: Update remaining files (api-reference, client-generation)

## Success Criteria

- [ ] All code examples use `package todo` with `API` and `Service` naming
- [ ] All paragraphs provide context and explanation suitable for beginners
- [ ] Technical terms are defined when first introduced
- [ ] Complex concepts include analogies or step-by-step breakdowns
- [ ] Each file follows consistent formatting and structure
