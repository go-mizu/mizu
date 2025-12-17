Your stated goal is “transport and fanout, minimal surface area, opaque payloads”. The current implementation succeeds on correctness of the wire protocol, but it is not minimal for learners. The biggest reason is that you are implementing a full RFC 6455 WebSocket stack yourself, plus a pubsub layer, plus a message envelope and JSON codec. That is too many concepts for the “live is just best-effort fanout” story.

Below is a reduction plan that removes the most code and the most things a learner must understand, while keeping the same user-facing promise.

## 1) Separate “what live is” from “how WebSocket works”

Right now, users must implicitly trust and reason about:

* hijacking, accept keys, masking, opcodes, control frames
* fragmentation rules you enforce (FIN required)
* buffering, reader/writer locks
* queue backpressure and close behavior

None of this is part of your package promise. It is implementation detail.

Recommendation: do not teach or expose RFC 6455 at all in the core package.

### Minimal public concept set

* A **Server** with sessions
* A **Session** that can `Send`
* A **topic pubsub** (`Subscribe`, `Unsubscribe`, `Publish`)
* A **Handler** that upgrades and binds to sessions
* A single **incoming message hook** (optional)

That is it.

If you keep the current file layout, at minimum move all WebSocket frame logic into an internal file/package (`internal/ws`) and keep `live` focused on sessions and topics.

## 2) Shrink the protocol surface: remove `Type`, `Ref`, and `Meta` from core

### A) Message envelope is more than “opaque payload”

Your `Message` includes `Type`, `Topic`, `Ref`, and `Body`. Then the docs say “live does not interpret type”, but you still made it part of the API and example. That creates a semantic contract you claim you do not own.

If the package is truly transport-only, the only thing you need is:

```go
type Message struct {
	Topic string `json:"topic"`
	Data  []byte `json:"data"`
}
```

or even: `Publish(topic string, data []byte)`.

Everything else (type/ref/ack semantics) belongs to a higher layer: `view/live` or `sync/live` integration.

### B) Meta is convenient, but not minimal

`Meta map[string]any` with helpers is another concept and encourages app-level identity to live inside the live session object. For minimal learners, prefer:

* `OnAuth` returns a `context.Context` or a single `any` value
* `Session` exposes `Value() any` (opaque)

Example minimal:

```go
type Session struct {
    id string
    value any
}
func (s *Session) Value() any { return s.value }
```

If you need convenience, provide helpers in a separate package, not in core.

## 3) Remove `OnMessage` from Options (or make it byte-level)

The package says it does not define the protocol, but `OnMessage` receives a decoded `Message`. That implies a wire format and a message envelope.

For maximum reduction:

* remove `OnMessage` entirely, and position `live` as server->client fanout only, or
* make it `OnMessage(ctx, s, topic, data []byte)` and let the app define decode and routing.

If you keep `OnMessage`, you should also own a stable wire format. That is more commitment than “transport-only”.

## 4) Remove JSON and base64 handling from the core

Your comment says: “When using JSON codec, Body is base64-encoded.” In the code, `Body []byte` with `json.Marshal` will indeed become base64. That is correct, but it is a surprising behavior to many learners.

If you keep JSON:

* switch to `json.RawMessage` for payloads and let callers decide encoding.
  If you want opaque binary payloads:
* use WebSocket binary frames and do not wrap in JSON at all.

Minimal approach for learners:

* support only text JSON frames, where the payload is a JSON object and `data` is `json.RawMessage`.
* or support only binary frames and let higher layers define the schema.

Do not support both in v1.

## 5) Simplify backpressure and shutdown semantics

Your “queue fills => close session” rule is good. But the current implementation has hidden edges:

* `writeLoop` reads from `session.sendCh` without checking ok. When `doneCh` closes you send a close frame, but you never close `sendCh`. That is fine, but it complicates mental models.
* `closeWithError` discards the error (`_ = err`). `OnClose` receives `readErr` from read loop, not necessarily the reason `Send` closed the session (queue full). This makes debugging harder.

For minimal and predictable behavior:

* store a `closeErr` on the session (atomic or mutex-protected)
* always call `OnClose(session, closeErr)` with the true reason
* make `Close()` idempotent and always unblock loops cleanly

This adds a few lines but reduces surprise.

## 6) Remove server-wide session bookkeeping complexity

* `sync.Map` plus an atomic counter is fine, but learners do not need it.
* You can keep `sync.Map` for easy concurrent iteration, but you can remove `SessionCount` entirely in v1.
* `Broadcast` is not essential either (nice-to-have). It is one extra concept.

Minimal API: keep only topics. If you want broadcast, implement it as publishing to a reserved topic like `""` or `"*"` at higher layer.

## 7) Biggest removal: do not implement WebSocket framing yourself (if you can)

If your project constraints allow any dependency, the simplest way to remove 60–70% of this file is to use a small well-tested WebSocket implementation.

If you have a “no deps” rule, keep your code, but hide it:

* `internal/ws` contains handshake + frame read/write.
* `live` package only sees an interface like:

```go
type conn interface {
    Read() (opcode int, payload []byte, err error)
    Write(opcode int, payload []byte) error
    Close() error
}
```

This turns RFC 6455 from a learner concern into a private implementation detail.

## 8) What I would keep in v1 (minimal public API)

For fastest learning and least surprise, I would make live a tiny pubsub over WebSocket with a single payload type:

* inbound: `{topic, data}` messages from client (optional)
* outbound: same struct

Public surface:

* `type Server`
* `func New(opts Options) *Server`
* `func (s *Server) Handler() http.Handler`
* `func (s *Server) Publish(topic string, data []byte)` (or `Message`)
* `func (s *Server) Subscribe(sess *Session, topic string)`
* `func (s *Server) Unsubscribe(sess *Session, topic string)`
* `type Session` with `ID()` and `Send(topic, data)` (or `Send(Message)`)

Options:

* `OnAuth(ctx, r) (any, error)` optional
* `OnMessage(ctx, sess, topic string, data []byte)` optional
* `QueueSize`, `ReadLimit`, `CheckOrigin`, `IDGenerator`

Everything else becomes internal.

## 9) Specific code you can delete immediately

If you want “remove as much as possible” without redesign:

* Delete `Meta.Get` and `Meta.GetString` (nice but not essential).
* Delete `Broadcast` and `SessionCount` (extra API surface).
* Delete `ErrInvalidVersion` (you never return it; you return HTTP errors instead).
* Delete `ErrAuthFailed` (same: you return HTTP errors; keep one model).
* Delete `count()` method on `memPubSub` if unused.
* Delete `ErrProtocolError` as exported; keep it internal to ws parsing.

These are low-risk reductions.

## Bottom line

The conceptual overhead is not in your pubsub logic, it is in (a) owning a full WebSocket implementation in the same package and (b) defining a message envelope that implies a protocol while claiming you do not.

If you want learners to “get it in 5 minutes”, the fastest win is:

1. hide WebSocket internals,
2. reduce Message to `{topic,data}` or `Publish(topic, []byte)`,
3. move subscribe/publish command semantics out of this package (higher layer),
4. remove Meta helpers and extra APIs like Broadcast/SessionCount.

If you want, I can provide a concrete reduced `live` package skeleton (public API plus internal ws boundary) that keeps your current behavior but cuts the exported surface to the minimum.
