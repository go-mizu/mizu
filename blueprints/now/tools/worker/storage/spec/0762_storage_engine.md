# 0762 — Event-Sourced Storage Engine

## Status: APPROVED

## Summary

Refactor the storage layer from a dual-write (R2 + D1 files table) model to an
**event-sourced, content-addressable** architecture. Every mutating operation
(write, move, delete) produces a numbered **transaction (tx)** and an **event
record** in D1. File data is stored as content-addressed **blobs** in R2 (keyed
by SHA-256 hash). A Cloudflare driver implements the abstract engine interface.

---

## Motivation

The current system has several weaknesses:

1. **No history** — overwriting a file destroys the previous version permanently.
2. **R2 keys are path-coupled** — `{actor}/{path}`, so a move copies the full
   blob and deletes the old key. Expensive for large files.
3. **No replayability** — the audit_log captures actions but not enough state to
   reconstruct the storage at a past point in time.
4. **No commit messages** — mutations are silent; there's no human-readable
   "why" for a change.
5. **Tight Cloudflare coupling** — every route handler calls `c.env.BUCKET` and
   `c.env.DB` directly. No way to swap the backend.

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    StorageEngine                      │  (engine.ts)
│                                                      │
│  write(actor, path, blob, msg?) → {tx, time, addr}   │
│  move(actor, from, to, msg?)   → {tx, time}         │
│  delete(actor, paths, msg?)    → {tx, time}         │
│  read(actor, path)             → {blob, tx, time}   │
│  list(actor, prefix, opts)     → FileEntry[]         │
│  head(actor, path)             → FileMeta | null     │
│  search(actor, query, opts)    → SearchResult[]      │
│  stats(actor)                  → {files, bytes}      │
│  log(actor, opts?)             → Event[]             │
│  snapshot(actor, path, tx?)    → {blob, tx, time}   │
└──────────────┬───────────────────────────────────────┘
               │ implements
┌──────────────▼───────────────────────────────────────┐
│              CloudflareEngine                         │  (cloudflare.ts)
│                                                      │
│  D1: events table, files table (materialized view)   │
│  R2: content-addressed blobs + GC lifecycle          │
└──────────────────────────────────────────────────────┘
```

---

## Design Decisions (needs discussion)

### Decision 1: Content addressing scheme

**Option A — SHA-256 of the blob content (recommended)**

```
R2 key: blobs/{hex(sha256(content))}
```

- Automatic dedup: uploading the same file twice stores only one blob.
- Move/rename is free — only the event changes, the blob stays.
- Read-after-write is guaranteed since R2 PUT is strongly consistent.
- Used by Git, IPFS, Docker.

**Option B — Random ID per blob version**

```
R2 key: blobs/{ulid}
```

- Simpler — no need to hash on write.
- But no dedup, and hashing a 10 MB file in a Worker takes ~30ms (acceptable).

**Tradeoff:** A requires computing SHA-256 for every write (CPU cost) but saves
storage on duplicate content. B is simpler but wastes R2 space and makes
dedup impossible.

**Recommendation:** A. SHA-256 is fast enough (WASM Web Crypto), dedup is a
real-world win for versioned files, and the content address doubles as an
integrity check.

---

### Decision 2: Event granularity — per-file or per-batch?

**Option A — One tx per mutating API call (recommended)**

A `storage_write` to `docs/a.md` gets tx=5. A bulk `storage_delete` of 3 files
gets a single tx=6 with multiple event rows sharing that tx number.

```sql
-- Single tx, multiple events
INSERT INTO events (tx, actor, action, path, addr, size, msg, ts)
VALUES
  (6, 'alice', 'delete', 'tmp/a.txt', NULL, 0, 'cleanup', 1711000000000),
  (6, 'alice', 'delete', 'tmp/b.txt', NULL, 0, 'cleanup', 1711000000000),
  (6, 'alice', 'delete', 'tmp/c.txt', NULL, 0, 'cleanup', 1711000000000);
