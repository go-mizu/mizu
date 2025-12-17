# Sync Package v1 Implementation Plan

This document details the implementation plan for the `sync` package as specified in [0021_sync_v1.md](./0021_sync_v1.md).

---

## Package Layout

```
sync/
  doc.go          // Package documentation
  types.go        // Mutation, Result, Change, Op
  errors.go       // Error definitions
  store.go        // Store interface
  log.go          // Log interface
  mutator.go      // Mutator interface
  applied.go      // Applied interface
  notify.go       // Notifier interface
  engine.go       // Engine and Options
  http.go         // HTTP handlers (push, pull, snapshot)

  engine_test.go  // Engine unit tests
  http_test.go    // HTTP handler tests

  memory/
    store.go      // In-memory Store implementation
    log.go        // In-memory Log implementation
    applied.go    // In-memory Applied implementation
    store_test.go
    log_test.go
    applied_test.go
```

---

## File-by-File Implementation

### 1. `doc.go`

Package-level documentation following Go conventions.

```go
// Package sync provides authoritative, offline-first state synchronization.
//
// It defines a durable mutation pipeline, an ordered change log, and
// cursor-based replication so clients can converge to correct state
// across retries, disconnects, offline operation, and server restarts.
//
// The package is transport-agnostic. HTTP is the default transport,
// but correctness does not depend on realtime delivery.
//
// Design principles:
//
//   - Authoritative: All durable state changes are applied on the server
//   - Offline-first: Clients may enqueue and replay mutations safely
//   - Idempotent: Replayed mutations must not apply twice
//   - Pull-based: Clients converge by pulling changes since a cursor
//   - Scoped: All data and cursors are partitioned by scope
package sync
```

---

### 2. `types.go`

Core data types: `Mutation`, `Result`, `Change`, `Op`.

```go
package sync

import "time"

// Mutation represents a client request to change state.
// It is a command, not a state patch.
type Mutation struct {
    // ID uniquely identifies this mutation for idempotency.
    ID string `json:"id"`

    // Name identifies the mutation type.
    Name string `json:"name"`

    // Scope identifies the data partition.
    Scope string `json:"scope,omitempty"`

    // Client identifies the originating client.
    Client string `json:"client,omitempty"`

    // Seq is a client-local sequence number.
    Seq uint64 `json:"seq,omitempty"`

    // Args contains mutation-specific arguments.
    Args map[string]any `json:"args,omitempty"`
}

// Result describes the outcome of applying a mutation.
type Result struct {
    OK      bool     `json:"ok"`
    Cursor  uint64   `json:"cursor,omitempty"`
    Code    string   `json:"code,omitempty"`
    Error   string   `json:"error,omitempty"`
    Changes []Change `json:"changes,omitempty"`
}

// Change is a single durable state change recorded in the log.
type Change struct {
    Cursor uint64    `json:"cursor"`
    Scope  string    `json:"scope"`
    Entity string    `json:"entity"`
    ID     string    `json:"id"`
    Op     Op        `json:"op"`
    Data   []byte    `json:"data,omitempty"`
    Time   time.Time `json:"time"`
}

// Op defines the type of change operation.
type Op string

const (
    Create Op = "create"
    Update Op = "update"
    Delete Op = "delete"
)
```

---

### 3. `errors.go`

Standard error definitions.

```go
package sync

import "errors"

var (
    // ErrNotFound is returned when an entity is not found.
    ErrNotFound = errors.New("sync: not found")

    // ErrUnknownMutation is returned when a mutation name has no handler.
    ErrUnknownMutation = errors.New("sync: unknown mutation")

    // ErrInvalidMutation is returned when a mutation is malformed.
    ErrInvalidMutation = errors.New("sync: invalid mutation")

    // ErrInvalidScope is returned when a scope is invalid or missing.
    ErrInvalidScope = errors.New("sync: invalid scope")

    // ErrCursorTooOld is returned when a cursor has been trimmed from the log.
    ErrCursorTooOld = errors.New("sync: cursor too old")

    // ErrConflict is returned when there is a conflict during mutation.
    ErrConflict = errors.New("sync: conflict")
)

// Error codes for Result.Code field.
const (
    CodeOK           = ""
    CodeNotFound     = "not_found"
    CodeUnknown      = "unknown_mutation"
    CodeInvalid      = "invalid_mutation"
    CodeCursorTooOld = "cursor_too_old"
    CodeConflict     = "conflict"
    CodeInternal     = "internal_error"
)
```

