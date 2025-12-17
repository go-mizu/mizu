# Sync Template Refactor Plan

## Overview

This document describes the refactor of `cli/templates/sync` to align with the simplified sync package API from spec 0029.

---

## Current Template Structure

```
cli/templates/sync/
  template.json
  Makefile.tmpl
  cmd/server/main.go.tmpl
  app/server/
    config.go.tmpl
    app.go.tmpl
    routes.go.tmpl
  handler/
    home.go.tmpl
  service/todo/
    mutator.go.tmpl
  assets/
    embed.go.tmpl
    views/...
    static/...
```

---

## Changes Required

### 1. app/server/app.go.tmpl

**Remove:** Usage of `sync.Engine.Store()` and `sync.Engine.Log()` accessors.

**Current:**
```go
a.syncEngine = sync.New(sync.Options{
    Store:   store,
    Log:     log,
    Applied: applied,
    Mutator: mutator,
})

// Later...
a.syncEngine = sync.New(sync.Options{
    Store:   a.syncEngine.Store(),  // REMOVED
    Log:     a.syncEngine.Log(),    // REMOVED
    Mutator: todo.NewMutator(),
    Notify:  live.SyncNotifier(a.liveServer, "sync:"),
})
```

**After:**
```go
// Store the components for reuse
store := memory.NewStore()
log := memory.NewLog()
applied := memory.NewApplied()
mutator := todo.NewMutator()

a.syncEngine = sync.New(sync.Options{
    Store:   store,
    Log:     log,
    Applied: applied,
    Mutator: mutator,
})

// ... after live setup ...
// Recreate with notifier (store/log/applied already connected)
a.syncEngine = sync.New(sync.Options{
    Store:   store,
    Log:     log,
    Applied: applied,
    Mutator: mutator,
    Notify:  live.SyncNotifier(a.liveServer, "sync:"),
})
```

Alternatively, restructure to set up live first, then create sync engine once.

### 2. service/todo/mutator.go.tmpl

**No changes required.** The Mutator interface remains the same:
- `Apply(ctx context.Context, store sync.Store, m sync.Mutation) ([]sync.Change, error)`

The template already uses:
- `sync.Store` interface
- `sync.Mutation` struct
- `sync.Change` struct
- `sync.Op` constants (`sync.Create`, `sync.Update`, `sync.Delete`)
- Error sentinels (`sync.ErrNotFound`, `sync.ErrInvalidMutation`, `sync.ErrConflict`, `sync.ErrUnknownMutation`)

All of these are kept in the refactored API.

### 3. handler/home.go.tmpl

**Check for:** Any usage of removed APIs.

Currently the home handler likely uses `sync.Engine.Snapshot()` which is kept.

### 4. routes.go.tmpl

**No changes required.** Uses `syncEngine.Mount(app)` which is kept.

### 5. JavaScript Client Updates

**File:** `assets/static/js/sync-client.js`

The JavaScript sync client must match the HTTP API. Since we're making HTTP request/response types internal but keeping the same wire format, no JS changes needed.

**Wire Protocol (unchanged):**
```
POST /_sync/push
  Request:  { mutations: [...] }
  Response: { results: [...] }

POST /_sync/pull
  Request:  { scope, cursor, limit }
  Response: { changes: [...], has_more }

POST /_sync/snapshot
  Request:  { scope }
  Response: { data: {...}, cursor }
```

### 6. live-client.js

**No changes required.** Live integration is separate from sync.

---

## Updated app.go.tmpl

