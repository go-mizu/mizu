# Live Template Refactor Plan

## Overview

This document describes the refactor of `cli/templates/live` to align with the simplified live package API from spec 0031.

---

## Current Template Structure

```
cli/templates/live/
  template.json
  Makefile.tmpl
  cmd/server/main.go.tmpl
  app/server/
    config.go.tmpl
    app.go.tmpl
    routes.go.tmpl
  handler/
    home.go.tmpl
    counter.go.tmpl
  assets/
    embed.go.tmpl
    views/
      layouts/default.html
      pages/home.html
      pages/counter.html
      partials/header.html
      partials/footer.html
      components/connection-status.html
    static/
      js/live-client.js
      js/live-view.js
```

---

## Changes Required

### 1. app/server/app.go.tmpl

**Change:** Replace `srv.PubSub().Subscribe()` with `srv.Subscribe()`

**Current:**
```go
func (a *App) handleLiveMessage(ctx context.Context, s *live.Session, msg live.Message) {
    switch msg.Type {
    case "subscribe":
        a.liveServer.PubSub().Subscribe(s, msg.Topic)
        _ = s.Send(live.Message{Type: "ack", Ref: msg.Ref, Topic: msg.Topic})
    case "unsubscribe":
        a.liveServer.PubSub().Unsubscribe(s, msg.Topic)
    // ...
    }
}
```

**After:**
```go
func (a *App) handleLiveMessage(ctx context.Context, s *live.Session, msg live.Message) {
    switch msg.Type {
    case "subscribe":
        a.liveServer.Subscribe(s, msg.Topic)
        _ = s.Send(live.Message{Type: "ack", Ref: msg.Ref, Topic: msg.Topic})
    case "unsubscribe":
        a.liveServer.Unsubscribe(s, msg.Topic)
    // ...
    }
}
```

### 2. handler/counter.go.tmpl

**No changes required.** Uses only:
- `*live.Session` (kept)
- `live.Message` (kept)
- `s.ID()` (kept)
- `s.Send()` (kept)

All used APIs are kept in the refactored package.

### 3. handler/home.go.tmpl

**No changes required.** Pure view rendering, no live package usage.

### 4. routes.go.tmpl

**No changes required.** Uses:
- `a.liveServer.Handler()` (kept)

### 5. config.go.tmpl

**No changes required.** No live package usage.

### 6. cmd/server/main.go.tmpl

**No changes required.** No direct live package usage.

### 7. assets/static/js/live-client.js

**No changes required.** Client-side code; wire protocol unchanged.

### 8. assets/static/js/live-view.js

**No changes required.** Client-side code; wire protocol unchanged.

### 9. assets/views/*

**No changes required.** HTML templates; no Go code.

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
	"github.com/go-mizu/mizu/view"

	"{{.Module}}/assets"
	"{{.Module}}/handler"
)

// App holds the application components.
type App struct {
	cfg        Config
	app        *mizu.App
	engine     *view.Engine
	liveServer *live.Server
	counter    *handler.CounterView
}

// New creates and configures a new application instance.
func New(cfg Config) *App {
	a := &App{
		cfg: cfg,
		app: mizu.New(),
	}

	// Setup view engine
	a.setupViews()

	// Setup live server
	a.setupLive()

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
		// Production: use embedded filesystem
		viewsFS, _ := fs.Sub(assets.ViewsFS, "views")
		opts.FS = viewsFS
	} else {
		// Development: use disk filesystem
		opts.Dir = "assets/views"
	}

	a.engine = view.New(opts)

	// Preload templates in production
	if !a.cfg.Dev {
		if err := a.engine.Preload(); err != nil {
			panic("failed to load templates: " + err.Error())
		}
	}

	// Add view middleware
	a.app.Use(view.Middleware(a.engine))
}

func (a *App) setupLive() {
	// Create counter view handler
	a.counter = handler.NewCounterView()

	// Create live server
	a.liveServer = live.New(live.Options{
		OnMessage: a.handleLiveMessage,
		OnClose: func(s *live.Session, err error) {
			// Cleanup session from all views
			a.counter.RemoveSession(s.ID())
		},
	})
}

