# cli/template/sync Proposal

## Purpose

The `sync` template provides a **full-featured offline-first application** that demonstrates:

- Server-side sync engine with mutations and change log
- Client-side reactive state with optimistic updates
- WebSocket-based live notifications for real-time sync
- Offline operation with mutation queueing
- Template-based server rendering with Mizu view engine

This template serves as both a learning resource and a production-ready starting point for building collaborative, offline-capable applications.

---

## Template description

```json
{
  "name": "sync",
  "description": "Offline-first app with sync, live updates, and reactive state",
  "tags": ["go", "mizu", "sync", "live", "offline-first", "realtime"]
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
│       ├── sync.go              # Sync engine setup
│       └── live.go              # Live server setup
├── handler/
│   ├── home.go                  # Home page handler
│   ├── api.go                   # REST API handlers
│   └── ws.go                    # WebSocket handler
├── service/
│   └── todo/
│       └── mutator.go           # Todo mutation handlers
├── assets/
│   ├── embed.go                 # Embedded assets
│   ├── views/
│   │   ├── layouts/
│   │   │   └── default.html     # Base layout
│   │   ├── pages/
│   │   │   └── home.html        # Home page
│   │   ├── components/
│   │   │   ├── todo-list.html   # Todo list component
│   │   │   ├── todo-item.html   # Todo item component
│   │   │   └── sync-status.html # Sync status indicator
│   │   └── partials/
│   │       ├── header.html      # Page header
│   │       └── footer.html      # Page footer
│   └── static/
│       ├── css/
│       │   └── app.css          # Application styles
│       └── js/
│           ├── app.js           # Main application script
│           ├── sync-client.js   # Sync client implementation
│           └── live-client.js   # WebSocket client
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Core features

### 1. Server-side sync engine

The sync engine handles mutations with:

- **Mutation processing**: Named mutation handlers for business logic
- **Change log**: Ordered, cursor-based change tracking
- **Idempotency**: Duplicate detection via mutation IDs
- **Scoping**: Data partitioned by user/scope

```go
// service/todo/mutator.go
func (m *Mutator) Apply(ctx context.Context, store sync.Store, mut sync.Mutation) ([]sync.Change, error) {
    switch mut.Name {
    case "todo.create":
        return m.createTodo(ctx, store, mut)
    case "todo.update":
        return m.updateTodo(ctx, store, mut)
    case "todo.delete":
        return m.deleteTodo(ctx, store, mut)
    default:
        return nil, sync.ErrUnknownMutation
    }
}
```

### 2. Live WebSocket server

Real-time notifications via WebSocket:

- **Topic-based subscriptions**: Clients subscribe to scope-specific topics
- **Sync integration**: Automatic notifications when data changes
- **Connection management**: Session tracking and cleanup

```go
// app/server/live.go
server := live.New(live.Options{
    OnAuth: authenticateSession,
    OnMessage: func(ctx context.Context, s *live.Session, msg live.Message) {
        switch msg.Type {
        case "subscribe":
            server.PubSub().Subscribe(s, msg.Topic)
        }
    },
})

// Notify sync clients
syncEngine := sync.New(sync.Options{
    Notify: live.SyncNotifier(server, "sync:"),
})
```

### 3. Client-side sync

JavaScript client implementing the sync protocol:

- **Mutation queue**: Persistent queue for offline mutations
- **Optimistic updates**: Apply changes immediately, reconcile on sync
- **Cursor tracking**: Efficient incremental sync
- **Conflict resolution**: Server-authoritative with retry

```javascript
// assets/static/js/sync-client.js
class SyncClient {
    constructor(options) {
        this.baseURL = options.baseURL;
        this.scope = options.scope;
        this.cursor = 0;
        this.queue = new MutationQueue();
        this.store = new LocalStore();
    }

    async mutate(name, args) {
        const mutation = { id: generateId(), name, args };
        this.queue.push(mutation);
        this.applyOptimistic(mutation);
        this.scheduleSync();
        return mutation.id;
    }

    async sync() {
        await this.push();
        await this.pull();
    }
}
```

### 4. Live WebSocket client

JavaScript WebSocket client for real-time updates:

- **Auto-reconnection**: Exponential backoff on disconnect
- **Topic subscription**: Subscribe to sync notifications
- **Event handling**: Trigger sync on data changes

```javascript
// assets/static/js/live-client.js
class LiveClient {
    constructor(options) {
        this.url = options.url;
        this.onMessage = options.onMessage;
    }

