# Live Package Refactor Plan

## Overview

This document describes an aggressive refactor of the `live` package to reduce API surface area, align with Go core library design principles, and consolidate files into single-file packages.

---

## Goals

1. **Minimal API surface** - Remove non-essential public types and functions
2. **Single-file packages** - Consolidate `live/*.go` into `live/live.go`
3. **Go idioms** - Follow Go standard library naming and design patterns
4. **Simplicity** - Remove unnecessary abstractions and indirection

---

## File Consolidation

### Before

```
live/
  doc.go
  errors.go
  message.go
  codec.go
  session.go
  pubsub.go
  server.go
  ws.go
  codec_test.go
  session_test.go
  pubsub_test.go
  server_test.go
  ws_test.go
```

### After

```
live/
  live.go         # All code combined (doc, types, server, session, pubsub, ws)
  live_test.go    # All tests combined
```

---

## Current Public API Analysis

### Types (8 exported)

| Type | Status | Reason |
|------|--------|--------|
| `Message` | **KEEP** | Essential transport envelope |
| `Meta` | **KEEP** | Auth metadata container |
| `Options` | **KEEP** | Server configuration |
| `Server` | **KEEP** | Main server type |
| `Session` | **KEEP** | Client connection |
| `Codec` | **REMOVE** | Interface not needed; always JSON |
| `JSONCodec` | **REMOVE** | Make internal |
| `PubSub` | **REMOVE** | Flatten into Server |

### Errors (5 exported)

| Error | Status | Reason |
|-------|--------|--------|
| `ErrSessionClosed` | **KEEP** | Essential for send error handling |
| `ErrQueueFull` | **KEEP** | Essential for backpressure handling |
| `ErrAuthFailed` | **KEEP** | Essential for auth callback |
| `ErrUpgradeFailed` | **REMOVE** | Internal; never returned to users |
| `ErrInvalidMessage` | **REMOVE** | Internal; messages just skipped |

### Functions (2 exported)

| Function | Status | Reason |
|----------|--------|--------|
| `New(Options) *Server` | **KEEP** | Essential constructor |
| `SyncNotifier` | **REMOVE** | Move to sync package |

### Server Methods (9 exported)

| Method | Status | Reason |
|--------|--------|--------|
| `Handler() http.Handler` | **KEEP** | Essential; returns WS handler |
| `Publish(topic, msg)` | **KEEP** | Essential; fanout to topic |
| `Broadcast(msg)` | **KEEP** | Useful; send to all |
| `Session(id) *Session` | **REMOVE** | Rarely needed; breaks encapsulation |
| `Sessions() []*Session` | **REMOVE** | Rarely needed; memory concern |
| `SessionCount() int` | **KEEP** | Useful for monitoring |
| `PubSub() PubSub` | **REMOVE** | Flatten methods onto Server |
| `Options() Options` | **REMOVE** | Breaks encapsulation |
| `Subscribe(s, topic)` | **ADD** | Flattened from PubSub |
| `Unsubscribe(s, topic)` | **ADD** | Flattened from PubSub |

### Session Methods (9 exported)

| Method | Status | Reason |
|--------|--------|--------|
| `ID() string` | **KEEP** | Essential; unique identifier |
| `Meta() Meta` | **KEEP** | Essential; access auth data |
| `Send(msg) error` | **KEEP** | Essential; send to client |
| `Close() error` | **KEEP** | Essential; graceful close |
| `CloseError() error` | **REMOVE** | Rarely needed |
| `Done() <-chan struct{}` | **REMOVE** | Internal; not needed by users |
| `IsClosed() bool` | **KEEP** | Essential; check state |
| `Topics() []string` | **REMOVE** | Rarely needed; leaks internals |

### Meta Methods (2 exported)

| Method | Status | Reason |
|--------|--------|--------|
| `Get(key) any` | **KEEP** | Essential; access metadata |
| `GetString(key) string` | **KEEP** | Convenient string access |

