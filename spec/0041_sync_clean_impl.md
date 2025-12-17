# Sync Package Cleanup - Implementation Plan

Based on the review in `spec/0041_sync_clean.md`, this document outlines the concrete implementation steps to simplify the sync package.

## Goal

Reduce the sync package to its minimal concept set for an offline-first, cursor-based replication system with idempotent pushes.

**Core primitives to keep:**
1. **Mutation** - client intent with stable ID for dedupe
2. **Apply** - application logic that mutates state and emits changes
3. **Log** - ordered changes served by cursor
4. **Snapshot** - bootstrap when cursor is too old

---

## Phase 1: Simplify Core Types

### 1.1 Simplify Mutation Struct

**Current:**
```go
type Mutation struct {
    ID     string
    Name   string
    Scope  string
    Client string         // REMOVE
    Seq    uint64         // REMOVE
    Args   map[string]any // CHANGE to json.RawMessage
}
```

**Target:**
```go
type Mutation struct {
    ID    string          `json:"id"`
    Scope string          `json:"scope,omitempty"`
    Name  string          `json:"name"`
    Args  json.RawMessage `json:"args,omitempty"`
}
```

**Rationale:** Client and Seq don't participate in correctness. If client identification is needed, get it from request context (auth).

### 1.2 Simplify Change Struct

**Current:**
```go
type Change struct {
    Cursor uint64
    Scope  string
    Entity string          // REMOVE
    ID     string          // REMOVE
    Op     Op              // REMOVE
    Data   json.RawMessage
    Time   time.Time
}
```

**Target:**
```go
type Change struct {
    Cursor uint64          `json:"cursor"`
    Scope  string          `json:"scope"`
    Time   time.Time       `json:"time"`
    Data   json.RawMessage `json:"data"`
}
```

**Rationale:** Make Change opaque. Let the app decide payload schema. Removes the CRUD mindset (Op, Entity, ID).

### 1.3 Remove Op Type

Delete the `Op` type and its constants (`Create`, `Update`, `Delete`). Apps can include operation semantics in their `Data` payload if needed.

---

## Phase 2: Simplify Interfaces

### 2.1 Remove Store Interface

**Current:** Public `Store` interface with Get/Set/Delete/Snapshot.

**Target:** Remove entirely from public API. Apps close over their own storage in ApplyFunc.

If helper store is still useful, move to `sync/memory` only, not exported from core.

### 2.2 Replace Mutator with ApplyFunc

**Current:**
```go
type Mutator interface {
    Apply(ctx context.Context, store Store, m Mutation) ([]Change, error)
}
```

**Target:**
```go
type ApplyFunc func(ctx context.Context, m Mutation) ([]Change, error)
```

**Rationale:** ApplyFunc closes over whatever storage the app uses. Removes Store from signature.

### 2.3 Replace Applied with Dedupe

**Current:**
```go
type Applied interface {
    Get(ctx context.Context, scope, key string) (Result, bool, error)
    Put(ctx context.Context, scope, key string, res Result) error
}
```

**Target:**
```go
type Dedupe interface {
    Seen(ctx context.Context, scope, id string) (bool, error)
    Mark(ctx context.Context, scope, id string) error
}
```

**Rationale:** Simpler interface. No need to persist full Result. Duplicates become no-ops.

### 2.4 Add SnapshotFunc

**Current:** `Engine.Snapshot()` uses Store.Snapshot()

**Target:**
```go
type SnapshotFunc func(ctx context.Context, scope string) (json.RawMessage, uint64, error)
```

**Rationale:** App-defined snapshot, returns opaque JSON + cursor.

### 2.5 Remove Notifier Interface

**Current:** `Notifier` interface with `Notify(scope, cursor)` callback.

**Target:** Remove entirely. Engine.Push returns cursor information. Caller can integrate with live/realtime themselves.

---

## Phase 3: Update Engine

### 3.1 Simplify Engine Struct

**Current:**
```go
type Engine struct {
    store        Store
    log          Log
    applied      Applied
    mutator      Mutator
    notify       Notifier
    now          func() time.Time
    scopeFunc    func(context.Context, string) (string, error)
    maxPullLimit int
    maxPushBatch int
}
```

**Target:**
```go
type Engine struct {
    log      Log
    apply    ApplyFunc
    snapshot SnapshotFunc
    dedupe   Dedupe // optional
    now      func() time.Time
}
```

**Note:** `scopeFunc`, `maxPullLimit`, `maxPushBatch` move to HTTP transport.

### 3.2 Simplify Options

**Current:**
```go
type Options struct {
    Store        Store
    Log          Log
    Applied      Applied
    Mutator      Mutator
    Notify       Notifier
    Now          func() time.Time
    ScopeFunc    func(context.Context, string) (string, error)
    MaxPullLimit int
    MaxPushBatch int
}
```

**Target:**
```go
type Options struct {
    Log      Log
    Apply    ApplyFunc
    Snapshot SnapshotFunc // optional
    Dedupe   Dedupe       // optional
    Now      func() time.Time // optional, defaults to time.Now
}
```

### 3.3 Simplify Result Struct

**Current:**
```go
type Result struct {
    OK      bool
    Cursor  uint64
    Code    string  // REMOVE - string error codes
    Error   string
    Changes []Change
}
```

**Target:**
```go
type Result struct {
    OK      bool
    Cursor  uint64
    Error   string
    Changes []Change
}
```

### 3.4 Update Push Method

- Remove Applied.Get/Put, replace with Dedupe.Seen/Mark
- Remove Store dependency (ApplyFunc closes over storage)
- Remove Notify callback
- Return results with enough info for caller to notify

