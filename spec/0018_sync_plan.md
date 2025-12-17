# 0018 Sync Package Implementation Plan

## Overview

The `mizu/sync` package provides offline-first data synchronization with push/pull semantics, change log tracking, and cursor-based convergence. It is designed to be the authoritative layer for durable state changes while `mizu/live` acts as a realtime accelerator.

## Design Principles

1. **Sync is authoritative**: All durable state changes flow through the sync pipeline
2. **Offline-first**: Clients can operate offline and sync when connectivity returns
3. **Cursor-based convergence**: Clients pull from a cursor and converge to consistent state
4. **Push/Pull model**: Clients push mutations, server applies and logs changes, clients pull changes

## Package Structure

```
sync/
  doc.go           # Package documentation
  mutation.go      # Mutation types and handlers
  changelog.go     # Change log storage interface and implementations
  store.go         # Scoped data store interface
  handler.go       # HTTP handlers for push/pull endpoints
  broker.go        # PokeBroker for live integration
  scope.go         # Scope management for data partitioning
  errors.go        # Error types
  sync.go          # Main Engine type
```

## Core Types

### Mutation

```go
// Mutation represents a client-originated state change request.
type Mutation struct {
    // Name identifies the mutation type (e.g., "todo/toggle", "user/update").
    Name string `json:"name"`

    // Args contains mutation-specific arguments.
    Args map[string]any `json:"args,omitempty"`

    // ClientID is the originating client identifier (for deduplication).
    ClientID string `json:"client_id,omitempty"`

    // ClientSeq is the client's sequence number (for ordering).
    ClientSeq uint64 `json:"client_seq,omitempty"`

    // Scope identifies the data partition this mutation affects.
    Scope string `json:"scope,omitempty"`
}

// MutationResult is returned after applying a mutation.
type MutationResult struct {
    // OK indicates success.
    OK bool `json:"ok"`

    // Cursor is the new change log cursor after this mutation.
    Cursor uint64 `json:"cursor"`

    // Error contains any error message.
    Error string `json:"error,omitempty"`

    // Changes lists the entities affected by this mutation.
    Changes []Change `json:"changes,omitempty"`
}
```

### Change and ChangeLog

```go
// Change represents a single entity change in the log.
type Change struct {
    // Cursor is the unique, monotonically increasing position.
    Cursor uint64 `json:"cursor"`

    // Scope identifies the data partition.
    Scope string `json:"scope"`

    // Entity is the entity type (e.g., "todo", "user").
    Entity string `json:"entity"`

    // ID is the entity identifier.
    ID string `json:"id"`

    // Op is the operation type.
    Op ChangeOp `json:"op"`

    // Data contains the entity data (for create/update).
    Data any `json:"data,omitempty"`

    // Timestamp is when the change occurred.
    Timestamp time.Time `json:"ts"`
}

// ChangeOp defines the type of change.
type ChangeOp string

const (
    OpCreate ChangeOp = "create"
    OpUpdate ChangeOp = "update"
    OpDelete ChangeOp = "delete"
)

// ChangeLog stores and retrieves changes.
type ChangeLog interface {
    // Append adds a change and returns the assigned cursor.
    Append(ctx context.Context, change Change) (uint64, error)

    // Since returns changes after the given cursor for a scope.
    Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)

    // Cursor returns the current latest cursor.
    Cursor(ctx context.Context) (uint64, error)

    // Trim removes changes older than the given cursor (for compaction).
    Trim(ctx context.Context, beforeCursor uint64) error
}
```

### Store (Scoped Data)

```go
// Store provides scoped key-value storage for entity data.
type Store interface {
    // Get retrieves an entity by scope/entity/id.
    Get(ctx context.Context, scope, entity, id string) (any, error)

    // Set stores an entity.
    Set(ctx context.Context, scope, entity, id string, data any) error

    // Delete removes an entity.
    Delete(ctx context.Context, scope, entity, id string) error

    // List returns all entities of a type in a scope.
    List(ctx context.Context, scope, entity string) ([]any, error)

    // Snapshot returns all data in a scope (for full sync).
    Snapshot(ctx context.Context, scope string) (map[string]map[string]any, error)
}
```

### Mutator (Business Logic)

```go
// Mutator applies mutations to the store and returns changes.
type Mutator interface {
    // Apply processes a mutation and returns the resulting changes.
    // The mutator should:
    // 1. Validate the mutation
    // 2. Apply changes to the store
    // 3. Return the list of changes for the log
    Apply(ctx context.Context, store Store, m Mutation) ([]Change, error)
}

// MutatorFunc is a function that implements Mutator.
type MutatorFunc func(ctx context.Context, store Store, m Mutation) ([]Change, error)

func (f MutatorFunc) Apply(ctx context.Context, store Store, m Mutation) ([]Change, error) {
    return f(ctx, store, m)
}
```

### PokeBroker (Live Integration)

```go
// PokeBroker notifies live connections when data changes.
type PokeBroker interface {
    // Poke notifies watchers of a scope that data has changed.
    Poke(scope string, cursor uint64)
}

// Poke is the message sent to live connections.
type Poke struct {
    Scope  string `json:"scope"`
    Cursor uint64 `json:"cursor"`
}
```

## HTTP Handlers

### Push Handler

```go
// PushRequest is the payload for POST /_sync/push
type PushRequest struct {
    Mutations []Mutation `json:"mutations"`
}

// PushResponse is returned from POST /_sync/push
type PushResponse struct {
    Results []MutationResult `json:"results"`
    Cursor  uint64           `json:"cursor"`
}
```

**Endpoint**: `POST /_sync/push`

**Flow**:
1. Parse mutations from request body
2. For each mutation:
   a. Call mutator.Apply()
   b. Append changes to changelog
   c. Collect results
