# Live Package Implementation Spec

## Overview

This document provides detailed implementation guidance for the `live` package based on the spec in `0023_live_v1.md`. The package provides low-latency realtime message delivery over WebSocket with topic-based publish/subscribe.

## Package Structure

```
live/
  doc.go        // Package documentation
  errors.go     // Error values
  message.go    // Message envelope
  codec.go      // Encoding interface + JSON/MessagePack codecs
  session.go    // Session management
  pubsub.go     // Topic routing (in-memory implementation)
  server.go     // Server + Options
  ws.go         // WebSocket upgrade and read/write loops
```

## Dependencies

- Standard library only for core functionality
- Use existing `middlewares/websocket` package for WebSocket framing
- JSON encoding via `encoding/json` or project's JSON abstraction
- MessagePack via optional build tag (defer to future if needed)

## Type Definitions

### Message (message.go)

```go
// Message is the transport envelope.
type Message struct {
    Type  string `json:"type"`
    Topic string `json:"topic,omitempty"`
    Ref   string `json:"ref,omitempty"`
    Body  []byte `json:"body,omitempty"`
}
```

**Design notes:**
- `Type` identifies message purpose (e.g., "subscribe", "unsubscribe", "publish", "ack")
- `Topic` is the routing key for pub/sub
- `Ref` correlates request/response pairs (client-generated)
- `Body` is opaque bytes; higher layers define schema

### Meta (message.go)

```go
// Meta holds authenticated connection metadata.
type Meta map[string]any
```

### Codec (codec.go)

```go
// Codec defines message encoding.
type Codec interface {
    Encode(Message) ([]byte, error)
    Decode([]byte) (Message, error)
}

// JSONCodec encodes messages as JSON.
type JSONCodec struct{}

func (JSONCodec) Encode(m Message) ([]byte, error)
func (JSONCodec) Decode(data []byte) (Message, error)
```

**Implementation:**
- JSON codec is the default
- MessagePack codec can be added later
- Codec is set per-server via Options

### Session (session.go)

```go
// Session represents a single connected client.
type Session struct {
    id       string
    meta     Meta
    sendCh   chan Message
    server   *Server
    conn     *websocket.Conn
    topics   map[string]struct{}
    mu       sync.RWMutex
    closed   atomic.Bool
    closeErr error
}

func (s *Session) ID() string
func (s *Session) Meta() Meta
func (s *Session) Send(msg Message) error  // Non-blocking, returns error if queue full
func (s *Session) Close() error
func (s *Session) Topics() []string        // Returns subscribed topics
```

**Implementation details:**
- `sendCh` is bounded by `QueueSize` from Options
- `Send()` does `select` with default case to detect backpressure
- If queue is full, close the session (as per spec)
- `topics` tracks current subscriptions for this session
- Thread-safe via `mu` for topic operations

### PubSub (pubsub.go)

```go
// PubSub routes messages by topic.
type PubSub interface {
    Subscribe(s *Session, topic string)
    Unsubscribe(s *Session, topic string)
    Publish(topic string, msg Message)
    Who(topic string) []*Session  // Optional: presence
}

// memPubSub is the default in-memory implementation.
type memPubSub struct {
    mu     sync.RWMutex
    topics map[string]map[*Session]struct{}
}

func newMemPubSub() *memPubSub
func (p *memPubSub) Subscribe(s *Session, topic string)
func (p *memPubSub) Unsubscribe(s *Session, topic string)
func (p *memPubSub) Publish(topic string, msg Message)
func (p *memPubSub) Who(topic string) []*Session
```

**Implementation details:**
- Use RWMutex for concurrent read access
- `topics` maps topic string to set of sessions
- `Publish` iterates subscribers and calls `s.Send()`
- `Who` returns a snapshot slice (not the internal map)

### Server (server.go)

```go
// Options configures the Server.
type Options struct {
    Codec     Codec
    QueueSize int  // Default: 256

    OnAuth    func(ctx context.Context, r *http.Request) (Meta, error)
    OnMessage func(ctx context.Context, s *Session, msg Message)
    OnClose   func(s *Session, err error)
}

// Server owns sessions, pubsub state, and the WebSocket handler.
type Server struct {
    opts     Options
    pubsub   *memPubSub
    sessions sync.Map  // map[string]*Session
    mu       sync.Mutex
}

func New(opts Options) *Server
func (srv *Server) Handler() http.Handler
func (srv *Server) Publish(topic string, msg Message)
func (srv *Server) Broadcast(msg Message)  // Publish to all sessions
func (srv *Server) Session(id string) *Session
func (srv *Server) Sessions() []*Session
func (srv *Server) PubSub() PubSub
```

**Implementation details:**
- `Handler()` returns http.Handler that upgrades to WebSocket
- Default codec is JSONCodec if not specified
- Default QueueSize is 256 if not specified
- Sessions are stored in sync.Map for concurrent access
- Session IDs are generated UUIDs (or configurable)

### WebSocket Handler (ws.go)

```go
// handleConn manages a single WebSocket connection lifecycle.
func (srv *Server) handleConn(w http.ResponseWriter, r *http.Request)
```

**Connection lifecycle:**
1. Call `OnAuth` (if set) to authenticate
2. Upgrade HTTP to WebSocket using existing middleware pattern
3. Create Session with generated ID
4. Start write goroutine (reads from sendCh, writes to conn)
5. Run read loop (decode messages, call OnMessage)
6. On disconnect: cleanup subscriptions, remove session, call OnClose