---

### 4. `store.go`

Store interface for authoritative state.

```go
package sync

import "context"

// Store is the authoritative state store.
// All data is stored as JSON bytes to avoid type ambiguity.
type Store interface {
    // Get retrieves an entity by scope/entity/id.
    // Returns ErrNotFound if the entity does not exist.
    Get(ctx context.Context, scope, entity, id string) ([]byte, error)

    // Set stores an entity.
    Set(ctx context.Context, scope, entity, id string, data []byte) error

    // Delete removes an entity.
    // Returns nil if the entity does not exist.
    Delete(ctx context.Context, scope, entity, id string) error

    // Snapshot returns all data in a scope.
    // Returns map[entity]map[id]data.
    Snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, error)
}
```

---

### 5. `log.go`

Log interface for ordered changes.

```go
package sync

import "context"

// Log records ordered changes and serves them by cursor.
type Log interface {
    // Append adds changes to the log and returns the final cursor.
    // Changes are assigned sequential cursor values starting after
    // the current cursor. The returned cursor is the last assigned.
    Append(ctx context.Context, scope string, changes []Change) (uint64, error)

    // Since returns changes after the given cursor for a scope.
    // Returns up to limit changes.
    Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)

    // Cursor returns the current latest cursor for a scope.
    Cursor(ctx context.Context, scope string) (uint64, error)

    // Trim removes changes before the given cursor (for compaction).
    Trim(ctx context.Context, scope string, before uint64) error
}
```

---

### 6. `mutator.go`

Mutator interface and helpers.

```go
package sync

import "context"

// Mutator contains application business logic.
// It processes mutations and returns the resulting changes.
type Mutator interface {
    // Apply processes a mutation and returns the resulting changes.
    // The mutator should:
    //   1. Validate the mutation
    //   2. Apply changes to the store
    //   3. Return the list of changes for the log
    Apply(ctx context.Context, store Store, m Mutation) ([]Change, error)
}

// MutatorFunc is a function that implements Mutator.
type MutatorFunc func(context.Context, Store, Mutation) ([]Change, error)

// Apply implements Mutator.
func (f MutatorFunc) Apply(ctx context.Context, s Store, m Mutation) ([]Change, error) {
    return f(ctx, s, m)
}

// MutatorMap dispatches to registered handlers by mutation name.
type MutatorMap struct {
    handlers map[string]MutatorFunc
}

// NewMutatorMap creates a new MutatorMap.
func NewMutatorMap() *MutatorMap {
    return &MutatorMap{handlers: make(map[string]MutatorFunc)}
}

// Register adds a handler for a mutation name.
func (m *MutatorMap) Register(name string, handler MutatorFunc) {
    m.handlers[name] = handler
}

// Apply dispatches to the registered handler.
func (m *MutatorMap) Apply(ctx context.Context, store Store, mut Mutation) ([]Change, error) {
    handler, ok := m.handlers[mut.Name]
    if !ok {
        return nil, ErrUnknownMutation
    }
    return handler(ctx, store, mut)
}
```

---

### 7. `applied.go`

Applied interface for idempotency tracking.

```go
package sync

import "context"

// Applied tracks mutations already processed.
// This enables strict idempotency - replayed mutations
// return their original result without re-execution.
type Applied interface {
    // Get retrieves a stored result for a mutation key.
    // Returns (result, true, nil) if found.
    // Returns (Result{}, false, nil) if not found.
    Get(ctx context.Context, scope, key string) (Result, bool, error)

    // Put stores a result for a mutation key.
    Put(ctx context.Context, scope, key string, res Result) error
}
```

---

### 8. `notify.go`

Notifier interface for optional integration.