```

**Option B — Strictly one event per tx**

Each file deletion in a batch gets its own tx. Simpler ordering but inflates tx
numbers.

**Tradeoff:** A gives atomic batch semantics (a "commit" can touch N files). B
is simpler but loses the concept of a batch.

**Recommendation:** A. It maps naturally to "commit"-style operations and keeps
the event log compact.

---

### Decision 3: Where does the `files` table fit?

**Option A — Keep `files` as a materialized view (recommended)**

The `files` table remains for fast listing, search, and stats. It's updated
transactionally alongside event insertion. But it's *derivable* — in theory,
you can reconstruct it by replaying all events.

**Option B — Drop `files`, query events directly**

Replace `SELECT FROM files` with aggregate queries on events:

```sql
SELECT path, addr, size, ... FROM events
WHERE actor = ? AND path = ?
ORDER BY tx DESC LIMIT 1
```

**Tradeoff:** B is purer event-sourcing but makes listing O(events) instead of
O(files). With D1's SQLite, a query like `WHERE actor=? AND path LIKE 'docs/%'
ORDER BY tx DESC` for every path is too slow at scale.

**Recommendation:** A. Keep `files` as a **read-optimized projection**. Update
it in the same D1 batch as the event insert. Add `tx` and `tx_time` columns to
`files` so reads return version information without joining.

---

### Decision 4: Blob garbage collection strategy

When a file is overwritten (new content address) or deleted, the old blob in R2
may become unreferenced. We need GC.

**Option A — Reference counting**

Add a `ref_count` column to a `blobs` table. Increment on write, decrement on
overwrite/delete. GC when `ref_count = 0`.

**Option B — Mark-and-sweep (recommended)**

Periodically (or probabilistically, like current audit cleanup):
1. Scan `events` table for all unique `addr` values currently live
   (`SELECT DISTINCT addr FROM files`).
2. List R2 blobs and delete any not in the live set.

**Option C — Lazy TTL-based eviction**

Blobs unused for >N days get deleted. Simple but requires tracking last-access.

**Tradeoff:** A is precise but adds write amplification (every write must update
two tables). B is simpler and batched but requires a full scan. C is simplest
but may delete blobs still referenced by old events (breaks replay).

**Recommendation:** B (mark-and-sweep), run as a cron-triggered Scheduled Event
on the Worker. In CloudflareEngine, also keep a `blobs` table in D1 that maps
`addr → {size, ref_count, created_at}` so we can track which blobs are orphaned
without scanning all events. Decrement ref_count on overwrite/delete; a cron job
deletes R2 objects where `ref_count = 0 AND created_at < now - 24h` (grace
period to avoid races).

---

### Decision 5: Event replay depth — full history or windowed?

**Option A — Keep all events forever**

Full audit trail. Can reconstruct state at any point.

**Option B — Retain events for N days, compact older ones (recommended)**

Keep individual events for 90 days. After that, compact into periodic
"snapshots" (a full state dump) and prune old events.

**Option C — Snapshot on every Nth tx**

Every 1000 tx, write a full snapshot to R2. Replay only from last snapshot.

**Tradeoff:** A has unbounded D1 growth. B balances auditability with storage
costs. C is good for replay but adds write cost.

**Recommendation:** B for v1 — 90 days matches current audit_log retention.
Optionally add C later for fast point-in-time recovery.

---

### Decision 6: MCP message support

Every mutating tool call should accept an optional `message` parameter (like a
git commit message). This appears in the event log.

```json
{
  "name": "storage_write",
  "arguments": {
    "path": "docs/readme.md",
    "content": "# Hello",
    "message": "Initial project readme"
  }
}
```

If no message is provided, auto-generate one:
- write → `"write {path}"`
- delete → `"delete {paths.join(', ')}"`
- move → `"move {from} → {to}"`

---

## Schema Changes

### New: `events` table

```sql
CREATE TABLE IF NOT EXISTS events (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  tx      INTEGER NOT NULL,           -- transaction number (per-actor)
  actor   TEXT    NOT NULL,
  action  TEXT    NOT NULL,           -- 'write' | 'move' | 'delete'
  path    TEXT    NOT NULL,           -- affected path
  addr    TEXT,                       -- content address (SHA-256 hex), NULL for delete
  size    INTEGER NOT NULL DEFAULT 0, -- blob size at time of event
  type    TEXT,                       -- MIME type
  meta    TEXT,                       -- JSON: {from?, content_type?, ...}
  msg     TEXT,                       -- commit message
  ts      INTEGER NOT NULL            -- unix millis
);
CREATE INDEX IF NOT EXISTS idx_events_actor_tx ON events(actor, tx);
CREATE INDEX IF NOT EXISTS idx_events_actor_path ON events(actor, path, tx);
```

### New: `blobs` table (tracking, not storage)

```sql
CREATE TABLE IF NOT EXISTS blobs (
  addr       TEXT PRIMARY KEY,        -- SHA-256 hex
  size       INTEGER NOT NULL,
  ref_count  INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL
);
```

### New: `tx_counter` table

```sql
CREATE TABLE IF NOT EXISTS tx_counter (
  actor TEXT PRIMARY KEY,
  next_tx INTEGER NOT NULL DEFAULT 1
);
```

### Modified: `files` table — add version columns

```sql
ALTER TABLE files ADD COLUMN addr    TEXT;     -- current content address
ALTER TABLE files ADD COLUMN tx      INTEGER;  -- last-write tx number
ALTER TABLE files ADD COLUMN tx_time INTEGER;  -- timestamp of last-write tx
```

---

## Engine Interface (engine.ts)

```typescript
// ── Types ────────────────────────────────────────────────────────────