3. Poke affected scopes via PokeBroker
4. Return results with new cursor

### Pull Handler

```go
// PullRequest is the payload for POST /_sync/pull
type PullRequest struct {
    Scope  string `json:"scope"`
    Cursor uint64 `json:"cursor"`
    Limit  int    `json:"limit,omitempty"` // default 100
}

// PullResponse is returned from POST /_sync/pull
type PullResponse struct {
    Changes []Change `json:"changes"`
    Cursor  uint64   `json:"cursor"`
    HasMore bool     `json:"has_more"`
}
```

**Endpoint**: `POST /_sync/pull`

**Flow**:
1. Parse scope and cursor from request
2. Query changelog.Since(scope, cursor, limit)
3. Return changes with new cursor and hasMore flag

### Snapshot Handler (Optional)

```go
// SnapshotRequest is the payload for POST /_sync/snapshot
type SnapshotRequest struct {
    Scope string `json:"scope"`
}

// SnapshotResponse is returned from POST /_sync/snapshot
type SnapshotResponse struct {
    Data   map[string]map[string]any `json:"data"`
    Cursor uint64                    `json:"cursor"`
}
```

**Endpoint**: `POST /_sync/snapshot`

**Flow**:
1. Get current cursor
2. Get store.Snapshot(scope)
3. Return full data with cursor

## Engine

```go
// Engine is the main sync coordinator.
type Engine struct {
    store     Store
    changelog ChangeLog
    mutator   Mutator
    broker    PokeBroker
}

// Options configures the sync engine.
type Options struct {
    Store     Store
    ChangeLog ChangeLog
    Mutator   Mutator
    Broker    PokeBroker
}

// New creates a new sync engine.
func New(opts Options) *Engine

// Push applies mutations and returns results.
func (e *Engine) Push(ctx context.Context, mutations []Mutation) ([]MutationResult, error)

// Pull returns changes since a cursor.
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, uint64, bool, error)

// Snapshot returns all data in a scope.
func (e *Engine) Snapshot(ctx context.Context, scope string) (map[string]map[string]any, uint64, error)

// Mount registers sync routes on a Mizu app.
func (e *Engine) Mount(app *mizu.App)
```

## In-Memory Implementations

### MemoryChangeLog

```go
type MemoryChangeLog struct {
    mu      sync.RWMutex
    entries []Change
    cursor  uint64
}

func NewMemoryChangeLog() *MemoryChangeLog
```

- Thread-safe in-memory change log
- Append increments cursor atomically
- Since scans entries for matching scope

### MemoryStore

```go
type MemoryStore struct {
    mu   sync.RWMutex
    data map[string]map[string]map[string]any // scope -> entity -> id -> data
}

func NewMemoryStore() *MemoryStore
```

- Thread-safe in-memory key-value store
- Supports scoped entity storage

## Integration with mizu/live

The sync package integrates with live via the `PokeBroker` interface:

1. When `Engine.Push()` commits changes, it calls `broker.Poke(scope, cursor)`
2. Live's PubSub receives the poke and broadcasts to watchers
3. Clients receive poke and immediately call `/_sync/pull`

### Live Protocol Changes

Add `MsgTypePoke` to the live protocol:

```go
const MsgTypePoke byte = 0x0B

type PokePayload struct {
    Scope  string `json:"scope"`
    Cursor uint64 `json:"cursor"`
}
```

Clients send `JOIN(scope)` to subscribe to pokes for a scope.

## Example Usage

```go
// Define a mutator
mutator := sync.MutatorFunc(func(ctx context.Context, store sync.Store, m sync.Mutation) ([]sync.Change, error) {
    switch m.Name {
    case "todo/create":
        id := uuid.NewString()
        todo := map[string]any{
            "id":        id,
            "title":     m.Args["title"],
            "completed": false,
        }
        if err := store.Set(ctx, m.Scope, "todo", id, todo); err != nil {
            return nil, err
        }
        return []sync.Change{{
            Scope:  m.Scope,
            Entity: "todo",
            ID:     id,
            Op:     sync.OpCreate,
            Data:   todo,
        }}, nil

    case "todo/toggle":
        id := m.Args["id"].(string)
        data, err := store.Get(ctx, m.Scope, "todo", id)
        if err != nil {
            return nil, err
        }
        todo := data.(map[string]any)
        todo["completed"] = !todo["completed"].(bool)
        if err := store.Set(ctx, m.Scope, "todo", id, todo); err != nil {
            return nil, err
        }
        return []sync.Change{{
            Scope:  m.Scope,
            Entity: "todo",
            ID:     id,
            Op:     sync.OpUpdate,
            Data:   todo,
        }}, nil
    }
    return nil, fmt.Errorf("unknown mutation: %s", m.Name)
})

// Create engine
engine := sync.New(sync.Options{
    Store:     sync.NewMemoryStore(),
    ChangeLog: sync.NewMemoryChangeLog(),
    Mutator:   mutator,
    Broker:    livePokeBroker, // From live package
})

// Mount on app
engine.Mount(app)
```

## Client-Side Responsibilities

The client runtime (JavaScript) should:

1. Maintain local IndexedDB for offline storage
2. Queue mutations when offline
3. Push loop with retry and backoff
4. Pull loop:
   - Periodic pull (failsafe every 30s)
   - Immediate pull on poke
5. UI subscribes to local DB and re-renders

## Future Considerations

1. **Conflict Resolution**: Could add last-write-wins or custom merge strategies
2. **Compression**: Batch changes and compress large payloads
3. **Cursor Compaction**: Periodically compact old changes
4. **Distributed ChangeLog**: Redis/PostgreSQL implementations
5. **Client Mutation Validation**: Server validates before applying
