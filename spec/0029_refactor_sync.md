# Sync Package Refactor Plan

## Overview

This document describes an aggressive refactor of the `sync` and `view/sync` packages to reduce API surface area, align with Go core library design principles, and consolidate files into single-file packages.

---

## Goals

1. **Minimal API surface** - Remove non-essential public types and functions
2. **Single-file packages** - Consolidate `sync/*.go` → `sync/sync.go`
3. **Go idioms** - Follow Go standard library naming and design patterns
4. **Clear separation** - Server-side (`sync`) vs client-side (`view/sync`)

---

## File Consolidation

### sync package

**Before:**
```
sync/
  doc.go
  types.go
  errors.go
  store.go
  log.go
  mutator.go
  applied.go
  notify.go
  engine.go
  http.go
  engine_test.go
  http_test.go
  memory/
    doc.go
    store.go
    log.go
    applied.go
    store_test.go
    log_test.go
    applied_test.go
```

**After:**
```
sync/
  sync.go         # All code combined
  sync_test.go    # All tests combined
  memory/
    memory.go     # All memory implementations combined
    memory_test.go
```

### view/sync package

**Before:**
```
view/sync/
  doc.go
  errors.go
  store.go
  queue.go
  entity.go
  http.go
  persistence.go
  client.go
  live.go
  signal.go
  *_test.go files
```

**After:**
```
view/sync/
  sync.go         # All code combined
  sync_test.go    # All tests combined
```

---

## API Surface Reduction

### sync package (server-side)

#### Keep (Essential)

**Types:**
```go
type Mutation struct { ... }  // Client request
type Result struct { ... }    // Mutation outcome
type Change struct { ... }    // Log entry
type Op string               // create, update, delete
```

**Interfaces:**
```go
type Store interface {
    Get(ctx, scope, entity, id string) ([]byte, error)
    Set(ctx, scope, entity, id string, data []byte) error
    Delete(ctx, scope, entity, id string) error
    Snapshot(ctx, scope string) (map[string]map[string][]byte, error)
}

type Log interface {
    Append(ctx, scope string, changes []Change) (uint64, error)
    Since(ctx, scope string, cursor uint64, limit int) ([]Change, error)
    Cursor(ctx, scope string) (uint64, error)
    Trim(ctx, scope string, before uint64) error
}

type Mutator interface {
    Apply(ctx, store Store, m Mutation) ([]Change, error)
}

type Applied interface {
    Get(ctx, scope, key string) (Result, bool, error)
    Put(ctx, scope, key string, res Result) error
}

type Notifier interface {
    Notify(scope string, cursor uint64)
}
```

**Engine:**
```go
type Engine struct { ... }
type Options struct { ... }
func New(opts Options) *Engine
func (e *Engine) Push(ctx, mutations) ([]Result, error)
func (e *Engine) Pull(ctx, scope, cursor, limit) ([]Change, bool, error)
func (e *Engine) Snapshot(ctx, scope) (data, cursor, error)
func (e *Engine) Mount(app *mizu.App)
func (e *Engine) MountAt(app *mizu.App, prefix string)
```

**Errors:**
```go
var ErrNotFound = errors.New("sync: not found")
var ErrUnknownMutation = errors.New("sync: unknown mutation")
var ErrInvalidMutation = errors.New("sync: invalid mutation")
var ErrConflict = errors.New("sync: conflict")
```

**Constants:**
```go
const (
    Create Op = "create"
    Update Op = "update"
    Delete Op = "delete"
)
```

#### Remove (Non-Essential)

| Type/Func | Reason |
|-----------|--------|
| `MutatorMap` | Users can implement dispatch trivially |
| `NewMutatorMap()` | Removed with MutatorMap |
| `MutatorFunc` | Keep for convenience (common pattern) |
| `NotifierFunc` | Simple adapter, keep for convenience |
| `MultiNotifier` | Trivial to implement externally |
| `ErrInvalidScope` | Never used in practice |
| `ErrCursorTooOld` | Make internal, handle via HTTP 410 |
| `Code*` constants | Make internal; only used in HTTP layer |
| `PushRequest/Response` | Make internal; HTTP transport detail |
| `PullRequest/Response` | Make internal; HTTP transport detail |
| `SnapshotRequest/Response` | Make internal; HTTP transport detail |
| `Handlers` struct | Users should use Mount/MountAt |
| `(e *Engine) Handlers()` | Removed with Handlers struct |
| `(e *Engine) Store()` | Breaks encapsulation |
| `(e *Engine) Log()` | Breaks encapsulation |

