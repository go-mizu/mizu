# Live Wire Protocol Specification

**Package**: `view/live`
**Status**: Draft
**Version**: 1.0

## Overview

This document specifies the WebSocket wire protocol between the Mizu Live client runtime and server. The protocol enables:

- Bidirectional event-driven communication
- Efficient HTML patch delivery
- Connection management (heartbeat, reconnection)
- Error handling and recovery

## Protocol Basics

### Transport

- **Protocol**: WebSocket (RFC 6455)
- **Endpoint**: `/_live/websocket`
- **Subprotocol**: `mizu-live-v1`
- **Encoding**: MessagePack (binary) or JSON (text, for debugging)

### Message Format

All messages follow this envelope structure:

```
┌─────────────────────────────────────────┐
│  type  │  ref  │  topic  │   payload   │
│  (u8)  │ (u32) │ (string)│   (varies)  │
└─────────────────────────────────────────┘
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | uint8 | Message type identifier |
| `ref` | uint32 | Request reference for replies (0 for push) |
| `topic` | string | Target topic/channel (empty for session messages) |
| `payload` | varies | Type-specific payload |

### Message Types

| Type | Code | Direction | Description |
|------|------|-----------|-------------|
| `JOIN` | 0x01 | C→S | Join/create session |
| `LEAVE` | 0x02 | C→S | Leave session |
| `EVENT` | 0x03 | C→S | User event (click, submit, etc.) |
| `HEARTBEAT` | 0x04 | C↔S | Keep-alive ping/pong |
| `REPLY` | 0x05 | S→C | Response to client message |
| `PATCH` | 0x06 | S→C | DOM patch |
| `COMMAND` | 0x07 | S→C | Client-side command |
| `ERROR` | 0x08 | S→C | Error notification |
| `REDIRECT` | 0x09 | S→C | Navigation redirect |
| `CLOSE` | 0x0A | S→C | Session close |

## Connection Lifecycle

### Initial Connection

```
┌────────┐                              ┌────────┐
│ Client │                              │ Server │
└────┬───┘                              └───┬────┘
     │                                      │
     │  HTTP GET /page (initial render)     │
     │─────────────────────────────────────>│
     │                                      │
     │  HTML + session token + runtime.js   │
     │<─────────────────────────────────────│
     │                                      │
     │  WS /_live/websocket                 │
     │─────────────────────────────────────>│
     │                                      │
     │  WS Handshake Complete               │
     │<─────────────────────────────────────│
     │                                      │
     │  JOIN { token, url, params }         │
     │─────────────────────────────────────>│
     │                                      │
     │  REPLY { status: "ok", session_id }  │
     │<─────────────────────────────────────│
     │                                      │
     │  PATCH { regions: [...] }            │
     │<─────────────────────────────────────│
     │                                      │
```

### Event Flow

```
┌────────┐                              ┌────────┐
│ Client │                              │ Server │
└────┬───┘                              └───┬────┘
     │                                      │
     │  EVENT { name: "inc", values: {} }   │
     │─────────────────────────────────────>│
     │                                      │
     │        [Process event, update state] │
     │                                      │
     │  REPLY { status: "ok" }              │
     │<─────────────────────────────────────│
     │                                      │
     │  PATCH { regions: [...] }            │
     │<─────────────────────────────────────│
     │                                      │
```

### Server Push Flow

```
┌────────┐                              ┌────────┐
│ Client │                              │ Server │
└────┬───┘                              └───┬────┘
     │                                      │
     │       [PubSub message received]      │
     │                                      │
     │  PATCH { regions: [...] }            │
     │<─────────────────────────────────────│
     │                                      │
```

### Disconnection

```
┌────────┐                              ┌────────┐
│ Client │                              │ Server │
└────┬───┘                              └───┬────┘
     │                                      │
     │  LEAVE { }                           │
     │─────────────────────────────────────>│
     │                                      │
     │  CLOSE { reason: "normal" }          │
     │<─────────────────────────────────────│
     │                                      │
     │  WebSocket Close                     │
     │<─────────────────────────────────────│
     │                                      │
