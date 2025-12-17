# Documentation Cleanup Plan

## Summary

Review and update all documentation in `docs/view/` to match the refactored view, sync, and live packages. Documentation should be beginner-friendly, detailed, and accurate to the current code.

## Documentation Standards

1. **Title Format**: Short, non-repetitive titles (e.g., "Overview" not "View Overview")
2. **No Level 1 Headers**: Since front matter has title, remove `# Title` headers
3. **Beginner-Friendly**: Detailed explanations for absolute beginners
4. **Code Accuracy**: All code examples must match current API

## Current Code Analysis

### View Package (`view/view.go`)

**Config struct:**
- `Dir` - Template directory path
- `FS` - Embedded filesystem (optional)
- `Extension` - File extension (default: ".html")
- `DefaultLayout` - Default layout name
- `Funcs` - Custom template functions
- `Delims` - Custom delimiters [left, right]
- `Development` - Enable hot reload

**Engine methods:**
- `New(Config)` - Create engine
- `Render(w, name, data, opts...)` - Render template
- `Load()` - Pre-load templates
- `Clear()` - Clear template cache
- `Middleware(app)` - Add to Mizu app
- `From(c)` - Get engine from context

**Render options:**
- `Layout(name)` - Set layout
- `NoLayout()` - Disable layout

**Built-in functions:**
- `dict`, `list`, `upper`, `lower`, `trim`, `contains`, `replace`, `split`, `join`, `hasPrefix`, `hasSuffix`

### Sync Package (`sync/sync.go`)

**Errors:**
- `ErrNotFound` - Entity not found
- `ErrInvalidMutation` - Invalid mutation format
- `ErrConflict` - Concurrent modification conflict
- `ErrCursorTooOld` - Cursor too old, full sync needed

**Types:**
- `Mutation` - ID, Scope, Name, Args
- `Result` - OK, Cursor, Error, Changes
- `Change` - Cursor, Scope, Time, Data

**Interfaces:**
- `Log` - Append, Since, Cursor, Trim
- `Dedupe` - Seen, Mark

**Functions:**
- `ApplyFunc` - `func(ctx, Mutation) ([]Change, error)`
- `SnapshotFunc` - `func(ctx, scope) (json.RawMessage, uint64, error)`

**Engine:**
- `New(Options)` - Create engine
- `Push(ctx, []Mutation)` - Apply mutations
- `Pull(ctx, scope, cursor, limit)` - Get changes since cursor
- `Snapshot(ctx, scope)` - Get full state

**Options:**
- `Log` - Change log implementation
- `Apply` - Mutation handler function
- `Snapshot` - Snapshot handler function
- `Dedupe` - Deduplication implementation
- `Now` - Time function (for testing)

### Live Package (`live/live.go`)

**Errors:**
- `ErrSessionClosed` - Session closed
- `ErrQueueFull` - Send queue full

**Types:**
- `Message` - Topic, Data

**Options:**
- `QueueSize` - Send buffer size
- `ReadLimit` - Max message size
- `OnAuth` - Authentication callback
- `OnMessage` - Message handler
- `OnClose` - Close handler
- `Origins` - Allowed origins
- `CheckOrigin` - Custom origin checker
- `IDGenerator` - Session ID generator

**Server:**
- `New(Options)` - Create server
- `Handler()` - HTTP handler
- `Publish(topic, data)` - Publish to topic
- `Subscribe(session, topic)` - Subscribe session
- `Unsubscribe(session, topic)` - Unsubscribe session

**Session:**
- `ID()` - Get session ID
- `Value()` - Get auth value
- `Send(data)` - Send message
- `Close()` - Close session
- `CloseError()` - Get close error

## Files to Update

### View Documentation

| File | Action | Notes |
|------|--------|-------|
| `overview.mdx` | Rewrite | Update to match actual Engine API |
| `quick-start.mdx` | Rewrite | Simple getting started guide |
| `engine.mdx` | Rewrite | Config options and engine methods |
| `templates.mdx` | Rewrite | Go template syntax basics |
| `layouts.mdx` | Rewrite | Layout system explanation |
| `functions.mdx` | Rewrite | Built-in and custom functions |
| `production.mdx` | Rewrite | Deployment best practices |
| `partials.mdx` | **Delete** | Not in current API |
| `components.mdx` | **Delete** | Not in current API |

### Sync Documentation

| File | Action | Notes |
|------|--------|-------|
| `sync-overview.mdx` | Rewrite | Match actual Engine API |
| `sync-quick-start.mdx` | Rewrite | Simple example |
| `sync-server.mdx` | Rewrite | Server setup and options |
| `sync-client.mdx` | **Delete** | No client-side sync in current API |
| `sync-reactive.mdx` | **Delete** | No reactive state in current API |
| `sync-collections.mdx` | **Delete** | No collections in current API |
| `sync-integration.mdx` | Rewrite | Live integration |

### Live Documentation

| File | Action | Notes |
|------|--------|-------|
| `live-overview.mdx` | Rewrite | Match actual Server API |
| `live-quick-start.mdx` | Rewrite | Simple WebSocket example |
| `live-server.mdx` | Rewrite | Server options |
| `live-sessions.mdx` | Rewrite | Session management |
| `live-pubsub.mdx` | Rewrite | Pub/sub patterns |

## Implementation Order

1. View documentation (core templating)
2. Sync documentation (state synchronization)
3. Live documentation (WebSocket communication)
4. Remove obsolete files
5. Final review

## Detailed Changes Per File

### `overview.mdx`
- Remove outdated feature list
- Focus on what view package actually provides
- Simple architecture diagram
- Link to other docs

### `quick-start.mdx`
- Create basic project structure
- Show minimal working example
- Step-by-step instructions
- Explain each part

### `engine.mdx`
- All Config options with examples
- Engine creation and methods
- Development vs production mode
- Embedded filesystem usage

### `templates.mdx`
- Go template syntax basics
- Variables and control flow
- Template inheritance concepts
- Common patterns

### `layouts.mdx`
- Layout concept explanation
- Default layout configuration
- Per-render layout override
- NoLayout option

### `functions.mdx`
- All built-in functions with examples
- Adding custom functions
- Function best practices

### `production.mdx`
- Pre-loading templates
- Embedded filesystem
- Caching strategies
- Error handling

### Sync Documentation
- Simple Log-based sync
- ApplyFunc implementation
- Pull/Push/Snapshot operations
- Integration with Live for real-time

### Live Documentation
- WebSocket server setup
- Authentication
- Pub/sub messaging
- Session management
