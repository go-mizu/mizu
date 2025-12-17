# Spec 0035: Refactor Live/Sync Templates for New View API

## Summary

Update CLI templates to use the refactored view package API from spec/0034, and clarify the relationship between `view/sync` and `sync` packages.

## Package Analysis

### Package Relationships

The codebase has three related but distinct packages:

1. **sync/** - Server-side authoritative sync engine
   - Handles mutations from clients
   - Maintains Store, Log, Applied state
   - Provides HTTP endpoints (`/_sync/push`, `/_sync/pull`, `/_sync/snapshot`)
   - Transport-agnostic correctness layer

2. **view/sync/** - Client-side reactive sync runtime
   - Designed for offline-first interactive applications
   - Provides reactive primitives: `Signal`, `Computed`, `Effect`
   - Local store with mutation queue
   - Pulls/pushes to server via HTTP
   - Integrates with live package for push notifications

3. **live/** - WebSocket realtime transport
   - Topic-based pub/sub
   - Best-effort message delivery
   - Independent of sync/view

### Decision: Keep view/sync Separate

The `view/sync` package should remain separate from `sync` because:

1. **Different contexts**: `sync` is server-side, `view/sync` is client-side
2. **Different responsibilities**:
   - `sync` = authoritative state, mutation processing
   - `view/sync` = local state, reactivity, offline support
3. **Different compilation targets**: `view/sync` may compile to WASM for browser use

The naming `view/sync` is intentional - it provides reactive sync capabilities for view-driven applications.

## Template Updates Required

The templates were written before spec/0034 refactored the view package API.

### API Changes (from spec/0034)

| Before | After |
|--------|-------|
| `view.Options{...}` | `view.Config{...}` |
| `view.Middleware(engine)` | `engine.Handler()` |
| `engine.Preload()` | `engine.Load()` |
| `view.GetEngine(c)` | `view.From(c)` |

### Files to Update

1. `cli/templates/live/app/server/app.go.tmpl`
2. `cli/templates/sync/app/server/app.go.tmpl`
3. `cli/templates/web/app/web/app.go.tmpl`

## Implementation

### 1. Update live template

```go
// Before
opts := view.Options{
    DefaultLayout: "default",
    Development:   a.cfg.Dev,
}
a.engine = view.New(opts)
if !a.cfg.Dev {
    if err := a.engine.Preload(); err != nil {
        panic("failed to load templates: " + err.Error())
    }
}
a.app.Use(view.Middleware(a.engine))

// After
cfg := view.Config{
    DefaultLayout: "default",
    Development:   a.cfg.Dev,
}
a.engine = view.New(cfg)
if !a.cfg.Dev {
    if err := a.engine.Load(); err != nil {
        panic("failed to load templates: " + err.Error())
    }
}
a.app.Use(a.engine.Handler())
```

### 2. Update sync template

Same changes as live template.

### 3. Update web template

Same changes as live template.

## Testing

After changes:
1. Run `make test` to verify packages compile
2. Generate a new project with each template and verify it builds

## Migration

Users with existing projects need to update their code:

```go
// Before
engine := view.New(view.Options{Dir: "views"})
app.Use(view.Middleware(engine))
e := view.GetEngine(c)

// After
engine := view.New(view.Config{Dir: "views"})
app.Use(engine.Handler())
e := view.From(c)
```
