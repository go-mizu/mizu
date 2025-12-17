# view/live Package Proposal

## Purpose

Package `view/live` provides a **browser-oriented live view runtime** built on top of `live` for low-latency UI interaction.

It defines the minimal client and server contract needed to:

- Mount a view session
- Send UI events (click, input, submit, custom)
- Receive UI updates as patches
- Handle reconnect and resubscribe
- Integrate with `view/sync` for offline-first correctness

`view/live` is **not authoritative**. If a connection drops, UI state must recover via `view/sync` (or snapshot and pull via sync). The live channel exists to reduce latency and improve UX.

---

## Design principles

1. **UI-focused transport**
   Optimized for browser-to-server UI event flow and server-to-browser patch delivery.

2. **Non-authoritative**
   Live updates are accelerators. Source of truth lives in `view/sync` or server state.

3. **Reconnect-safe**
   Sessions can reconnect and resume without data loss (via sync recovery).

4. **Event-driven**
   UI events trigger server handlers that return patches or redirect to sync.

5. **Minimal surface area**
   Thin layer over `live` with UI-specific conventions.

6. **Progressive enhancement**
   Works with or without JavaScript; enhances existing HTML.

---

## Package layout

```
view/live/
  doc.go          // Package documentation
  handler.go      // Live view handler
  session.go      // View session management
  events.go       // UI event types
  patches.go      // Patch types for UI updates
  mount.go        // View mounting logic
  errors.go       // Error values
```

---

## Core concepts

### Handler

`Handler` manages live view sessions for a specific view.

```go
type Handler struct {
    // unexported fields
}
```

```go
type Options struct {
    // Live is the underlying live server
    Live *live.Server

    // Mount is called when a new session mounts the view
    Mount func(ctx context.Context, s *Session) error

    // HandleEvent is called when a UI event is received
    HandleEvent func(ctx context.Context, s *Session, event Event) error

    // OnDisconnect is called when a session disconnects
    OnDisconnect func(s *Session)

    // TopicPrefix is the prefix for view topics (default: "view:")
    TopicPrefix string
}
```

```go
func NewHandler(opts Options) *Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (h *Handler) HandleMessage(ctx context.Context, s *live.Session, msg live.Message)
```

---

### Session

`Session` represents a connected live view session.

```go
type Session struct {
    // unexported
}
```

```go
func (s *Session) ID() string
func (s *Session) Meta() live.Meta
func (s *Session) ViewID() string

// State management
func (s *Session) Get(key string) any
func (s *Session) Set(key string, value any)
func (s *Session) Assign(assigns map[string]any)
func (s *Session) Assigns() map[string]any

// Sending updates
func (s *Session) Push(patch Patch) error
func (s *Session) PushMulti(patches []Patch) error
func (s *Session) Redirect(url string) error

// Lifecycle
func (s *Session) Close() error
```

Notes:

* `Assign` is the primary way to update view state
* `Push` sends patches to the connected client
* State is lost on disconnect (recovered via sync)

---

### Event

`Event` represents a UI event from the client.

```go
type Event struct {
    Type    string         `json:"type"`    // "click", "input", "submit", "custom"
    Target  string         `json:"target"`  // Element identifier (data-live-* attribute)
    Name    string         `json:"name"`    // Event name for custom events
    Value   string         `json:"value"`   // Input value or form data
    Data    map[string]any `json:"data"`    // Additional event data
}
```

Standard event types:

| Type | Description |
|------|-------------|
| `click` | Element click |
| `input` | Input value change |
| `change` | Select/checkbox change |
| `submit` | Form submission |
| `focus` | Element focus |
| `blur` | Element blur |
| `keydown` | Key press |
| `keyup` | Key release |
| `custom` | Application-defined event |

---

### Patch

`Patch` represents a UI update to send to the client.

