# Contract Async v2: HTTP-Based Async Transport

**Status:** Implementation Ready
**Depends on:** 0049_contract_v2.md

## Overview

This specification defines `contract/v2/transport/async` - an HTTP-based async transport that exposes contract methods via Server-Sent Events (SSE). It enables fire-and-forget requests and async request-reply patterns over standard HTTP.

## Goals

1. **Native mizu integration** - Use `mizu.Handler` directly
2. **Minimal API surface** - Follow Go standard library design philosophy
3. **Consistent with other transports** - Same API pattern: `Mount`, `Handler`, options
4. **No external dependencies** - Works with any HTTP client, no message broker required
5. **Simple wire protocol** - JSON over SSE, similar to JSON-RPC

## Non-Goals

- Full message broker functionality (use `Broker` interface directly for that)
- Guaranteed delivery (HTTP has no delivery guarantees)
- Message persistence (in-memory only, bring your own store if needed)

## Design

### Core API

```go
package async

// Mount registers async endpoints on a mizu router.
// POST {path} - Submit async request
// GET  {path} - SSE connection for receiving responses
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error

// Handler returns a mizu.Handler for the async endpoint.
// The handler routes based on HTTP method (POST=submit, GET=stream).
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error)
```

### Options

```go
// Option configures async transport behavior.
type Option func(*options)

// WithErrorMapper sets custom error mapping.
func WithErrorMapper(m ErrorMapper) Option

// WithMaxBodySize limits request body size (default: 1MB).
func WithMaxBodySize(n int64) Option

// WithBufferSize sets the per-client event buffer size (default: 16).
func WithBufferSize(n int) Option

// WithOnConnect sets a callback when a client connects to SSE stream.
func WithOnConnect(fn func(clientID string)) Option

// WithOnDisconnect sets a callback when a client disconnects.
func WithOnDisconnect(fn func(clientID string)) Option
```

### Error Mapping

```go
// ErrorMapper converts Go errors to async error responses.
// Returns: error code and message
type ErrorMapper func(error) (code string, message string)

// Default: code="error", message=err.Error()
```

## Wire Protocol

### Submit Request (POST)

```json
{
  "id": "request-123",
  "method": "todos.create",
  "params": {"title": "Buy milk"}
}
```

- `id` (string, required): Client-generated correlation ID
- `method` (string, required): Method name in `resource.method` format
- `params` (object, optional): Method input parameters

### Submit Response

```
HTTP/1.1 202 Accepted
Content-Type: application/json

{"id": "request-123", "status": "accepted"}
```

### SSE Stream (GET)

```
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: result
data: {"id":"request-123","result":{"id":"1","title":"Buy milk"}}

event: error
data: {"id":"request-456","error":{"code":"not_found","message":"todo not found"}}
```

### Event Types

| Event | Description |
|-------|-------------|
| `result` | Successful method invocation result |
| `error` | Method invocation error |
| `ping` | Keepalive (sent every 30s) |

### SSE Data Format

Result event:
```json
{"id": "correlation-id", "result": {...}}
```

Error event:
```json
{"id": "correlation-id", "error": {"code": "...", "message": "..."}}
```

## Usage Examples

### Simple Mount

```go
svc := contract.Register[TodoAPI](impl)
r := mizu.NewRouter()

async.Mount(r, "/async", svc)
```

### With Options

```go
async.Mount(r, "/async", svc,
    async.WithBufferSize(32),
    async.WithErrorMapper(func(err error) (string, string) {
        if errors.Is(err, ErrNotFound) {
            return "not_found", "resource not found"
        }
        return "error", err.Error()
    }),
)
```

### Client Usage (JavaScript)

```javascript
// Connect to SSE stream
const events = new EventSource('/async');
events.addEventListener('result', (e) => {
    const data = JSON.parse(e.data);
    console.log('Result:', data.id, data.result);
});
events.addEventListener('error', (e) => {
    const data = JSON.parse(e.data);
    console.log('Error:', data.id, data.error);
});

// Submit request
fetch('/async', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
        id: 'req-1',
        method: 'todos.create',
        params: {title: 'Buy milk'}
    })
});
```

### Client Usage (Go)

