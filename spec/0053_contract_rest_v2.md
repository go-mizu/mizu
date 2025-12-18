# Contract REST v2: Mizu Integration

## Overview

This specification redesigns `contract/v2/transport/rest` to integrate natively with `mizu.Handler` and `mizu.Router`, providing excellent developer experience while maintaining Go standard library conventions.

## Goals

1. **Native mizu integration** - Use `mizu.Handler` and `mizu.Ctx` directly
2. **Minimal API surface** - Follow Go standard library design philosophy
3. **Composable** - Work with mizu middleware system
4. **Idiomatic naming** - Follow Go conventions (no stuttering, clear names)
5. **Zero magic** - Explicit routing, predictable behavior

## Current Problems

1. **No mizu integration** - Returns `http.Handler`, not `mizu.Handler`
2. **Separate router** - Uses internal `http.ServeMux`, can't share with mizu.Router
3. **No middleware support** - Can't apply mizu middleware to contract routes
4. **No access to `mizu.Ctx`** - Contract handlers can't use Ctx helpers

## Design

### Core API

```go
package rest

// Mount registers all contract routes on a mizu router.
// This is the primary API for integrating contracts with mizu.
func Mount(r *mizu.Router, inv contract.Invoker) error

// MountAt registers contract routes under a path prefix.
func MountAt(r *mizu.Router, prefix string, inv contract.Invoker) error

// Handler returns a single mizu.Handler for all contract routes.
// Useful when you need manual control over routing.
func Handler(inv contract.Invoker) (mizu.Handler, error)

// Routes returns route definitions for manual registration.
// Each Route contains Method, Path, and Handler.
func Routes(inv contract.Invoker) ([]Route, error)
```

### Route Type

```go
// Route represents a single HTTP endpoint from a contract.
type Route struct {
    Method   string       // HTTP method (GET, POST, etc.)
    Path     string       // URL path with {param} placeholders
    Resource string       // Contract resource name
    Name     string       // Contract method name
    Handler  mizu.Handler // Handler function
}
```

### Usage Examples

#### Simple Mount

```go
// Register service
svc := contract.Register[TodoAPI](impl)

// Mount on router
r := mizu.NewRouter()
rest.Mount(r, svc)

// Or with prefix
rest.MountAt(r, "/api/v1", svc)
```

#### With Middleware

```go
r := mizu.NewRouter()

// Apply auth middleware to all contract routes
api := r.Prefix("/api").With(authMiddleware)
rest.Mount(api, svc)
```

#### Manual Route Registration

```go
routes, _ := rest.Routes(svc)
for _, rt := range routes {
    // Customize individual routes
    r.Handle(rt.Method, rt.Path, rt.Handler)
}
```

#### Single Handler (for mounting on http.ServeMux)

```go
handler, _ := rest.Handler(svc)
mux.Handle("/api/", mizu.Adapt(handler)) // Using mizu.Adapt helper
```

### Path Translation

Contract paths use `{param}` syntax, translated to Go 1.22+ patterns:

| Contract Path | Mizu/ServeMux Pattern |
|--------------|----------------------|
| `/todos` | `/todos` |
| `/todos/{id}` | `/todos/{id}` |
| `/users/{user_id}/posts/{post_id}` | `/users/{user_id}/posts/{post_id}` |

Note: Go 1.22+ ServeMux uses the same `{param}` syntax, so no translation needed.

### Input Binding

The handler extracts input from multiple sources:

1. **Path parameters** - Via `c.Param(name)` from mizu.Ctx
2. **Query parameters** - Via `c.QueryValues()` for GET requests
3. **JSON body** - Via `c.BindJSON()` for non-GET requests

```go
// Binding priority for a GET request:
// 1. Path params override everything
// 2. Query params fill remaining fields

// Binding priority for POST/PUT/PATCH:
// 1. Path params override everything
// 2. JSON body fills remaining fields
// 3. Query params can supplement (rare)
```

### Response Handling

```go
// Success with output
c.JSON(200, output)

// Success without output
c.NoContent()

// Errors
c.JSON(statusCode, ErrorResponse{
    Error:   "error_code",
    Message: "Human readable message",
})
```

### Error Mapping

Contract errors are mapped to HTTP status codes:

```go
// ErrorMapper customizes error-to-status mapping.
type ErrorMapper func(error) (status int, code string, message string)

// DefaultErrorMapper returns 400 for all errors.
func DefaultErrorMapper(err error) (int, string, string) {
    return http.StatusBadRequest, "request_error", err.Error()
}

// WithErrorMapper sets a custom error mapper.
func WithErrorMapper(m ErrorMapper) Option
```

### Options

```go
// Option configures REST transport behavior.
type Option func(*options)

// WithErrorMapper sets custom error mapping.
func WithErrorMapper(m ErrorMapper) Option

// WithMaxBodySize limits request body size (default: 1MB).
func WithMaxBodySize(n int64) Option

// WithStrictRouting disables query param fallback for non-GET.
func WithStrictRouting() Option
```

## Implementation

### Handler Factory

```go
func makeHandler(inv contract.Invoker, rt route, opts *options) mizu.Handler {
    return func(c *mizu.Ctx) error {
        ctx := c.Context()

        // Create input if method has one
        var in any
        if rt.hasInput {
            var err error
            in, err = inv.NewInput(rt.resource, rt.method)
            if err != nil {
                return c.JSON(500, errorResponse("internal_error", err.Error()))
            }

            // Fill from path params
            if err := fillFromPath(in, c, rt.pathParams); err != nil {
                return c.JSON(400, errorResponse("bad_path", err.Error()))
            }

            // Fill from query (GET) or body (other methods)
            if c.Request().Method == http.MethodGet {
                if err := fillFromQuery(in, c.QueryValues()); err != nil {
                    return c.JSON(400, errorResponse("bad_query", err.Error()))
                }
            } else if c.Request().ContentLength > 0 {
                if err := c.BindJSON(in, opts.maxBodySize); err != nil {
                    return c.JSON(400, errorResponse("bad_json", err.Error()))
                }
            }
        }

        // Invoke contract method
        out, err := inv.Call(ctx, rt.resource, rt.method, in)
        if err != nil {
            status, code, msg := opts.errorMapper(err)
            return c.JSON(status, errorResponse(code, msg))
        }

        // Write response
        if !rt.hasOutput {
            return c.NoContent()
        }
        return c.JSON(200, out)
    }
}
```

### Path Parameter Extraction

```go
func fillFromPath(dst any, c *mizu.Ctx, params []string) error {
    v := reflect.ValueOf(dst).Elem()
    t := v.Type()

    for _, param := range params {
        value := c.Param(param)
        if value == "" {
            continue
        }

        fi, ok := findField(t, param)
        if !ok {
            continue
        }

        if err := setField(v.Field(fi), value); err != nil {
            return fmt.Errorf("%s: %w", param, err)
        }
    }
    return nil
}
```

### Route Registration

```go
func Mount(r *mizu.Router, inv contract.Invoker, opts ...Option) error {
    routes, err := buildRoutes(inv, mergeOptions(opts))
    if err != nil {
        return err
    }

    for _, rt := range routes {
        r.Handle(rt.Method, rt.Path, rt.Handler)
    }
    return nil
}

func MountAt(r *mizu.Router, prefix string, inv contract.Invoker, opts ...Option) error {
    return Mount(r.Prefix(prefix), inv, opts...)
}
```

## Naming Conventions

### Package-Level Functions

| Name | Purpose |
|------|---------|
| `Mount` | Register routes on router |
| `MountAt` | Register routes under prefix |
| `Handler` | Get single handler for all routes |
| `Routes` | Get route definitions |
| `OpenAPI` | Generate OpenAPI spec |

### Types

| Name | Purpose |
|------|---------|
| `Route` | Single route definition |
| `Option` | Configuration function |
| `ErrorMapper` | Error to HTTP mapping |

### Removed/Renamed

| Old | New | Reason |
|-----|-----|--------|
| `Server` | (removed) | Not needed with Mount |
| `NewServer` | `Mount` | Simpler API |
| `Server.Handler()` | `Handler()` | Package-level function |

## File Structure

```
contract/v2/transport/rest/
├── mount.go       # Mount, MountAt, Handler, Routes
├── handler.go     # Handler factory and input binding
├── route.go       # Route type and building
├── bind.go        # Input binding helpers
├── options.go     # Option types and defaults
├── openapi.go     # OpenAPI generation (unchanged)
├── client.go      # HTTP client (unchanged)
└── rest_test.go   # Comprehensive tests
```

