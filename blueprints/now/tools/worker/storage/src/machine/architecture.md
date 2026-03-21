# Storage Architecture

> Purpose-built for AI agents and humans to collaborate on files.

Storage is an edge-first file storage platform. Every design decision, from content addressing to event sourcing to the dual identity model, serves both human users and AI agents through one unified API.

## System Overview

```
Client (Human or Agent)
     │
     │  HTTPS
     ▼
┌─────────────────────────────────────────────┐
│              Edge Runtime                    │
│  ┌────────┐  ┌──────────┐  ┌─────────────┐ │
│  │  Auth   │  │  Router  │  │  MCP Server │ │
│  │  Layer  │  │  (Hono)  │  │  (Tools)    │ │
│  └────────┘  └──────────┘  └─────────────┘ │
└──────────────────┬──────────────────────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
┌───────────────┐    ┌───────────────┐
│  Meta Plane   │    │ Object Storage│
│               │    │   (Blobs)     │
│  files (inode)│    │               │
│  events       │    │  blobs/       │
│  blobs (ref)  │    │   {actor}/    │
│  sessions     │    │    {aa}/{bb}/ │
│  tx_counter   │    │     {hash}    │
└───────────────┘    └───────────────┘
```

### Layers

| Layer | Technology | Role |
|-------|-----------|------|
| Edge Runtime | TypeScript on V8 isolates | Request handling, auth, routing, MCP |
| Meta Plane | Structured metadata store | File index, events, sessions, blob refs |
| Blob Store | S3-compatible Object Storage | File content, content-addressed by SHA-256 |
| Protocol | REST + MCP | Unified access for humans and AI agents |

## Meta Plane

The Meta Plane stores all structured data: file entries, events, sessions, blob references, and transaction counters. It separates file identity from file content, similar to how UNIX inodes separate a filename from disk blocks.

### Data Model

```
files table
┌──────┬──────────────────┬──────────────┬──────────┬───────┐
│ id   │ path             │ addr (hash)  │ size     │ tx    │
├──────┼──────────────────┼──────────────┼──────────┼───────┤
│ 1    │ report.pdf       │ a1b2c3d4...  │ 204800   │ 1     │
│ 2    │ data/config.json │ d4e5f6a7...  │ 512      │ 2     │
│ 3    │ backup/report.pdf│ a1b2c3d4...  │ 204800   │ 6     │
└──────┴──────────────────┴──────────────┴──────────┴───────┘

blobs table (ref counting)
┌──────────────┬───────────┐
│ addr (hash)  │ ref_count │
├──────────────┼───────────┤
│ a1b2c3d4...  │ 2         │  ← two files, one blob
│ d4e5f6a7...  │ 1         │
└──────────────┴───────────┘
```

### Why This Enables Fast Operations

- **Move / Rename**: Update the `path` column. Zero blob copies. O(1).
- **Deduplication**: Same `addr` means same blob. Upload once, reference many times.
- **Delete**: Decrement `ref_count`. Only delete the blob when it reaches zero.
- **Integrity**: The hash IS the address. Corruption is self-evident.

## Request Lifecycle

### Upload Flow

```
1. Client ──POST /files/uploads──> Edge (auth + validate)
2. Edge ──returns presigned URL──> Client
3. Client ──PUT bytes──> Object Storage (direct, no proxy)
4. Client ──POST /files/uploads/complete──> Edge (confirm + index)
```

- File bytes never touch the API server
- Presigned URL generated in under 50ms
- Confirmation writes to Meta Plane: file entry + event + blob ref

### Download Flow

```
1. Client ──GET /files/{path}──> Edge (auth + lookup)
2. Edge ──302 redirect──> Presigned Object Storage URL
3. Client ──GET──> Object Storage (direct download)
```

- Sub-50ms metadata lookup
- Zero bandwidth through API server
- No egress fees from Object Storage

## Content Addressing

Every file is stored by the SHA-256 hash of its content:

```
Path:    blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
Example: blobs/alice/a1/b2/a1b2c3d4e5f67890...
```

### Properties

- **Automatic deduplication**: same content = same blob, always
- **Free integrity verification**: the hash IS the identifier
- **Zero-cost rename and move**: only the Meta Plane path changes
- **Collision risk ~2^-128**: effectively zero

Upload the same file twice, one blob stored. Rename a file, zero bytes copied.

## Event Architecture

Every mutation in the system produces an immutable event. This is not just logging. It is the primary source of truth for what happened, when, and by whom.

### Event Log

```
┌─────┬────────┬──────────────┬──────────────┬───────────┐
│ tx  │ action │ path         │ addr         │ timestamp │
├─────┼────────┼──────────────┼──────────────┼───────────┤
│ 1   │ put    │ readme.md    │ a1b2c3...    │ 1711065600│
│ 2   │ put    │ data/cfg.json│ d4e5f6...    │ 1711065601│
│ 3   │ move   │ readme.md    │ a1b2c3...    │ 1711065602│
│ 4   │ delete │ old/file.txt │ (null)       │ 1711065603│
│ 5   │ put    │ data/cfg.json│ f7g8h9...    │ 1711065604│
└─────┴────────┴──────────────┴──────────────┴───────────┘
```

### Properties

- **Append-only**: events are never updated or deleted (until GC compaction)
- **Per-actor isolation**: each actor has its own transaction counter
- **Dense sequence**: tx numbers have no gaps, ordered by insertion time
- **Monotonic**: tx counter never resets

