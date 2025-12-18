# Contract JSON-RPC v2: Mizu Integration

**Status:** Implementation Ready
**Depends on:** 0053_contract_rest_v2.md

## Overview

This specification redesigns `contract/v2/transport/jsonrpc` to integrate natively with `mizu.Handler` and `mizu.Router`, mirroring the REST transport API for consistency and excellent developer experience.

## Goals

1. **Native mizu integration** - Use `mizu.Handler` directly, not `http.Handler`
2. **Minimal API surface** - Follow Go standard library design philosophy
3. **Consistent with REST** - Same API pattern: `Mount`, `Handler`, options
4. **Composable** - Work with mizu middleware system
5. **Idiomatic naming** - Follow Go conventions (no stuttering, clear names)

## Current Problems

1. **Returns `http.Handler`** - `Server.Handler()` returns `http.Handler`, not `mizu.Handler`
2. **Requires `Server` type** - Two-step: `NewServer()` then `.Handler()`
3. **No options** - No error mapper or configuration
4. **No mizu.Ctx access** - Can't use Ctx helpers for context propagation
5. **Inconsistent with REST** - Different API pattern than `rest.Mount()`

## Design

### Core API

```go
package jsonrpc

// Mount registers a JSON-RPC endpoint on a mizu router.
// The endpoint accepts POST requests with JSON-RPC 2.0 payloads.
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error

// Handler returns a mizu.Handler for JSON-RPC 2.0 requests.
// This is the primary API when you need direct control.
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error)

// OpenRPC generates an OpenRPC 1.2 specification from a contract descriptor.
func OpenRPC(svc *contract.Service) ([]byte, error)
```

### Removed Types

| Old | New | Reason |
|-----|-----|--------|
| `Server` | (removed) | Not needed with Mount/Handler |
| `NewServer` | `Handler` | Simpler, returns mizu.Handler |
| `Server.Handler()` | (removed) | Handler() returns directly |

### Usage Examples

#### Simple Mount

```go
// Register service
svc := contract.Register[TodoAPI](impl)

// Mount JSON-RPC endpoint
r := mizu.NewRouter()
jsonrpc.Mount(r, "/rpc", svc)
```

#### With Middleware

```go
r := mizu.NewRouter()

// Apply auth middleware to JSON-RPC endpoint
rpc := r.Prefix("/rpc").With(authMiddleware)
jsonrpc.Mount(rpc, "", svc)
```

#### Manual Handler Registration

```go
handler, _ := jsonrpc.Handler(svc)
r.Post("/rpc", handler)
```

#### With Options

```go
jsonrpc.Mount(r, "/rpc", svc,
    jsonrpc.WithErrorMapper(customMapper),
    jsonrpc.WithMaxBodySize(2<<20), // 2MB
)
```

### Options

```go
// Option configures JSON-RPC transport behavior.
type Option func(*options)

// WithErrorMapper sets custom error-to-JSON-RPC-error mapping.
// Default maps all errors to code -32000 (server error).
func WithErrorMapper(m ErrorMapper) Option

// WithMaxBodySize limits request body size (default: 1MB).
func WithMaxBodySize(n int64) Option
```

### Error Mapping

```go
// ErrorMapper converts Go errors to JSON-RPC error responses.
// Returns: code (JSON-RPC error code), message, data (optional)
type ErrorMapper func(error) (code int, message string, data any)

// DefaultErrorMapper returns -32000 (server error) for all errors.
func DefaultErrorMapper(err error) (int, string, any) {
    return -32000, "server error", err.Error()
}
```

### Standard JSON-RPC Error Codes

| Code | Name | Description |
|------|------|-------------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid Request | Missing jsonrpc/method |
| -32601 | Method not found | Unknown resource.method |
| -32602 | Invalid params | Params decode failure |
| -32603 | Internal error | Internal server error |
| -32000 to -32099 | Server error | Application errors |

## Implementation

### Handler Factory

