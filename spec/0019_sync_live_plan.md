# 0019 Sync + Live Integration Plan

## Overview

This document details how `mizu/sync` and `mizu/live` integrate to provide offline-first realtime applications. The key insight is that Live acts as a "realtime accelerator" for Sync, not as the primary data path.

## Integration Architecture

```
                                  +-----------------+
                                  |   mizu/live     |
                                  |  (PokeBroker)   |
                                  +--------+--------+
                                           |
                                           | Poke(scope, cursor)
                                           v
+----------+  Mutation   +-----------+  Changes   +------------+
|  Client  | ----------> | mizu/sync | ---------> | ChangeLog  |
+----------+             +-----------+            +------------+
     ^                         |
     |                         | Poke via WebSocket
     |                         v
     |                  +-----------+
     +------ Pull ----> |   Client  |
                        +-----------+
```

## Changes to mizu/live

### 1. Add Poke Message Type

Add to `protocol.go`:

```go
const MsgTypePoke byte = 0x0B

// PokePayload is the payload for POKE messages.
type PokePayload struct {
    Scope  string `json:"scope"`
    Cursor uint64 `json:"cursor"`
}
```

### 2. Add SyncPokeBroker

Create a bridge between sync and live in `sync_bridge.go`:

```go
// SyncPokeBroker implements sync.PokeBroker using live's PubSub.
type SyncPokeBroker struct {
    pubsub PubSub
}

// NewSyncPokeBroker creates a poke broker backed by live pubsub.
func NewSyncPokeBroker(pubsub PubSub) *SyncPokeBroker {
    return &SyncPokeBroker{pubsub: pubsub}
}

// Poke sends a poke message to all watchers of a scope.
func (b *SyncPokeBroker) Poke(scope string, cursor uint64) {
    // The scope becomes the pubsub topic
    b.pubsub.Publish(scope, Poke{
        Scope:  scope,
        Cursor: cursor,
    })
}
```

### 3. Handle Poke in Session Handler

Update `handler.go` to handle poke messages from server channel:

```go
func (h *sessionHandler) handleServerMessage(msg any) error {
    h.session.lock()
    defer h.session.unlock()

    // Check if this is a sync poke
    if poke, ok := msg.(Poke); ok {
        return h.sendPoke(poke)
    }

    // Existing Info handling...
    if err := h.page.info(h.ctx, h.session, msg); err != nil {
        // ...
    }
    return h.sendPatches()
}

func (h *sessionHandler) sendPoke(poke Poke) error {
    h.send(MsgTypePoke, 0, PokePayload{
        Scope:  poke.Scope,
        Cursor: poke.Cursor,
    })
    return nil
}
```

### 4. Add Scope Subscription Support

Modify session handling to support scope subscriptions:

```go
// In JoinPayload, add Scopes field
type JoinPayload struct {
    Token     string            `json:"token"`
    URL       string            `json:"url"`
    Params    map[string]string `json:"params,omitempty"`
    SessionID string            `json:"session,omitempty"`
    Reconnect bool              `json:"reconnect,omitempty"`
    Scopes    []string          `json:"scopes,omitempty"` // NEW
}

// In sessionHandler.run(), subscribe to scopes
func (h *sessionHandler) run(ctx context.Context, conn *wsConn, join *JoinPayload) error {
    // ...existing code...

    // Subscribe to sync scopes
    if len(join.Scopes) > 0 && h.live.pubsub != nil {
        h.live.pubsub.Subscribe(h.session.getID(), join.Scopes...)
    }

    // ...rest of code...
}
```

### 5. Add Subscribe/Unsubscribe Messages

```go
const (
    MsgTypeSubscribe   byte = 0x0C
    MsgTypeUnsubscribe byte = 0x0D
)

type SubscribePayload struct {
    Scopes []string `json:"scopes"`
}

type UnsubscribePayload struct {
    Scopes []string `json:"scopes"`
}
```

Handle in `handleClientMessage`:

```go
case MsgTypeSubscribe:
    var payload SubscribePayload
    if err := msg.parsePayload(&payload); err != nil {
        return err
    }
    if h.live.pubsub != nil {
        h.live.pubsub.Subscribe(h.session.getID(), payload.Scopes...)
    }
    h.send(MsgTypeReply, msg.Ref, ReplyPayload{Status: "ok"})
    return nil

case MsgTypeUnsubscribe:
    var payload UnsubscribePayload
    if err := msg.parsePayload(&payload); err != nil {
        return err
    }
    if h.live.pubsub != nil {
        h.live.pubsub.Unsubscribe(h.session.getID(), payload.Scopes...)
    }
    h.send(MsgTypeReply, msg.Ref, ReplyPayload{Status: "ok"})
    return nil
```

## Changes to Client Runtime

### 1. Handle Poke Messages

Update `runtime.go` (JavaScript):

```javascript
// In message handler
case MSG_TYPE_POKE:
    const poke = payload;
    this.emit('poke', poke);
    // Trigger immediate pull
    if (this.syncClient) {
        this.syncClient.pull(poke.scope, poke.cursor);
    }
    break;
```

### 2. Add Scope Subscription API

```javascript
class MizuLive {
    subscribe(scopes) {
        this.send({
            type: MSG_TYPE_SUBSCRIBE,
            payload: { scopes: Array.isArray(scopes) ? scopes : [scopes] }
        });
    }

    unsubscribe(scopes) {
        this.send({
            type: MSG_TYPE_UNSUBSCRIBE,
            payload: { scopes: Array.isArray(scopes) ? scopes : [scopes] }
        });
    }
}
```

### 3. Sync Client Integration