### Smart Versioning

Every `put` to the same path creates a new event with a new `addr`. The previous version is still in the event log. You can reconstruct the full history of any file by reading events for that path.

### Replayable

Given the event log, you can reconstruct the complete state of any actor's storage at any point in time. Start from tx 0, apply events in order, and you get the exact file tree.

### Auditable

Every event records who did what, when. No action goes untracked. For AI agents operating autonomously, this provides complete accountability.

### Realtime Change Tracking

Agents can poll for events since their last known `tx` number. No need to list all files and diff. Just ask "what changed since tx 42?" and get exactly the new events.

### Why This Matters for AI Agents

- Incremental sync without full directory scans
- No race conditions between concurrent agent operations
- Full audit trail of every action an agent took
- Deterministic replay for debugging agent behavior
- Version history for every file, built in

## Dual Identity Model

Storage treats humans and AI agents as equal participants. Both are "actors" with the same API surface and permissions model.

### Human Authentication

```
1. Request magic link via email
2. Click link to activate session
3. Session cookie set automatically
4. Use browser, API, or CLI
```

### Agent Authentication

```
1. Register Ed25519 public key
2. POST /auth/challenge to receive nonce
3. Sign nonce with private key
4. POST /auth/verify to get bearer token
```

### Same API, Different Auth

| Capability | Human | Agent |
|-----------|-------|-------|
| Upload files | Yes | Yes |
| Download files | Yes | Yes |
| Share files | Yes | Yes |
| Search files | Yes | Yes |
| Create API keys | Yes | Yes |
| MCP tools | Yes | Yes |

No special "service account" or "bot mode". An agent is just another actor.

## Edge-First Design

The entire application runs at the edge, as close to the client as possible.

### Performance

| Operation | Latency |
|-----------|---------|
| Auth check | < 1ms |
| File metadata lookup | < 50ms |
| Presigned URL generation | < 50ms |
| List files | < 50ms |
| Search | < 100ms |

### Why Edge Matters for AI Agents

- AI agents in cloud functions have cold starts, so every millisecond counts
- MCP tool calls add latency on top of LLM inference, so the API must be fast
- Global distribution means agents in any region get consistent performance
- No single region bottleneck for multi-agent coordination

## MCP: Native AI Integration

Storage implements the Model Context Protocol as a first-class interface, not an adapter bolted on top.

### Tools

| Tool | Maps to | Description |
|------|---------|-------------|
| `storage_read` | `GET /files/{path}` | Read file contents |
| `storage_write` | `POST /files/uploads` + confirm | Write or overwrite a file |
| `storage_list` | `GET /files?prefix=` | List files in a folder |
| `storage_search` | `GET /files/search?q=` | Search files by name |
| `storage_share` | `POST /files/share` | Create a temporary public link |
| `storage_move` | `POST /files/move` | Move or rename a file |
| `storage_delete` | `DELETE /files/{path}` | Delete a file |
| `storage_stats` | `GET /files/stats` | Get storage usage |

### MCP Architecture

```
AI Client (Claude, ChatGPT, custom)
     │
     │  MCP Protocol (SSE transport)
     ▼
┌─────────────────────────────────┐
│         MCP Server              │
│  ┌──────────┐  ┌─────────────┐ │
│  │  OAuth    │  │  Tool       │ │
│  │  Auth     │  │  Handlers   │ │
│  └──────────┘  └──────────────┘│
└────────────────┬────────────────┘
                 │  Same storage engine
                 ▼
          ┌─────────────┐
          │  Storage    │
          │  Engine     │
          └─────────────┘
```

MCP tools and REST API share the same storage engine. A file uploaded via REST is immediately visible via MCP, and vice versa.

## Technology Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| Runtime | Edge Workers (V8 isolates) | Sub-5ms cold start, global distribution |
| Framework | Hono + OpenAPI | Type-safe routes, auto-generated API docs |
| Meta Plane | Structured metadata store | Fast reads, strong consistency, zero config |
| Blob Storage | S3-compatible Object Store | Durable, zero egress fees, presigned URLs |
| Auth | Ed25519 + Magic Links | No passwords, no shared secrets |
| API Keys | Scoped, prefixed (`sk_*`) | Path restrictions, 90-day TTL |
| Protocol | REST + MCP | Universal access, AI-native |
| Schema | OpenAPI 3.1 | Auto-generated from route definitions |
| Validation | Zod | Runtime type safety, schema generation |

## Why This Architecture Suits AI Agents

1. **No SDK required**: plain HTTP. Any agent runtime can call it.
2. **Deterministic responses**: JSON with consistent schemas. No HTML parsing.
3. **MCP native**: tools map directly to storage operations.
4. **Event sourcing**: agents can sync incrementally, not poll everything.
5. **Content addressing**: deduplication means agents don't waste storage re-uploading.
6. **Sub-50ms latency**: fast enough for tool calls inside LLM inference loops.
7. **Dual identity**: agents are first-class actors, not hacks on top of user accounts.
8. **Scoped keys**: agents get exactly the permissions they need, nothing more.
9. **Audit trail**: every agent action is logged and traceable.
10. **Zero egress**: agents can read files as often as they need without cost anxiety.

## Links

- [Developer Guide](https://storage.liteio.dev/developers)
- [API Reference](https://storage.liteio.dev/api)
- [CLI Documentation](https://storage.liteio.dev/cli)
- [Pricing](https://storage.liteio.dev/pricing)
