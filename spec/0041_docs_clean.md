# Documentation Cleanup Plan

## Summary

Review and update all documentation in `docs/view/` to match the refactored view, sync, and live packages. Documentation should be beginner-friendly, detailed, and accurate to the current code.

## Documentation Standards

1. **Title Format**: Short, non-repetitive titles (e.g., "Overview" not "View Overview")
2. **No Level 1 Headers**: Since front matter has title, remove `# Title` headers
3. **Beginner-Friendly**: Detailed explanations for absolute beginners
4. **Code Accuracy**: All code examples must match current API

## Current Code Analysis

### View Package (`view/view.go`)

The view package provides server-side HTML templating using Go's `html/template`.

**Config struct:**
- `Dir` - Template directory path (default: "views")
- `FS` - Embedded filesystem (optional, for production)
- `Extension` - File extension (default: ".html")
- `DefaultLayout` - Default layout name (default: "default")
- `Funcs` - Custom template functions
- `Delims` - Custom delimiters [left, right] (default: "{{", "}}")
- `Development` - Enable hot reload (default: false)

**Engine methods:**
- `New(Config) *Engine` - Create engine
- `Load() error` - Pre-load all templates (call at startup in production)
- `Clear()` - Clear template cache
- `Render(w io.Writer, page string, data any, opts ...option) error` - Render template
- `Middleware() mizu.Middleware` - Returns middleware for Mizu app

**Package-level functions:**
- `From(c *mizu.Ctx) *Engine` - Get engine from context
- `Render(c *mizu.Ctx, page string, data any, opts ...option) error` - Render using context engine

**Render options:**
- `Layout(name string) option` - Override layout
- `NoLayout() option` - Render without layout

**Built-in functions:**
- `dict` - Create map from key-value pairs
- `list` - Create slice from values
- `upper`, `lower` - Case conversion
- `trim` - Trim whitespace
- `contains`, `hasPrefix`, `hasSuffix` - String checks
- `replace` - String replacement
- `split`, `join` - String/slice conversion

**Directory structure:**
```
views/
├── layouts/
│   └── default.html    # Layout templates
├── pages/
│   ├── home.html       # Page templates
│   └── about.html
```

**Template data structure:**
```go
type pageData struct {
    Page    pageMeta      // Page info
    Data    any           // User data
    Content template.HTML // Rendered content (for layouts)
}
```

### Sync Package (`view/sync/sync.go`)

The sync package is a **client-side** runtime for offline-first applications with reactive state management.

**Errors:**
- `ErrNotStarted` - Client operations called before Start
- `ErrAlreadyStarted` - Start called on running client

**Core Types:**

*Operations:*
- `Op` - Operation type: `OpCreate`, `OpUpdate`, `OpDelete`

*Reactive Primitives:*
- `Signal[T]` - Reactive value container
- `Computed[T]` - Derived reactive value
- `Effect` - Side effect that runs on dependency changes

*Sync Types:*
- `Client` - Main sync runtime
- `Collection[T]` - Manages synchronized entities
- `Entity[T]` - Single synchronized record
- `Persistence` - Interface for state persistence

**Signal API:**
```go
count := sync.NewSignal(0)
count.Get()           // Get value (registers dependency)
count.Set(1)          // Set value (notifies subscribers)
count.Update(fn)      // Update with function
```

**Computed API:**
```go
doubled := sync.NewComputed(func() int {
    return count.Get() * 2
})
doubled.Get()  // Lazy evaluation, auto-tracks dependencies
```

**Effect API:**
```go
effect := sync.NewEffect(func() {
    fmt.Println("Count:", count.Get())
})
effect.Stop()  // Stop future runs
```

**Client Options:**
- `BaseURL` - Sync server endpoint
- `Scope` - Data partition identifier
- `HTTP` - HTTP client (optional)
- `Persistence` - State persistence (optional)
- `OnError`, `OnSync`, `OnOnline`, `OnOffline` - Callbacks
- `PushInterval`, `PullInterval` - Timing configuration

**Client API:**
```go
client := sync.New(sync.Options{...})
client.Start(ctx)     // Start background sync
client.Stop()         // Stop sync
client.Sync()         // Force immediate sync
client.Mutate(name, args)  // Queue mutation
client.Cursor()       // Get current cursor
client.IsOnline()     // Check online status
client.NotifyLive(cursor)  // Handle live notification
```