```go
type Patch struct {
    Op     string `json:"op"`     // Operation type
    Target string `json:"target"` // Target element selector or ID
    HTML   string `json:"html"`   // HTML content for replace/append/prepend
    Attr   string `json:"attr"`   // Attribute name for attr operations
    Value  string `json:"value"`  // Value for attr/class operations
}
```

Patch operations:

| Op | Description |
|----|-------------|
| `replace` | Replace element innerHTML |
| `outer` | Replace element outerHTML |
| `append` | Append to element |
| `prepend` | Prepend to element |
| `remove` | Remove element |
| `attr` | Set attribute |
| `removeAttr` | Remove attribute |
| `addClass` | Add CSS class |
| `removeClass` | Remove CSS class |
| `toggleClass` | Toggle CSS class |
| `show` | Show element (remove hidden) |
| `hide` | Hide element (add hidden) |
| `focus` | Focus element |
| `blur` | Blur element |
| `dispatch` | Dispatch custom event |
| `redirect` | Navigate to URL |
| `reload` | Reload page |

---

### Mount

`Mount` configures the initial view state on connection.

```go
type Mount struct {
    ViewID  string         // Unique view identifier
    Assigns map[string]any // Initial state
    Params  map[string]any // URL parameters
}
```

Mount lifecycle:

1. Client connects and sends mount message
2. Server creates session with initial state
3. `Mount` callback invoked
4. Initial patches sent to client
5. Session ready for events

---

## Message protocol

### Client to server

**Mount request:**
```json
{
    "type": "mount",
    "topic": "view:counter",
    "ref": "1",
    "body": {"viewId": "abc123", "params": {"id": "42"}}
}
```

**Event message:**
```json
{
    "type": "event",
    "topic": "view:counter",
    "ref": "2",
    "body": {
        "type": "click",
        "target": "increment-btn",
        "data": {}
    }
}
```

**Heartbeat:**
```json
{
    "type": "heartbeat",
    "topic": "view:counter",
    "ref": "3"
}
```

### Server to client

**Mount success:**
```json
{
    "type": "mounted",
    "topic": "view:counter",
    "ref": "1",
    "body": {"sessionId": "xyz789"}
}
```

**Patch message:**
```json
{
    "type": "patch",
    "topic": "view:counter",
    "body": {
        "patches": [
            {"op": "replace", "target": "#count", "html": "<span>5</span>"}
        ]
    }
}
```

**Error message:**
```json
{
    "type": "error",
    "topic": "view:counter",
    "ref": "2",
    "body": {"code": "invalid_event", "message": "Unknown event type"}
}
```

---

## HTML integration

### Declarative event binding

```html
<!-- Click events -->
<button data-live-click="increment">+1</button>
<button data-live-click="decrement">-1</button>

<!-- Input events -->
<input data-live-input="search" data-live-debounce="300" />

<!-- Form submission -->
<form data-live-submit="create-todo">
    <input name="title" />
    <button type="submit">Add</button>
</form>

<!-- Change events -->
<select data-live-change="filter">
    <option value="all">All</option>
    <option value="active">Active</option>
</select>

<!-- Key events -->
<input data-live-keydown="handle-key" data-live-key="Enter" />

<!-- Custom events with data -->
<div data-live-click="select-item" data-live-value="{{.ID}}">
    {{.Name}}
</div>
```

### Element identification

```html
<!-- Target for patches -->
<div id="todo-list">
    {{range .Todos}}
        <div id="todo-{{.ID}}">{{.Title}}</div>
    {{end}}
</div>

<!-- Container for append/prepend -->
<ul data-live-container="messages">
    <!-- New items appended here -->
</ul>
```

---

## Server-side usage

### Basic handler setup