```go
package sync

// Notifier is an optional integration hook.
// It is called after changes are committed to notify
// external systems (e.g., live/websocket connections).
type Notifier interface {
    Notify(scope string, cursor uint64)
}

// NotifierFunc wraps a function as a Notifier.
type NotifierFunc func(scope string, cursor uint64)

// Notify implements Notifier.
func (f NotifierFunc) Notify(scope string, cursor uint64) {
    f(scope, cursor)
}

// MultiNotifier fans out notifications to multiple notifiers.
type MultiNotifier []Notifier

// Notify calls all notifiers.
func (m MultiNotifier) Notify(scope string, cursor uint64) {
    for _, n := range m {
        n.Notify(scope, cursor)
    }
}
```

---

### 9. `engine.go`

Engine coordinates mutation application and replication.

```go
package sync

import (
    "context"
    "time"
)

// Engine coordinates mutation application and replication.
type Engine struct {
    store   Store
    log     Log
    applied Applied
    mutator Mutator
    notify  Notifier
}

// Options configures the sync engine.
type Options struct {
    Store   Store    // Required: authoritative state store
    Log     Log      // Required: change log
    Applied Applied  // Optional: idempotency tracker (nil disables deduplication)
    Mutator Mutator  // Required: mutation processor
    Notify  Notifier // Optional: notification hook
}

// New creates a new sync engine.
func New(opts Options) *Engine {
    return &Engine{
        store:   opts.Store,
        log:     opts.Log,
        applied: opts.Applied,
        mutator: opts.Mutator,
        notify:  opts.Notify,
    }
}

// Push applies mutations and returns results.
// For each mutation:
//  1. Compute dedupe key (mutation.ID)
//  2. If already applied, return stored result
//  3. Call Mutator.Apply
//  4. Append changes to Log (with timestamps)
//  5. Store Result in Applied
//  6. Call Notify with latest cursor
func (e *Engine) Push(ctx context.Context, mutations []Mutation) ([]Result, error) {
    results := make([]Result, len(mutations))
    affectedScopes := make(map[string]uint64) // scope -> max cursor

    for i, mut := range mutations {
        result := e.processMutation(ctx, mut)
        results[i] = result

        if result.OK && result.Cursor > 0 {
            scope := mut.Scope
            if scope == "" {
                scope = "_default"
            }
            if result.Cursor > affectedScopes[scope] {
                affectedScopes[scope] = result.Cursor
            }
        }
    }

    // Notify affected scopes
    if e.notify != nil {
        for scope, cursor := range affectedScopes {
            e.notify.Notify(scope, cursor)
        }
    }

    return results, nil
}

func (e *Engine) processMutation(ctx context.Context, mut Mutation) Result {
    scope := mut.Scope
    if scope == "" {
        scope = "_default"
    }

    // Check idempotency
    if e.applied != nil && mut.ID != "" {
        if res, found, err := e.applied.Get(ctx, scope, mut.ID); err != nil {
            return Result{OK: false, Code: CodeInternal, Error: err.Error()}
        } else if found {
            return res
        }
    }

    // Apply mutation
    changes, err := e.mutator.Apply(ctx, e.store, mut)
    if err != nil {
        return e.errorResult(err)
    }

    // Set timestamps on changes
    now := time.Now()
    for i := range changes {
        if changes[i].Time.IsZero() {
            changes[i].Time = now
        }
        if changes[i].Scope == "" {
            changes[i].Scope = scope
        }
    }

    // Append to log
    var cursor uint64
    if len(changes) > 0 {
        var err error
        cursor, err = e.log.Append(ctx, scope, changes)
        if err != nil {
            return Result{OK: false, Code: CodeInternal, Error: err.Error()}
        }
    } else {
        cursor, _ = e.log.Cursor(ctx, scope)
    }

    result := Result{
        OK:      true,
        Cursor:  cursor,
        Changes: changes,
    }

    // Store for idempotency
    if e.applied != nil && mut.ID != "" {
        _ = e.applied.Put(ctx, scope, mut.ID, result)
    }

    return result
}

func (e *Engine) errorResult(err error) Result {
    code := CodeInternal
    switch err {
    case ErrNotFound:
        code = CodeNotFound
    case ErrUnknownMutation:
        code = CodeUnknown
    case ErrInvalidMutation:
        code = CodeInvalid
    case ErrConflict:
        code = CodeConflict
    }
    return Result{OK: false, Code: code, Error: err.Error()}
}

// Pull returns changes since a cursor.
// Returns (changes, hasMore) where hasMore indicates more data exists.
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, bool, error) {
    if scope == "" {
        scope = "_default"
    }
    if limit <= 0 {
        limit = 100
    }

    // Request one extra to detect hasMore
    changes, err := e.log.Since(ctx, scope, cursor, limit+1)
    if err != nil {
        return nil, false, err
    }

    hasMore := len(changes) > limit
    if hasMore {
        changes = changes[:limit]
    }

    return changes, hasMore, nil
}

// Snapshot returns all data in a scope with the current cursor.
func (e *Engine) Snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, uint64, error) {
    if scope == "" {
        scope = "_default"
    }

    cursor, err := e.log.Cursor(ctx, scope)
    if err != nil {
        return nil, 0, err
    }

    data, err := e.store.Snapshot(ctx, scope)
    if err != nil {
        return nil, 0, err
    }

    return data, cursor, nil
}

// Store returns the underlying store.
func (e *Engine) Store() Store { return e.store }

// Log returns the underlying log.
func (e *Engine) Log() Log { return e.log }
```