**Read loop:**
```go
for {
    msgType, data, err := conn.ReadMessage()
    if err != nil {
        break
    }
    if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
        continue  // Ignore control frames handled by websocket layer
    }
    msg, err := srv.opts.Codec.Decode(data)
    if err != nil {
        // Log and continue or close?
        continue
    }
    if srv.opts.OnMessage != nil {
        srv.opts.OnMessage(r.Context(), session, msg)
    }
}
```

**Write loop:**
```go
for msg := range session.sendCh {
    data, err := srv.opts.Codec.Encode(msg)
    if err != nil {
        continue
    }
    if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
        break
    }
}
```

### Errors (errors.go)

```go
var (
    ErrSessionClosed = errors.New("live: session closed")
    ErrQueueFull     = errors.New("live: send queue full")
    ErrAuthFailed    = errors.New("live: authentication failed")
)
```

## Wire Protocol

### Message Types (conventions, not enforced by live)

Higher layers define message types. Common conventions:

| Type | Direction | Purpose |
|------|-----------|---------|
| `subscribe` | C->S | Subscribe to topic |
| `unsubscribe` | C->S | Unsubscribe from topic |
| `message` | Both | Application message |
| `ack` | S->C | Acknowledge subscription |
| `error` | S->C | Error response |
| `ping` | Both | Keep-alive (WebSocket level) |

### Example Messages

**Subscribe:**
```json
{"type":"subscribe","topic":"room:123","ref":"1"}
```

**Ack:**
```json
{"type":"ack","topic":"room:123","ref":"1"}
```

**Message:**
```json
{"type":"message","topic":"room:123","body":"eyJzZW5kZXIiOiJ1c2VyMSIsInRleHQiOiJoZWxsbyJ9"}
```

Body is base64-encoded when using JSON codec for binary safety.

## Integration with Mizu

### Mounting

```go
server := live.New(live.Options{
    OnAuth: func(ctx context.Context, r *http.Request) (live.Meta, error) {
        // Validate token, return user info
        return live.Meta{"user_id": "123"}, nil
    },
    OnMessage: func(ctx context.Context, s *live.Session, msg live.Message) {
        switch msg.Type {
        case "subscribe":
            server.PubSub().Subscribe(s, msg.Topic)
        case "unsubscribe":
            server.PubSub().Unsubscribe(s, msg.Topic)
        case "broadcast":
            server.PubSub().Publish(msg.Topic, msg)
        }
    },
})

app := mizu.New()
app.Get("/ws", mizu.Compat(server.Handler()))
```

### Integration with sync

```go
// Adapter to notify live clients when sync cursor advances
type liveNotifier struct {
    server *live.Server
}

func (n *liveNotifier) Notify(scope string, cursor uint64) {
    n.server.Publish("sync:"+scope, live.Message{
        Type:  "sync",
        Topic: "sync:" + scope,
        Body:  []byte(fmt.Sprintf(`{"cursor":%d}`, cursor)),
    })
}

// Usage
engine := sync.New(sync.Options{
    // ...
    Notify: &liveNotifier{server: liveServer},
})
```

## Testing Strategy

### Unit Tests

1. **Codec tests** (codec_test.go)
   - JSON encode/decode round-trip
   - Handle empty fields
   - Handle nil Body
   - Invalid JSON handling

2. **Session tests** (session_test.go)
   - Send to open session
   - Send to closed session
   - Queue full behavior (closes session)
   - Topic subscription tracking

3. **PubSub tests** (pubsub_test.go)
   - Subscribe adds session to topic
   - Unsubscribe removes session
   - Publish delivers to all subscribers
   - Publish to empty topic (no error)
   - Who returns correct sessions
   - Concurrent subscribe/unsubscribe safety

4. **Server tests** (server_test.go)
   - Default options applied
   - Session lifecycle
   - Broadcast to all sessions

### Integration Tests

1. **WebSocket tests** (ws_test.go)
   - Full connection lifecycle
   - Auth success/failure
   - Message round-trip
   - Graceful close
   - Abrupt disconnect handling

### Test Helpers

```go
// mockConn simulates a WebSocket connection for testing
type mockConn struct {
    readCh  chan []byte
    writeCh chan []byte
    closed  bool
}

// testServer creates a server with test defaults
func testServer(opts ...func(*live.Options)) *live.Server
```

## Performance Considerations

1. **Send queue size**: Default 256 balances memory vs latency
2. **Topic map**: Use sync.Map or sharded maps for high-concurrency
3. **Message encoding**: JSON is simple; MessagePack for efficiency
4. **Broadcast optimization**: Consider batch writes for large fanouts

## Future Extensions

1. **MessagePack codec**: Add with build tag `msgpack`
2. **Redis PubSub adapter**: For horizontal scaling
3. **Presence tracking**: Who is subscribed to a topic
4. **Rate limiting**: Per-session message rate limits
5. **Compression**: Per-message compression for large payloads

## Implementation Order

1. errors.go - Simple, no dependencies
2. message.go - Types only
3. codec.go - JSON codec
4. session.go - Depends on message
5. pubsub.go - Depends on session, message
6. server.go - Depends on session, pubsub, codec
7. ws.go - WebSocket handling, depends on server
8. doc.go - Package documentation

## Checklist

- [ ] All types match spec in 0023_live_v1.md
- [ ] Backpressure policy enforced (close on full queue)
- [ ] Thread-safe session and pubsub operations
- [ ] Clean connection lifecycle (auth -> connect -> close)
- [ ] Comprehensive test coverage
- [ ] No external dependencies beyond std lib + mizu
- [ ] Integration example with sync package
