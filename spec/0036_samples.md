# Spec 0036: Sample Apps from Templates

## Summary

Create sample apps in `cli/samples/` for all available templates to verify they work correctly and serve as reference implementations.

## Available Templates

| Template | Description |
|----------|-------------|
| `minimal` | Smallest runnable Mizu project |
| `api` | JSON API service with a recommended layout |
| `contract` | Transport-neutral service contract with REST, JSON-RPC, and OpenAPI |
| `web` | Full-stack web app with views, components, and static assets |
| `sync` | Offline-first app with sync, live updates, and reactive state |
| `live` | Real-time interactive app with live views and instant updates |

## Target Structure

```
cli/samples/
├── minimal/     # mizu new --template minimal
├── api/         # mizu new --template api
├── contract/    # mizu new --template contract
├── web/         # mizu new --template web
├── sync/        # mizu new --template sync
└── live/        # mizu new --template live
```

## Implementation Plan

### 1. Build CLI

Build the mizu CLI to ensure we're testing with the latest code:

```bash
make install
```

### 2. Create Samples Directory

```bash
mkdir -p cli/samples
```

### 3. Generate Sample Apps

For each template, run:

```bash
mizu new --template <template> --module example.com/<template> cli/samples/<template>
```

### 4. Verify Each Sample

For each sample app:

1. Add replace directive: `replace github.com/go-mizu/mizu => ../../..`
2. Run `go mod tidy` to resolve dependencies
3. Run `go build ./...` to verify compilation

## Verification Checklist

- [x] `minimal` - builds and runs
- [x] `api` - builds and runs
- [x] `contract` - builds and runs
- [x] `web` - builds and runs
- [x] `sync` - builds and runs
- [x] `live` - builds and runs

## Template Fixes Applied

During implementation, several template issues were discovered and fixed:

### 1. HTML View Templates (sync, live)

HTML files containing Go template syntax (like `{{slot ...}}`, `{{partial ...}}`) were being processed by the CLI template engine. Fixed by escaping with `{{ "{{" }}` pattern.

Files fixed:
- `cli/templates/sync/assets/views/layouts/default.html`
- `cli/templates/sync/assets/views/pages/home.html`
- `cli/templates/sync/assets/views/components/todo-item.html`
- `cli/templates/live/assets/views/layouts/default.html`
- `cli/templates/live/assets/views/pages/home.html`
- `cli/templates/live/assets/views/pages/counter.html`

### 2. Go Struct Literals with `{{` (sync)

Go code like `[]sync.Change{{...}}` was being misinterpreted as template syntax. Fixed by reformatting to avoid `{{` on single line.

Files fixed:
- `cli/templates/sync/service/todo/mutator.go.tmpl`

### 3. API Changes (api, live, sync, contract)

Templates referenced APIs that had changed:
- `c.BindJSON(&req)` → `c.BindJSON(&req, 1<<20)` (added max size)
- `mizu.Compat(handler)` → `a.app.Mount(path, handler)` (correct http.Handler mounting)
- `github.com/go-mizu/mizu/contract` → `github.com/go-mizu/mizu/contract/v1` (correct package path)
- `a.app.Shutdown()` removed (mizu handles graceful shutdown internally)

Files fixed:
- `cli/templates/api/feature/echo/http.go.tmpl`
- `cli/templates/api/app/api/app.go.tmpl`
- `cli/templates/api/cmd/api/main.go.tmpl`
- `cli/templates/live/app/server/routes.go.tmpl`
- `cli/templates/sync/app/server/routes.go.tmpl`
- `cli/templates/contract/app/server/server.go.tmpl`

## Notes

- Sample apps use `example.com/<name>` as the module path
- Samples require a `replace` directive to use local mizu (not published to GitHub yet)
- Samples are in `.gitignore` and not committed (they're generated for validation)
- This validates that templates are up-to-date with current API (spec/0034, spec/0035)
