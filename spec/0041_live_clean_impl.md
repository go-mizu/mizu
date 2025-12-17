# Live Package Cleanup - Implementation Plan

Based on review in `spec/0041_live_clean.md`, this document outlines the implementation steps to minimize the live package's public surface area while maintaining its core functionality.

## Goals

1. Reduce conceptual overhead for learners
2. Separate WebSocket implementation details from live API
3. Simplify Message envelope to true "opaque payload" model
4. Remove unused/redundant errors and helpers
5. Make OnClose receive the true close reason

## Implementation Steps

### Phase 1: Move WebSocket to Internal Package

**Create `live/internal/ws/` package:**

- Move WebSocket constants (opcodes, close codes, GUID)
- Move `wsConn` struct and its methods (`readMessage`, `writeMessage`)
- Move handshake helpers (`isWebSocketUpgrade`, `computeAcceptKey`, `validateWebSocketKey`)
- Export a minimal interface:
  ```go
  type Conn interface {
      ReadMessage() (opcode int, data []byte, err error)
      WriteMessage(opcode int, data []byte) error
      Close() error
  }

  func Upgrade(w http.ResponseWriter, r *http.Request, readLimit int) (Conn, error)
  ```

**Files:**
- `live/internal/ws/ws.go` - WebSocket implementation
- `live/internal/ws/errors.go` - WS-specific errors (ErrProtocolError, ErrMessageTooLarge)

### Phase 2: Simplify Message Struct

**Before:**
```go
type Message struct {
    Type  string `json:"type"`
    Topic string `json:"topic,omitempty"`
    Ref   string `json:"ref,omitempty"`
    Body  []byte `json:"body,omitempty"`
}
```

**After:**
```go
type Message struct {
    Topic string          `json:"topic,omitempty"`
    Data  json.RawMessage `json:"data,omitempty"`
}
```

**Rationale:**
- `Type` and `Ref` are application protocol concerns, not transport concerns
- `json.RawMessage` avoids base64 encoding surprise for JSON payloads
- Higher layers (view/live, sync/live) define their own envelope if needed

### Phase 3: Replace Meta with Value Pattern

**Before:**
```go
type Meta map[string]any

func (m Meta) Get(key string) any
func (m Meta) GetString(key string) string

// OnAuth returns Meta
OnAuth func(ctx context.Context, r *http.Request) (Meta, error)

// Session exposes Meta
func (s *Session) Meta() Meta
```

**After:**
```go
// OnAuth returns opaque value
OnAuth func(ctx context.Context, r *http.Request) (any, error)

// Session exposes opaque value
func (s *Session) Value() any
```

**Rationale:**
- Simpler mental model
- No convenience helpers that encourage storing app state in session
- Application decides what to store and how to type-assert

### Phase 4: Simplify OnMessage

**Before:**
```go
OnMessage func(ctx context.Context, s *Session, msg Message)
```

**After:**
```go
OnMessage func(ctx context.Context, s *Session, topic string, data []byte)
```

**Rationale:**
- Byte-level interface lets app define decode/routing
- No implied wire format
- If app wants Message struct, they decode it themselves

### Phase 5: Remove Extra API Surface

**Remove from Server:**
- `Broadcast(msg Message)` - Can be done via reserved topic or user iteration
- `SessionCount() int` - Nice-to-have, not essential for v1

**Remove from memPubSub:**
- `count(topic string) int` - Only used in tests, internal detail

**Remove from Session:**
- Nothing directly, but Meta becomes Value

### Phase 6: Clean Up Errors

**Remove (use HTTP errors directly):**
- `ErrInvalidVersion` - Server returns HTTP 426
- `ErrAuthFailed` - Server returns HTTP 401

**Move to internal/ws:**
- `ErrProtocolError` - WebSocket protocol violation
- `ErrMessageTooLarge` - Message exceeds limit

**Keep in live package:**
- `ErrSessionClosed` - Returned by Send()
- `ErrQueueFull` - Returned by Send(), also closes session

### Phase 7: Fix OnClose Semantics

**Problem:** closeWithError discards the error; OnClose receives readErr, not the actual close reason.

**Solution:**
```go
type Session struct {
    // ...
    closeErr atomic.Value // stores error
}

func (s *Session) closeWithError(err error) error {
    if !s.closed.CompareAndSwap(false, true) {
        return nil
    }
    s.closeErr.Store(err) // Store the actual reason
    close(s.doneCh)
    if s.conn != nil {
        _ = s.conn.Close()
    }
    return nil
}
```

Then in handleConn, OnClose receives `session.closeErr.Load().(error)`.

## File Changes Summary

| File | Action |
|------|--------|
| `live/internal/ws/ws.go` | Create - WebSocket implementation |
| `live/internal/ws/errors.go` | Create - WS errors |
| `live/live.go` | Modify - Simplified API |
| `live/live_test.go` | Modify - Update tests |

## Migration Notes for Existing Code

1. `Message.Type` -> Move to `Data` as part of app envelope
2. `Message.Ref` -> Move to `Data` as part of app envelope
3. `Message.Body` -> `Message.Data` (now json.RawMessage)
4. `Session.Meta()` -> `Session.Value().(YourType)`
5. `OnAuth` return `Meta` -> return `any`
6. `OnMessage(ctx, s, msg)` -> `OnMessage(ctx, s, topic, data)`
7. `srv.Broadcast(msg)` -> Publish to `"*"` topic or iterate sessions manually
8. `srv.SessionCount()` -> Track manually if needed

## Minimal Public API After Cleanup

```go
package live

// Errors
var (
    ErrSessionClosed = errors.New("live: session closed")
    ErrQueueFull     = errors.New("live: send queue full")
)

// Message is the transport envelope
type Message struct {
    Topic string          `json:"topic,omitempty"`
    Data  json.RawMessage `json:"data,omitempty"`
}

// Options configures the Server
type Options struct {
    QueueSize   int
    ReadLimit   int
    OnAuth      func(ctx context.Context, r *http.Request) (any, error)
    OnMessage   func(ctx context.Context, s *Session, topic string, data []byte)
    OnClose     func(s *Session, err error)
    Origins     []string
    CheckOrigin func(r *http.Request) bool
    IDGenerator func() string
}

// Server owns sessions and pubsub
type Server struct { /* ... */ }

func New(opts Options) *Server
func (srv *Server) Handler() http.Handler
func (srv *Server) Publish(topic string, data []byte)
func (srv *Server) Subscribe(s *Session, topic string)
func (srv *Server) Unsubscribe(s *Session, topic string)

// Session represents a connected client
type Session struct { /* ... */ }

func (s *Session) ID() string
func (s *Session) Value() any
func (s *Session) Send(msg Message) error
func (s *Session) Close() error
func (s *Session) IsClosed() bool
```

Total exported identifiers: ~20 (down from ~30+)