```

## Message Specifications

### JOIN (0x01)

Client sends to establish a live session.

**Request Payload:**
```json
{
  "token": "csrf_token_here",
  "url": "/counter",
  "params": {
    "id": "123"
  },
  "session": "existing_session_id_or_null",
  "reconnect": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | CSRF token from initial page |
| `url` | string | Yes | Page URL path |
| `params` | object | No | URL path parameters |
| `session` | string | No | Existing session ID for reconnection |
| `reconnect` | bool | No | True if reconnecting after disconnect |

**Reply Payload (success):**
```json
{
  "status": "ok",
  "session_id": "abc123",
  "rendered": {
    "lv-root": "<div>...</div>"
  }
}
```

**Reply Payload (error):**
```json
{
  "status": "error",
  "reason": "invalid_token"
}
```

Error reasons:
- `invalid_token`: CSRF token invalid or expired
- `session_not_found`: Session ID doesn't exist (reconnection failed)
- `session_expired`: Session has timed out
- `page_not_found`: URL doesn't match a live page

### LEAVE (0x02)

Client sends to cleanly close the session.

**Payload:**
```json
{
  "reason": "navigation"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `reason` | string | No | Why the client is leaving |

Reasons:
- `navigation`: User navigating to another page
- `close`: Tab/window closing
- `reload`: Page reload
- `timeout`: Client-side timeout

### EVENT (0x03)

Client sends user interaction events.

**Payload:**
```json
{
  "name": "save",
  "target": null,
  "values": {
    "id": "123",
    "action": "archive"
  },
  "form": {
    "title": ["My Title"],
    "tags": ["go", "web"]
  },
  "key": null,
  "meta": {
    "shift": false,
    "ctrl": false,
    "alt": false,
    "meta": false
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Event name from `data-lv-*` attribute |
| `target` | string | No | Component target ID |
| `values` | object | No | Values from `data-lv-value-*` attributes |
| `form` | object | No | Form field values (multimap) |
| `key` | string | No | Keyboard key for key events |
| `meta` | object | No | Modifier key state |

**Reply Payload:**
```json
{
  "status": "ok"
}
```

or on error:
```json
{
  "status": "error",
  "reason": "handler_error",
  "message": "Internal server error"
}
```

### HEARTBEAT (0x04)

Bidirectional keep-alive.

**Client → Server (ping):**
```json
{
  "ping": 1702500000
}
```

**Server → Client (pong):**
```json
{
  "pong": 1702500000
}
```

The timestamp is the client's local time in milliseconds. Server echoes it back for latency measurement.

### REPLY (0x05)

Server response to client requests.

**Payload:**
```json
{
  "status": "ok",
  ...additional fields based on request type
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | "ok" or "error" |
| `reason` | string | If error | Error code |
| `message` | string | No | Human-readable error message |

### PATCH (0x06)

Server sends DOM updates.

**Payload:**
```json
{
  "regions": [
    {
      "id": "stats",
      "html": "<div class=\"stats\">...</div>",
      "action": "replace"
    },
    {
      "id": "messages",
      "html": "<div class=\"message\">New!</div>",
      "action": "append"
    }
  ],
  "title": "New Page Title"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `regions` | array | Yes | List of region updates |
| `title` | string | No | New document title |

**Region Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | DOM element ID to update |
| `html` | string | Yes | New HTML content |
| `action` | string | No | Update action (default: "replace") |

Actions:
- `replace`: Replace element's innerHTML (default)
- `morph`: Morphdom-style DOM diffing
- `append`: Append to element
- `prepend`: Prepend to element
- `before`: Insert before element
- `after`: Insert after element
- `remove`: Remove element

### COMMAND (0x07)

Server sends client-side commands.

**Redirect Command:**
```json
{
  "cmd": "redirect",
  "to": "/dashboard",
  "replace": false
}
```

**Focus Command:**
```json
{
  "cmd": "focus",
  "selector": "#email-input"
}
```

**Scroll Command:**
```json
{
  "cmd": "scroll",
  "selector": "#messages",
  "block": "end"
}
```

**Download Command:**
```json
{
  "cmd": "download",
  "url": "/api/export.csv",
  "filename": "report.csv"
}
```

**JavaScript Command:**
```json
{
  "cmd": "js",
  "code": "console.log('hello')",
  "args": {}
}
```

**Flash Command:**
```json
{
  "cmd": "flash",
  "type": "success",
  "message": "Saved successfully!"
}
```

**Loading Command:**
```json
{
  "cmd": "loading",
  "show": true,
  "target": "#submit-btn"
}
```

### ERROR (0x08)

Server sends error notifications.

**Payload:**
```json
{
  "code": "session_expired",
  "message": "Your session has expired",
  "recoverable": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | string | Yes | Error code |
| `message` | string | No | Human-readable message |
| `recoverable` | bool | No | Client should retry |

Error codes:
- `session_expired`: Session timed out, reconnect needed
- `handler_error`: Error in event handler
- `render_error`: Error rendering template
- `rate_limited`: Too many events
- `invalid_event`: Malformed event
- `internal_error`: Server error

### REDIRECT (0x09)

Server sends navigation command (convenience wrapper).

**Payload:**
```json
{
  "to": "/login",
  "replace": false
}
```

### CLOSE (0x0A)

Server sends before closing connection.

**Payload:**
```json
{
  "reason": "session_expired",
  "message": "Please refresh the page"
}
```

## Binary Encoding (MessagePack)

For production, messages are encoded with MessagePack for efficiency.

### Envelope Format

```
┌─────┬─────┬────────────┬──────────────┐
│ 1B  │ 4B  │  2B + str  │   payload    │
│type │ ref │topic_len+s │   msgpack    │
└─────┴─────┴────────────┴──────────────┘
```

- Type: 1 byte unsigned integer
- Ref: 4 bytes big-endian unsigned integer
- Topic: 2 bytes length prefix + UTF-8 string
- Payload: MessagePack-encoded object

### Size Limits

| Limit | Default | Description |
|-------|---------|-------------|
| Max message size | 64 KB | Maximum WebSocket message |
| Max event values | 100 | Maximum data-lv-value-* count |
| Max form fields | 1000 | Maximum form field count |
| Max form value size | 1 MB | Maximum single form value |
| Max patch size | 1 MB | Maximum single patch response |

## Client Runtime Behavior

### Initialization

```javascript
// Runtime loaded from /_live/runtime.js
window.MizuLive = {
  connect(options) {
    // 1. Extract session token from page
    // 2. Open WebSocket connection
    // 3. Send JOIN message
    // 4. Setup event listeners
    // 5. Start heartbeat timer
  }
}

// Auto-initialize if data-lv present
document.addEventListener('DOMContentLoaded', () => {
  if (document.querySelector('[data-lv]')) {
    MizuLive.connect()
  }
})
```

### Event Capture

```javascript
// Click events
document.addEventListener('click', (e) => {
  const target = e.target.closest('[data-lv-click]')
  if (target) {
    e.preventDefault()
    sendEvent('click', target)
  }
}, true)

// Form submit
document.addEventListener('submit', (e) => {
  const form = e.target.closest('[data-lv-submit]')
  if (form) {
    e.preventDefault()
    sendEvent('submit', form)
  }
}, true)

// Change events (with debounce)
document.addEventListener('change', (e) => {
  const target = e.target.closest('[data-lv-change]')
  if (target) {
    const debounce = parseInt(target.dataset.lvDebounce || '150')
    debounceEvent('change', target, debounce)
  }
}, true)
```

### DOM Patching

```javascript
function applyPatch(patch) {
  for (const region of patch.regions) {
    const el = document.getElementById(region.id)
    if (!el) continue

    switch (region.action) {
      case 'replace':
      case 'morph':
        morphdom(el, region.html, {
          onBeforeElUpdated(from, to) {
            // Preserve focus, selection, scroll
            return !from.isEqualNode(to)
          }
        })
        break
      case 'append':
        el.insertAdjacentHTML('beforeend', region.html)
        break
      case 'prepend':
        el.insertAdjacentHTML('afterbegin', region.html)
        break
      case 'remove':
        el.remove()
        break
    }
  }

  if (patch.title) {
    document.title = patch.title
  }
}
```

### Reconnection Strategy

```javascript
const RECONNECT_DELAYS = [1000, 2000, 4000, 8000, 16000, 30000]

function reconnect(attempt = 0) {
  const delay = RECONNECT_DELAYS[Math.min(attempt, RECONNECT_DELAYS.length - 1)]

  setTimeout(() => {
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      send({
        type: JOIN,
        payload: {
          token: getToken(),
          url: location.pathname,
          session: sessionId,
          reconnect: true
        }
      })
    }

    ws.onerror = () => reconnect(attempt + 1)
  }, delay)
}
```

### Loading States

```javascript
function setLoading(event, loading) {
  const target = event.target.closest('[data-lv-loading-class], [data-lv-loading-target]')
  if (!target) return

  const loadingTarget = target.dataset.lvLoadingTarget
    ? document.querySelector(target.dataset.lvLoadingTarget)
    : target

  if (loading) {
    // Add loading class
    if (target.dataset.lvLoadingClass) {
      loadingTarget.classList.add(...target.dataset.lvLoadingClass.split(' '))
    }
    // Show/hide elements
    loadingTarget.querySelectorAll('[data-lv-loading-show]').forEach(el => el.hidden = false)
    loadingTarget.querySelectorAll('[data-lv-loading-hide]').forEach(el => el.hidden = true)
    // Disable inputs
    loadingTarget.querySelectorAll('[data-lv-loading-disable]').forEach(el => el.disabled = true)
  } else {
    // Remove loading states
    if (target.dataset.lvLoadingClass) {
      loadingTarget.classList.remove(...target.dataset.lvLoadingClass.split(' '))
    }
    loadingTarget.querySelectorAll('[data-lv-loading-show]').forEach(el => el.hidden = true)
    loadingTarget.querySelectorAll('[data-lv-loading-hide]').forEach(el => el.hidden = false)
    loadingTarget.querySelectorAll('[data-lv-loading-disable]').forEach(el => el.disabled = false)
  }
}
```

## Server Implementation Notes

### WebSocket Handler

```go
func (l *Live) wsHandler(c *mizu.Ctx) error {
    // Upgrade connection
    conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
        Subprotocols: []string{"mizu-live-v1"},
    })
    if err != nil {
        return err
    }
    defer conn.Close(websocket.StatusNormalClosure, "")

    // Set limits
    conn.SetReadLimit(l.opts.MaxMessageSize)

    // Create session handler
    handler := newSessionHandler(l, conn)

    // Run event loop
    return handler.run(c.Request.Context())
}
```

### Event Loop

```go
func (h *sessionHandler) run(ctx context.Context) error {
    // Channels
    events := make(chan *Message, 16)
    serverMsgs := make(chan any, 16)

    // Read goroutine
    go func() {
        for {
            msg, err := h.readMessage(ctx)
            if err != nil {
                close(events)
                return
            }
            events <- msg
        }
    }()

    // Heartbeat ticker
    heartbeat := time.NewTicker(h.live.opts.HeartbeatInterval)
    defer heartbeat.Stop()

    // Timeout timer
    timeout := time.NewTimer(h.live.opts.SessionTimeout)
    defer timeout.Stop()

    for {
        select {
        case msg, ok := <-events:
            if !ok {
                return nil // Client disconnected
            }
            timeout.Reset(h.live.opts.SessionTimeout)

            if err := h.handleMessage(ctx, msg); err != nil {
                h.sendError(err)
            }

        case msg := <-serverMsgs:
            if err := h.handleServerMessage(ctx, msg); err != nil {
                h.sendError(err)
            }

        case <-heartbeat.C:
            h.sendHeartbeat()

        case <-timeout.C:
            h.sendClose("session_expired", "Session timed out")
            return nil

        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### Patch Generation

```go
func (h *sessionHandler) generatePatches() (*PatchMessage, error) {
    view, err := h.page.Render(h.ctx, h.session)
    if err != nil {
        return nil, err
    }

    patches := &PatchMessage{
        Regions: make([]RegionPatch, 0),
    }

    for id := range h.session.dirty.List() {
        templateName, ok := view.Regions[id]
        if !ok {
            // Not a region, use full page
            templateName = view.Page
        }

        // Render region
        html, err := h.renderRegion(templateName)
        if err != nil {
            return nil, err
        }

        // Check if changed
        oldHTML := h.session.regions[id]
        if html != oldHTML {
            patches.Regions = append(patches.Regions, RegionPatch{
                ID:     id,
                HTML:   html,
                Action: "morph",
            })
            h.session.regions[id] = html
        }
    }

    h.session.dirty.Clear()
    return patches, nil
}
```

## Security Considerations

### CSRF Protection

1. Initial page render includes CSRF token in HTML:
   ```html
   <meta name="csrf-token" content="abc123">
   ```

2. Client includes token in JOIN message

3. Server validates token before creating session

4. Token rotates on authentication changes

### Rate Limiting

```go
type rateLimiter struct {
    events    int
    window    time.Time
    maxEvents int
    windowDur time.Duration
}

func (r *rateLimiter) allow() bool {
    now := time.Now()
    if now.Sub(r.window) > r.windowDur {
        r.events = 0
        r.window = now
    }
    r.events++
    return r.events <= r.maxEvents
}
```

Default limits:
- 100 events per second per session
- 10 JOIN attempts per minute per IP
- 1000 concurrent sessions per server

### Input Validation

- All string values are sanitized server-side
- Maximum lengths enforced
- Form values validated against expected types
- Template output auto-escaped by Go templates

## Debugging

### JSON Mode

For debugging, client can request JSON encoding:

```javascript
const ws = new WebSocket(url + '?format=json')
```

Server responds with JSON-encoded messages instead of MessagePack.

### Debug Logging

Enable verbose logging:

```go
lv := live.New(live.Options{
    Dev: true, // Enables debug logging
})
```

Logs include:
- Connection events
- Message contents
- Patch sizes
- Timing information

### Client Debug Mode

```html
<div data-lv data-lv-debug>
```

Enables:
- Console logging of all messages
- Network panel shows readable JSON
- DOM mutation highlights

## Version Negotiation

### Subprotocol

Client requests subprotocol in WebSocket handshake:

```
Sec-WebSocket-Protocol: mizu-live-v1
```

Server responds with accepted version. If server doesn't support requested version, connection fails.

### Future Versions

- `mizu-live-v1`: Current version (this spec)
- `mizu-live-v2`: Reserved for future incompatible changes

Minor additions (new message types, new fields) are backwards compatible within a major version.

## Metrics

Servers should expose these metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `live_sessions_active` | gauge | Current active sessions |
| `live_connections_total` | counter | Total connections |
| `live_events_total` | counter | Total events processed |
| `live_patches_total` | counter | Total patches sent |
| `live_patch_bytes_total` | counter | Total patch bytes |
| `live_event_duration_seconds` | histogram | Event processing time |
| `live_render_duration_seconds` | histogram | Render time |
| `live_errors_total` | counter | Total errors by type |
