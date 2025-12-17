# cli/template/live Proposal

## Purpose

The `live` template provides a **real-time interactive application** that demonstrates:

- Server-side live view handler with event processing
- WebSocket-based bidirectional communication
- DOM patching for instant UI updates
- Integration with sync for offline-first correctness
- Progressive enhancement from server-rendered HTML

This template serves as both a learning resource and a production-ready starting point for building interactive, real-time applications with Mizu.

---

## Template description

```json
{
  "name": "live",
  "description": "Real-time interactive app with live views and instant updates",
  "tags": ["go", "mizu", "live", "websocket", "realtime", "interactive"]
}
```

---

## Directory structure

```
{{.Name}}/
├── cmd/
│   └── server/
│       └── main.go              # Server entry point
├── app/
│   └── server/
│       ├── app.go               # Application setup
│       ├── config.go            # Configuration
│       ├── routes.go            # HTTP routes
│       ├── live.go              # Live server setup
│       └── sync.go              # Sync engine setup
├── handler/
│   ├── home.go                  # Home page handler
│   └── counter.go               # Counter live view handler
├── service/
│   └── counter/
│       └── mutator.go           # Counter mutation handlers
├── assets/
│   ├── embed.go                 # Embedded assets
│   ├── views/
│   │   ├── layouts/
│   │   │   └── default.html     # Base layout
│   │   ├── pages/
│   │   │   ├── home.html        # Home page
│   │   │   └── counter.html     # Counter page
│   │   └── components/
│   │       ├── counter.html     # Counter component
│   │       └── sync-status.html # Connection status
│   └── static/
│       ├── css/
│       │   └── app.css          # Application styles
│       └── js/
│           ├── app.js           # Main application script
│           ├── live-client.js   # WebSocket client
│           ├── live-view.js     # Live view implementation
│           └── sync-client.js   # Sync client (optional)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Core features

### 1. Live server setup

WebSocket server with session management:

```go
// app/server/live.go
func NewLiveServer(cfg *Config) *live.Server {
    return live.New(live.Options{
        QueueSize: 256,
        OnAuth: func(ctx context.Context, r *http.Request) (live.Meta, error) {
            // Extract user info from session/token
            return live.Meta{
                "user_id": getUserID(r),
            }, nil
        },
        OnMessage: func(ctx context.Context, s *live.Session, msg live.Message) {
            // Route to appropriate handler based on topic prefix
            switch {
            case strings.HasPrefix(msg.Topic, "view:"):
                handleViewMessage(ctx, s, msg)
            case strings.HasPrefix(msg.Topic, "sync:"):
                handleSyncMessage(ctx, s, msg)
            }
        },
        OnClose: func(s *live.Session, err error) {
            log.Printf("Session %s closed: %v", s.ID(), err)
        },
    })
}
```

### 2. Live view handler

Counter live view with event handling:

```go
// handler/counter.go
type CounterHandler struct {
    live      *live.Server
    sessions  map[string]*CounterSession
    mu        sync.RWMutex
}

type CounterSession struct {
    session *live.Session
    count   int
}

func (h *CounterHandler) HandleMessage(ctx context.Context, s *live.Session, msg live.Message) {
    switch msg.Type {
    case "mount":
        h.handleMount(ctx, s, msg)
    case "event":
        h.handleEvent(ctx, s, msg)
    }
}

func (h *CounterHandler) handleMount(ctx context.Context, s *live.Session, msg live.Message) {
    h.mu.Lock()
    h.sessions[s.ID()] = &CounterSession{
        session: s,
        count:   0,
    }
    h.mu.Unlock()

    // Send mounted confirmation
    s.Send(live.Message{
        Type:  "mounted",
        Topic: msg.Topic,
        Ref:   msg.Ref,
        Body:  []byte(`{"sessionId":"` + s.ID() + `"}`),
    })
}

func (h *CounterHandler) handleEvent(ctx context.Context, s *live.Session, msg live.Message) {
    var event struct {
        Type   string `json:"type"`
        Target string `json:"target"`
    }
    json.Unmarshal(msg.Body, &event)

    h.mu.Lock()
    cs := h.sessions[s.ID()]
    if cs == nil {
        h.mu.Unlock()
        return
    }

    switch event.Target {
    case "increment":
        cs.count++
    case "decrement":
        cs.count--
    case "reset":
        cs.count = 0
    }
    count := cs.count
    h.mu.Unlock()

    // Send patch
    patch := fmt.Sprintf(`{"patches":[{"op":"replace","target":"#count","html":"%d"}]}`, count)
    s.Send(live.Message{
        Type:  "patch",
        Topic: msg.Topic,
        Body:  []byte(patch),
    })
}
```

### 3. Server-rendered initial state

Counter page with live attributes:

```html
<!-- assets/views/pages/counter.html -->
{{define "content"}}
<div class="counter-container" data-live-view="counter">
    <h1>Live Counter</h1>

    <div class="counter-display">
        <span id="count">{{.Count}}</span>
    </div>

    <div class="counter-controls">
        <button data-live-click="decrement" class="btn btn-secondary">
            -
        </button>
        <button data-live-click="reset" class="btn btn-outline">
            Reset
        </button>
        <button data-live-click="increment" class="btn btn-primary">
            +
        </button>
    </div>

    <div id="connection-status" class="status">
        <span class="status-dot"></span>
        <span class="status-text">Connecting...</span>
    </div>