```go
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
    if inv == nil {
        return nil, errors.New("jsonrpc: nil invoker")
    }
    svc := inv.Descriptor()
    if svc == nil {
        return nil, errors.New("jsonrpc: nil descriptor")
    }

    o := applyOptions(opts)

    return func(c *mizu.Ctx) error {
        if c.Request().Method != http.MethodPost {
            return writeError(c, nil, errInvalidRequest, "method not allowed", nil)
        }

        // Read and parse request body
        body, err := io.ReadAll(io.LimitReader(c.Request().Body, o.maxBodySize))
        if err != nil {
            return writeError(c, nil, errParse, "parse error", err.Error())
        }

        raw := strings.TrimSpace(string(body))
        if raw == "" {
            return writeError(c, nil, errInvalidRequest, "invalid request", "empty body")
        }

        // Handle batch or single request
        if raw[0] == '[' {
            return handleBatch(c, inv, svc, body, o)
        }
        return handleSingle(c, inv, svc, body, o)
    }, nil
}
```

### Mount Function

```go
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error {
    handler, err := Handler(inv, opts...)
    if err != nil {
        return err
    }
    if path == "" {
        path = "/"
    }
    r.Post(path, handler)
    return nil
}
```

### Single Request Handler

```go
func handleSingle(c *mizu.Ctx, inv contract.Invoker, svc *contract.Service, body []byte, o *options) error {
    var req request
    if err := json.Unmarshal(body, &req); err != nil {
        return writeError(c, nil, errInvalidRequest, "invalid request", err.Error())
    }

    resp := processRequest(c.Context(), inv, svc, &req, o)
    if resp == nil {
        // Notification (no id) - no response
        return c.NoContent()
    }

    return c.JSON(http.StatusOK, resp)
}
```

### Batch Request Handler

```go
func handleBatch(c *mizu.Ctx, inv contract.Invoker, svc *contract.Service, body []byte, o *options) error {
    var batch []json.RawMessage
    if err := json.Unmarshal(body, &batch); err != nil {
        return writeError(c, nil, errParse, "parse error", err.Error())
    }
    if len(batch) == 0 {
        return writeError(c, nil, errInvalidRequest, "invalid request", "empty batch")
    }

    var responses []any
    for _, raw := range batch {
        var req request
        if err := json.Unmarshal(raw, &req); err != nil {
            responses = append(responses, errorResponse(nil, errInvalidRequest, "invalid request", err.Error()))
            continue
        }
        if resp := processRequest(c.Context(), inv, svc, &req, o); resp != nil {
            responses = append(responses, resp)
        }
    }

    if len(responses) == 0 {
        // All notifications - no response
        return c.NoContent()
    }

    return c.JSON(http.StatusOK, responses)
}
```

### Wire Types

```go
// request is the JSON-RPC 2.0 request structure.
type request struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
    ID      any             `json:"id,omitempty"`
}

// response is the JSON-RPC 2.0 response structure.
type response struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      any             `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *rpcError       `json:"error,omitempty"`
}

// rpcError is the JSON-RPC 2.0 error structure.
type rpcError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

## File Structure

```
contract/v2/transport/jsonrpc/
├── jsonrpc.go      # Mount, Handler (main API)
├── options.go      # Option types and defaults
├── wire.go         # Wire types (request, response, rpcError)
├── openrpc.go      # OpenRPC generation (unchanged)
├── client.go       # JSON-RPC client (unchanged)
└── jsonrpc_test.go # Comprehensive tests
```

### File Removals

| File | Reason |
|------|--------|
| `server.go` | Replaced by `jsonrpc.go` |

## Tests

### Unit Tests

```go
func TestHandler(t *testing.T)           // Handler creation
func TestMount(t *testing.T)             // Mount on router
func TestSingleRequest(t *testing.T)     // Single JSON-RPC call
func TestBatchRequest(t *testing.T)      // Batch JSON-RPC calls
func TestNotification(t *testing.T)      // Request without id
func TestParseError(t *testing.T)        // Invalid JSON
func TestInvalidRequest(t *testing.T)    // Missing jsonrpc/method
func TestMethodNotFound(t *testing.T)    // Unknown method
func TestInvalidParams(t *testing.T)     // Params decode failure
func TestServerError(t *testing.T)       // Handler returns error
func TestErrorMapper(t *testing.T)       // Custom error mapping
func TestMaxBodySize(t *testing.T)       // Body size limit
func TestWithMiddleware(t *testing.T)    // Middleware integration
func TestNoInput(t *testing.T)           // Method without input
func TestNoOutput(t *testing.T)          // Method without output
```

### Integration Tests

```go
func TestTodoAPI(t *testing.T)           // Full CRUD via JSON-RPC
func TestMultiResource(t *testing.T)     // Multiple resources
func TestOpenRPC(t *testing.T)           // OpenRPC spec generation
```