```go
package server

import (
	"context"
	"io/fs"
	"net/http"
	"os"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/live"
	"github.com/go-mizu/mizu/sync"
	"github.com/go-mizu/mizu/sync/memory"
	"github.com/go-mizu/mizu/view"

	"{{.Module}}/assets"
	"{{.Module}}/service/todo"
)

// App holds the application components.
type App struct {
	cfg        Config
	app        *mizu.App
	engine     *view.Engine
	syncEngine *sync.Engine
	liveServer *live.Server
}

// New creates and configures a new application instance.
func New(cfg Config) *App {
	a := &App{
		cfg: cfg,
		app: mizu.New(),
	}

	// Setup view engine
	a.setupViews()

	// Setup live server first (needed for sync notifier)
	a.setupLive()

	// Setup sync engine with live notifier
	a.setupSync()

	// Setup routes
	a.routes()

	return a
}

// Listen starts the HTTP server.
func (a *App) Listen(addr string) error {
	return a.app.Listen(addr)
}

func (a *App) setupViews() {
	opts := view.Options{
		DefaultLayout: "default",
		Development:   a.cfg.Dev,
	}

	if !a.cfg.Dev {
		viewsFS, _ := fs.Sub(assets.ViewsFS, "views")
		opts.FS = viewsFS
	} else {
		opts.Dir = "assets/views"
	}

	a.engine = view.New(opts)

	if !a.cfg.Dev {
		if err := a.engine.Preload(); err != nil {
			panic("failed to load templates: " + err.Error())
		}
	}

	a.app.Use(view.Middleware(a.engine))
}

func (a *App) setupLive() {
	a.liveServer = live.New(live.Options{
		OnMessage: func(ctx context.Context, s *live.Session, msg live.Message) {
			switch msg.Type {
			case "subscribe":
				a.liveServer.PubSub().Subscribe(s, msg.Topic)
				_ = s.Send(live.Message{Type: "ack", Ref: msg.Ref, Topic: msg.Topic})
			case "unsubscribe":
				a.liveServer.PubSub().Unsubscribe(s, msg.Topic)
			}
		},
	})
}

func (a *App) setupSync() {
	// Create storage backends
	store := memory.NewStore()
	log := memory.NewLog()
	applied := memory.NewApplied()

	// Create mutator
	mutator := todo.NewMutator()

	// Create sync engine with live notifier
	a.syncEngine = sync.New(sync.Options{
		Store:   store,
		Log:     log,
		Applied: applied,
		Mutator: mutator,
		Notify:  live.SyncNotifier(a.liveServer, "sync:"),
	})
}

// staticHandler serves embedded static files
func staticHandler(dev bool) http.Handler {
	var staticFS fs.FS
	if dev {
		staticFS = os.DirFS("assets/static")
	} else {
		staticFS, _ = fs.Sub(assets.StaticFS, "static")
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
}
```

---

## Validation Checklist

After refactoring both packages and template:

- [ ] `mizu new ./testapp --template sync` generates valid code
- [ ] Generated app compiles without errors
- [ ] Generated app runs and serves pages
- [ ] Sync push/pull/snapshot endpoints work
- [ ] Live WebSocket connection works
- [ ] Todo CRUD operations work end-to-end
- [ ] Offline mutations queue and replay correctly

---

## Test Commands

```bash
# Generate test app
cd /tmp
rm -rf testapp
mizu new ./testapp --template sync --module example.com/testapp

# Build and run
cd testapp
go mod tidy
go build ./...
go run ./cmd/server

# Test endpoints
curl -X POST http://localhost:8080/_sync/snapshot -d '{"scope":""}'
curl -X POST http://localhost:8080/_sync/pull -d '{"scope":"","cursor":0}'
```

---

## Dependencies

This template refactor depends on:

1. **spec/0029_refactor_sync.md** - Core sync package changes
2. Sync package file consolidation complete
3. Removed APIs actually removed from sync package

---

## Backward Compatibility

This is a **breaking change**. Existing projects using the sync template will need to:

1. Update app.go to not use `Engine.Store()` / `Engine.Log()` accessors
2. Restructure initialization order (live before sync)

Or regenerate from template and migrate custom code.

---

## Summary

The template changes are minimal:

| File | Change Required |
|------|-----------------|
| template.json | None |
| Makefile.tmpl | None |
| cmd/server/main.go.tmpl | None |
| app/server/config.go.tmpl | None |
| app/server/app.go.tmpl | **Restructure init order** |
| app/server/routes.go.tmpl | None |
| handler/home.go.tmpl | None |
| service/todo/mutator.go.tmpl | None |
| assets/... | None |

Only `app.go.tmpl` needs modification to avoid removed `Engine.Store()` and `Engine.Log()` accessors.