</div>
{{end}}

{{define "scripts"}}
<script src="/static/js/live-client.js"></script>
<script src="/static/js/live-view.js"></script>
<script>
    const liveClient = new LiveClient({
        url: 'ws://' + window.location.host + '/ws',
        onConnect: () => updateStatus('connected'),
        onDisconnect: () => updateStatus('disconnected')
    });

    const counterView = new LiveView({
        socket: liveClient,
        viewId: 'counter',
        element: document.querySelector('[data-live-view="counter"]')
    });

    liveClient.connect();
    counterView.mount();
</script>
{{end}}
```

### 4. JavaScript live client

WebSocket client with reconnection:

```javascript
// assets/static/js/live-client.js
class LiveClient {
    constructor(options) {
        this.url = options.url;
        this.onConnect = options.onConnect || (() => {});
        this.onDisconnect = options.onDisconnect || (() => {});
        this.onMessage = options.onMessage || (() => {});

        this.ws = null;
        this.connected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 1000;
        this.handlers = new Map();
    }

    connect() {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            this.connected = true;
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;
            this.onConnect();

            // Resubscribe to topics
            this.handlers.forEach((handler, topic) => {
                this.subscribe(topic);
            });
        };

        this.ws.onclose = () => {
            this.connected = false;
            this.onDisconnect();
            this.scheduleReconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            this.onMessage(msg);

            // Route to registered handler
            if (msg.topic && this.handlers.has(msg.topic)) {
                this.handlers.get(msg.topic)(msg);
            }
        };
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    send(msg) {
        if (this.connected && this.ws) {
            this.ws.send(JSON.stringify(msg));
        }
    }

    subscribe(topic) {
        this.send({
            type: 'subscribe',
            topic: topic
        });
    }

    unsubscribe(topic) {
        this.send({
            type: 'unsubscribe',
            topic: topic
        });
    }

    register(topic, handler) {
        this.handlers.set(topic, handler);
        if (this.connected) {
            this.subscribe(topic);
        }
    }

    unregister(topic) {
        this.handlers.delete(topic);
        if (this.connected) {
            this.unsubscribe(topic);
        }
    }

    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('Max reconnection attempts reached');
            return;
        }

        setTimeout(() => {
            this.reconnectAttempts++;
            this.connect();
        }, this.reconnectDelay);

        // Exponential backoff
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
    }
}
```

### 5. Live view JavaScript

Live view abstraction with auto-binding:

```javascript
// assets/static/js/live-view.js
class LiveView {
    constructor(options) {
        this.socket = options.socket;
        this.viewId = options.viewId;
        this.element = options.element;
        this.topic = `view:${this.viewId}`;
        this.sessionId = null;
        this.mounted = false;
        this.refCounter = 0;
        this.pendingRefs = new Map();

        // Register message handler
        this.socket.register(this.topic, (msg) => this.handleMessage(msg));
    }

    mount(params = {}) {
        const ref = this.nextRef();
        return new Promise((resolve, reject) => {
            this.pendingRefs.set(ref, { resolve, reject, type: 'mount' });
            this.socket.send({
                type: 'mount',
                topic: this.topic,
                ref: ref,
                body: JSON.stringify({
                    viewId: this.viewId,
                    params: params
                })
            });
        }).then((result) => {
            this.sessionId = result.sessionId;
            this.mounted = true;
            this.bindEvents();
            return result;
        });
    }

    unmount() {
        this.socket.unregister(this.topic);
        this.unbindEvents();
        this.mounted = false;
    }

    sendEvent(event) {
        if (!this.mounted) return Promise.reject(new Error('View not mounted'));

        const ref = this.nextRef();
        this.socket.send({
            type: 'event',
            topic: this.topic,
            ref: ref,
            body: JSON.stringify(event)
        });
    }

    handleMessage(msg) {
        // Handle pending refs
        if (msg.ref && this.pendingRefs.has(msg.ref)) {
            const pending = this.pendingRefs.get(msg.ref);
            this.pendingRefs.delete(msg.ref);

            if (msg.type === 'error') {
                pending.reject(new Error(JSON.parse(msg.body).message));
            } else {
                pending.resolve(JSON.parse(msg.body));
            }
            return;
        }

        // Handle patches
        if (msg.type === 'patch') {
            const data = JSON.parse(msg.body);
            this.applyPatches(data.patches);
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
                this.rebindEvents();
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
            case 'redirect':
                window.location.href = patch.value;
                break;
            case 'reload':
                window.location.reload();
                break;
        }
    }