## CLI Template Changes

### Before (server.go.tmpl)

```go
// Mount JSON-RPC 2.0 endpoint
rpcServer, err := jsonrpc.NewServer(svc)
if err != nil {
    return nil, err
}
app.Router.Mount("/rpc", rpcServer.Handler())
```

### After (server.go.tmpl)

```go
// Mount JSON-RPC 2.0 endpoint
if err := jsonrpc.Mount(app.Router, "/rpc", svc); err != nil {
    return nil, err
}
```

### Full Template

```go
package server

import (
    "github.com/go-mizu/mizu"
    contract "github.com/go-mizu/mizu/contract/v2"
    "github.com/go-mizu/mizu/contract/v2/transport/jsonrpc"
    "github.com/go-mizu/mizu/contract/v2/transport/rest"
    "{{.Module}}/service/todo"
)

func New(cfg Config, todoSvc *todo.Service) (*mizu.App, error) {
    // Register service using code-first approach
    svc := contract.Register[todo.API](todoSvc,
        contract.WithDefaultResource("todos"),
        contract.WithName("Todo"),
        contract.WithDescription("Todo management service"),
    )

    // Create mizu app
    app := mizu.New()

    // Mount REST API
    if err := rest.Mount(app.Router, svc); err != nil {
        return nil, err
    }

    // Mount JSON-RPC 2.0 endpoint
    if err := jsonrpc.Mount(app.Router, "/rpc", svc); err != nil {
        return nil, err
    }

    // Serve OpenAPI 3.0 spec
    app.Router.Get("/openapi.json", func(c *mizu.Ctx) error {
        spec, err := rest.OpenAPI(svc.Descriptor())
        if err != nil {
            return c.JSON(500, map[string]string{"error": err.Error()})
        }
        c.Header().Set("Content-Type", "application/json")
        _, err = c.Write(spec)
        return err
    })

    // Serve OpenRPC spec
    app.Router.Get("/openrpc.json", func(c *mizu.Ctx) error {
        spec, err := jsonrpc.OpenRPC(svc.Descriptor())
        if err != nil {
            return c.JSON(500, map[string]string{"error": err.Error()})
        }
        c.Header().Set("Content-Type", "application/json")
        _, err = c.Write(spec)
        return err
    })

    return app, nil
}
```

## Migration Guide

### From Server-based API

```go
// Before
rpcServer, err := jsonrpc.NewServer(svc)
if err != nil {
    return nil, err
}
mux.Handle("/rpc", rpcServer.Handler())

// After
jsonrpc.Mount(r, "/rpc", svc)
```

### Adding Middleware

```go
// Before: Not possible without wrapping http.Handler

// After
rpc := r.Prefix("/rpc").With(authMiddleware, logMiddleware)
jsonrpc.Mount(rpc, "", svc)
```

### Custom Error Handling

```go
// Before: Modify source code

// After
jsonrpc.Mount(r, "/rpc", svc, jsonrpc.WithErrorMapper(func(err error) (int, string, any) {
    if errors.Is(err, ErrNotFound) {
        return -32001, "not found", nil
    }
    return -32000, "server error", err.Error()
}))
```

## Comparison with REST Transport

| Feature | REST | JSON-RPC |
|---------|------|----------|
| Mount function | `Mount(r, inv)` | `Mount(r, path, inv)` |
| Handler function | `Handler(inv)` | `Handler(inv)` |
| Routes function | `Routes(inv)` | N/A (single endpoint) |
| Error mapper | `WithErrorMapper` | `WithErrorMapper` |
| Body size limit | `WithMaxBodySize` | `WithMaxBodySize` |
| Spec generation | `OpenAPI(svc)` | `OpenRPC(svc)` |

The JSON-RPC API has one difference: `Mount` takes a `path` parameter because JSON-RPC uses a single endpoint rather than multiple routes.

## Summary

This redesign:

1. **Integrates with mizu** - Uses `mizu.Handler` natively, enabling middleware
2. **Simplifies API** - `Mount` and `Handler` functions replace `Server` type
3. **Adds configurability** - Error mapper and body size limit options
4. **Consistent with REST** - Same API pattern for both transports
5. **Follows Go conventions** - Minimal types, clear function names
6. **Maintains compatibility** - Same wire protocol, same OpenRPC generation
