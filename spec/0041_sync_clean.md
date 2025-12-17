Below is a “remove as much as possible” review of `sync`, with an explicit target: smallest concept set that still yields an offline-first, cursor-based replication system with idempotent pushes.

## What to keep as the core model

You only need four primitives:

1. **Mutation**: client intent, with stable ID for dedupe.
2. **Apply**: application logic that mutates authoritative state and emits changes.
3. **Log**: ordered changes, served by cursor.
4. **Snapshot**: bootstrap when the cursor is too old.

Everything else is optional or can be derived.

## High-impact removals

### 1) Remove `Store` from the public surface

Right now `Store` is “the authoritative state store”, but it is only used by the `Mutator` and by `Snapshot`. For minimal learning, treat the store as an implementation detail of the application, not the sync framework.

Replace:

* `Store` interface (public)
* `Mutator.Apply(ctx, store, m)`
* `Engine.Snapshot()` returning a generic nested map

With:

* `Apply(ctx, m) ([]Change, error)` where Apply is provided by the app and closes over whatever storage it uses.
* `Snapshot(ctx, scope) (Snapshot, cursor, error)` where Snapshot is app-defined or raw JSON.

This eliminates one large interface and a big concept. It also avoids forcing everyone into `map[entity]map[id]json`.

If you still want an optional helper store, put it in `sync/memory` or `sync/store`, not in the core package.

### 2) Remove `Entity`, `ID`, `Op`, `Data` fields from `Change` (make it opaque)

For offline-first, the only hard requirement is: changes are **ordered** and **serializable**.

The current shape is prescriptive and encourages people to model everything as CRUD, which is often not what an offline-first domain wants.

Replace `Change` with:

```go
type Change struct {
	Cursor uint64          `json:"cursor"`
	Scope  string          `json:"scope"`
	Time   time.Time       `json:"time"`
	Data   json.RawMessage `json:"data"`
}
```

or even:

```go
type Change struct {
	Cursor uint64 `json:"cursor"`
	Data   []byte `json:"data"`
}
```

Let the app decide the payload schema. This removes `Op`, the CRUD mindset, and four fields worth of learner burden.

### 3) Remove `Client` and `Seq` from `Mutation`

They do not participate in correctness in your current engine. Keeping them invites false expectations about ordering and conflict resolution.

Keep only:

```go
type Mutation struct {
	ID    string          `json:"id"`
	Scope string          `json:"scope,omitempty"`
	Name  string          `json:"name"`
	Args  json.RawMessage `json:"args,omitempty"`
}
```

If you want client identification for rate limiting or auditing, get it from request context (auth), not from the mutation.

### 4) Remove `Notifier`

It is a convenience hook, but it is not required for offline-first correctness and adds a concept that beginners must evaluate.

If you want integration with `live`, expose the cursor from Push and let the caller notify. In other words: return enough information, not callbacks.

### 5) Remove the `Applied` interface and make idempotency minimal

This is the hardest trade-off.

If your goal is “learn fast”, you can make dedupe optional and simpler:

* Require `Mutation.ID` and provide a single interface `Dedupe` that can answer “seen?” and “mark seen” without storing the full `Result`.
* Or accept at-least-once semantics and state: “Mutator must be idempotent”. That is common in distributed systems, but it shifts complexity to app code.

Best minimal compromise for DX:

```go
type Dedupe interface {
	Seen(ctx context.Context, scope, id string) (bool, error)
	Mark(ctx context.Context, scope, id string) error
}
```

Then `Push` returns the fresh result each time (no need to persist `Result`), and duplicates just become no-ops. This removes storing `Result` forever and a bunch of corner cases.

If you truly need “replay returns identical result”, keep `Applied`, but move it out of v1 minimal core.

### 6) Remove `ErrUnknownMutation` by removing “Name-based dispatch” from core

The “mutation name to handler” registry is not implemented in this file anyway, but the error suggests it exists.

To reduce concepts, do not make the sync engine responsible for dispatching by name. That is application logic:

* The app receives `Mutation{Name, Args}` and dispatches inside its `Apply` function.

So you can drop:

* `ErrUnknownMutation`
* any mention of “mutation has no handler” in core docs

### 7) Remove the HTTP transport from the core package

This is the biggest simplification and also the most Go-idiomatic:

* `sync` should be correctness and primitives.
* `sync/http` (or `sync/transport/http`) should be the transport.

Benefits:

* The core package loses `net/http`, `mizu`, request/response structs, error codes, and bind limits.
* Learners can learn sync without learning your HTTP decisions.
* You can iterate transport independently.

If you insist on keeping HTTP in the same package for v1 DX, at least hide it behind one method (`Mount`) and avoid exporting any of the wire structs and error codes.

## Medium-impact simplifications (still worthwhile)

### A) Remove string error codes

You already return typed errors (`ErrCursorTooOld`, `ErrConflict`, etc.). The HTTP layer can map those to codes if needed.

Drop:

* `codeNotFound`, `codeUnknown`, etc.

Keep:

* exported sentinel errors
* a helper in transport layer to map errors to status/code

### B) Remove `DefaultScope` or make scope mandatory

Default scope is convenient, but it adds a rule. Minimal model is: scope is required.

If you keep it, hide it inside transport: if request has empty scope, set `_default`.

### C) Remove `MaxPullLimit` and `MaxPushBatch` from core Options

Those are transport-level concerns. If you split HTTP out, they naturally belong there.

## What the minimal v1 core API could look like

This is the shape that is easiest to teach:

```go
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

type Log interface {
	Append(ctx context.Context, scope string, changes []Change) (uint64, error)
	Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)
	Cursor(ctx context.Context, scope string) (uint64, error)
}

type ApplyFunc func(ctx context.Context, m Mutation) ([]Change, error)

type SnapshotFunc func(ctx context.Context, scope string) (json.RawMessage, uint64, error)

type Engine struct {
	log      Log
	apply    ApplyFunc
	snapshot SnapshotFunc
	dedupe   Dedupe // optional
	now      func() time.Time
}

func (e *Engine) Push(ctx context.Context, muts []Mutation) ([]Result, error)
func (e *Engine) Pull(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, uint64, bool, error)
func (e *Engine) Snapshot(ctx context.Context, scope string) (json.RawMessage, uint64, error)
```

Notable simplifications:

* No `Store` interface.
* `Change.Data` is opaque, app-defined.
* `Pull` returns `nextCursor` explicitly (less client surprise).
* No notifier.
* Dedupe is minimal or omitted.

## If you only do three changes now

These yield big reduction without a full redesign:

1. Change `Mutation.Args` from `map[string]any` to `json.RawMessage` (or add `ArgsRaw` and deprecate map).
2. Add `NextCursor` to `pullResponse` (so clients do not compute it).
3. Split HTTP transport into `sync/transport/http` and delete HTTP-only constants/types from core.

That will make the package materially easier to learn while preserving your current semantics.

If you want, I can provide a concrete “reduced” version of this file (core-only plus a tiny `transport/http` file) in the same Go style you are using, keeping names aligned with your preference for Go core library conventions.
