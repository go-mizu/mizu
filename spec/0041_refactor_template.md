# 0041: Template Refactoring for view/live/sync API Changes

## Summary

This spec documents the required template changes following the refactoring of the `view`, `live`, and `sync` packages. The templates in `cli/templates/*` needed updates to match the new APIs.

## Background

Recent commits refactored three core packages:
- **view** (commits 85f70ac, 4737a0e): Simplified API, removed template functions, renamed `Handler()` to `Middleware()`
- **live** (commit 5fc77ab): Simplified Message structure, changed OnMessage signature
- **sync** (commit 02bda1c+): New Options structure with `Apply`, `Snapshot`, `Dedupe`; removed Store abstraction

## Breaking Changes

### View Package

| Before | After |
|--------|-------|
| `engine.Handler()` | `engine.Middleware()` |
| `{{default ...}}` template func | Removed |
| `{{empty ...}}` template func | Removed |
| `{{safeHTML ...}}` template func | Removed |
| `{{eq ...}}` / `{{ne ...}}` template funcs | Removed |
| `pageData.CSRF` | Removed |
| `pageMeta.Title` | Removed |
| `Status(code int) option` | Removed (hardcoded 200) |

**Retained template functions:** `dict`, `list`, `upper`, `lower`, `trim`, `contains`, `replace`, `split`, `join`, `hasPrefix`, `hasSuffix`

### Live Package

| Before | After |
|--------|-------|
| `Message.Type`, `Message.Ref`, `Message.Body` | `Message.Topic`, `Message.Data` |
| `OnMessage(ctx, s, msg Message)` | `OnMessage(ctx, s, topic string, data []byte)` |
| `OnAuth(...) (Meta, error)` | `OnAuth(...) (any, error)` |
| `Session.Meta()` | `Session.Value()` |
| `Publish(topic, Message)` | `Publish(topic string, data []byte)` |
| `Broadcast()`, `SessionCount()` | Removed |

### Sync Package

Complete API redesign:

| Before | After |
|--------|-------|
| `sync.Options.Store` | Removed |
| `sync.Options.Applied` | Removed |
| `sync.Options.Mutator` | `sync.Options.Apply` (ApplyFunc) |
| `sync.Options.Notify` | Removed |
| - | `sync.Options.Log` (required) |
| - | `sync.Options.Snapshot` (optional) |
| - | `sync.Options.Dedupe` (optional) |
| `sync.Store` interface | Removed (use custom store) |
| `sync.Mutator` interface | Removed (use ApplyFunc) |
| `sync.Notifier`, `sync.NotifierFunc` | Removed |
| `sync.Engine.Mount()` | Use `sync/http.Transport.Mount()` |
| `memory.NewStore()` | Removed |
| `memory.NewApplied()` | Removed |
| - | `memory.NewLog()` |
| - | `memory.NewDedupe()` |

## Template Changes

### web/app/web/app.go.tmpl

```diff
-	a.app.Use(a.engine.Handler())
+	a.app.Use(a.engine.Middleware())
```

### live/app/server/app.go.tmpl

```diff
-	a.app.Use(a.engine.Handler())
+	a.app.Use(a.engine.Middleware())
```

### sync/app/server/app.go.tmpl

Complete rewrite:

1. Import changes:
```go
import (
	synchttp "github.com/go-mizu/mizu/sync/http"  // NEW
	"github.com/go-mizu/mizu/sync/memory"
)
```

2. App struct changes:
```diff
 type App struct {
 	cfg           Config
 	app           *mizu.App
 	engine        *view.Engine
 	syncEngine    *sync.Engine
+	syncTransport *synchttp.Transport
 	liveServer    *live.Server
+	store         *todo.Store  // Application-level store
 }
```

3. View middleware fix:
```diff
-	a.app.Use(a.engine.Handler())
+	a.app.Use(a.engine.Middleware())
```

4. Sync setup changes:
```diff
 func (a *App) setupSync() {
-	store := memory.NewStore()
-	log := memory.NewLog()
-	applied := memory.NewApplied()
-	mutator := todo.NewMutator()
-
-	a.syncEngine = sync.New(sync.Options{
-		Store:   store,
-		Log:     log,
-		Applied: applied,
-		Mutator: mutator,
-		Notify:  a.syncNotifier("sync:"),
-	})
+	log := memory.NewLog()
+	dedupe := memory.NewDedupe()
+
+	a.syncEngine = sync.New(sync.Options{
+		Log:      log,
+		Apply:    a.store.Apply,
+		Snapshot: a.store.Snapshot,
+		Dedupe:   dedupe,
+	})
+
+	a.syncTransport = synchttp.New(synchttp.Options{
+		Engine: a.syncEngine,
+	})
 }
```