#### Final Public API Count

- Types: 5 (Mutation, Result, Change, Op, Options)
- Interfaces: 5 (Store, Log, Mutator, Applied, Notifier)
- Structs: 1 (Engine)
- Functions: 1 (New)
- Methods: 5 (Push, Pull, Snapshot, Mount, MountAt)
- Errors: 4
- Constants: 3 (Op values)
- Helpers: 2 (MutatorFunc, NotifierFunc)

**Total: ~26 exported symbols** (down from ~40+)

---

### view/sync package (client-side)

#### Keep (Essential)

**Client:**
```go
type Client struct { ... }
type Options struct { ... }
func New(opts Options) *Client
func (c *Client) Start(ctx) error
func (c *Client) Stop()
func (c *Client) Sync() error
func (c *Client) Mutate(name string, args map[string]any) string
func (c *Client) Cursor() uint64
func (c *Client) IsOnline() bool
```

**Reactive Primitives:**
```go
type Signal[T any] struct { ... }
func NewSignal[T any](initial T) *Signal[T]
func (s *Signal[T]) Get() T
func (s *Signal[T]) Set(value T)
func (s *Signal[T]) Update(fn func(T) T)

type Computed[T any] struct { ... }
func NewComputed[T any](fn func() T) *Computed[T]
func (c *Computed[T]) Get() T

type Effect struct { ... }
func NewEffect(fn func()) *Effect
func (e *Effect) Stop()
```

**Collections:**
```go
type Collection[T any] struct { ... }
func NewCollection[T any](client *Client, name string) *Collection[T]
func (c *Collection[T]) Get(id string) *Entity[T]
func (c *Collection[T]) Create(id string, value T) *Entity[T]
func (c *Collection[T]) All() []*Entity[T]
func (c *Collection[T]) Count() int
func (c *Collection[T]) Find(predicate func(T) bool) []*Entity[T]
func (c *Collection[T]) Has(id string) bool

type Entity[T any] struct { ... }
func (e *Entity[T]) ID() string
func (e *Entity[T]) Get() T
func (e *Entity[T]) Set(value T)
func (e *Entity[T]) Delete()
func (e *Entity[T]) Exists() bool
```

**Persistence:**
```go
type Persistence interface {
    SaveState(cursor uint64, queue []Mutation, store map[string]map[string][]byte) error
    LoadState() (cursor uint64, queue []Mutation, store map[string]map[string][]byte, err error)
}
```

**Errors:**
```go
var ErrNotStarted = errors.New("sync: client not started")
var ErrAlreadyStarted = errors.New("sync: client already started")
```

#### Remove (Non-Essential)

| Type/Func | Reason |
|-----------|--------|
| `Store` (exported) | Make internal; clients use Collection |
| `Queue` (exported) | Make internal; implementation detail |
| `Transport` | Make internal; HTTP detail |
| `Mutation` | Duplicate from sync package; use internal |
| `Change`, `Result`, `Op` | Duplicate from sync package |
| `PushRequest/Response` | Internal HTTP types |
| `PullRequest/Response` | Internal HTTP types |
| `SnapshotRequest/Response` | Internal HTTP types |
| `LiveMessage` | Simplify live integration |
| `ParseLiveMessage` | Simplify live integration |
| `WithLive` | Simplify; use NotifyLive directly |
| `(c *Client) LiveHandler()` | Simplify; use NotifyLive directly |
| `(c *Client) LiveTopic()` | Simplify; caller knows scope |
| `MemoryPersistence` | Keep only Persistence interface |
| `NopPersistence` | Use nil Persistence instead |
| `(c *Client) Store()` | Breaks encapsulation |
| `(c *Client) Queue()` | Breaks encapsulation |
| `ErrOffline` | Never returned; internal state |
| `ErrConflict` | Duplicate from server sync |
| `ErrNotFound` | Duplicate from server sync |
| `ErrInvalidState` | Internal error |
| `ErrCursorTooOld` | Internal; handled by re-sync |
| `CollectionOption` | No options currently used |
| `(c *Collection) First()` | Just use Find()[0] |
| `(c *Collection) IDs()` | Use All() and map |
| `(c *Collection) Clear()` | Dangerous; rarely needed |
| `(c *Collection) Name()` | Rarely needed externally |
| `(e *Entity) IsPending()` | Implementation detail |
| `Signal.Version()` | Internal optimization detail |
| `Batch()` | Unimplemented placeholder |