```go
// Submit request
resp, _ := http.Post(url+"/async", "application/json",
    strings.NewReader(`{"id":"1","method":"todos.create","params":{"title":"Test"}}`))

// Connect to SSE (using bufio.Scanner)
resp, _ = http.Get(url + "/async")
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "data: ") {
        data := line[6:]
        // Parse JSON response
    }
}
```

## Implementation

### Internal Types

```go
// request is the submit request structure.
type request struct {
    ID     string          `json:"id"`
    Method string          `json:"method"`
    Params json.RawMessage `json:"params,omitempty"`
}

// response is the SSE event data structure.
type response struct {
    ID     string          `json:"id"`
    Result json.RawMessage `json:"result,omitempty"`
    Error  *asyncError     `json:"error,omitempty"`
}

// asyncError is the error structure.
type asyncError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// client represents a connected SSE client.
type client struct {
    id     string
    events chan []byte
    done   chan struct{}
}
```

### Hub Pattern

```go
// hub manages SSE connections and message routing.
type hub struct {
    mu      sync.RWMutex
    clients map[string]*client
    bufSize int
}

func (h *hub) register(c *client) {
    h.mu.Lock()
    h.clients[c.id] = c
    h.mu.Unlock()
}

func (h *hub) unregister(id string) {
    h.mu.Lock()
    if c, ok := h.clients[id]; ok {
        close(c.events)
        delete(h.clients, id)
    }
    h.mu.Unlock()
}

func (h *hub) broadcast(data []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for _, c := range h.clients {
        select {
        case c.events <- data:
        default:
            // Buffer full, drop message
        }
    }
}
```

### Handler Implementation

```go
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
    if inv == nil {
        return nil, errors.New("async: nil invoker")
    }
    svc := inv.Descriptor()
    if svc == nil {
        return nil, errors.New("async: nil descriptor")
    }

    o := applyOptions(opts)
    h := newHub(o.bufferSize)

    return func(c *mizu.Ctx) error {
        switch c.Request().Method {
        case http.MethodPost:
            return handleSubmit(c, inv, svc, h, o)
        case http.MethodGet:
            return handleStream(c, h, o)
        default:
            c.Header().Set("Allow", "GET, POST")
            return c.Status(http.StatusMethodNotAllowed).Text("method not allowed")
        }
    }, nil
}
```

### Submit Handler

```go
func handleSubmit(c *mizu.Ctx, inv contract.Invoker, svc *contract.Service, h *hub, o *options) error {
    body, err := io.ReadAll(io.LimitReader(c.Request().Body, o.maxBodySize+1))
    if err != nil {
        return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "read error"})
    }
    if int64(len(body)) > o.maxBodySize {
        return c.Status(http.StatusRequestEntityTooLarge).JSON(map[string]string{"error": "body too large"})
    }

    var req request
    if err := json.Unmarshal(body, &req); err != nil {
        return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "invalid json"})
    }
    if req.ID == "" || req.Method == "" {
        return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "missing id or method"})
    }

    // Parse method: resource.method
    resource, method, ok := parseMethod(req.Method)
    if !ok {
        return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "invalid method format"})
    }

    // Accept immediately
    c.Status(http.StatusAccepted)
    c.JSON(http.StatusAccepted, map[string]string{"id": req.ID, "status": "accepted"})

    // Process async
    go func() {
        ctx := context.Background()
        resp := processRequest(ctx, inv, svc, resource, method, req.ID, req.Params, o)
        data, _ := json.Marshal(resp)

        eventType := "result"
        if resp.Error != nil {
            eventType = "error"
        }

        // Format SSE event
        event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
        h.broadcast([]byte(event))
    }()

    return nil
}
```

### Stream Handler