### sync/app/server/routes.go.tmpl

```diff
 func (a *App) routes() {
 	a.app.Mount("/static/", staticHandler(a.cfg.Dev))
-	a.app.Get("/", handler.Home(a.syncEngine))
-	a.syncEngine.Mount(a.app)
+	a.app.Get("/", handler.Home(a.store))
+	a.syncTransport.Mount(a.app)  // Mounts at /_sync/*
 	a.app.Mount("/ws", a.liveServer.Handler())
 }
```

### sync/service/todo/mutator.go.tmpl

Complete rewrite from `Mutator` to `Store`:

1. Store type with in-memory data:
```go
type Store struct {
	mu    gosync.RWMutex
	todos map[string]map[string]*Todo // scope -> id -> todo
}

func NewStore() *Store {
	return &Store{
		todos: make(map[string]map[string]*Todo),
	}
}
```

2. ApplyFunc implementation:
```go
func (s *Store) Apply(ctx context.Context, m sync.Mutation) ([]sync.Change, error) {
	// Parse args from m.Args (json.RawMessage)
	// Dispatch to createTodo/updateTodo/deleteTodo/toggleTodo
}
```

3. SnapshotFunc implementation:
```go
func (s *Store) Snapshot(ctx context.Context, scope string) (json.RawMessage, uint64, error) {
	// Return all todos as JSON
}
```

4. GetAll for handler:
```go
func (s *Store) GetAll(scope string) []*Todo {
	// Return todos for view rendering
}
```

5. Helper to avoid template escaping issues:
```go
func makeChanges(scope string, data json.RawMessage) []sync.Change {
	return []sync.Change{
		{Scope: scope, Data: data},
	}
}
```

### sync/handler/home.go.tmpl

```diff
-func Home(engine *sync.Engine) mizu.Handler {
+func Home(store *todo.Store) mizu.Handler {
 	return func(c *mizu.Ctx) error {
 		scope := sync.DefaultScope
-		data, cursor, err := engine.Snapshot(c.Context(), scope)
-		// Parse structured data from map[string]map[string]json.RawMessage
-		...
+		todos := store.GetAll(scope)
+		// Sort and render
 	}
 }
```

## Template Escaping

Go templates interpret `{{` as directive start. When generating Go code with struct literals like `[]sync.Change{{Scope: scope}}`, the template engine fails.

**Solution:** Use a helper function:
```go
func makeChanges(scope string, data json.RawMessage) []sync.Change {
	return []sync.Change{
		{Scope: scope, Data: data},
	}
}
```

## Testing

All templates were regenerated and tested:

```bash
# Generate samples
mizu new -t minimal ./minimal
mizu new -t api ./api
mizu new -t contract ./contract
mizu new -t web ./web
mizu new -t live ./live
mizu new -t sync ./sync

# Build each sample
cd minimal && go build .
cd api && go build ./cmd/api
cd contract && go build ./cmd/api
cd web && go build ./cmd/web
cd live && go build ./cmd/server
cd sync && go build ./cmd/server
```

All samples compile successfully.

## Files Changed

Templates modified:
- `cli/templates/web/app/web/app.go.tmpl`
- `cli/templates/live/app/server/app.go.tmpl`
- `cli/templates/sync/app/server/app.go.tmpl`
- `cli/templates/sync/app/server/routes.go.tmpl`
- `cli/templates/sync/handler/home.go.tmpl`
- `cli/templates/sync/service/todo/mutator.go.tmpl`

Samples regenerated:
- `cli/samples/minimal/*`
- `cli/samples/api/*`
- `cli/samples/contract/*`
- `cli/samples/web/*`
- `cli/samples/live/*`
- `cli/samples/sync/*`

## Migration Guide

For users with existing projects based on old templates:

### Web/Live Templates
1. Replace `engine.Handler()` with `engine.Middleware()`

### Sync Templates
1. Replace `memory.NewStore()` with custom store implementation
2. Replace `memory.NewApplied()` with `memory.NewDedupe()`
3. Change `sync.Options` to use `Log`, `Apply`, `Snapshot`, `Dedupe`
4. Use `sync/http.Transport` for HTTP routes
5. Update handler to use store directly instead of `Engine.Snapshot()`