---

### 10. `http.go`

HTTP transport handlers.

```go
package sync

import (
    "net/http"

    "github.com/go-mizu/mizu"
)

// PushRequest is the payload for POST /_sync/push
type PushRequest struct {
    Mutations []Mutation `json:"mutations"`
}

// PushResponse is returned from POST /_sync/push
type PushResponse struct {
    Results []Result `json:"results"`
}

// PullRequest is the payload for POST /_sync/pull
type PullRequest struct {
    Scope  string `json:"scope,omitempty"`
    Cursor uint64 `json:"cursor"`
    Limit  int    `json:"limit,omitempty"`
}

// PullResponse is returned from POST /_sync/pull
type PullResponse struct {
    Changes []Change `json:"changes"`
    HasMore bool     `json:"has_more"`
}

// SnapshotRequest is the payload for POST /_sync/snapshot
type SnapshotRequest struct {
    Scope string `json:"scope,omitempty"`
}

// SnapshotResponse is returned from POST /_sync/snapshot
type SnapshotResponse struct {
    Data   map[string]map[string][]byte `json:"data"`
    Cursor uint64                       `json:"cursor"`
}

// Mount registers sync routes on a Mizu app at /_sync/*.
func (e *Engine) Mount(app *mizu.App) {
    e.MountAt(app, "/_sync")
}

// MountAt registers sync routes at a custom prefix.
func (e *Engine) MountAt(app *mizu.App, prefix string) {
    app.Post(prefix+"/push", e.handlePush())
    app.Post(prefix+"/pull", e.handlePull())
    app.Post(prefix+"/snapshot", e.handleSnapshot())
}

func (e *Engine) handlePush() mizu.Handler {
    return func(c *mizu.Ctx) error {
        var req PushRequest
        if err := c.BindJSON(&req, 1<<20); err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{
                "error": "invalid request body",
            })
        }

        if len(req.Mutations) == 0 {
            return c.JSON(http.StatusBadRequest, map[string]string{
                "error": "no mutations provided",
            })
        }

        results, err := e.Push(c.Context(), req.Mutations)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{
                "error": err.Error(),
            })
        }

        return c.JSON(http.StatusOK, PushResponse{Results: results})
    }
}

func (e *Engine) handlePull() mizu.Handler {
    return func(c *mizu.Ctx) error {
        var req PullRequest
        if err := c.BindJSON(&req, 1<<16); err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{
                "error": "invalid request body",
            })
        }

        changes, hasMore, err := e.Pull(c.Context(), req.Scope, req.Cursor, req.Limit)
        if err != nil {
            code := http.StatusInternalServerError
            errCode := CodeInternal
            if err == ErrCursorTooOld {
                code = http.StatusGone
                errCode = CodeCursorTooOld
            }
            return c.JSON(code, map[string]string{
                "code":  errCode,
                "error": err.Error(),
            })
        }

        return c.JSON(http.StatusOK, PullResponse{
            Changes: changes,
            HasMore: hasMore,
        })
    }
}

func (e *Engine) handleSnapshot() mizu.Handler {
    return func(c *mizu.Ctx) error {
        var req SnapshotRequest
        if err := c.BindJSON(&req, 1<<16); err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{
                "error": "invalid request body",
            })
        }

        data, cursor, err := e.Snapshot(c.Context(), req.Scope)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{
                "error": err.Error(),
            })
        }

        return c.JSON(http.StatusOK, SnapshotResponse{
            Data:   data,
            Cursor: cursor,
        })
    }
}

// Handlers provides individual handlers for custom mounting.
type Handlers struct {
    Push     mizu.Handler
    Pull     mizu.Handler
    Snapshot mizu.Handler
}

// Handlers returns individual sync handlers.
func (e *Engine) Handlers() Handlers {
    return Handlers{
        Push:     e.handlePush(),
        Pull:     e.handlePull(),
        Snapshot: e.handleSnapshot(),
    }
}
```