```javascript
class MizuSync {
    constructor(options) {
        this.baseUrl = options.baseUrl || '';
        this.cursors = new Map(); // scope -> cursor
        this.db = null; // IndexedDB instance
    }

    async push(mutations) {
        const response = await fetch(`${this.baseUrl}/_sync/push`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mutations })
        });
        return response.json();
    }

    async pull(scope, fromCursor = 0) {
        const cursor = fromCursor || this.cursors.get(scope) || 0;
        const response = await fetch(`${this.baseUrl}/_sync/pull`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ scope, cursor })
        });
        const result = await response.json();

        // Apply changes to local DB
        await this.applyChanges(result.changes);

        // Update cursor
        this.cursors.set(scope, result.cursor);

        // Continue if more
        if (result.has_more) {
            return this.pull(scope, result.cursor);
        }

        return result;
    }

    async applyChanges(changes) {
        // Apply each change to IndexedDB
        for (const change of changes) {
            switch (change.op) {
                case 'create':
                case 'update':
                    await this.db.put(change.entity, change.data);
                    break;
                case 'delete':
                    await this.db.delete(change.entity, change.id);
                    break;
            }
        }
        // Emit change event for UI updates
        this.emit('changes', changes);
    }
}
```

## Usage Pattern

### Server Setup

```go
func main() {
    app := mizu.New()

    // Create view engine
    viewEngine := view.New(view.Options{Dir: "templates"})

    // Create live with pubsub
    pubsub := live.NewInmemPubSub()
    lv := live.New(live.Options{
        View:   viewEngine,
        PubSub: pubsub,
    })

    // Create poke broker bridging sync to live
    pokeBroker := live.NewSyncPokeBroker(pubsub)

    // Create sync engine with poke broker
    syncEngine := sync.New(sync.Options{
        Store:     sync.NewMemoryStore(),
        ChangeLog: sync.NewMemoryChangeLog(),
        Mutator:   todoMutator,
        Broker:    pokeBroker,
    })

    // Mount both
    lv.Mount(app)
    syncEngine.Mount(app)

    // Register pages
    todoPage := &TodoPage{}
    lv.RegisterPage("/todos", live.Wrap(todoPage))
    app.Get("/todos", live.Handle(lv, todoPage))

    app.Start(":3000")
}
```

### Client Setup

```javascript
// Initialize sync client
const sync = new MizuSync({ baseUrl: '' });
await sync.openDB('myapp');

// Initialize live connection with scope subscription
const live = new MizuLive({
    url: '/todos',
    scopes: ['user:123:todos'] // Subscribe to user's todo scope
});

// Handle pokes - triggers pull
live.on('poke', async (poke) => {
    console.log('Data changed in scope', poke.scope);
    await sync.pull(poke.scope, poke.cursor);
});

// Handle sync changes - update UI
sync.on('changes', (changes) => {
    for (const change of changes) {
        updateUI(change);
    }
});

// Initial pull
await sync.pull('user:123:todos');

// Connect live for realtime pokes
live.connect();
```

### Live Page with Sync Events

```go
type TodoPage struct{}

func (p *TodoPage) Handle(ctx *live.Ctx, s *live.Session[TodoState], e live.Event) error {
    switch e.Name {
    case "create":
        // Instead of directly mutating, emit a sync mutation
        mutation := sync.Mutation{
            Name:  "todo/create",
            Scope: s.State.Scope,
            Args: map[string]any{
                "title": e.Get("title"),
            },
        }
        // Push mutation (could be done client-side too)
        ctx.PushMutation(mutation)
        return nil

    case "toggle":
        mutation := sync.Mutation{
            Name:  "todo/toggle",
            Scope: s.State.Scope,
            Args: map[string]any{
                "id": e.Get("id"),
            },
        }
        ctx.PushMutation(mutation)
        return nil
    }
    return nil
}
```

## Ephemeral Events (Non-Sync)

Some events should NOT go through sync:

1. **Typing indicators** - Send directly via live
2. **Cursor position** - Ephemeral, no persistence needed
3. **Presence** - Managed by live, not sync
4. **Transient UI state** - Hover, focus, selection

For these, continue using live's existing Info/Handle pattern:

```go
func (p *TodoPage) Handle(ctx *live.Ctx, s *live.Session[TodoState], e live.Event) error {
    switch e.Name {
    case "typing":
        // Ephemeral - broadcast directly, no sync
        ctx.Broadcast("typing", map[string]any{
            "user": s.UserID,
            "typing": e.GetBool("typing"),
        })
        return nil

    case "create":
        // Durable - goes through sync
        // ...
    }
}
```

## File Changes Summary

### New Files
- `sync/doc.go`
- `sync/mutation.go`
- `sync/changelog.go`
- `sync/store.go`
- `sync/handler.go`
- `sync/broker.go`
- `sync/scope.go`
- `sync/errors.go`
- `sync/sync.go`

### Modified Files in view/live
- `protocol.go` - Add MsgTypePoke, MsgTypeSubscribe, MsgTypeUnsubscribe
- `handler.go` - Handle poke messages and scope subscriptions
- `sync_bridge.go` - New file for SyncPokeBroker
- `runtime.go` - Update client JS for poke handling

## Migration Path

For existing live pages:

1. **No changes required** for ephemeral interactions
2. **For durable state**: Refactor Handle to emit sync mutations
3. **Add scope subscriptions** in page Mount for sync-driven updates
4. **Update client** to use sync client for local state

## Benefits

1. **Offline-first**: Clients work without connection
2. **Realtime**: Pokes provide instant notification
3. **Simplicity**: Live protocol stays minimal
4. **Scalability**: Stateless HTTP + lightweight WS fanout
5. **Clear separation**: Sync = data, Live = UI coordination