```go
handler := viewlive.NewHandler(viewlive.Options{
    Live: liveServer,
    Mount: func(ctx context.Context, s *viewlive.Session) error {
        // Initialize state
        s.Assign(map[string]any{
            "count": 0,
        })
        return nil
    },
    HandleEvent: func(ctx context.Context, s *viewlive.Session, event viewlive.Event) error {
        switch event.Target {
        case "increment":
            count := s.Get("count").(int) + 1
            s.Set("count", count)
            return s.Push(viewlive.Patch{
                Op:     "replace",
                Target: "#count",
                HTML:   fmt.Sprintf("<span>%d</span>", count),
            })
        case "decrement":
            count := s.Get("count").(int) - 1
            s.Set("count", count)
            return s.Push(viewlive.Patch{
                Op:     "replace",
                Target: "#count",
                HTML:   fmt.Sprintf("<span>%d</span>", count),
            })
        }
        return nil
    },
})

app.Get("/counter", handler.ServeHTTP)
```

### Integration with live server

```go
liveServer := live.New(live.Options{
    OnAuth: authenticate,
    OnMessage: func(ctx context.Context, s *live.Session, msg live.Message) {
        // Route view messages to handler
        if strings.HasPrefix(msg.Topic, "view:") {
            viewHandler.HandleMessage(ctx, s, msg)
            return
        }
        // Handle other messages...
    },
})
```

---

## Client-side implementation

### JavaScript client

```javascript
class LiveView {
    constructor(options) {
        this.socket = options.socket;    // live.Client instance
        this.viewId = options.viewId;
        this.topic = options.topic || `view:${this.viewId}`;
        this.sessionId = null;
        this.refCounter = 0;
        this.pendingRefs = new Map();
    }

    mount(params = {}) {
        return this.send('mount', {
            viewId: this.viewId,
            params
        });
    }

    sendEvent(event) {
        return this.send('event', event);
    }

    send(type, body) {
        const ref = String(++this.refCounter);
        return new Promise((resolve, reject) => {
            this.pendingRefs.set(ref, { resolve, reject });
            this.socket.send({
                type,
                topic: this.topic,
                ref,
                body
            });
        });
    }

    handleMessage(msg) {
        // Resolve pending ref
        if (msg.ref && this.pendingRefs.has(msg.ref)) {
            const { resolve, reject } = this.pendingRefs.get(msg.ref);
            this.pendingRefs.delete(msg.ref);
            if (msg.type === 'error') {
                reject(new Error(msg.body.message));
            } else {
                resolve(msg.body);
            }
        }

        // Handle patches
        if (msg.type === 'patch') {
            this.applyPatches(msg.body.patches);
        }
    }

    applyPatches(patches) {
        for (const patch of patches) {
            this.applyPatch(patch);
        }
    }

    applyPatch(patch) {
        const el = document.querySelector(patch.target);
        if (!el) return;

        switch (patch.op) {
            case 'replace':
                el.innerHTML = patch.html;
                break;
            case 'outer':
                el.outerHTML = patch.html;
                break;
            case 'append':
                el.insertAdjacentHTML('beforeend', patch.html);
                break;
            case 'prepend':
                el.insertAdjacentHTML('afterbegin', patch.html);
                break;
            case 'remove':
                el.remove();
                break;
            case 'attr':
                el.setAttribute(patch.attr, patch.value);
                break;
            case 'removeAttr':
                el.removeAttribute(patch.attr);
                break;
            case 'addClass':
                el.classList.add(patch.value);
                break;
            case 'removeClass':
                el.classList.remove(patch.value);
                break;
            case 'toggleClass':
                el.classList.toggle(patch.value);
                break;
            case 'show':
                el.hidden = false;
                break;
            case 'hide':
                el.hidden = true;
                break;
            case 'focus':
                el.focus();
                break;
            case 'blur':
                el.blur();
                break;
            case 'redirect':
                window.location.href = patch.value;
                break;
            case 'reload':
                window.location.reload();
                break;
        }
    }
}
```

### Auto-binding events