---

### 11. `memory/store.go`

In-memory Store implementation.

```go
package memory

import (
    "context"
    "sync"

    gosync "github.com/go-mizu/mizu/sync"
)

// Store is an in-memory implementation of sync.Store.
type Store struct {
    mu   sync.RWMutex
    data map[string]map[string]map[string][]byte // scope -> entity -> id -> data
}

// NewStore creates a new in-memory store.
func NewStore() *Store {
    return &Store{
        data: make(map[string]map[string]map[string][]byte),
    }
}

// Get retrieves an entity by scope/entity/id.
func (s *Store) Get(ctx context.Context, scope, entity, id string) ([]byte, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    entities, ok := s.data[scope]
    if !ok {
        return nil, gosync.ErrNotFound
    }
    items, ok := entities[entity]
    if !ok {
        return nil, gosync.ErrNotFound
    }
    data, ok := items[id]
    if !ok {
        return nil, gosync.ErrNotFound
    }

    // Return a copy to avoid mutation
    cp := make([]byte, len(data))
    copy(cp, data)
    return cp, nil
}

// Set stores an entity.
func (s *Store) Set(ctx context.Context, scope, entity, id string, data []byte) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.data[scope] == nil {
        s.data[scope] = make(map[string]map[string][]byte)
    }
    if s.data[scope][entity] == nil {
        s.data[scope][entity] = make(map[string][]byte)
    }

    // Store a copy
    cp := make([]byte, len(data))
    copy(cp, data)
    s.data[scope][entity][id] = cp
    return nil
}

// Delete removes an entity.
func (s *Store) Delete(ctx context.Context, scope, entity, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if entities, ok := s.data[scope]; ok {
        if items, ok := entities[entity]; ok {
            delete(items, id)
            if len(items) == 0 {
                delete(entities, entity)
            }
            if len(entities) == 0 {
                delete(s.data, scope)
            }
        }
    }
    return nil
}

// Snapshot returns all data in a scope.
func (s *Store) Snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    entities, ok := s.data[scope]
    if !ok {
        return make(map[string]map[string][]byte), nil
    }

    // Deep copy
    result := make(map[string]map[string][]byte, len(entities))
    for entity, items := range entities {
        result[entity] = make(map[string][]byte, len(items))
        for id, data := range items {
            cp := make([]byte, len(data))
            copy(cp, data)
            result[entity][id] = cp
        }
    }
    return result, nil
}

// Clear removes all data from the store.
func (s *Store) Clear() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data = make(map[string]map[string]map[string][]byte)
}
```

---

### 12. `memory/log.go`

In-memory Log implementation.