    bindEvents() {
        this.unbindEvents();
        this.boundHandlers = [];

        // Click events
        this.element.querySelectorAll('[data-live-click]').forEach(el => {
            const handler = (e) => {
                e.preventDefault();
                this.sendEvent({
                    type: 'click',
                    target: el.dataset.liveClick,
                    value: el.dataset.liveValue || '',
                    data: this.getDataAttributes(el)
                });
            };
            el.addEventListener('click', handler);
            this.boundHandlers.push({ el, event: 'click', handler });
        });

        // Input events
        this.element.querySelectorAll('[data-live-input]').forEach(el => {
            const debounce = parseInt(el.dataset.liveDebounce) || 0;
            let timeout;
            const handler = (e) => {
                clearTimeout(timeout);
                timeout = setTimeout(() => {
                    this.sendEvent({
                        type: 'input',
                        target: el.dataset.liveInput,
                        value: el.value
                    });
                }, debounce);
            };
            el.addEventListener('input', handler);
            this.boundHandlers.push({ el, event: 'input', handler });
        });

        // Change events
        this.element.querySelectorAll('[data-live-change]').forEach(el => {
            const handler = (e) => {
                this.sendEvent({
                    type: 'change',
                    target: el.dataset.liveChange,
                    value: el.type === 'checkbox' ? el.checked : el.value
                });
            };
            el.addEventListener('change', handler);
            this.boundHandlers.push({ el, event: 'change', handler });
        });

        // Form submissions
        this.element.querySelectorAll('[data-live-submit]').forEach(form => {
            const handler = (e) => {
                e.preventDefault();
                const formData = new FormData(form);
                this.sendEvent({
                    type: 'submit',
                    target: form.dataset.liveSubmit,
                    data: Object.fromEntries(formData)
                });
            };
            form.addEventListener('submit', handler);
            this.boundHandlers.push({ el: form, event: 'submit', handler });
        });

        // Key events
        this.element.querySelectorAll('[data-live-keydown]').forEach(el => {
            const targetKey = el.dataset.liveKey;
            const handler = (e) => {
                if (!targetKey || e.key === targetKey) {
                    this.sendEvent({
                        type: 'keydown',
                        target: el.dataset.liveKeydown,
                        value: el.value,
                        data: { key: e.key }
                    });
                }
            };
            el.addEventListener('keydown', handler);
            this.boundHandlers.push({ el, event: 'keydown', handler });
        });
    }

    unbindEvents() {
        if (this.boundHandlers) {
            this.boundHandlers.forEach(({ el, event, handler }) => {
                el.removeEventListener(event, handler);
            });
            this.boundHandlers = [];
        }
    }

    rebindEvents() {
        // Re-query element after outer replace
        this.element = document.querySelector(`[data-live-view="${this.viewId}"]`);
        if (this.element) {
            this.bindEvents();
        }
    }

    getDataAttributes(el) {
        const data = {};
        for (const attr of el.attributes) {
            if (attr.name.startsWith('data-') && !attr.name.startsWith('data-live-')) {
                const key = attr.name.slice(5).replace(/-./g, x => x[1].toUpperCase());
                data[key] = attr.value;
            }
        }
        return data;
    }

    nextRef() {
        return String(++this.refCounter);
    }
}
```

---

## Data flow

### Event flow (click)

1. User clicks button with `data-live-click="increment"`
2. LiveView sends event message to server
3. Server handler updates state
4. Server sends patch message
5. LiveView applies patch to DOM
6. User sees updated UI (typically <50ms)

### Reconnection flow

1. WebSocket disconnects
2. UI shows "Disconnected" status
3. Client attempts reconnect with backoff
4. On reconnect, view re-mounts
5. Server rebuilds session state
6. UI shows "Connected" status

---

## API endpoints

### WebSocket endpoint

```
GET /ws    # WebSocket upgrade
```

### HTTP endpoints

```
GET /            # Home page
GET /counter     # Counter page (initial render)
```

---

## Configuration

```go
type Config struct {
    Addr string // Server address
    Dev  bool   // Development mode
}
```

Environment variables:

```
ADDR=:8080
DEV=true
```

---

## Template variables

| Variable | Description | Default |
|----------|-------------|---------|
| `Name` | Project name | Directory name |
| `Module` | Go module path | `example.com/{name}` |
| `License` | License identifier | `MIT` |

---

## Usage

```bash
# Create new project
mizu new myapp --template live

# Navigate and install
cd myapp
go mod tidy

# Run development server
mizu dev

# Or run directly
go run ./cmd/server