### PubSub Methods (6 exported via interface)

| Method | Status | Reason |
|--------|--------|--------|
| `Subscribe(s, topic)` | **FLATTEN** | Move to Server |
| `Unsubscribe(s, topic)` | **FLATTEN** | Move to Server |
| `Publish(topic, msg)` | Already on Server | Via `srv.Publish()` |
| `Who(topic) []*Session` | **REMOVE** | Rarely needed |
| `Topics() []string` | **REMOVE** | Rarely needed |
| `Count(topic) int` | **REMOVE** | Rarely needed; use SessionCount |

---

## New Public API

### Types (5 exported)

```go
// Message is the transport envelope for all live communications.
type Message struct {
    Type  string `json:"type"`           // Message purpose
    Topic string `json:"topic,omitempty"` // Routing key
    Ref   string `json:"ref,omitempty"`   // Request correlation
    Body  []byte `json:"body,omitempty"`  // Opaque payload
}

// Meta holds authenticated connection metadata.
type Meta map[string]any

// Options configures the Server.
type Options struct {
    QueueSize   int                                              // Per-session queue (default: 256)
    OnAuth      func(ctx context.Context, r *http.Request) (Meta, error)
    OnMessage   func(ctx context.Context, s *Session, msg Message)
    OnClose     func(s *Session, err error)
    Origins     []string  // Allowed origins (empty = all)
    IDGenerator func() string // Custom session IDs
}

// Server owns sessions and handles WebSocket connections.
type Server struct { ... }

// Session represents a connected WebSocket client.
type Session struct { ... }
```

### Errors (3 exported)

```go
var (
    ErrSessionClosed = errors.New("live: session closed")
    ErrQueueFull     = errors.New("live: send queue full")
    ErrAuthFailed    = errors.New("live: authentication failed")
)
```

### Constructor (1 exported)

```go
func New(opts Options) *Server
```

### Server Methods (6 exported)

```go
func (srv *Server) Handler() http.Handler
func (srv *Server) Publish(topic string, msg Message)
func (srv *Server) Broadcast(msg Message)
func (srv *Server) SessionCount() int
func (srv *Server) Subscribe(s *Session, topic string)
func (srv *Server) Unsubscribe(s *Session, topic string)
```

### Session Methods (5 exported)

```go
func (s *Session) ID() string
func (s *Session) Meta() Meta
func (s *Session) Send(msg Message) error
func (s *Session) Close() error
func (s *Session) IsClosed() bool
```

### Meta Methods (2 exported)

```go
func (m Meta) Get(key string) any
func (m Meta) GetString(key string) string
```

---

## Final Symbol Count

| Category | Before | After | Removed |
|----------|--------|-------|---------|
| Types | 8 | 5 | 3 |
| Errors | 5 | 3 | 2 |
| Functions | 2 | 1 | 1 |
| Server Methods | 9 | 6 | 3 |
| Session Methods | 9 | 5 | 4 |
| Meta Methods | 2 | 2 | 0 |
| PubSub Methods | 6 | 0 | 6 (flattened) |
| **Total** | **41** | **22** | **19 (~46%)** |

---

## Key Design Decisions

### 1. Remove Codec Interface

The Codec interface adds complexity without benefit. JSON is the only reasonable wire format for browser WebSockets. Users needing binary formats can encode into `Message.Body`.

**Before:**
```go
type Codec interface {
    Encode(Message) ([]byte, error)
    Decode([]byte) (Message, error)
}

// Options
Codec: JSONCodec{}, // configurable
```

**After:**
```go
// Internal only - always JSON
func encodeMessage(m Message) ([]byte, error)
func decodeMessage(data []byte) (Message, error)
```

### 2. Flatten PubSub into Server

The PubSub interface is an over-abstraction. Users only need Subscribe/Unsubscribe, and those should be on Server.