```go
package memory

import (
    "context"
    "sync"

    gosync "github.com/go-mizu/mizu/sync"
)

// Log is an in-memory implementation of sync.Log.
type Log struct {
    mu      sync.RWMutex
    entries map[string][]gosync.Change // scope -> changes
    cursors map[string]uint64          // scope -> current cursor
    global  uint64                     // global cursor counter
}

// NewLog creates a new in-memory log.
func NewLog() *Log {
    return &Log{
        entries: make(map[string][]gosync.Change),
        cursors: make(map[string]uint64),
    }
}

// Append adds changes to the log and returns the final cursor.
func (l *Log) Append(ctx context.Context, scope string, changes []gosync.Change) (uint64, error) {
    if len(changes) == 0 {
        return l.Cursor(ctx, scope)
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    for i := range changes {
        l.global++
        changes[i].Cursor = l.global
        if changes[i].Scope == "" {
            changes[i].Scope = scope
        }
        l.entries[scope] = append(l.entries[scope], changes[i])
    }

    l.cursors[scope] = l.global
    return l.global, nil
}

// Since returns changes after the given cursor for a scope.
func (l *Log) Since(ctx context.Context, scope string, cursor uint64, limit int) ([]gosync.Change, error) {
    l.mu.RLock()
    defer l.mu.RUnlock()

    if limit <= 0 {
        limit = 100
    }

    entries := l.entries[scope]
    var result []gosync.Change

    for _, entry := range entries {
        if entry.Cursor <= cursor {
            continue
        }
        result = append(result, entry)
        if len(result) >= limit {
            break
        }
    }

    return result, nil
}

// Cursor returns the current latest cursor for a scope.
func (l *Log) Cursor(ctx context.Context, scope string) (uint64, error) {
    l.mu.RLock()
    defer l.mu.RUnlock()
    return l.cursors[scope], nil
}

// Trim removes changes before the given cursor.
func (l *Log) Trim(ctx context.Context, scope string, before uint64) error {
    l.mu.Lock()
    defer l.mu.Unlock()

    entries := l.entries[scope]
    idx := 0
    for i, entry := range entries {
        if entry.Cursor >= before {
            idx = i
            break
        }
    }

    if idx > 0 {
        l.entries[scope] = entries[idx:]
    }
    return nil
}

// Len returns the number of entries in the log for a scope.
func (l *Log) Len(scope string) int {
    l.mu.RLock()
    defer l.mu.RUnlock()
    return len(l.entries[scope])
}

// Clear removes all entries from the log.
func (l *Log) Clear() {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.entries = make(map[string][]gosync.Change)
    l.cursors = make(map[string]uint64)
    l.global = 0
}
```

---

### 13. `memory/applied.go`

In-memory Applied implementation.

```go
package memory

import (
    "context"
    "sync"

    gosync "github.com/go-mizu/mizu/sync"
)

// Applied is an in-memory implementation of sync.Applied.
type Applied struct {
    mu    sync.RWMutex
    store map[string]map[string]gosync.Result // scope -> key -> result
}

// NewApplied creates a new in-memory applied tracker.
func NewApplied() *Applied {
    return &Applied{
        store: make(map[string]map[string]gosync.Result),
    }
}

// Get retrieves a stored result for a mutation key.
func (a *Applied) Get(ctx context.Context, scope, key string) (gosync.Result, bool, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    scoped, ok := a.store[scope]
    if !ok {
        return gosync.Result{}, false, nil
    }
    result, ok := scoped[key]
    return result, ok, nil
}

// Put stores a result for a mutation key.
func (a *Applied) Put(ctx context.Context, scope, key string, res gosync.Result) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.store[scope] == nil {
        a.store[scope] = make(map[string]gosync.Result)
    }
    a.store[scope][key] = res
    return nil
}

// Clear removes all stored results.
func (a *Applied) Clear() {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.store = make(map[string]map[string]gosync.Result)
}

// Len returns the number of stored results for a scope.
func (a *Applied) Len(scope string) int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return len(a.store[scope])
}
```

---

## Test Plan

### Unit Tests

#### `engine_test.go`

```go
package sync_test

import (
    "context"
    "testing"

    "github.com/go-mizu/mizu/sync"
    "github.com/go-mizu/mizu/sync/memory"
)

func TestEngine_Push_Success(t *testing.T)
func TestEngine_Push_Idempotency(t *testing.T)
func TestEngine_Push_UnknownMutation(t *testing.T)
func TestEngine_Push_MutationError(t *testing.T)
func TestEngine_Push_MultipleScopes(t *testing.T)
func TestEngine_Push_NotifyCalled(t *testing.T)

func TestEngine_Pull_Empty(t *testing.T)
func TestEngine_Pull_WithChanges(t *testing.T)
func TestEngine_Pull_Pagination(t *testing.T)
func TestEngine_Pull_HasMore(t *testing.T)

func TestEngine_Snapshot_Empty(t *testing.T)
func TestEngine_Snapshot_WithData(t *testing.T)
```

#### `http_test.go`