#### Final Public API Count

- Types: 6 (Client, Options, Signal, Computed, Effect, Persistence)
- Generic Types: 2 (Collection[T], Entity[T])
- Functions: 4 (New, NewSignal, NewComputed, NewEffect)
- Client Methods: 6 (Start, Stop, Sync, Mutate, Cursor, IsOnline, NotifyLive)
- Signal Methods: 3 (Get, Set, Update)
- Computed Methods: 1 (Get)
- Effect Methods: 1 (Stop)
- Collection Methods: 5 (Get, Create, All, Count, Find, Has)
- Entity Methods: 5 (ID, Get, Set, Delete, Exists)
- Errors: 2

**Total: ~30 exported symbols** (down from ~60+)

---

## Simplified Persistence Interface

The current `Persistence` interface has 8 methods. Simplify to 2:

```go
type Persistence interface {
    // Save persists all client state atomically.
    Save(state State) error

    // Load restores client state. Returns zero State if none saved.
    Load() (State, error)
}

type State struct {
    Cursor   uint64
    ClientID string
    Queue    []mutation
    Store    map[string]map[string][]byte
}
```

---

## Live Integration Simplification

Remove LiveMessage, ParseLiveMessage, WithLive, LiveHandler, LiveTopic.

Replace with single method:

```go
// NotifyLive notifies the client of a cursor update.
// Called by live connection handler when sync notification received.
func (c *Client) NotifyLive(cursor uint64)
```

Usage:
```go
liveConn.OnMessage = func(msg live.Message) {
    if msg.Type == "sync" {
        var data struct{ Cursor uint64 }
        json.Unmarshal(msg.Body, &data)
        syncClient.NotifyLive(data.Cursor)
    }
}
```

---

## memory/ Subpackage

### Keep

```go
func NewStore() *Store
func NewLog() *Log
func NewApplied() *Applied
```

### Remove

| Type/Method | Reason |
|-------------|--------|
| `(s *Store) Clear()` | Testing helper only |
| `(l *Log) Clear()` | Testing helper only |
| `(l *Log) Len()` | Testing helper only |
| `(a *Applied) Clear()` | Testing helper only |
| `(a *Applied) Len()` | Testing helper only |

---

## Migration Path

1. **Phase 1:** Write spec files (this document)
2. **Phase 2:** Combine files without changing API
3. **Phase 3:** Deprecate removed symbols (if backward compat needed)
4. **Phase 4:** Remove deprecated symbols

For this refactor, we proceed directly to Phase 2-4 (aggressive, breaking changes allowed).

---

## Risks

1. **Breaking changes** - All usages of removed APIs must be updated
2. **Template sync** - cli/templates/sync must be updated to match
3. **Test coverage** - Combined test files must maintain coverage

---

## Success Criteria

- [ ] sync/sync.go compiles and passes all tests
- [ ] sync/sync_test.go contains all tests
- [ ] view/sync/sync.go compiles and passes all tests
- [ ] view/sync/sync_test.go contains all tests
- [ ] memory/memory.go compiles and passes all tests
- [ ] cli/templates/sync works with refactored packages
- [ ] Public API count reduced by ~40%
