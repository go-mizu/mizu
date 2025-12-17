# Spec 0034: Refactor View Package

## Summary

Consolidate the view package into fewer files and simplify the public API following Go core library conventions.

## Current Structure

```
view/
├── doc.go           # Package documentation (keep separate)
├── engine.go        # Engine type and methods
├── options.go       # Options, Data, PageData, RenderOption
├── errors.go        # Error types and sentinel errors
├── render.go        # renderContext for slots/stacks
├── context.go       # getEngine helper
├── funcs.go         # Template functions
├── middleware.go    # Mizu middleware, Render, RenderComponent
├── view_test.go     # Engine tests
├── funcs_test.go    # Template function tests
├── testdata/        # Test fixtures (excluded by .gitignore - BUG)
└── sync/            # Separate subpackage (not part of this refactor)
```

## Target Structure

```
view/
├── doc.go           # Package documentation (unchanged)
├── view.go          # All implementation code
├── view_test.go     # All tests
├── testdata/        # Test fixtures (properly tracked)
└── sync/            # Separate subpackage (unchanged)
```

## API Simplification

### Before (Public API)

```go
// Types
type Engine struct { ... }
type Options struct { ... }
type Data map[string]any
type PageData struct { ... }
type PageMeta struct { ... }
type RenderOption func(*renderConfig)
type TemplateError struct { ... }
type NotFoundError struct { ... }

// Errors
var ErrTemplateNotFound
var ErrLayoutNotFound
var ErrComponentNotFound
var ErrPartialNotFound
var ErrSlotNotDefined

// Functions
func New(Options) *Engine
func Middleware(*Engine) mizu.Middleware
func Render(*mizu.Ctx, string, any, ...RenderOption) error
func RenderComponent(*mizu.Ctx, string, any) error
func GetEngine(*mizu.Ctx) *Engine
func Status(int) RenderOption
func Layout(string) RenderOption
func NoLayout() RenderOption

// Engine methods
func (*Engine) Render(io.Writer, string, any, ...RenderOption) error
func (*Engine) RenderComponent(io.Writer, string, any) error
func (*Engine) RenderPartial(io.Writer, string, any) error
func (*Engine) Preload() error
func (*Engine) ClearCache()
```

### After (Simplified API)

```go
// Types (minimized)
type Engine struct { ... }         // Keep - core type
type Config struct { ... }         // Rename Options -> Config (shorter)
type Data = map[string]any         // Keep - convenience alias
type Error struct { ... }          // Combine error types into one

// Errors (reduced)
var ErrNotFound error              // Single error for all not-found cases

// Functions (simplified)
func New(Config) *Engine           // Keep
func (e *Engine) Handler() mizu.Middleware  // Rename Middleware -> Handler (verb)
func (e *Engine) Render(w io.Writer, page string, data any) error
func (e *Engine) Component(w io.Writer, name string, data any) error
func (e *Engine) Partial(w io.Writer, name string, data any) error
func (e *Engine) Load() error      // Rename Preload -> Load (shorter)
func (e *Engine) Clear()           // Rename ClearCache -> Clear (shorter)

// Context helpers
func Render(c *mizu.Ctx, page string, data any) error
func Component(c *mizu.Ctx, name string, data any) error
func From(c *mizu.Ctx) *Engine     // Rename GetEngine -> From (shorter)

// Render options via method chaining on Engine or direct parameters
// Remove RenderOption pattern - use explicit parameters instead
```

## Changes

### 1. File Consolidation

- Merge `engine.go`, `options.go`, `errors.go`, `render.go`, `context.go`, `funcs.go`, `middleware.go` into `view.go`
- Merge `view_test.go`, `funcs_test.go` into `view_test.go`
- Keep `doc.go` separate (Go convention for package docs)
- Keep `sync/` subpackage unchanged

### 2. Naming Simplifications

| Before | After | Rationale |
|--------|-------|-----------|
| `Options` | `Config` | Shorter, common Go pattern |
| `Middleware()` | `Handler()` | More verb-like, describes action |
| `RenderComponent()` | `Component()` | Shorter, Render prefix redundant |
| `RenderPartial()` | `Partial()` | Shorter |
| `Preload()` | `Load()` | Shorter |
| `ClearCache()` | `Clear()` | Shorter |
| `GetEngine()` | `From()` | Shorter, "get from context" pattern |
| `PageData` | Remove | Internal only |
| `PageMeta` | Remove | Internal only |
| `renderConfig` | Keep private | Internal only |
| `TemplateError` | `Error` | Single error type |
| `NotFoundError` | Remove | Use Error with Kind field |

### 3. Error Handling

Consolidate into single error type:

```go
type Error struct {
    Kind string // "page", "layout", "component", "partial"
    Name string
    Line int    // Optional
    Err  error  // Wrapped error
}

var ErrNotFound = errors.New("view: not found")

func (e *Error) Is(target error) bool {
    return target == ErrNotFound && e.Err == nil
}
```

### 4. Remove RenderOption Pattern

Instead of functional options for render, use direct parameters or method on context:

```go
// Before
Render(c, "page", data, Status(404), Layout("bare"), NoLayout())

// After - explicit methods
c.Status(404)
e.Render(w, "page", data)

// Or keep simple with layout parameter
e.RenderWithLayout(w, "page", "bare", data)
```

Decision: Keep RenderOption for backward compat, but make it private. Expose simpler public API.

### 5. Fix .gitignore

Remove `testdata/` from .gitignore - it was incorrectly excluding test fixtures needed for CI/CD.

## Implementation Plan

1. Fix .gitignore first (unblock CI/CD)
2. Create view/view.go with consolidated code
3. Apply naming simplifications
4. Consolidate error types
5. Create view/view_test.go with all tests
6. Delete old files
7. Run tests

## Breaking Changes

- `Options` renamed to `Config`
- `Middleware(*Engine)` becomes `(*Engine).Handler()`
- `GetEngine(*mizu.Ctx)` becomes `From(*mizu.Ctx)`
- Error types consolidated
- `PageData`, `PageMeta` removed from public API

## Migration

```go
// Before
e := view.New(view.Options{Dir: "views"})
app.Use(view.Middleware(e))
engine := view.GetEngine(c)

// After
e := view.New(view.Config{Dir: "views"})
app.Use(e.Handler())
engine := view.From(c)
```