```go
package sync_test

import (
    "testing"
)

func TestHTTP_Push_Success(t *testing.T)
func TestHTTP_Push_BadRequest(t *testing.T)
func TestHTTP_Push_NoMutations(t *testing.T)

func TestHTTP_Pull_Success(t *testing.T)
func TestHTTP_Pull_BadRequest(t *testing.T)

func TestHTTP_Snapshot_Success(t *testing.T)
func TestHTTP_Snapshot_BadRequest(t *testing.T)
```

#### `memory/store_test.go`

```go
package memory_test

import (
    "testing"
)

func TestStore_GetSetDelete(t *testing.T)
func TestStore_GetNotFound(t *testing.T)
func TestStore_Snapshot_Empty(t *testing.T)
func TestStore_Snapshot_WithData(t *testing.T)
func TestStore_Snapshot_IsCopy(t *testing.T)
func TestStore_Concurrency(t *testing.T)
```

#### `memory/log_test.go`

```go
package memory_test

import (
    "testing"
)

func TestLog_Append_SingleChange(t *testing.T)
func TestLog_Append_MultipleChanges(t *testing.T)
func TestLog_Since_Empty(t *testing.T)
func TestLog_Since_WithCursor(t *testing.T)
func TestLog_Since_Limit(t *testing.T)
func TestLog_Cursor(t *testing.T)
func TestLog_Trim(t *testing.T)
func TestLog_Scoped(t *testing.T)
func TestLog_Concurrency(t *testing.T)
```

#### `memory/applied_test.go`

```go
package memory_test

import (
    "testing"
)

func TestApplied_GetPut(t *testing.T)
func TestApplied_GetNotFound(t *testing.T)
func TestApplied_Scoped(t *testing.T)
func TestApplied_Concurrency(t *testing.T)
```

### Integration Tests

#### `engine_integration_test.go`

```go
package sync_test

func TestIntegration_OfflineReplay(t *testing.T)
func TestIntegration_CursorConvergence(t *testing.T)
func TestIntegration_MultiClientSync(t *testing.T)
```

---

## Implementation Order

1. **Types first**: `types.go`, `errors.go` (no dependencies)
2. **Interfaces**: `store.go`, `log.go`, `mutator.go`, `applied.go`, `notify.go`
3. **Memory implementations**: `memory/store.go`, `memory/log.go`, `memory/applied.go`
4. **Memory tests**: Verify implementations work correctly
5. **Engine**: `engine.go` (depends on interfaces)
6. **Engine tests**: Core functionality
7. **HTTP handlers**: `http.go` (depends on engine)
8. **HTTP tests**: Full HTTP integration
9. **Documentation**: `doc.go`

---

## Key Design Decisions

### 1. `[]byte` for Data Storage

The spec uses `[]byte` for all data storage to maintain JSON-native representation and avoid type ambiguity. This differs from the previous implementation that used `any`.

### 2. Scoped Cursors

Cursors are scoped, meaning each scope has its own cursor sequence. This allows for independent partitions.

### 3. Global Cursor Counter in Memory Log

The in-memory log uses a global counter across all scopes to ensure cursor uniqueness. This simplifies the implementation while still supporting scoped queries.

### 4. Idempotency is Optional

The `Applied` interface is optional. If not provided, mutations will not be deduplicated. This allows simpler setups that don't need idempotency.

### 5. Notifier is Optional

The `Notifier` interface is optional. If not provided, no notifications are sent. This decouples sync from realtime systems.

### 6. Default Scope

When scope is empty, the engine uses `"_default"` as the scope name. This provides a sensible default for single-scope applications.

---

## Error Handling

| Error | Code | HTTP Status |
|-------|------|-------------|
| `ErrNotFound` | `not_found` | 404 (in context) |
| `ErrUnknownMutation` | `unknown_mutation` | 200 (in Result) |
| `ErrInvalidMutation` | `invalid_mutation` | 200 (in Result) |
| `ErrCursorTooOld` | `cursor_too_old` | 410 Gone |
| `ErrConflict` | `conflict` | 200 (in Result) |
| Internal errors | `internal_error` | 500 |

---

## Future Considerations

1. **SQL implementations**: Postgres, SQLite schemas can be derived from the interfaces
2. **TTL for Applied**: Add expiration for idempotency entries
3. **Batched notifications**: Debounce notifications for bulk operations
4. **Cursor validation**: Return `ErrCursorTooOld` when cursor is trimmed