export interface FileEntry {
  name: string;
  type: string;        // MIME or 'directory'
  size?: number;
  updated_at?: number;
  tx?: number;
  tx_time?: number;
}

export interface FileMeta {
  path: string;
  name: string;
  size: number;
  type: string;
  addr: string | null;
  tx: number;
  tx_time: number;
}

export interface WriteResult {
  tx: number;
  time: number;
  addr: string;
  size: number;
}

export interface MutationResult {
  tx: number;
  time: number;
}

export interface ReadResult {
  body: ReadableStream | ArrayBuffer;
  meta: FileMeta;
}

export interface SearchResult {
  path: string;
  name: string;
  size: number;
  type: string;
  tx: number;
}

export interface StorageEvent {
  tx: number;
  action: 'write' | 'move' | 'delete';
  path: string;
  addr: string | null;
  size: number;
  msg: string | null;
  ts: number;
}

export interface ListOptions {
  prefix?: string;
  limit?: number;
  offset?: number;
}

export interface LogOptions {
  path?: string;      // filter by path
  since_tx?: number;  // events after this tx
  limit?: number;
}

// ── Abstract engine ──────────────────────────────────────────────────

export interface StorageEngine {
  /** Write a file. Returns tx, content address, and timestamp. */
  write(actor: string, path: string, body: ArrayBuffer | ReadableStream,
        contentType: string, msg?: string): Promise<WriteResult>;

  /** Move/rename a file. */
  move(actor: string, from: string, to: string,
       msg?: string): Promise<MutationResult>;

  /** Delete file(s). Paths ending with / delete recursively. */
  delete(actor: string, paths: string[],
         msg?: string): Promise<MutationResult & { deleted: number }>;

  /** Read a file's content + metadata. */
  read(actor: string, path: string): Promise<ReadResult | null>;

  /** Check if a file exists and get metadata. */
  head(actor: string, path: string): Promise<FileMeta | null>;

  /** List files/folders at a prefix. */
  list(actor: string, opts?: ListOptions): Promise<{
    entries: FileEntry[];
    truncated: boolean;
  }>;

  /** Search files by name. */
  search(actor: string, query: string, opts?: {
    limit?: number;
    prefix?: string;
  }): Promise<SearchResult[]>;

  /** Storage usage stats. */
  stats(actor: string): Promise<{ files: number; bytes: number }>;

  /** Get event log. */
  log(actor: string, opts?: LogOptions): Promise<StorageEvent[]>;

  /** Read a file at a specific past tx. */
  snapshot(actor: string, path: string, tx: number): Promise<ReadResult | null>;

  /** Generate a presigned download URL. */
  presignRead(actor: string, path: string,
              expiresIn?: number): Promise<string | null>;

  /** Generate a presigned upload URL. */
  presignUpload(actor: string, path: string, contentType: string,
                expiresIn?: number): Promise<string>;
}
```

---

## CloudflareEngine Implementation Sketch

### write()

```
1. Hash the body → addr = hex(SHA-256(body))
2. Check blobs table: if addr exists, increment ref_count; else PUT to R2
3. Allocate next tx: UPDATE tx_counter SET next_tx = next_tx + 1 ... RETURNING
4. If overwriting, read old addr from files table, decrement old blob ref_count
5. D1 batch:
   a. INSERT INTO events (tx, actor, action, path, addr, size, type, msg, ts)
   b. UPSERT INTO files  (owner, path, name, size, type, addr, tx, tx_time, updated_at)
   c. UPSERT INTO blobs  (addr, size, ref_count, created_at)
   d. UPDATE blobs SET ref_count = ref_count - 1 WHERE addr = old_addr (if overwrite)
6. Return { tx, time, addr, size }
```

### move()

```
1. Read current file entry → get addr, size, type
2. Allocate next tx
3. D1 batch:
   a. INSERT INTO events (tx, action='move', path=to, meta={from}, ...)
   b. DELETE FROM files WHERE path = from
   c. UPSERT INTO files (path=to, addr=same, tx, ...)
4. NO R2 operation needed — blob hasn't changed
```

This is a **major win** over the current system, which copies the full blob
for every move.

### delete()

```
1. For each path, read addr from files table
2. Allocate next tx
3. D1 batch:
   a. INSERT INTO events (tx, action='delete', ...) for each path
   b. DELETE FROM files for each path
   c. UPDATE blobs SET ref_count = ref_count - 1 for each addr
4. R2 blobs are NOT immediately deleted — GC handles it
```

### read()

```
1. SELECT addr, tx, tx_time, ... FROM files WHERE owner=? AND path=?
2. GET from R2: blobs/{addr}
3. Return stream + meta (including tx, tx_time)
```

### snapshot(actor, path, tx)

```
1. SELECT addr FROM events WHERE actor=? AND path=? AND tx<=? AND action='write'
   ORDER BY tx DESC LIMIT 1