```javascript
function bindLiveEvents(view) {
    // Click events
    document.querySelectorAll('[data-live-click]').forEach(el => {
        el.addEventListener('click', (e) => {
            e.preventDefault();
            view.sendEvent({
                type: 'click',
                target: el.dataset.liveClick,
                value: el.dataset.liveValue || '',
                data: getDataAttributes(el)
            });
        });
    });

    // Input events with debounce
    document.querySelectorAll('[data-live-input]').forEach(el => {
        const debounce = parseInt(el.dataset.liveDebounce) || 0;
        let timeout;
        el.addEventListener('input', (e) => {
            clearTimeout(timeout);
            timeout = setTimeout(() => {
                view.sendEvent({
                    type: 'input',
                    target: el.dataset.liveInput,
                    value: el.value
                });
            }, debounce);
        });
    });

    // Form submission
    document.querySelectorAll('[data-live-submit]').forEach(form => {
        form.addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            view.sendEvent({
                type: 'submit',
                target: form.dataset.liveSubmit,
                data
            });
        });
    });
}
```

---

## Integration with view/sync

### Offline-first pattern

When `view/live` is combined with `view/sync`:

1. **Initial render**: Server renders HTML using sync data
2. **Live connection**: Client connects to live view
3. **Events via live**: UI events sent through live for instant feedback
4. **Mutations via sync**: State changes queue sync mutations
5. **Patches via live**: Server pushes UI patches immediately
6. **Disconnect recovery**: On reconnect, sync restores state

```go
handler := viewlive.NewHandler(viewlive.Options{
    HandleEvent: func(ctx context.Context, s *viewlive.Session, event viewlive.Event) error {
        switch event.Target {
        case "toggle-todo":
            todoID := event.Data["id"].(string)

            // Queue mutation via sync
            syncEngine.Mutate(ctx, sync.Mutation{
                Name:  "todo.toggle",
                Scope: s.Meta().GetString("scope"),
                Args:  map[string]any{"id": todoID},
            })

            // Immediate UI feedback via live
            return s.Push(viewlive.Patch{
                Op:     "toggleClass",
                Target: fmt.Sprintf("#todo-%s", todoID),
                Value:  "done",
            })
        }
        return nil
    },
})
```

### Reconnection flow

```
1. Client disconnects
2. view/live session closed
3. Client reconnects
4. view/sync pulls latest state
5. Client re-mounts view/live
6. Server rebuilds session from sync state
7. UI reflects current state
```

---

## Error handling

```go
var (
    ErrSessionNotFound  = errors.New("viewlive: session not found")
    ErrViewNotMounted   = errors.New("viewlive: view not mounted")
    ErrInvalidEvent     = errors.New("viewlive: invalid event")
    ErrEventFailed      = errors.New("viewlive: event handler failed")
)
```

Errors are handled via:

* Error messages sent to client
* `OnDisconnect` callback for cleanup
* Sync recovery for state restoration

---

## What this package deliberately avoids

* Full server-side DOM diffing (not Phoenix LiveView)
* Component lifecycle (use templates)
* Client-side state management (use sync)
* Automatic optimistic updates (explicit via sync)
* Complex routing (use Mizu router)

---

## Relationship to other packages

* **live**: Provides WebSocket transport and pub/sub
* **view**: Template rendering engine
* **view/sync**: Client-side state and offline support
* **sync**: Server-side mutation engine

The architecture:

```
Browser                    Server
-------                    ------
view/sync  <--> HTTP <-->  sync
    |                        |
    v                        v
view/live  <--> WS  <-->  live
    |                        |
    v                        v
DOM patches              view templates
```

---

## Why this matches Go core library style

* Short names (`Handler`, `Session`, `Event`, `Patch`)
* Interfaces describe behavior
* Explicit over magic
* Clear ownership of state
* Transport separate from rendering

---

## Summary

* `view/live` is the **browser-oriented live view runtime**
* Thin layer over `live` for UI events and patches
* Non-authoritative: sync provides correctness
* Enables low-latency UI updates
* Progressive enhancement compatible
* Integrates cleanly with sync for offline-first apps