**Collection API:**
```go
todos := sync.NewCollection[Todo](client, "todo")
todo := todos.Create("id", Todo{...})  // Create entity
todo := todos.Get("id")                // Get entity
todos.All()                            // All entities (reactive)
todos.Count()                          // Count (reactive)
todos.Find(predicate)                  // Find matching
todos.Has("id")                        // Check existence
```

**Entity API:**
```go
entity.ID()      // Get ID
entity.Get()     // Get value (reactive)
entity.Set(val)  // Update value
entity.Delete()  // Delete entity
entity.Exists()  // Check existence (reactive)
```

### Live Package (`live/live.go`)

The live package provides WebSocket-based real-time messaging with pub/sub.

**Errors:**
- `ErrSessionClosed` - Session closed
- `ErrQueueFull` - Send queue full

**Types:**
- `Message` - Transport envelope with Topic and Data

**Options:**
- `QueueSize` - Per-session send queue size (default: 256)
- `ReadLimit` - Max message size (default: 4MB)
- `OnAuth` - Authentication callback
- `OnMessage` - Message handler callback
- `OnClose` - Close handler callback
- `Origins` - Allowed origins list
- `CheckOrigin` - Custom origin validation
- `IDGenerator` - Custom session ID generator

**Server API:**
```go
server := live.New(live.Options{...})
server.Handler()                 // HTTP handler for WebSocket
server.Publish(topic, data)      // Publish to topic
server.Subscribe(session, topic) // Subscribe session
server.Unsubscribe(session, topic)
```

**Session API:**
```go
session.ID()         // Unique identifier
session.Value()      // Auth value
session.Send(msg)    // Send message (non-blocking)
session.Close()      // Close session
session.IsClosed()   // Check if closed
session.CloseError() // Get close error
```

## Files to Update

### View Documentation

| File | Action | Notes |
|------|--------|-------|
| `overview.mdx` | Update | Remove # header, update API references |
| `quick-start.mdx` | Update | Fix directory structure (pages/, layouts/) |
| `engine.mdx` | Update | Fix Middleware() API, add pageData structure |
| `templates.mdx` | Update | Add pageData access patterns |
| `layouts.mdx` | Update | Fix directory structure |
| `functions.mdx` | Update | Verify function list matches code |
| `production.mdx` | Update | Minor updates |
| `partials.mdx` | **DELETE** | Not in current API |
| `components.mdx` | **DELETE** | Not in current API |

### Sync Documentation

| File | Action | Notes |
|------|--------|-------|
| `sync-overview.mdx` | **REWRITE** | Now client-side runtime with reactive |
| `sync-quick-start.mdx` | **REWRITE** | Client usage example |
| `sync-server.mdx` | **DELETE** | No server-side sync in this package |
| `sync-client.mdx` | Update | Matches current Client API |
| `sync-reactive.mdx` | **REWRITE** | Signal, Computed, Effect |
| `sync-collections.mdx` | **REWRITE** | Collection and Entity API |
| `sync-integration.mdx` | Update | Live integration via NotifyLive |

### Live Documentation

| File | Action | Notes |
|------|--------|-------|
| `live-overview.mdx` | Update | Remove # header, verify examples |
| `live-quick-start.mdx` | Update | Verify example code |
| `live-server.mdx` | Update | Verify Options match code |
| `live-sessions.mdx` | Update | Session API |
| `live-pubsub.mdx` | Update | Publish/Subscribe patterns |

## Implementation Order

1. Delete obsolete files (partials.mdx, components.mdx)
2. View documentation updates
3. Sync documentation (complete rewrite for client-side focus)
4. Live documentation updates
5. Final review

## Key Changes Summary

### View Package
- Directory structure uses `pages/` and `layouts/` subdirectories
- `Middleware()` is a method that returns middleware, not a function that takes app
- Templates receive `pageData` struct with `.Page`, `.Data`, `.Content`
- `From(c)` and `Render(c, ...)` are package-level functions

### Sync Package
- **Client-side** runtime, not server-side engine
- Includes reactive primitives: Signal, Computed, Effect
- Collections manage synchronized entities
- Integrates with live package via `NotifyLive()`
- Import path is `github.com/go-mizu/mizu/view/sync`

### Live Package
- Simple WebSocket server with pub/sub
- Message type has Topic and Data fields
- No changes from documented API

## Documentation Style

### Do
- Explain concepts before showing code
- Use clear, simple language
- Provide complete, runnable examples
- Link between related docs
- Show common patterns

### Don't
- Use level 1 headers (title is in front matter)
- Repeat title in content (e.g., no "Sync Overview" header)
- Assume prior knowledge
- Skip error handling in examples
- Use placeholder code