### 3.5 Update Pull Method

- Keep signature, return (changes, nextCursor, hasMore, error)
- Consider adding NextCursor to response

### 3.6 Update Snapshot Method

- Delegate to SnapshotFunc
- Return (json.RawMessage, cursor, error)

---

## Phase 4: Split HTTP Transport

### 4.1 Create sync/http Package

Move all HTTP-related code to `sync/http/`:
- `pushRequest`, `pushResponse`
- `pullRequest`, `pullResponse`
- `snapshotRequest`, `snapshotResponse`
- `handlePush`, `handlePull`, `handleSnapshot`
- `Mount`, `MountAt`
- String error codes (codeNotFound, etc.)

### 4.2 HTTP Transport Options

```go
type TransportOptions struct {
    ScopeFunc    func(context.Context, string) (string, error)
    MaxPullLimit int  // default 1000
    MaxPushBatch int  // default 100
}
```

### 4.3 Error Code Mapping

In transport layer, map sentinel errors to HTTP status/codes:
- `ErrNotFound` → 404, code "not_found"
- `ErrCursorTooOld` → 410, code "cursor_too_old"
- `ErrConflict` → 409, code "conflict"
- `ErrInvalidMutation` → 400, code "invalid_mutation"

---

## Phase 5: Clean Up Errors

### 5.1 Remove ErrUnknownMutation

Apps dispatch mutations in their ApplyFunc. Remove this error from core.

### 5.2 Keep Sentinel Errors

Keep in core:
- `ErrNotFound`
- `ErrInvalidMutation`
- `ErrConflict`
- `ErrCursorTooOld`

---

## Phase 6: Update Memory Package

### 6.1 Remove memory.Store

Or keep it as unexported helper for testing only.

### 6.2 Update memory.Log

Keep as-is, it implements the Log interface correctly.

### 6.3 Replace memory.Applied with memory.Dedupe

```go
type Dedupe struct {
    mu   sync.RWMutex
    seen map[string]map[string]bool // scope -> id -> seen
}

func (d *Dedupe) Seen(ctx context.Context, scope, id string) (bool, error)
func (d *Dedupe) Mark(ctx context.Context, scope, id string) error
```

---

## Phase 7: Update Tests

- Update all tests to use new simplified types
- Add tests for HTTP transport package
- Ensure idempotency still works with Dedupe

---

## Summary of Removals

| Item | Action |
|------|--------|
| `Store` interface | Remove from core, keep in memory as unexported |
| `Mutation.Client` | Remove |
| `Mutation.Seq` | Remove |
| `Mutation.Args` | Change from `map[string]any` to `json.RawMessage` |
| `Change.Entity` | Remove |
| `Change.ID` | Remove |
| `Change.Op` | Remove |
| `Op` type | Remove |
| `Mutator` interface | Replace with `ApplyFunc` |
| `MutatorFunc` | Remove (use ApplyFunc directly) |
| `Applied` interface | Replace with `Dedupe` |
| `Notifier` interface | Remove |
| `NotifierFunc` | Remove |
| `ErrUnknownMutation` | Remove |
| `Result.Code` | Remove |
| String error codes | Move to HTTP transport |
| HTTP handlers | Move to `sync/http` package |
| `Options.Store` | Remove |
| `Options.ScopeFunc` | Move to HTTP transport |
| `Options.MaxPullLimit` | Move to HTTP transport |
| `Options.MaxPushBatch` | Move to HTTP transport |

---

## Final Minimal Core API

```go
// sync/sync.go

type Mutation struct {
    ID    string          `json:"id"`
    Scope string          `json:"scope,omitempty"`
    Name  string          `json:"name"`
    Args  json.RawMessage `json:"args,omitempty"`
}

type Change struct {
    Cursor uint64          `json:"cursor"`
    Scope  string          `json:"scope"`
    Time   time.Time       `json:"time"`
    Data   json.RawMessage `json:"data"`
}

type Result struct {
    OK      bool
    Cursor  uint64
    Error   string
    Changes []Change
}

type Log interface {
    Append(ctx context.Context, scope string, changes []Change) (uint64, error)
    Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)
    Cursor(ctx context.Context, scope string) (uint64, error)
    Trim(ctx context.Context, scope string, before uint64) error
}

type ApplyFunc func(ctx context.Context, m Mutation) ([]Change, error)

type SnapshotFunc func(ctx context.Context, scope string) (json.RawMessage, uint64, error)

type Dedupe interface {
    Seen(ctx context.Context, scope, id string) (bool, error)
    Mark(ctx context.Context, scope, id string) error
}

type Options struct {
    Log      Log
    Apply    ApplyFunc
    Snapshot SnapshotFunc
    Dedupe   Dedupe
    Now      func() time.Time
}

type Engine struct { ... }

func New(opts Options) (*Engine, error)
func (e *Engine) Push(ctx context.Context, muts []Mutation) ([]Result, error)
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, uint64, bool, error)
func (e *Engine) Snapshot(ctx context.Context, scope string) (json.RawMessage, uint64, error)
```

---

## Implementation Order

1. ✅ Create this implementation plan
2. Simplify `Mutation` struct
3. Simplify `Change` struct (remove Op, Entity, ID)
4. Replace `Mutator` with `ApplyFunc`
5. Replace `Applied` with `Dedupe`
6. Remove `Store` interface
7. Remove `Notifier` interface
8. Remove `ErrUnknownMutation`
9. Update `Engine` struct and methods
10. Update `Options`
11. Create `sync/http` package with transport
12. Update `memory` package
13. Update all tests
