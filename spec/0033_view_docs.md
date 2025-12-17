# View Documentation Plan

This document outlines the comprehensive documentation plan for Mizu's View system, which includes three interconnected packages:

1. **view** - Template engine for server-side rendering
2. **live** - WebSocket-based realtime message delivery
3. **sync** - Offline-first state synchronization

---

## Documentation Structure

A new "View" tab will be added to `docs/docs.json` containing the following sections:

### 1. View (Template Engine)

The view package provides a standardized template system built on Go's `html/template`.

#### Pages:
- **overview.mdx** - Introduction to the view system and its core concepts
- **quick-start.mdx** - Step-by-step guide to get started with views
- **engine.mdx** - Engine configuration and options
- **templates.mdx** - Understanding template types (pages, layouts, components, partials)
- **layouts.mdx** - Layout templates and slot mechanism
- **components.mdx** - Reusable component templates
- **partials.mdx** - Template fragments
- **functions.mdx** - Built-in template helper functions
- **production.mdx** - Production deployment with embed.FS

### 2. Live (Realtime)

The live package provides low-latency realtime message delivery over WebSocket.

#### Pages:
- **overview.mdx** - Introduction to the live package
- **quick-start.mdx** - Getting started with WebSocket connections
- **server.mdx** - Server configuration and options
- **sessions.mdx** - Session management and authentication
- **pubsub.mdx** - Topic-based publish/subscribe

### 3. Sync (State Synchronization)

The sync package provides offline-first state synchronization with both server and client components.

#### Pages:
- **overview.mdx** - Introduction to the sync system
- **quick-start.mdx** - Getting started with sync
- **server.mdx** - Server-side sync engine
- **client.mdx** - Client-side sync runtime
- **reactive.mdx** - Reactive state management (Signal, Computed, Effect)
- **collections.mdx** - Entity and Collection management
- **integration.mdx** - Integrating live with sync for realtime updates

---

## Target Audience

The documentation is written for **absolute beginners** who may not be familiar with:
- Go template syntax
- WebSocket protocols
- State synchronization patterns
- Reactive programming

Each page will:
1. Start with a clear explanation of what the concept is
2. Explain why it's useful
3. Show complete, runnable code examples
4. Explain each line of code
5. Provide common use cases and best practices

---

## Documentation Style Guidelines

1. **Friendly tone** - Use conversational language
2. **Complete examples** - Always show full, working code
3. **Step-by-step** - Break down complex tasks into numbered steps
4. **Visual aids** - Use diagrams and tables where helpful
5. **Cross-references** - Link to related documentation
6. **Common mistakes** - Highlight pitfalls and how to avoid them

---

## Package Relationships

```
                    ┌─────────────────────────────────────┐
                    │           Your Application          │
                    └─────────────────────────────────────┘
                                      │
         ┌────────────────────────────┼────────────────────────────┐
         │                            │                            │
         ▼                            ▼                            ▼
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│      view       │         │      live       │         │      sync       │
│                 │         │                 │         │                 │
│ Template Engine │         │   WebSocket     │         │ State Sync      │
│ - Layouts       │         │   Pub/Sub       │◀────────│ - Server Engine │
│ - Components    │         │ - Sessions      │         │ - Client Runtime│
│ - Partials      │         │ - Topics        │         │ - Reactive State│
└─────────────────┘         └─────────────────┘         └─────────────────┘
```

Key relationships:
- **view** is independent - for server-side HTML rendering
- **live** is independent - for realtime WebSocket communication
- **sync** optionally integrates with **live** for immediate sync triggers
- **view/sync** provides a client-side reactive runtime that can receive live notifications

---

## docs.json Tab Structure

```json
{
  "tab": "View",
  "groups": [
    {
      "group": "View Engine",
      "pages": [
        "view/overview",
        "view/quick-start",
        "view/engine",
        "view/templates",
        "view/layouts",
        "view/components",
        "view/partials",
        "view/functions",
        "view/production"
      ]
    },
    {
      "group": "Live (Realtime)",
      "pages": [
        "view/live-overview",
        "view/live-quick-start",
        "view/live-server",
        "view/live-sessions",
        "view/live-pubsub"
      ]
    },
    {
      "group": "Sync (State)",
      "pages": [
        "view/sync-overview",
        "view/sync-quick-start",
        "view/sync-server",
        "view/sync-client",
        "view/sync-reactive",
        "view/sync-collections",
        "view/sync-integration"
      ]
    }
  ]
}
```

---

## Implementation Order

1. Create spec file (this document)
2. Update docs/docs.json with View tab
3. Create all View Engine documentation pages
4. Create all Live documentation pages
5. Create all Sync documentation pages
6. Review and cross-link pages

---

## Key Concepts to Cover

### View Engine
- Directory structure convention (views/layouts, views/pages, etc.)
- Template syntax (Go templates)
- Slot mechanism for layout composition
- Stack mechanism for asset aggregation
- Component isolation and data passing
- Development vs production modes
- Template caching and preloading
- Custom template functions

### Live Package
- WebSocket protocol basics
- Connection lifecycle
- Authentication via OnAuth callback
- Topic-based pub/sub model
- Session management
- Backpressure handling
- Error handling

### Sync Package
- Offline-first architecture
- Mutation pipeline
- Idempotency guarantees
- Cursor-based replication
- Store, Log, and Applied interfaces
- Reactive state (Signal, Computed, Effect)
- Collection and Entity abstractions
- Live integration for immediate sync