    connect() {
        this.ws = new WebSocket(this.url);
        this.ws.onmessage = (e) => {
            const msg = JSON.parse(e.data);
            this.onMessage(msg);
        };
    }

    subscribe(topic) {
        this.send({ type: 'subscribe', topic });
    }
}
```

### 5. Reactive UI

Server-rendered HTML with client-side enhancement:

- **Progressive enhancement**: Works without JavaScript
- **Optimistic rendering**: Immediate UI updates
- **Sync status**: Visual indicator for online/offline state
- **Form handling**: Standard form submission with JS enhancement

```html
<!-- assets/views/components/todo-item.html -->
<div class="todo-item" data-id="{{.ID}}" data-pending="{{.Pending}}">
    <input type="checkbox" {{if .Done}}checked{{end}}
           onchange="todoApp.toggleTodo('{{.ID}}')">
    <span class="{{if .Done}}done{{end}}">{{.Title}}</span>
    <button onclick="todoApp.deleteTodo('{{.ID}}')" aria-label="Delete">
        &times;
    </button>
</div>
```

---

## Data flow

### Creating a todo (online)

1. User submits form
2. JS intercepts, generates mutation ID
3. Mutation queued locally
4. UI updated optimistically
5. Mutation pushed to server
6. Server validates, applies to store
7. Change appended to log
8. Live notification sent to topic
9. Other clients receive notification
10. Other clients pull changes

### Creating a todo (offline)

1. User submits form
2. JS intercepts, generates mutation ID
3. Mutation queued locally (persisted)
4. UI updated optimistically
5. Sync fails (offline)
6. Mutation remains in queue
7. Later: connectivity restored
8. Queued mutations pushed
9. Server processes in order
10. UI reconciled with server state

---

## API endpoints

### Sync endpoints

```
POST /_sync/push      # Push mutations
POST /_sync/pull      # Pull changes since cursor
POST /_sync/snapshot  # Get full state snapshot
```

### REST endpoints (optional)

```
GET  /api/todos       # List todos (JSON)
POST /api/todos       # Create todo
PUT  /api/todos/:id   # Update todo
DELETE /api/todos/:id # Delete todo
```

### WebSocket endpoint

```
GET /ws               # WebSocket upgrade
```

---

## Configuration

```go
type Config struct {
    Addr string // Server address
    Dev  bool   // Development mode

    // Database (future)
    DatabaseURL string

    // Auth (future)
    SessionSecret string
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
mizu new myapp --template sync

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
3. **Modify mutations**: Add handlers to mutator
4. **Test offline**: Disable network in browser
5. **Verify sync**: Check mutations queue and replay

---

## Production considerations

### Persistence

The template uses in-memory stores by default. For production:

- Replace `memory.Store` with database-backed implementation
- Replace `memory.Log` with persistent change log
- Add proper session management

### Scaling

For multiple instances:

- Use shared database for sync engine
- Use Redis pub/sub for live notifications
- Consider cursor trimming for log growth

### Security

- Add authentication middleware
- Validate scope access per user
- Sanitize mutation arguments
- Rate limit sync endpoints

---

## Learning path

The template demonstrates:

1. **Sync protocol**: Push/pull/snapshot flow
2. **Optimistic updates**: Client-side state management
3. **Conflict resolution**: Server-authoritative model
4. **Real-time updates**: WebSocket integration
5. **Offline support**: Mutation queueing
6. **Progressive enhancement**: Works without JS

---

## Dependencies

Server:
- `github.com/go-mizu/mizu` - Web framework
- `github.com/go-mizu/mizu/sync` - Sync engine
- `github.com/go-mizu/mizu/live` - WebSocket server
- `github.com/go-mizu/mizu/view` - Template engine

Client (bundled):
- No external dependencies
- Vanilla JavaScript

---

## Summary

The `sync` template provides:

- Complete offline-first architecture
- Server and client sync implementations
- Real-time WebSocket updates
- Production-ready patterns
- Clear learning progression

It serves as both a reference implementation and a starting point for building collaborative, offline-capable applications with Mizu.