```go
func handleStream(c *mizu.Ctx, h *hub, o *options) error {
    // Set SSE headers
    c.Header().Set("Content-Type", "text/event-stream")
    c.Header().Set("Cache-Control", "no-cache")
    c.Header().Set("Connection", "keep-alive")
    c.Header().Set("X-Accel-Buffering", "no")

    // Generate client ID
    clientID := generateID()

    client := &client{
        id:     clientID,
        events: make(chan []byte, o.bufferSize),
        done:   make(chan struct{}),
    }
    h.register(client)
    defer h.unregister(clientID)

    if o.onConnect != nil {
        o.onConnect(clientID)
    }
    defer func() {
        if o.onDisconnect != nil {
            o.onDisconnect(clientID)
        }
    }()

    // Flush headers
    if f, ok := c.ResponseWriter().(http.Flusher); ok {
        f.Flush()
    }

    ctx := c.Request().Context()
    ping := time.NewTicker(30 * time.Second)
    defer ping.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case event := <-client.events:
            c.ResponseWriter().Write(event)
            if f, ok := c.ResponseWriter().(http.Flusher); ok {
                f.Flush()
            }
        case <-ping.C:
            c.ResponseWriter().Write([]byte(": ping\n\n"))
            if f, ok := c.ResponseWriter().(http.Flusher); ok {
                f.Flush()
            }
        }
    }
}
```

## File Structure

```
contract/v2/transport/async/
├── async.go        # Mount, Handler (main API)
├── hub.go          # SSE client hub
├── wire.go         # Wire types (request, response)
├── options.go      # Options
└── async_test.go   # Comprehensive tests
```

## Comparison with Other Transports

| Feature | REST | JSON-RPC | MCP | Async |
|---------|------|----------|-----|-------|
| Mount signature | `Mount(r, inv)` | `Mount(r, path, inv)` | `Mount(r, path, inv)` | `Mount(r, path, inv)` |
| Handler | `Handler(inv)` | `Handler(inv)` | `Handler(inv)` | `Handler(inv)` |
| HTTP Method | varies | POST | POST | POST (submit), GET (stream) |
| Response | immediate | immediate | immediate | async via SSE |
| Error mapper | `WithErrorMapper` | `WithErrorMapper` | `WithErrorMapper` | `WithErrorMapper` |
| Body size limit | `WithMaxBodySize` | `WithMaxBodySize` | `WithMaxBodySize` | `WithMaxBodySize` |

## Tests

### Unit Tests

```go
func TestHandler(t *testing.T)              // Handler creation
func TestMount(t *testing.T)                // Mount on router
func TestSubmit(t *testing.T)               // Submit async request
func TestStream(t *testing.T)               // SSE stream connection
func TestSubmitAndReceive(t *testing.T)     // Full round-trip
func TestInvalidMethod(t *testing.T)        // Invalid method format
func TestInvalidJSON(t *testing.T)          // Malformed JSON
func TestMissingID(t *testing.T)            // Missing request ID
func TestMaxBodySize(t *testing.T)          // Body size limit
func TestErrorMapper(t *testing.T)          // Custom error mapping
func TestBufferSize(t *testing.T)           // Event buffer configuration
func TestPing(t *testing.T)                 // SSE keepalive
func TestClientDisconnect(t *testing.T)     // Client disconnect handling
func TestMultipleClients(t *testing.T)      // Multiple SSE clients
```

### Integration Tests

```go
func TestTodoAPI(t *testing.T)              // Full CRUD via async
func TestConcurrentRequests(t *testing.T)   // Multiple concurrent requests
func TestReconnect(t *testing.T)            // Client reconnection
```

## CLI Template Changes

Update `server.go.tmpl` to optionally include async transport:

```go
import (
    // ... existing imports
    "github.com/go-mizu/mizu/contract/v2/transport/async"
)

func New(cfg Config, todoSvc *todo.Service) (*mizu.App, error) {
    // ... existing code

    // Mount async endpoint (SSE-based)
    if err := async.Mount(app.Router, "/async", svc); err != nil {
        return nil, err
    }

    return app, nil
}
```

Update `main.go.tmpl` logging:

```go
log.Printf("Async:    http://localhost%s/async", cfg.Addr)
```

## Summary

This specification defines:

1. **HTTP-based async transport** - Fire-and-forget and request-reply over HTTP
2. **SSE for responses** - Real-time event delivery without WebSocket complexity
3. **Consistent API** - Same patterns as REST, JSON-RPC, and MCP transports
4. **Minimal surface** - Mount, Handler, and a few options
5. **No external dependencies** - Works with any HTTP infrastructure