func (a *App) handleLiveMessage(ctx context.Context, s *live.Session, msg live.Message) {
	switch msg.Type {
	case "subscribe":
		a.liveServer.Subscribe(s, msg.Topic)  // Changed from PubSub().Subscribe
		_ = s.Send(live.Message{Type: "ack", Ref: msg.Ref, Topic: msg.Topic})
	case "unsubscribe":
		a.liveServer.Unsubscribe(s, msg.Topic)  // Changed from PubSub().Unsubscribe
	case "mount":
		a.handleMount(ctx, s, msg)
	case "event":
		a.handleEvent(ctx, s, msg)
	}
}

func (a *App) handleMount(ctx context.Context, s *live.Session, msg live.Message) {
	// Route to appropriate view handler based on topic
	switch msg.Topic {
	case "view:counter":
		a.counter.Mount(s, msg)
	}
}

func (a *App) handleEvent(ctx context.Context, s *live.Session, msg live.Message) {
	// Route to appropriate view handler based on topic
	switch msg.Topic {
	case "view:counter":
		a.counter.HandleEvent(s, msg)
	}
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

After refactoring the live package and template:

- [ ] `mizu new ./testapp --template live` generates valid code
- [ ] Generated app compiles without errors
- [ ] Generated app runs and serves pages
- [ ] WebSocket connection establishes successfully
- [ ] Counter view mounts correctly
- [ ] Counter increment/decrement/reset work
- [ ] Session cleanup works on disconnect
- [ ] Reconnection works after temporary disconnect

---

## Test Commands

```bash
# Generate test app
cd /tmp
rm -rf testapp
mizu new ./testapp --template live --module example.com/testapp

# Build and run
cd testapp
go mod tidy
go build ./...
go run ./cmd/server

# Open in browser
open http://localhost:8080/counter
```

---

## Dependencies

This template refactor depends on:

1. **spec/0031_refactor.live.md** - Core live package changes
2. Live package file consolidation complete
3. PubSub flattened into Server

---

## Backward Compatibility

This is a **breaking change**. Existing projects using the live template will need to:

1. Replace `liveServer.PubSub().Subscribe()` with `liveServer.Subscribe()`
2. Replace `liveServer.PubSub().Unsubscribe()` with `liveServer.Unsubscribe()`

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
| app/server/app.go.tmpl | **Change PubSub() calls** |
| app/server/routes.go.tmpl | None |
| handler/home.go.tmpl | None |
| handler/counter.go.tmpl | None |
| assets/embed.go.tmpl | None |
| assets/views/* | None |
| assets/static/js/* | None |

Only `app.go.tmpl` needs modification to use the flattened Subscribe/Unsubscribe methods.

---

## Coordination with 0031

The template changes are minimal because the live package refactor focuses on:

1. **Internal consolidation** - Multiple files into one
2. **Removing rarely-used accessors** - `Options()`, `Session()`, `Sessions()`
3. **Flattening PubSub** - Only affects Subscribe/Unsubscribe calls

The template already uses a clean subset of the API:
- `live.New()` - kept
- `live.Options{}` - kept
- `live.Server.Handler()` - kept
- `live.Server.PubSub().Subscribe()` - changed to `Server.Subscribe()`
- `live.Server.PubSub().Unsubscribe()` - changed to `Server.Unsubscribe()`
- `live.Session.ID()` - kept
- `live.Session.Send()` - kept
- `live.Message{}` - kept

No template code uses the removed APIs:
- `Codec` / `JSONCodec`
- `srv.Options()`
- `srv.Session(id)`
- `srv.Sessions()`
- `srv.PubSub().Who()`
- `srv.PubSub().Topics()`
- `srv.PubSub().Count()`
- `session.Done()`
- `session.CloseError()`
- `session.Topics()`
- `ErrUpgradeFailed`
- `ErrInvalidMessage`
- `SyncNotifier()`
