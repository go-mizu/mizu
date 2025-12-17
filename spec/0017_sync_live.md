Yes. The sync protocol above and `mizu/live` can coexist cleanly, but you should treat them as two different layers:

* **`mizu/sync`**: data correctness, offline-first, convergence (push/pull, local store)
* **`mizu/live`**: low-latency UI interactions and server push (events, realtime)

The best design is not to merge them into one wire protocol. Instead, integrate them by making Live act as a “realtime accelerator” for Sync.

## The recommended integration model

### Rule 1: Sync is authoritative

All durable state changes flow through the sync pipeline:

* client applies local mutation
* pushes it to server
* server commits and emits change log
* all clients pull and converge

### Rule 2: Live is only a wake-up channel

Live should not ship DOM patches as the primary data path. Live should mainly send:

* “poke: pull now”
* optional “presence” or “ephemeral” events (typing indicators, cursor, live selection)

This keeps offline-first semantics intact.

## How the pieces connect

### Server-side flow

1. Client pushes mutation to `/_sync/push`
2. Server applies it and appends to change log
3. Server publishes a poke to watchers on that scope
4. Clients receive poke via `mizu/live` and immediately call `/_sync/pull`

This yields real-time updates without making Live responsible for correctness.

## What `mizu/live` looks like in this model

### Live channel types

Keep Live extremely small:

* `JOIN(scope)`
* `LEAVE`
* server push: `POKE(cursor)`
* optional: `EPHEMERAL(event)` (non-durable)

You do not need `PATCH`, `COMMAND`, DOM actions, or templating concerns.

### Suggested endpoints

* `WS /_live` (or `/ _sync/poke/ws`)
* `POST /_sync/push`
* `POST /_sync/pull`

## Client runtime responsibilities

1. Maintain local DB (IndexedDB)
2. Maintain mutation queue
3. Push loop (retry, backoff)
4. Pull loop:

   * periodic pull (failsafe)
   * immediate pull on poke
5. UI subscribes to local DB and re-renders

Live only triggers “pull now”.

## Where server-rendered templates fit

You have two good options.

### Option A (preferred): Client-side rendering for synced data

* local DB drives UI directly (React/Svelte/vanilla)
* `mizu/view` still used for initial HTML shell and a few SSR pages

This is the strongest offline-first DX.

### Option B: Live SSR pages that read from local DB (hybrid)

If you want server-rendered HTML for pages, do not stream HTML patches. Instead:

* page shell SSR once
* region updates re-render on the client using local DB
* or client fetches server-rendered fragments over HTTP after pull

Example:

* pull completes
* client fetches `/partials/todos/list` if you insist on server templates

This keeps sync as the source of truth.

## What to do about LiveView-style “HandleEvent”

You can still offer `mizu/live` developer ergonomics, but map it onto sync primitives.

### Recommended DX: events produce mutations

In a live page, when user clicks:

* handler returns a mutation request
* runtime applies it locally and enqueues push
* server commits, pokes others
* everyone pulls

In other words, Live event handlers should not “mutate server session state” for durable data. They should emit sync mutations.

Pseudo shape:

```go
func (p *TodoPage) Handle(e live.Event) (sync.Mutation, error) {
  switch e.Name {
  case "toggle":
    return sync.Mutation{Name: "todo/toggle", Args: map[string]any{"id": e.Values["id"]}}, nil
  }
}
```

The live runtime then:

* runs the same mutation locally (client mutator)
* pushes to server

This is exactly the offline-first contract.

## What Live is still good for (beyond poke)

There are two categories where Live should own state:

### 1) Ephemeral UX state

* typing indicators
* presence
* cursor position
* hover/selection
* transient toasts, focus control

These do not belong in the durable sync log.

### 2) Server-side computation that is not “state”

* running validations
* search suggestions
* streaming AI responses

These can be WS messages that update UI directly, without polluting durable data sync.

## A concrete “wiring” plan for Mizu

### Packages

* `mizu/sync`

  * client: local store, queue, push/pull
  * server: push handler, pull handler, changelog store
* `mizu/live`

  * ws broker: subscribe(scope), poke(scope)
  * optional ephemeral events

### Server implementation

On successful push:

* compute affected scope(s)
* publish `poke(scope, cursor)` to live broker

### Client implementation

* connect live broker and subscribe to scope
* on poke:

  * call pull immediately
* UI reacts to local DB changes

## Why this works well

* Offline correctness remains simple and testable
* Realtime is fast because poke is tiny
* You avoid complex DOM patch protocol completely
* Server scale is easier: stateless HTTP + lightweight WS fanout
* Clear separation of concerns for developers

## When you would still want Live DOM patches

Only if your primary product is server-rendered pages with no client-side state. But that conflicts with “offline-first” as a priority. Offline-first almost always implies local client state and client rendering.

---

If you want, I can propose a minimal `mizu/live` v1 wire protocol that only supports `join/leave/poke` and an idiomatic Go API for a `PokeBroker` that is called from `mizu/sync` after commit.