**Before:**
```go
srv.PubSub().Subscribe(session, topic)
srv.PubSub().Unsubscribe(session, topic)
srv.PubSub().Publish(topic, msg) // duplicate of srv.Publish
```

**After:**
```go
srv.Subscribe(session, topic)
srv.Unsubscribe(session, topic)
srv.Publish(topic, msg)
```

### 3. Remove SyncNotifier

The `SyncNotifier` helper couples live to sync. Instead, sync package should implement its own notifier using the public live API.

**Before (in live package):**
```go
func SyncNotifier(srv *Server, prefix string) sync.Notifier
```

**After (in sync package):**
```go
func LiveNotifier(srv *live.Server, prefix string) Notifier {
    return NotifierFunc(func(scope string, cursor uint64) {
        srv.Publish(prefix+scope, live.Message{
            Type:  "sync",
            Topic: prefix + scope,
            Body:  []byte(`{"cursor":` + strconv.FormatUint(cursor, 10) + `}`),
        })
    })
}
```

### 4. Remove Accessor Methods

`Options()`, `Session(id)`, `Sessions()`, `PubSub()` break encapsulation and leak implementation details. Remove them.

### 5. Simplify Session

Remove `Done()`, `CloseError()`, `Topics()` - these are internal details users don't need. `IsClosed()` is sufficient for state checking.

---

## Migration Guide

### Removed: `srv.PubSub().Subscribe()`

```go
// Before
srv.PubSub().Subscribe(session, topic)
srv.PubSub().Unsubscribe(session, topic)

// After
srv.Subscribe(session, topic)
srv.Unsubscribe(session, topic)
```

### Removed: `live.SyncNotifier()`

```go
// Before (in app code)
notifier := live.SyncNotifier(liveServer, "sync:")

// After (sync package provides this)
notifier := sync.LiveNotifier(liveServer, "sync:")
```

### Removed: `srv.Options()`

Store options locally if needed:

```go
// Before
opts := srv.Options()

// After
opts := live.Options{...}
srv := live.New(opts)
// Use opts directly if needed
```

### Removed: `srv.Session(id)` / `srv.Sessions()`

Track sessions externally if needed:

```go
// Before
session := srv.Session(id)

// After (if tracking needed)
sessions := make(map[string]*live.Session)
srv := live.New(live.Options{
    OnClose: func(s *live.Session, err error) {
        delete(sessions, s.ID())
    },
})
// In OnMessage, store: sessions[s.ID()] = s
```

### Removed: `session.Topics()`

Track subscriptions externally if needed (rarely necessary).

### Removed: `session.Done()` / `session.CloseError()`

Use `session.IsClosed()` for state checking. Close errors are passed to `OnClose` callback.

---

## Implementation Steps

1. **Create live/live.go**
   - Combine doc.go header comment
   - Add all type definitions
   - Add all internal functions
   - Add all methods
   - Order: doc, types, errors, constructor, server methods, session methods, ws internals

2. **Create live/live_test.go**
   - Combine all test files
   - Update tests for removed APIs
   - Ensure all kept APIs are tested

3. **Delete old files**
   - doc.go, errors.go, message.go, codec.go, session.go, pubsub.go, server.go, ws.go
   - All *_test.go files except live_test.go

4. **Update sync package**
   - Move SyncNotifier to sync.LiveNotifier
   - Update imports

5. **Run tests**
   - `go test ./live/...`
   - Verify all pass

---

## Risks

1. **Breaking changes** - All usages of removed APIs must be updated
2. **Template sync** - cli/templates/live must be updated to match
3. **Test coverage** - Combined test file must maintain coverage
4. **Sync integration** - SyncNotifier move needs sync package update

---

## Success Criteria

- [ ] live/live.go compiles
- [ ] live/live_test.go passes all tests
- [ ] No other .go files in live/ directory
- [ ] Public API reduced by ~46%
- [ ] cli/templates/live works with refactored package
- [ ] sync package provides LiveNotifier if needed
