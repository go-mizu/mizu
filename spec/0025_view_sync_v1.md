# view/sync Package Proposal

## Purpose

Package `view/sync` provides a **client-side synchronization runtime** that integrates:

- `sync` as the authoritative correctness layer
- Reactive state management for UI binding
- Optional `live` as a latency accelerator

Its primary goal is to enable **offline-first interactive applications** where UI state remains responsive, mutations are safely queued, and authoritative state convergence happens transparently.

---

## Design principles

1. **Offline-first**
   Mutations are queued locally and applied optimistically. Network is optional.

2. **Reactive**
   State changes automatically propagate to dependent computations and effects.

3. **Pull-based convergence**
   Authoritative state is pulled from the server via sync protocol.

4. **Push-accelerated**
   Optional live connection triggers immediate pulls when data changes.

5. **Scoped**
   All data is partitioned by scope, matching the sync model.

6. **Minimal surface area**
   Few types, predictable behavior, explicit contracts.

---

## Package layout

```
view/sync/
  doc.go        // Package documentation
  client.go     // Client runtime
  store.go      // Local state store
  queue.go      // Mutation queue
  signal.go     // Reactive primitives (Signal, Computed, Effect)
  entity.go     // Entity and Collection types
  http.go       // HTTP transport
  live.go       // Live integration (optional)
  errors.go     // Error values
```

---

## Core concepts

### Client

`Client` is the main runtime that coordinates local state, mutation queue, and sync transport.

```go
type Client struct {
    // unexported fields
}
```

```go
type Options struct {
    // BaseURL is the sync server endpoint (e.g., "https://api.example.com/_sync")
    BaseURL string

    // Scope is the data partition identifier
    Scope string

    // HTTP is the HTTP client to use. Default: http.DefaultClient
    HTTP *http.Client

    // Live is an optional live connection for push notifications
    Live *live.Conn

    // OnError is called when background operations fail
    OnError func(error)

    // OnSync is called after successful sync
    OnSync func(cursor uint64)
}
```

```go
func New(opts Options) *Client
func (c *Client) Start(ctx context.Context) error
func (c *Client) Stop()
```

---

### Signal

`Signal` is a reactive value container that notifies dependents on change.

```go
type Signal[T any] struct {
    // unexported
}
```

```go
func NewSignal[T any](initial T) *Signal[T]
func (s *Signal[T]) Get() T
func (s *Signal[T]) Set(value T)
func (s *Signal[T]) Update(fn func(T) T)
```

Notes:

* `Get` registers dependency when called within `Computed` or `Effect`
* `Set` triggers recomputation of dependents
* Zero-allocation for primitive types

---

### Computed

`Computed` is a derived value that recomputes when dependencies change.

```go
type Computed[T any] struct {
    // unexported
}
```

```go
func NewComputed[T any](fn func() T) *Computed[T]
func (c *Computed[T]) Get() T
```

Notes:

* Automatically tracks dependencies
* Lazily recomputes on access after invalidation
* Cached until dependencies change

---

### Effect

`Effect` runs a side effect when dependencies change.

```go
type Effect struct {
    // unexported
}
```

```go
func NewEffect(fn func()) *Effect
func (e *Effect) Stop()
```

Notes:

* Runs immediately on creation
* Re-runs when any tracked signal changes
* Can be stopped to prevent future runs

---

### Entity

`Entity` represents a single synchronized record with reactive access.

```go
type Entity[T any] struct {
    // unexported
}
```

```go
func (e *Entity[T]) ID() string
func (e *Entity[T]) Get() T
func (e *Entity[T]) Set(value T)
func (e *Entity[T]) Delete()
func (e *Entity[T]) Exists() bool
```

Notes:

* `Get` returns current value (reactive)
* `Set` queues a mutation
* Changes are optimistic until confirmed

---

### Collection

`Collection` manages a set of entities of the same type.

```go
type Collection[T any] struct {
    // unexported
}
```

```go
func (c *Client) Collection(name string, opts ...CollectionOption) *Collection[T]
func (col *Collection[T]) Get(id string) *Entity[T]
func (col *Collection[T]) Create(id string, value T) *Entity[T]
func (col *Collection[T]) All() []*Entity[T]
func (col *Collection[T]) Count() int
func (col *Collection[T]) Find(predicate func(T) bool) []*Entity[T]
```

Notes:

* Entities are created lazily on first access
* `All` returns reactive slice that updates on changes
* CRUD operations queue mutations automatically

---

### Store

`Store` is the local state container.

```go
type Store struct {
    // unexported
}
```

```go
func (s *Store) Get(entity, id string) ([]byte, bool)
func (s *Store) Set(entity, id string, data []byte)
func (s *Store) Delete(entity, id string)
func (s *Store) Snapshot() map[string]map[string][]byte
func (s *Store) Load(data map[string]map[string][]byte)
func (s *Store) Clear()
```

Notes:

* Thread-safe in-memory store
* Supports bulk load for hydration
* Notifies signals on change

---

### Queue

`Queue` manages pending mutations.

```go
type Queue struct {
    // unexported
}
```

```go
func (q *Queue) Push(m Mutation) string
func (q *Queue) Pending() []Mutation
func (q *Queue) Len() int
func (q *Queue) Clear()
func (q *Queue) Remove(id string)
```