2. If found, GET from R2: blobs/{addr}
3. If the blob has been GC'd (ref_count=0 and deleted from R2), return error
```

---

## API Response Changes

All read/list endpoints now include version info:

```json
// GET /files
{
  "prefix": "docs/",
  "entries": [
    { "name": "readme.md", "type": "text/markdown", "size": 1234,
      "updated_at": 1711000000000, "tx": 42, "tx_time": 1711000000000 }
  ]
}

// POST /files/uploads/complete (and storage_write)
{
  "path": "docs/readme.md",
  "tx": 43,
  "time": 1711000001000,
  "addr": "a1b2c3d4...",
  "size": 1234
}
```

### New endpoint: GET /files/log

```json
// GET /files/log?path=docs/readme.md&limit=20
{
  "events": [
    { "tx": 43, "action": "write", "path": "docs/readme.md",
      "addr": "a1b2c3d4...", "size": 1234, "msg": "update readme",
      "ts": 1711000001000 },
    { "tx": 12, "action": "write", "path": "docs/readme.md",
      "addr": "e5f6a7b8...", "size": 890, "msg": null,
      "ts": 1710900000000 }
  ]
}
```

---

## Migration Strategy

1. **Add new tables** (`events`, `blobs`, `tx_counter`) and new columns on
   `files` (`addr`, `tx`, `tx_time`) via a D1 migration.
2. **Backfill**: For every existing file in `files`, compute its SHA-256 from
   R2, set `addr`, insert a synthetic `tx=0` event, and copy the R2 object to
   `blobs/{addr}`. Run as a one-time Worker script.
3. **Dual-write period**: Keep old R2 keys (`{actor}/{path}`) alive during
   migration. New writes go to both old key and content-addressed key.
4. **Cutover**: Once backfill is verified, switch reads to content-addressed
   keys. Delete old R2 keys.

---

## Performance Considerations

| Operation   | Current cost          | New cost                        | Notes                              |
|-------------|----------------------|---------------------------------|------------------------------------|
| Write       | 1 R2 PUT + 1 D1 UPSERT | 1 R2 PUT + 1 D1 batch (3-4 stmts) | Slight D1 overhead for events    |
| Move        | 1 R2 GET + 1 R2 PUT + 1 R2 DELETE + 1 D1 batch | **0 R2 ops** + 1 D1 batch | Major improvement             |
| Delete      | 1 R2 DELETE + 1 D1 DELETE | 1 D1 batch (no R2)             | R2 cleanup deferred to GC         |
| Read        | 1 R2 GET (by path)   | 1 D1 SELECT + 1 R2 GET (by addr) | One extra D1 query               |
| List        | 1 D1 SELECT          | 1 D1 SELECT (same)              | Same — `files` table unchanged     |

**Net impact:** Move and delete get dramatically cheaper. Read adds one D1 query
(~1ms on Cloudflare edge). Write adds ~3ms for SHA-256 hashing + extra D1 rows.

---

## Resolved Decisions

1. **Tx counter: per-actor.** Each actor has independent tx numbering. Simpler,
   no contention, matches the per-actor storage model.

2. **Content hash NOT exposed in public API.** `addr` is internal only. API
   responses include `tx` and `tx_time` but not the content address.

3. **No public snapshot endpoint for now.** `snapshot()` stays internal-only.
   Can add a `GET /files/{path}?at_tx=N` later if needed.

4. **R2 blob key format: `blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}`.**
   Per-actor sharding makes account deletion trivial (delete prefix
   `blobs/{actor}/`). Minor duplication across actors is acceptable for
   simpler data lifecycle management.

5. **`audit_log` renamed to `audit`, kept separate from `events`.** `audit` is
   for audit purposes (reads, auth, login, etc.). `events` is for data
   mutations only (write, move, delete). No overlap in purpose.

---

## Deliverables

- [x] `src/storage/engine.ts` — interface + types
- [x] `src/storage/cloudflare.ts` — Cloudflare D1+R2 implementation
- [x] D1 migration: `migrations/0769_events.sql`
- [x] Refactor `routes/files-v2.ts` to use engine
- [x] Refactor `routes/mcp.ts` to use engine + add `message` param
- [x] Refactor `routes/share.ts` to use engine
- [x] Add `GET /files/log` endpoint
- [x] Add `message` param to MCP write/move/delete tools
- [x] Update all responses to include `tx`, `tx_time`
- [x] Rename `audit_log` → `audit`
- [x] Engine injected via middleware in `index.ts`
- [ ] GC cron job for orphaned blobs (future: Scheduled Event)
- [ ] Backfill migration script (one-time, post-deploy)