# Build for production
go build -o server ./cmd/server
```

---

## Development workflow

1. **Start server**: `mizu dev` watches for changes
2. **Edit templates**: Hot reload in development mode
3. **Add views**: Create handler and template
4. **Add events**: Use data-live-* attributes
5. **Test reconnection**: Disconnect/reconnect in browser

---

## Adding a new live view

### 1. Create handler

```go
// handler/chat.go
type ChatHandler struct {
    live *live.Server
    // ...
}

func (h *ChatHandler) HandleMessage(ctx context.Context, s *live.Session, msg live.Message) {
    // Handle mount and events
}
```

### 2. Create template

```html
<!-- assets/views/pages/chat.html -->
{{define "content"}}
<div data-live-view="chat">
    <div id="messages">
        {{range .Messages}}
            <div class="message">{{.Text}}</div>
        {{end}}
    </div>
    <form data-live-submit="send-message">
        <input name="text" data-live-input="typing" />
        <button type="submit">Send</button>
    </form>
</div>
{{end}}
```

### 3. Register route

```go
// app/server/routes.go
app.Get("/chat", handler.ChatPage)
```

### 4. Register handler

```go
// app/server/live.go
chatHandler := handler.NewChatHandler(liveServer)
// Route messages in OnMessage callback
```

---

## Production considerations

### Performance

- Batch patches when possible
- Use debounce for input events
- Limit concurrent sessions per server
- Consider sticky sessions for scaling

### Security

- Validate all event data
- Authenticate WebSocket connections
- Rate limit event processing
- Sanitize HTML in patches

### Scaling

- Use Redis pub/sub for multi-instance
- Implement session affinity
- Consider connection limits
- Monitor WebSocket memory usage

---

## Learning path

The template demonstrates:

1. **WebSocket basics**: Connection, messages, reconnection
2. **Live views**: Server-driven UI updates
3. **Event handling**: Click, input, submit
4. **DOM patching**: Efficient partial updates
5. **Progressive enhancement**: Works without JS initially

---

## Dependencies

Server:
- `github.com/go-mizu/mizu` - Web framework
- `github.com/go-mizu/mizu/live` - WebSocket server
- `github.com/go-mizu/mizu/view` - Template engine

Client (bundled):
- No external dependencies
- Vanilla JavaScript

---

## Example: Todo list with live updates

### Template

```html
<div data-live-view="todos">
    <form data-live-submit="add-todo">
        <input name="title" placeholder="What needs to be done?"
               data-live-keydown="add-todo" data-live-key="Enter" />
    </form>

    <ul id="todo-list">
        {{range .Todos}}
        <li id="todo-{{.ID}}" class="{{if .Done}}done{{end}}">
            <input type="checkbox" {{if .Done}}checked{{end}}
                   data-live-change="toggle" data-live-value="{{.ID}}" />
            <span>{{.Title}}</span>
            <button data-live-click="delete" data-live-value="{{.ID}}">×</button>
        </li>
        {{end}}
    </ul>

    <div class="filters">
        <button data-live-click="filter" data-live-value="all">All</button>
        <button data-live-click="filter" data-live-value="active">Active</button>
        <button data-live-click="filter" data-live-value="done">Done</button>
    </div>
</div>
```

### Handler

```go
func (h *TodoHandler) handleEvent(ctx context.Context, s *live.Session, msg live.Message) {
    var event Event
    json.Unmarshal(msg.Body, &event)

    session := h.getSession(s.ID())

    switch event.Target {
    case "add-todo":
        if event.Type == "submit" || (event.Type == "keydown" && event.Data["key"] == "Enter") {
            todo := &Todo{
                ID:    generateID(),
                Title: event.Data["title"].(string),
            }
            session.Todos = append(session.Todos, todo)

            // Append new todo to list
            html := renderTodoItem(todo)
            s.Send(patchMessage(msg.Topic, []Patch{
                {Op: "append", Target: "#todo-list", HTML: html},
            }))
        }

    case "toggle":
        todoID := event.Value
        for _, t := range session.Todos {
            if t.ID == todoID {
                t.Done = !t.Done
                s.Send(patchMessage(msg.Topic, []Patch{
                    {Op: "toggleClass", Target: "#todo-" + todoID, Value: "done"},
                }))
                break
            }
        }

    case "delete":
        todoID := event.Value
        session.removeTodo(todoID)
        s.Send(patchMessage(msg.Topic, []Patch{
            {Op: "remove", Target: "#todo-" + todoID},
        }))
    }
}
```

---

## Summary

The `live` template provides:

- Complete WebSocket-based live view implementation
- Server and client code for real-time interaction
- DOM patching for efficient updates
- Reconnection handling
- Progressive enhancement patterns
- Clear examples for building interactive features

It serves as both a reference implementation and a starting point for building real-time interactive applications with Mizu.