Notes:

* Mutations are persisted (e.g., localStorage in WASM)
* Order is preserved
* Deduplication by mutation ID

---

## Mutation

`Mutation` mirrors the sync package type for client use.

```go
type Mutation struct {
    ID     string         `json:"id"`
    Name   string         `json:"name"`
    Scope  string         `json:"scope,omitempty"`
    Client string         `json:"client,omitempty"`
    Seq    uint64         `json:"seq,omitempty"`
    Args   map[string]any `json:"args,omitempty"`
}
```

Helper for creating mutations:

```go
func (c *Client) Mutate(name string, args map[string]any) string
```

---

## Sync behavior

### Push (upload mutations)

1. Collect pending mutations from queue
2. POST to `{baseURL}/push`
3. Process results:
   - Success: remove from queue, update cursor
   - Conflict: trigger full sync, re-apply
   - Error: retry with backoff
4. Notify `OnSync` callback

### Pull (download changes)

1. POST to `{baseURL}/pull` with current cursor
2. Apply changes to local store
3. Trigger reactive updates
4. If `hasMore`, continue pulling
5. Notify `OnSync` callback

### Snapshot (full sync)

1. POST to `{baseURL}/snapshot`
2. Replace local store contents
3. Update cursor
4. Clear mutation queue (if cursor advanced)
5. Trigger reactive updates

---

## Live integration

When a `live.Conn` is provided, the client subscribes to scope-specific topics:

```go
// Automatic subscription on start
topic := "sync:" + scope
conn.Subscribe(topic)

// On message
conn.OnMessage = func(msg live.Message) {
    if msg.Type == "sync" {
        // Trigger pull
        client.Sync()
    }
}
```

Notes:

* Live is purely optional
* Without live, client polls or syncs on demand
* Live accelerates convergence but doesn't guarantee delivery

---

## Optimistic updates

When a mutation is queued:

1. Apply change to local store immediately
2. Mark entity as "pending"
3. Trigger reactive updates
4. Queue mutation for push

When push succeeds:

1. Update cursor
2. Clear pending flag
3. Confirm optimistic state

When push fails with conflict:

1. Rollback optimistic change
2. Trigger full sync
3. Re-apply valid mutations

---

## Offline support

The client operates fully offline:

1. Mutations are persisted to local storage
2. Reads always succeed from local store
3. When online, mutations are pushed in order
4. Conflicts are resolved server-side

Persistence interface:

```go
type Persistence interface {
    Load() ([]Mutation, uint64, error)  // Load queue and cursor
    Save(mutations []Mutation, cursor uint64) error
    SaveStore(data map[string]map[string][]byte) error
    LoadStore() (map[string]map[string][]byte, error)
}
```

Default implementations:

* `MemoryPersistence` - ephemeral, testing
* `LocalStoragePersistence` - browser (WASM)
* `FilePersistence` - disk-based

---

## Client lifecycle

```go
client := viewsync.New(viewsync.Options{
    BaseURL: "https://api.example.com/_sync",
    Scope:   "user:123",
})

// Start background sync
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := client.Start(ctx); err != nil {
    log.Fatal(err)
}

// Use collections
todos := viewsync.Collection[Todo](client, "todo")

// Create entity (queues mutation)
todo := todos.Create("abc", Todo{Title: "Buy milk", Done: false})

// Read reactively
title := todo.Get().Title

// Update (queues mutation)
todo.Set(Todo{Title: "Buy milk", Done: true})

// Computed values
doneTodos := viewsync.NewComputed(func() []*viewsync.Entity[Todo] {
    return todos.Find(func(t Todo) bool { return t.Done })
})

// Effects
viewsync.NewEffect(func() {
    fmt.Printf("Done: %d/%d\n", len(doneTodos.Get()), todos.Count())
})
```

---

## Error handling

```go
var (
    ErrOffline       = errors.New("viewsync: offline")
    ErrConflict      = errors.New("viewsync: conflict")
    ErrNotFound      = errors.New("viewsync: not found")
    ErrInvalidState  = errors.New("viewsync: invalid state")
)
```

Errors are handled via:

* `OnError` callback for background operations
* Return values for explicit calls
* Rollback and retry for conflicts

---

## What this package deliberately avoids

* Server-side rendering or template logic
* CRDT conflict resolution (server is authoritative)
* Real-time collaboration primitives
* Complex query language
* Schema validation (handled by mutators)

---

## Relationship to other packages

* **sync**: Provides server-side engine, types, and protocol
* **live**: Optional push notifications for latency reduction
* **view**: Independent template engine (not integrated directly)

The name `view/sync` indicates it's the client-side sync runtime intended for view layers, but it doesn't import or depend on the `view` package.

---

## Why this matches Go core library style

* Short names (`Signal`, `Entity`, `Queue`)
* Generics for type safety
* Interfaces describe behavior, not structure
* Explicit error handling
* Small, composable pieces
* Clear ownership of state

---

## Summary

* `view/sync` is the **client-side sync runtime**
* Provides reactive state management
* Enables offline-first operation
* Integrates optionally with `live`
* Follows sync protocol for correctness
* Minimal, composable API