## Tests

### Unit Tests

```go
func TestMount(t *testing.T)
func TestMountAt(t *testing.T)
func TestHandler(t *testing.T)
func TestRoutes(t *testing.T)
func TestPathParams(t *testing.T)
func TestQueryParams(t *testing.T)
func TestJSONBody(t *testing.T)
func TestErrorMapping(t *testing.T)
func TestWithMiddleware(t *testing.T)
func TestNoInput(t *testing.T)
func TestNoOutput(t *testing.T)
```

### Integration Tests

```go
func TestTodoAPI(t *testing.T)
func TestCRUDOperations(t *testing.T)
func TestMiddlewareChain(t *testing.T)
```

## CLI Template Changes

### Before (server.go.tmpl)

```go
mux := http.NewServeMux()
restServer, err := rest.NewServer(invoker)
mux.Handle("/", restServer.Handler())
```

### After (server.go.tmpl)

```go
r := mizu.NewRouter()
rest.Mount(r, svc)
// or with prefix:
rest.MountAt(r, "/api", svc)
```

### Updated Template Structure

```
templates/contract/
├── app/server/
│   ├── server.go.tmpl   # Uses mizu.Router + rest.Mount
│   └── config.go.tmpl   # (unchanged)
├── service/todo/
│   ├── api.go.tmpl      # Interface definition (NEW)
│   ├── service.go.tmpl  # Implementation (renamed from todo.go)
│   └── service_test.go.tmpl
├── cmd/api/
│   └── main.go.tmpl     # Uses mizu.App
└── go.mod.tmpl
```

### Code-First Template (api.go.tmpl)

```go
package todo

import "context"

// API defines the Todo service contract.
type API interface {
    Create(ctx context.Context, in *CreateInput) (*Todo, error)
    List(ctx context.Context) (*ListOutput, error)
    Get(ctx context.Context, in *GetInput) (*Todo, error)
    Update(ctx context.Context, in *UpdateInput) (*Todo, error)
    Delete(ctx context.Context, in *DeleteInput) error
}
```

### Server Template (server.go.tmpl)

```go
package server

import (
    "github.com/go-mizu/mizu"
    contract "github.com/go-mizu/mizu/contract/v2"
    "github.com/go-mizu/mizu/contract/v2/transport/rest"
    "{{.Module}}/service/todo"
)

func New(cfg Config, todoSvc *todo.Service) (*mizu.App, error) {
    // Register service using code-first approach
    svc := contract.Register[todo.API](todoSvc,
        contract.WithResource("todos"),
    )

    // Create mizu app
    app := mizu.New()

    // Mount REST API with middleware
    api := app.Router.Prefix("/api")
    rest.Mount(api, svc)

    // Serve OpenAPI spec
    app.Router.Get("/openapi.json", func(c *mizu.Ctx) error {
        spec, err := rest.OpenAPI(svc.Descriptor())
        if err != nil {
            return c.JSON(500, map[string]string{"error": err.Error()})
        }
        return c.JSON(200, spec)
    })

    return app, nil
}
```

## Migration Guide

### From v1 (http.Handler based)

```go
// Before
restServer, _ := rest.NewServer(invoker)
mux.Handle("/", restServer.Handler())

// After
rest.Mount(r, invoker)
```

### Adding Middleware

```go
// Before: Not possible without wrapping http.Handler

// After
api := r.Prefix("/api").With(authMiddleware, logMiddleware)
rest.Mount(api, invoker)
```

### Custom Error Handling

```go
// Before: Modify source code

// After
rest.Mount(r, invoker, rest.WithErrorMapper(func(err error) (int, string, string) {
    if errors.Is(err, ErrNotFound) {
        return 404, "not_found", "Resource not found"
    }
    return 500, "internal_error", err.Error()
}))
```

## Summary

This redesign:

1. **Integrates with mizu** - Uses `mizu.Handler` and `mizu.Ctx` natively
2. **Simplifies API** - Single `Mount` function replaces `Server` type
3. **Enables middleware** - Contract routes work with mizu middleware
4. **Follows conventions** - Go standard library naming, no stuttering
5. **Maintains compatibility** - Same input binding, same OpenAPI generation
6. **Improves templates** - Code-first approach, cleaner structure
