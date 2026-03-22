# 0763 — Storage Engine Architecture Deep Dive

This document explains every technical decision, trade-off, consistency
guarantee, cost implication, and scaling boundary of the event-sourced storage
engine deployed at `storage.liteio.dev`. It is the canonical reference for
anyone modifying the engine or reasoning about its behavior under load.

---

## 1. System Model

The storage engine manages per-actor namespaces of files. An **actor** is a
human user or an agent — each has an isolated view of storage, its own
transaction counter, and its own blob namespace in R2. There is no cross-actor
file sharing at the engine level (share links are built above the engine in the
route layer).

### Data flow

```
Client request
     │
     ▼
┌─────────────┐
│  Hono route │  files-v2.ts, mcp.ts, share.ts
│  handlers   │  path validation, auth, prefix scoping
└──────┬──────┘
       │  c.get("engine")
       ▼
┌─────────────┐
│  Storage    │  engine.ts interface
│  Engine     │  actor + path + body → {tx, time}
└──────┬──────┘
       │  implements
       ▼
┌─────────────────────────────────────────────┐
│            CloudflareEngine                  │  cloudflare.ts
│                                              │
│  ┌──────────┐    ┌───────────┐              │
│  │    D1    │    │     R2    │              │
│  │ (SQLite) │    │  (S3-like │              │
│  │          │    │   object  │              │
│  │ events   │    │   store)  │              │
│  │ files    │    │           │              │
│  │ blobs    │    │ blobs/    │              │
│  │ tx_count │    │  {actor}/ │              │
│  └──────────┘    │   {aa}/   │              │
│                  │    {bb}/  │              │
│                  │     {hash}│              │
│                  └───────────┘              │
└─────────────────────────────────────────────┘
```

### Invariants

1. Every file in `files` references at most one `addr` (content address).
2. Every `addr` referenced by a live file has `ref_count >= 1` in `blobs`.
3. Every `addr` referenced by a live file has a corresponding object in R2.
4. The `events` table is append-only. No row is ever updated or deleted
   (until GC compaction, which only deletes events older than the retention
   window).
5. `tx_counter.next_tx` is monotonically increasing per actor and never
   resets.
6. For any actor, `tx` values in `events` are dense (no gaps) and ordered by
   insertion time.

---

## 2. Content Addressing

### Scheme

Every blob stored in R2 is keyed by the SHA-256 hash of its content:

```
blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
```

Example: actor `alice`, hash `a1b2c3d4e5f6...`:

```
blobs/alice/a1/b2/a1b2c3d4e5f6...
```

### Why SHA-256

| Property         | SHA-256          | Random ID (ULID)  |
|------------------|------------------|--------------------|
| Deduplication    | Automatic        | Impossible         |
| Integrity check  | Free (hash = ID) | Separate checksum  |
| Move cost        | Zero R2 ops      | Zero R2 ops        |
| Write CPU cost   | ~3ms for 10 MB   | Negligible         |
| Collision risk   | ~2^-128          | N/A                |

SHA-256 collision probability is effectively zero. Two distinct files producing
the same hash requires finding a collision in a 256-bit space. The birthday
bound is 2^128 operations. At 10 billion files per second, it would take
10^21 years.

### Why per-actor blob namespacing

Blobs are namespaced to actors: the same file content uploaded by two different
actors produces two R2 objects with the same hash but different key prefixes.

This is intentional. Trade-offs:

| Factor              | Cross-actor dedup       | Per-actor dedup           |
|---------------------|-------------------------|---------------------------|
| Storage efficiency  | Optimal (one copy)      | Slight waste (~1.5x at most) |
| Account deletion    | Complex (must check other actors' refs) | Trivial: `DELETE blobs/{actor}/*` |
| GDPR compliance     | Hard (shared blobs)     | Clean (isolated data)     |
| Blast radius        | A bug in one actor's GC can orphan another's data | Isolated |
| Implementation      | Global ref_count coordination | Simple per-actor ref_count |

We chose per-actor. The duplication cost is negligible because cross-actor
content overlap is rare in practice (each user stores their own files). And the
operational simplicity of "delete everything under `blobs/alice/`" is
invaluable for account deletion, GDPR right-to-erasure, and incident response.

### Directory-style sharding: `{hash[0:2]}/{hash[2:4]}/`

R2 (and S3) have no "directories" — the key is flat. But listing objects with a
common prefix uses a B-tree scan on the key. Sharding by the first 4 hex
characters creates 65,536 pseudo-directories per actor, ensuring:

- No single prefix accumulates more than `N / 65536` objects.
- `ListObjects` with a prefix scans a narrow key range.
- The shard prefix is deterministic from the hash, so no lookup table is needed.

At 1 million files per actor, each shard has ~15 objects. At 10 million, ~152.
This keeps list operations fast regardless of total object count.

---

## 3. Transaction Model

### Per-actor monotonic counter

Each actor has an independent transaction counter stored in
`tx_counter(actor, next_tx)`. Allocation is a two-step process:

```sql
-- Step 1: Atomic increment
INSERT INTO tx_counter (actor, next_tx) VALUES (?, 1)
  ON CONFLICT (actor) DO UPDATE SET next_tx = next_tx + 1;

-- Step 2: Read the allocated value
SELECT next_tx FROM tx_counter WHERE actor = ?;
```

This works because D1 (SQLite) executes each statement serially per database.
There is no concurrent write contention on a single actor's row within a single
Worker invocation. Cross-isolate races are mitigated by D1's per-database write
serialization.

### Why not RETURNING?

D1 does not support `INSERT ... RETURNING`. The two-step approach is required.
The gap between step 1 and step 2 is not a correctness issue because:

1. D1 is a single-writer database — concurrent writes to the same row are
   serialized.
2. Even if two requests race, each will see a different `next_tx` value
   because the increment happened before the read.

### Why per-actor, not global?

| Factor            | Per-actor tx       | Global tx            |
|-------------------|--------------------|----------------------|
| Contention        | None (actor isolated) | Hot row under load |
| Comparability     | tx=5 for alice ≠ tx=5 for bob | Globally ordered |
| Counter overflow  | Effectively infinite (per-actor scale) | Shared ceiling |
| Implementation    | Simple upsert      | Needs distributed lock or Durable Object |
| D1 write limits   | 1 write per mutation | Every mutation touches the same row |

Per-actor tx is the right choice for a multi-tenant storage service. Global tx
would create a serialization bottleneck: every write from every actor would
contend on a single row, limiting throughput to D1's single-writer capacity
(~100 writes/second). Per-actor tx distributes load across rows.

The downside — no global ordering — doesn't matter. Actors don't need to
compare their tx numbers with each other. The event timestamp (`ts`) provides
wall-clock ordering when cross-actor comparison is needed (e.g., admin audit
views).

### Batch semantics

A single API call (e.g., `storage_delete` with 3 paths) allocates one tx and
produces multiple event rows sharing that tx number. This gives "commit"-style
atomicity at the D1 level:

```sql
-- tx=6, three deletions
INSERT INTO events (tx, actor, action, path, ...) VALUES (6, 'alice', 'delete', 'a.txt', ...);
INSERT INTO events (tx, actor, action, path, ...) VALUES (6, 'alice', 'delete', 'b.txt', ...);
INSERT INTO events (tx, actor, action, path, ...) VALUES (6, 'alice', 'delete', 'c.txt', ...);
DELETE FROM files WHERE owner = 'alice' AND path IN ('a.txt', 'b.txt', 'c.txt');
```

All statements run in a single `db.batch()` call. D1 executes batched
statements in a single SQLite transaction — if any statement fails, the entire
batch rolls back. This ensures the events table and files projection are
always consistent.

---

## 4. The Dual-Table Design: `events` + `files`

### Why not pure event-sourcing?

In a pure event-sourced system, current state is derived by replaying events.
For storage, "list all files in `docs/`" would require:

```sql
-- For each unique path, find the latest event
SELECT path, addr, size, type FROM events
WHERE actor = ? AND path LIKE 'docs/%'
GROUP BY path
HAVING MAX(tx)
AND action != 'delete';
```

This is O(total events matching the prefix), not O(current files). With 10,000
events and 50 current files, the query scans 10,000 rows to return 50. D1 has
no materialized views, no streaming aggregation — it's raw SQLite. This
approach doesn't scale.

### The `files` table as a read-optimized projection

The `files` table is a denormalized, always-current view of what files exist.
It is updated transactionally alongside event insertion in the same `db.batch()`
call. It stores:

```
files(owner, path, name, size, type, addr, tx, tx_time, updated_at)
```

Read operations (`list`, `search`, `stats`, `head`) query `files` only. They
never touch `events`. This means:

- **list** is O(matching files), not O(events).
- **search** uses LIKE on the name/path columns, hitting the existing index.
- **stats** uses COUNT/SUM on the owner-filtered rows.
- **head** is a single primary key lookup.

The `events` table is only queried by:
- `log()` — the event history endpoint.
- Future `snapshot()` — read-at-tx (not yet exposed).

### Consistency between tables

Both `events` and `files` are updated in the same `db.batch()` call. D1
executes batched statements as a single SQLite transaction. If the batch fails
(e.g., D1 transient error), neither the event nor the files update is committed.
This guarantees:

- **No orphaned events:** Every event has a corresponding files update.
- **No stale files:** The files table always reflects the latest committed event.
- **Crash safety:** A Worker crash mid-handler produces either the complete
  transaction or nothing (SQLite's atomic commit).

The one exception is R2 writes. R2 PUT happens *before* the D1 batch (we need
the blob in R2 first). If the D1 batch fails after the R2 PUT, we have an
orphaned blob in R2. This is safe: the blob is content-addressed, so it's just
unused storage. The GC cron will eventually clean it up (ref_count = 0 or not
referenced by any file row).

---

## 5. Write Path in Detail

```
write(actor, path, body, contentType, msg?)
  │
  ├─ 1. Buffer body into ArrayBuffer (if ReadableStream)
  │     Cost: memory = O(file size), capped at 10 MB for MCP writes
  │
  ├─ 2. SHA-256 hash the buffer
  │     Cost: ~3ms per 10 MB (Web Crypto API, hardware-accelerated)
  │     Output: 64 hex character address
  │
  ├─ 3. Check R2 for existing blob: HEAD blobs/{actor}/{aa}/{bb}/{hash}
  │     Cost: 1 R2 HEAD (Class A op, ~$0.0000044)
  │     If exists: skip PUT (dedup hit)
  │     If not: PUT the blob to R2
  │     Cost: 1 R2 PUT (Class A op, ~$0.0000044) + storage
  │
  ├─ 4. Read current file entry from D1 (for old addr)
  │     Cost: 1 D1 read (~0.2ms, 1 row read)
  │
  ├─ 5. Allocate next tx
  │     Cost: 1 D1 write (upsert) + 1 D1 read
  │
  ├─ 6. D1 batch (3-5 statements, single transaction):
  │     a. INSERT INTO events (...)
  │     b. UPSERT INTO files (...)
  │     c. UPSERT INTO blobs (... ref_count + 1)
  │     d. UPDATE blobs SET ref_count - 1 WHERE addr = old_addr (if overwrite)
  │     e. UPDATE blobs SET ref_count - 1 WHERE addr = new_addr (if same-addr overwrite)
  │     Cost: 1 D1 batch (~1-3ms, 3-5 row writes)
  │
  └─ Return: { tx, time, size }
```

### Total cost per write

| Resource      | New file    | Overwrite (different content) | Overwrite (same content) |
|---------------|-------------|-------------------------------|--------------------------|
| R2 HEAD       | 1           | 1                             | 1                        |
| R2 PUT        | 1           | 1 (or 0 if dedup)            | 0 (dedup hit)            |
| D1 reads      | 2           | 2                             | 2                        |
| D1 writes     | 3 statements| 4 statements                  | 4 statements             |
| SHA-256       | 1           | 1                             | 1                        |

Compared to the old system (1 R2 PUT + 1 D1 UPSERT), the new write path adds
1 R2 HEAD + 1-3 D1 writes + SHA-256 hash. At D1's pricing ($0.75 per million
row writes), the extra D1 cost is ~$0.000002 per write. The R2 HEAD is
~$0.0000044. Total overhead: less than a ten-thousandth of a cent per write.

### Same-content overwrite (idempotent write)

When a file is overwritten with identical content:
1. SHA-256 produces the same addr.
2. R2 HEAD finds the blob already exists — skip PUT.
3. The blobs upsert increments ref_count (new ref), then the same-addr
   correction decrements it (old ref released). Net: ref_count unchanged.
4. Files projection updated with new tx, tx_time.
5. An event is still recorded (the write happened, even if content didn't
   change).

This means: even an idempotent write produces a new tx. This is intentional —
the event log records "user wrote to this path at this time", regardless of
whether the bytes changed. The caller explicitly requested a write; we record
it. For conditional writes (only write if content differs), the caller can
compare sizes or use `head()` first.

---

## 6. Move Path in Detail

```
move(actor, from, to, msg?)
  │
  ├─ 1. Read source file entry from D1
  │     Cost: 1 D1 read
  │     Get: addr, size, type
  │     If not found: throw error
  │
  ├─ 2. Allocate next tx
  │     Cost: 1 D1 write + 1 D1 read
  │
  ├─ 3. D1 batch (3 statements):
  │     a. INSERT INTO events (action='move', path=to, meta={"from": from})
  │     b. DELETE FROM files WHERE path = from
  │     c. UPSERT INTO files (path=to, addr=SAME, tx, ...)
  │     Cost: 1 D1 batch (~1ms)
  │
  ├─ 4. NO R2 operations
  │
  └─ Return: { tx, time }
```

### Why move is free

The old system had to:
1. `BUCKET.get(actor/from)` — download the full blob.
2. `BUCKET.put(actor/to, body)` — re-upload it under the new key.
3. `BUCKET.delete(actor/from)` — delete the old key.

For a 1 GB file, this meant 1 GB download + 1 GB upload + 1 delete. On R2,
that's $0.36/GB egress within the same region (actually free for same-bucket
copy, but still 1 Class A GET + 1 Class A PUT + 1 Class A DELETE + the data
transfer time).

With content addressing, a move changes only the D1 metadata. The blob stays
at `blobs/{actor}/{hash}` and both old and new path point to the same addr.
The cost drops from O(file_size) to O(1) — a fixed ~1ms D1 batch regardless
of file size. Renaming a 1 GB file is exactly as fast as renaming a 1 byte
file.

### Move event metadata

The move event stores `meta: {"from": "old/path.md"}` as JSON. This allows
reconstructing the full rename history from the event log. The event's `path`
field is the destination (new path), and `meta.from` is the source (old path).

---

## 7. Delete Path in Detail

```
delete(actor, paths, msg?)
  │
  ├─ For each path:
  │   ├─ If path ends with "/":
  │   │   Query all files with prefix → get addrs
  │   │   Add event row per file, ref_count decrement per addr
  │   │   Add DELETE FROM files WHERE path LIKE 'prefix%'
  │   └─ Else:
  │       Query single file → get addr
  │       Add event row, ref_count decrement
  │       Add DELETE FROM files WHERE path = ?
  │
  ├─ D1 batch: all events + file deletes + blob decrements
  │
  ├─ NO R2 deletions (deferred to GC)
  │
  └─ Return: { tx, time, deleted }
```

### Why R2 deletions are deferred

Immediately deleting the R2 blob on file deletion has problems:

1. **Shared references.** Two files at different paths might point to the same
   addr (content dedup). Deleting the blob when one file is deleted would
   corrupt the other.
2. **Replay window.** If we want to support reading historical file versions
   from the event log, the blob must exist even after the file entry is deleted.
3. **Race conditions.** A concurrent write might be uploading a blob with the
   same addr at the same time a delete is running.

Deferred deletion via the `blobs` ref_count table solves all three. The GC cron
only deletes blobs where `ref_count = 0 AND created_at < now - 24h`. The 24h
grace period covers:
- In-flight writes that haven't committed their D1 batch yet.
- Retry windows for failed writes.
- Admin investigation of recent deletions.

---

## 8. Read Path in Detail

```
read(actor, path)
  │
  ├─ 1. SELECT addr, tx, tx_time, ... FROM files WHERE owner=? AND path=?
  │     Cost: 1 D1 read (~0.2ms, primary key lookup)
  │
  ├─ 2. If addr is not null (content-addressed):
  │     GET from R2: blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
  │     If R2 returns null (blob missing): fall back to legacy path
  │
  ├─ 3. If addr is null (pre-migration file):
  │     GET from R2: {actor}/{path}   (legacy key format)
  │
  └─ Return: { body: ReadableStream, meta: { path, name, size, type, tx, tx_time } }
```

### Read cost: old vs new

| System | D1 reads | R2 GETs | Notes                      |
|--------|----------|---------|----------------------------|
| Old    | 0        | 1       | Direct R2 GET by path key  |
| New    | 1        | 1       | D1 lookup for addr, then R2 GET |

The new system adds one D1 read per file read. D1 reads from the nearest edge
location are ~0.2ms. The trade-off:

- **+0.2ms latency** on every read.
- **Enables content addressing** (all the move/dedup/versioning benefits).
- **D1 cost:** $0.75 per 5 million reads = ~$0.00000015 per read.

For download-heavy workloads (e.g., CDN-like access patterns), the extra D1
hop might matter. Mitigation: the `presignRead()` path generates a presigned
URL pointing directly to the R2 blob. Once the client has the URL, subsequent
fetches bypass both the Worker and D1 entirely — the browser/client hits R2
directly. The presigned URL lifetime is 1 hour, so for repeated access to the
same file, the D1 overhead is amortized to once per hour.

### Legacy fallback

Pre-migration files have `addr = NULL` in the files table. For these, the
engine falls back to the old key format: `{actor}/{path}`. This ensures zero
downtime during migration — existing files continue to work immediately after
deploying the new engine, before the backfill script runs.

The fallback also handles a safety case: if a content-addressed blob is
missing from R2 (e.g., GC deleted it prematurely, or R2 had a transient error),
the engine tries the legacy key. This is defense in depth — it should never
happen in practice.

---

## 9. Presigned Upload Flow

Direct file uploads bypass the Worker for the data transfer. The flow:

```
Client                     Worker                    R2
  │                          │                        │
  ├── POST /files/uploads ──►│                        │
  │   {path, content_type}   │                        │
  │                          ├── presign PUT URL ────►│
  │◄── {url, expires_in} ───┤                        │
  │                          │                        │
  ├── PUT <presigned URL> ──────────────────────────►│
  │   (data goes direct to R2, bypasses Worker)       │
  │◄── 200 OK ──────────────────────────────────────┤
  │                          │                        │
  ├── POST /uploads/complete►│                        │
  │   {path, message}        │                        │
  │                          ├── GET legacy key ─────►│
  │                          │◄── body ──────────────┤
  │                          ├── SHA-256(body)         │
  │                          ├── PUT blob key ────────►│
  │                          ├── DELETE legacy key ───►│
  │                          ├── D1 batch              │
  │◄── {tx, time, size} ────┤                        │
```

### Why uploads land at the legacy key first

Presigned PUT URLs are generated before the content exists. We can't compute
the SHA-256 hash until we have the content. So uploads land at a temporary
"legacy" key (`{actor}/{path}`), and `confirmUpload()` reads the blob,
hashes it, writes to the content-addressed key, and deletes the legacy key.

This double-write (legacy → content-addressed) adds one extra R2 GET + PUT +
DELETE during confirmation. For most files (<100 MB), this completes in under
a second. For very large files, the multipart flow is available.

### Multipart uploads

Same pattern: multipart parts assemble at the legacy key via R2's native
multipart API. `completeMultipart()` reads the assembled object, hashes it,
and content-addresses it. The legacy key is cleaned up.

For files >100 MB, this means the assembled object is read back into Worker
memory for hashing. Workers have a 128 MB memory limit. Files approaching this
limit should use chunked hashing in the future (stream the object and hash
incrementally using Web Crypto's streaming digest — not yet implemented but
straightforward).

---

## 10. Garbage Collection

### Orphaned blobs

A blob becomes orphaned when its `ref_count` reaches 0. This happens when:
- A file is deleted.
- A file is overwritten with different content.

Orphaned blobs are **not immediately deleted from R2**. They remain until the
GC cron runs.

### GC algorithm (planned)

```
Scheduled Event (cron: every 6 hours):
  1. SELECT addr, actor FROM blobs WHERE ref_count = 0 AND created_at < now() - 24h
  2. For each (addr, actor):
     a. Verify no file in `files` references this addr (safety check)
     b. DELETE from R2: blobs/{actor}/{aa}/{bb}/{hash}
     c. DELETE FROM blobs WHERE addr = ? AND actor = ?
  3. Log: {deleted_count, freed_bytes, duration}
```

### Grace period

The 24-hour grace period between `ref_count = 0` and actual deletion covers:

- **In-flight writes:** A write that hasn't committed its D1 batch yet might
  reference the same addr. Without the grace period, GC could delete the blob
  between the R2 PUT and the D1 commit.
- **Rollback recovery:** If we ever need to manually restore a file, having
  24 hours of blob retention gives ops time to react.
- **Replay window:** Historical event log queries can still resolve blob addrs
  for recently-deleted files (within 24h).

### Cost of deferred deletion

Orphaned blobs consume R2 storage until GC runs. At R2's pricing
($0.015/GB/month), keeping 1 GB of orphaned blobs for 24 hours costs:

```
$0.015 × (1/30) × 1 = $0.0005
```

Half a thousandth of a cent per GB per day. Negligible.

### Why not immediate deletion?

Immediate deletion would save this tiny storage cost but introduces:
- **Correctness risk:** Must check all files for the same addr before deleting.
  This requires a query per deletion, inside the write transaction path, adding
  latency.
- **Concurrency hazard:** Two concurrent writes to different paths might produce
  the same addr (uploading the same file). If one completes and the other's
  D1 batch fails, immediate deletion from the first could remove the blob that
  the second just PUT.
- **Complexity:** The write path would need to be a two-phase commit (check
  refs, delete if zero, handle races). Not worth it.

---

## 11. D1 Scaling Characteristics

### Row limits

D1 has no hard row limit per table. The practical ceiling is the 10 GB
database size limit. Approximate row counts at that limit:

| Table    | Avg row size | Rows at 10 GB  |
|----------|-------------|----------------|
| files    | ~150 bytes  | ~66 million    |
| events   | ~200 bytes  | ~50 million    |
| blobs    | ~100 bytes  | ~100 million   |

For a single-actor storage, 50 million events represents years of heavy use.
For multi-tenant, the total is shared across all actors.

### Write throughput

D1 is a single-writer database (SQLite WAL mode). Write throughput is bounded
by:
- ~100-500 writes per second for individual statements.
- `db.batch()` executes multiple statements in a single transaction, counting
  as one write operation. Our write path does 3-5 statements per batch = 1
  write operation.

At 500 batch writes per second across all actors, the system supports ~500
file mutations per second globally. This is sufficient for a storage service
where writes are infrequent relative to reads. If write throughput becomes a
bottleneck, the mitigation is:

1. **Shard by actor prefix** — route actors to different D1 databases based on
   a hash of their actor ID.
2. **Move to Durable Objects** — each actor gets a DO with its own SQLite
   (this is the long-term Cloudflare-native answer to write scaling).

### Read throughput

D1 reads are served from edge replicas. Read throughput is effectively
unlimited — each Cloudflare PoP maintains a read replica of the database.
Read latency is ~0.2ms from the nearest PoP.

### Query performance

Critical indexes:

```sql
PRIMARY KEY (owner, path) ON files     -- list, head, read lookups
idx_files_name ON files(owner, name)   -- search
idx_events_actor_tx ON events(actor, tx)   -- log by actor
idx_events_actor_path ON events(actor, path, tx)  -- log by path
PRIMARY KEY (addr, actor) ON blobs     -- dedup check, ref_count update
PRIMARY KEY (actor) ON tx_counter      -- tx allocation
```

All queries hit a covering index or primary key. No full table scans in any
read path. The `LIKE 'prefix%'` pattern in `list` and `search` uses the index
prefix (SQLite optimizes leading-prefix LIKE queries into range scans).

---

## 12. R2 Scaling Characteristics

### Object limits

R2 has no per-bucket object limit. The pricing-relevant limits:

| Metric              | Limit / Pricing                |
|---------------------|-------------------------------|
| Object size         | 5 TB per object               |
| Storage             | $0.015/GB/month               |
| Class A (PUT, HEAD) | $4.50 per million             |
| Class B (GET)       | $0.36 per million             |
| Free tier           | 10 GB storage, 1M Class A, 10M Class B/month |

### Key distribution

The `{hash[0:2]}/{hash[2:4]}` sharding ensures uniform key distribution.
SHA-256 output is uniformly distributed, so the first 4 hex characters create
an even spread across the 65,536 shard prefixes. R2 (like S3) partitions by key
prefix for parallel access, so this sharding maximizes read/write parallelism.

### Egress

R2 has zero egress fees for public access. Presigned URLs serve data directly
from R2 to the client, bypassing the Worker. This means:

- Worker CPU is not consumed for data transfer.
- No bandwidth costs (R2 → client is free).
- The Worker only handles metadata and URL generation.

---

## 13. Consistency Model

### D1 consistency

D1 uses SQLite in WAL mode with a single writer. This guarantees:

- **Serializable writes:** All write transactions execute sequentially. No
  write-write conflicts.
- **Read-after-write consistency:** On the primary (the region where the
  database was created), reads immediately see the latest writes.
- **Eventual consistency on replicas:** Edge read replicas may lag behind the
  primary by ~10-100ms. A write in Singapore may not be visible on a read in
  Frankfurt for up to 100ms.

For our use case: a user writes a file, then immediately lists files. If the
list query hits the primary, they see the new file. If it hits a stale replica,
they might not see it for ~100ms. This is acceptable for a storage service.

### R2 consistency

R2 provides strong read-after-write consistency:

- A PUT followed by a GET to the same key always returns the new data.
- A DELETE followed by a GET to the same key always returns 404.
- LIST is eventually consistent (new objects may take a few seconds to appear).

We don't use R2 LIST in the hot path (listing is served from D1). R2 LIST is
only used in GC (where eventual consistency is fine — if we miss a blob in one
GC pass, we catch it in the next).

### Cross-service consistency: D1 + R2

The write path has a potential inconsistency window:

```
Time 0: R2 PUT blob            ← blob exists in R2
Time 1: D1 batch (events, files) ← IF THIS FAILS, blob is orphaned
```

If the D1 batch fails (transient error, timeout), we have a blob in R2 with
no corresponding events or files entry. This is safe:

- The blob is unreferenced (no files row points to it).
- The blobs table doesn't have a row for it (the batch that would insert it
  failed).
- GC won't find it via the blobs table, but it also won't be referenced by
  any file. It's inert storage.
- A future write of the same content will find it via the R2 HEAD check and
  reuse it (dedup).

The reverse — D1 committed but R2 PUT failed — cannot happen because R2 PUT
executes first. If R2 PUT fails, the function throws before reaching the D1
batch.

### The `db.batch()` atomicity guarantee

D1's `batch()` runs all statements in a single SQLite transaction:

- If any statement fails, all statements roll back.
- The events insertion and files update are atomic.
- The blobs ref_count update is atomic with the event.

This is the critical guarantee that keeps the events table and files projection
in sync.

---

## 14. Event Log Design

### Schema

```sql
events(id, tx, actor, action, path, addr, size, type, meta, msg, ts)
```

- `id`: Auto-increment for insertion ordering within a tx.
- `tx`: Per-actor transaction number. Multiple events can share a tx (batch
  operations).
- `action`: Enum of `'write'`, `'move'`, `'delete'`.
- `path`: The affected path (for moves, the destination).
- `addr`: Content address (NULL for deletes).
- `meta`: JSON bag for action-specific data (e.g., `{"from": "old/path"}` for
  moves).
- `msg`: Human-readable commit message.
- `ts`: Unix millisecond timestamp.

### Separation from audit

The `audit` table (formerly `audit_log`) records all user-facing actions:
reads, auth attempts, share link creation, etc. The `events` table records
only data mutations. This separation exists because:

1. **Different retention needs.** Audit logs may have compliance-driven
   retention (90 days). Event logs may need longer retention for replay.
2. **Different access patterns.** Event logs are queried by tx number and path.
   Audit logs are queried by actor and time.
3. **Different write rates.** Every API call generates an audit entry. Only
   mutations generate events. The events table grows much more slowly.

### Replay capability

The events table enables reconstructing the state of any file at any past tx:

```sql
-- What was at docs/readme.md at tx=42?
SELECT addr, size, type FROM events
WHERE actor = ? AND path = 'docs/readme.md' AND tx <= 42 AND action = 'write'
ORDER BY tx DESC LIMIT 1;
```

If the blob at that addr still exists in R2 (not yet GC'd), the content can
be retrieved. This is the `snapshot()` method (currently internal only).

### Event compaction (future)

After the retention window (90 days), events can be compacted:

1. Take a snapshot of the current files table.
2. Delete events older than the retention window.
3. Store the snapshot as a single "checkpoint" event.

Replay from any point after the checkpoint is still possible. Replay before
the checkpoint requires the snapshot.

---

## 15. Cost Model

### Per-operation costs (approximate, USD)

| Operation       | D1 reads | D1 writes | R2 ops   | Total cost    |
|-----------------|----------|-----------|----------|---------------|
| write (10 KB)   | 2        | 1 batch   | 1-2      | ~$0.000008    |
| move            | 1        | 1 batch   | 0        | ~$0.000003    |
| delete (1 file) | 1        | 1 batch   | 0        | ~$0.000003    |
| read (metadata) | 1        | 0         | 0        | ~$0.00000015  |
| read (download) | 1        | 0         | 1        | ~$0.0000005   |
| list (100 files)| 1        | 0         | 0        | ~$0.00000015  |
| search          | 1        | 0         | 0        | ~$0.00000015  |
| log (50 events) | 1        | 0         | 0        | ~$0.00000015  |

### Monthly cost at scale (estimates)

| Scenario          | Files   | Writes/day | Reads/day | Storage | Monthly cost |
|-------------------|---------|------------|-----------|---------|--------------|
| Personal storage  | 1,000   | 50         | 500       | 1 GB    | ~$0.05       |
| Team workspace    | 10,000  | 500        | 5,000     | 10 GB   | ~$0.50       |
| Heavy agent usage | 100,000 | 5,000      | 50,000    | 100 GB  | ~$5.00       |

These are dominated by R2 storage costs ($0.015/GB/month). The D1 and R2
operation costs are negligible at these scales. Cloudflare's free tier covers
most personal usage entirely.

---

## 16. Failure Modes and Recovery

### Worker crash during write

- **R2 PUT completed, D1 batch not reached:** Orphaned blob. Inert. GC will
  eventually clean it, or a future write of the same content reuses it.
- **D1 batch partially committed:** Cannot happen. `db.batch()` is atomic.
- **D1 batch failed with transient error:** Write returns 500. Client retries.
  The next write attempt re-hashes, finds the blob already in R2 (dedup), and
  commits the D1 batch.

### D1 outage

All write and list operations fail. Read operations that use presigned URLs
(cached by the client) continue to work since they hit R2 directly.

### R2 outage

- **Writes fail** at the `bucket.head()` or `bucket.put()` step.
- **Reads fail** for content-addressed files.
- **Metadata operations** (list, search, stats, head) still work (D1 only).
- **Presigned URLs already issued** may fail (R2 is down).

### Blob corruption

If an R2 blob is corrupted (bit rot, although R2 uses checksums internally),
the content address acts as a checksum. On read, the caller can hash the
response and compare to the addr in the files table. If they don't match, the
blob is corrupt. Recovery: re-upload the file.

### Ref count drift

If ref_count drifts (e.g., due to a partial D1 batch failure that somehow
committed the event but not the blob decrement), the GC safety check prevents
data loss:

```
-- GC step: verify no file references this addr before deleting
SELECT 1 FROM files WHERE owner = ? AND addr = ? LIMIT 1;
```

If any file still points to the addr, GC skips it regardless of ref_count.
The ref_count is an optimization to avoid scanning the files table for every
blob — the safety check is the ground truth.

---

## 17. Security Considerations

### Content address leakage

The content address (SHA-256 hash) reveals whether two files have the same
content. This is why `addr` is not exposed in the public API. An attacker who
knows the hash of a target file could confirm its existence by uploading the
same content and comparing hashes. With per-actor namespacing, this attack is
limited: the attacker can only confirm content within their own namespace.

### Path traversal

The engine trusts that `path` has been validated by the route layer
(`validatePath()` in `lib/path.ts`). It does not re-validate. The path
validation rejects:
- `..` segments (directory traversal).
- Null bytes.
- Empty segments.
- Paths longer than 1024 characters or segments longer than 255 characters.

### Actor isolation

The engine takes `actor` as a parameter and scopes all queries with
`WHERE owner = ?` or `WHERE actor = ?`. There is no mechanism to access
another actor's data through the engine interface. Actor identity is
established by the auth middleware before the engine is called.

### Presigned URL security

Presigned URLs are generated using S3 V4 signing with the R2 access key.
They expire after the specified TTL (default 1 hour). The URL contains the
full blob key path, but this is not a security concern — the URL is
authenticated via the HMAC signature.

---

## 18. Migration Path

### Current state

The engine is deployed and handling all new writes via content addressing.
Pre-migration files have `addr = NULL` in the files table and are served via
the legacy `{actor}/{path}` R2 key.

### Backfill procedure (not yet run)

```
For each file in files WHERE addr IS NULL:
  1. GET from R2: {actor}/{path}
  2. SHA-256 hash the content
  3. PUT to R2: blobs/{actor}/{hash[0:2]}/{hash[2:4]}/{hash}
  4. UPDATE files SET addr = hash, tx = 0, tx_time = updated_at
  5. INSERT INTO blobs (addr, actor, size, ref_count, created_at) VALUES (...)
  6. INSERT INTO events (tx=0, action='write', msg='backfill', ...)
```

After backfill:
- All files have `addr` set.
- All files have a tx=0 event in the log.
- Legacy R2 keys can be deleted (but no rush — they cost only storage).

### Rollback

If the engine needs to be rolled back:
1. Deploy the previous Worker version (routes use `c.env.BUCKET` directly).
2. Pre-migration files still work (legacy keys untouched).
3. Post-migration files written via content addressing need their legacy keys
   restored. This is a migration script that reads `addr` from files, GETs
   from the blob key, and PUTs to the legacy key.

---

## 19. Future Directions

### Streaming hash for large uploads

Currently, `confirmUpload()` reads the entire blob into Worker memory to hash
it. For files >100 MB, this will hit the 128 MB Worker memory limit. The fix:
use Web Crypto's `crypto.subtle.digest()` with a streaming approach — read the
R2 object in chunks and update a running hash.

### Cross-actor dedup (optional)

If storage costs become significant, a second dedup layer could map `addr →
canonical_blob_key` globally. This would store one copy across all actors while
maintaining per-actor ref_counts. Complex, and only worth it if >30% of total
storage is duplicate across actors.

### Durable Objects migration

For actors with very high write throughput, the D1 single-writer bottleneck
can be bypassed by giving each actor a Durable Object with its own SQLite
instance. The `StorageEngine` interface makes this a drop-in replacement —
implement a `DurableObjectEngine` that uses the DO's SQLite for events/files/
blobs and R2 for blob storage.

### Event streaming

The events table enables real-time change feeds. A future endpoint could expose
a WebSocket or SSE stream of events as they happen, allowing clients to sync
their local state without polling.

### Snapshot endpoint

`GET /files/{path}?at_tx=N` — read a file's content at a specific past
transaction. Requires the blob to still exist in R2 (within the GC grace
period or retention window). Enables "file history" UIs and undo functionality.

---

## 20. Per-Actor Table Sharding (D1)

### Motivation

The original design stored all actors in shared tables with `WHERE owner = ?`
filtering. This has two problems:

1. **Query cost grows with total actors.** A `SELECT FROM files WHERE owner = ?`
   scans the B-tree branch for that owner. With 10,000 actors and 1M files each,
   the index is massive even though each query only touches one actor's data.

2. **Single table contention.** D1 is single-writer. All actors' writes compete
   for the same `files` table lock. A high-throughput actor's writes block
   everyone.

### Design

Each actor gets dedicated tables named by a deterministic shard:

```
shard = sha256(actor).slice(0, 16)    # 16 hex chars

f_{shard}  — files (replaces shared `files` table)
e_{shard}  — events (replaces shared `events` table)
b_{shard}  — blobs (replaces shared `blobs` table)
```

The `shards` registry table maps `actor → shard + next_tx`:

```sql
CREATE TABLE shards (
  actor      TEXT PRIMARY KEY,
  shard      TEXT NOT NULL UNIQUE,
  next_tx    INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL
);
```

### Why SHA-256 for shard names?

- **Deterministic:** No lookup needed — the shard can be computed from the
  actor name alone.
- **Collision-free:** 64-bit hash space (16 hex chars) supports ~4 billion
  actors before birthday collisions become probable. Collision is caught by
  the UNIQUE constraint on `shards.shard`.
- **Safe as SQL identifier:** Hexadecimal only — no injection risk in
  `f_{shard}` table names.

### Lazy creation

Tables are created on first access via `ensureShard()`:

```
ensureShard(actor)
  ├─ Check in-memory cache → return if hit
  ├─ SELECT shard FROM shards WHERE actor = ? → return if exists
  ├─ Compute shard = sha256(actor).slice(0, 16)
  ├─ INSERT INTO shards (handle race via UNIQUE conflict)
  ├─ CREATE TABLE IF NOT EXISTS f_{shard}, e_{shard}, b_{shard}
  ├─ Migrate legacy data from shared tables (INSERT OR IGNORE)
  └─ Cache shard in memory
```

### Legacy migration

On first access, existing data is copied from the shared tables:

```sql
INSERT OR IGNORE INTO f_{shard} SELECT ... FROM files WHERE owner = ?;
INSERT OR IGNORE INTO e_{shard} SELECT ... FROM events WHERE actor = ?;
INSERT OR IGNORE INTO b_{shard} SELECT ... FROM blobs WHERE actor = ?;
```

`INSERT OR IGNORE` ensures idempotency — the migration can be re-triggered
safely (e.g., if a previous attempt crashed mid-way).

### Trade-offs

| Factor                | Shared tables          | Per-actor sharding     |
|-----------------------|------------------------|------------------------|
| Query scope           | Full index scan        | Table IS the scope     |
| Write contention      | All actors compete     | Reduced (smaller tables)|
| Schema overhead       | 3 tables total         | 3 tables per actor     |
| Max actors per D1 DB  | Unlimited              | ~3,000 (SQLite limit)  |
| Cross-actor queries   | Trivial JOIN           | Requires iterating shards |
| Admin tooling         | Simple                 | Must resolve shard names |

For >3,000 actors, the system needs either multiple D1 databases or the
Durable Objects driver (see next section).

---

## 21. Durable Objects Driver

### Architecture

The `DOEngine` uses Cloudflare Durable Objects — each actor gets a dedicated
DO instance with its own SQLite database.

```
Worker ──RPC──► StorageDO[actor] ──► local SQLite
  │
  └──────────► R2 (blob storage, direct from Worker)
```

The DO handles metadata only. R2 blob operations happen directly in the Worker
to avoid routing large files through the DO's network hop.

### DO SQLite schema

Identical to a single D1 shard, but simpler — no `owner` column, no sharding:

```sql
files  (path PK, name, size, type, addr, tx, tx_time, updated_at)
events (id PK, tx, action, path, ...)
blobs  (addr PK, size, ref_count, created_at)
meta   (key PK, value)   -- stores 'next_tx' counter
```

### RPC split

The `DOEngine` (in the Worker) delegates to the DO via RPC:

| Operation        | Worker (DOEngine)         | DO (StorageDO)        |
|------------------|---------------------------|-----------------------|
| write            | R2 HEAD + PUT (dedup)     | recordWrite (SQL tx)  |
| read             | R2 GET (blob fetch)       | getFileAddr (SQL)     |
| move             | Nothing                   | recordMove (SQL tx)   |
| delete           | Nothing (GC later)        | recordDelete (SQL tx) |
| list/search/stats| Nothing                   | SQL query             |
| presign          | S3 V4 signing             | getFileAddr (for key) |

### Advantages over D1

1. **Zero write contention.** Each actor has its own SQLite — writes don't
   compete with other actors.
2. **Synchronous SQL.** `transactionSync()` provides real ACID transactions
   with no async overhead.
3. **Process-level isolation.** A bug or resource leak in one actor's DO
   cannot affect another actor.
4. **Instant actor deletion.** Delete the DO — all data gone.

### Disadvantages

1. **No edge read replicas.** All requests route to the DO's region. Read
   latency is ~10-30ms vs D1's ~0.2ms from edge.
2. **Higher cost.** DO request charge ($0.15/M) vs D1 read ($0.001/M).
3. **1 GB storage cap per DO.** Heavy actors with many files may hit this.
4. **No cross-actor queries.** Each DO is a separate database.
5. **Single-region.** The DO runs in one data center (auto-selected by
   Cloudflare based on request origin).

### When to choose DO over D1

- Actor generates >100 writes/second (D1 write contention).
- Process-level isolation required (compliance, security).
- Need WebSocket push from the DO for real-time file sync.
- Account deletion must be instantaneous and provably complete.

See `spec/0764_drivers_benchmark.md` for detailed cost and performance
comparison.

---

## 22. PostgreSQL Drivers (Hyperdrive & Neon)

Two additional drivers connect to PostgreSQL instead of Cloudflare-native
storage. Both extend `PgEngineBase` (shared SQL + R2 logic) and differ only
in transport.

### Architecture

```
PgEngineBase (abstract)
├── PostgreSQL schema (stg_files, stg_events, stg_blobs, stg_tx)
├── All StorageEngine methods (15 methods)
├── R2 blob operations (same as D1/DO)
├── query<T>()     → abstract (subclass provides)
└── transaction<R>() → abstract (subclass provides)

HyperdriveEngine                          NeonEngine
├── postgres lib (TCP)                    ├── neon() HTTP for reads
├── Hyperdrive edge proxy                 ├── Pool WebSocket for writes
└── Any PostgreSQL provider               └── Neon-specific
```

### PostgreSQL Schema

Unlike D1's per-actor table sharding (`f_{shard}`, `e_{shard}`), Postgres
uses standard multi-tenant tables with `owner`/`actor` columns:

```sql
stg_files  (owner TEXT, path TEXT, ...) PRIMARY KEY (owner, path)
stg_events (id BIGSERIAL, actor TEXT, tx INTEGER, ...)
stg_blobs  (addr TEXT, actor TEXT, ...) PRIMARY KEY (addr, actor)
stg_tx     (actor TEXT PRIMARY KEY, next_tx INTEGER)
```

Indexes on `(owner, ...)` make per-actor queries efficient. PostgreSQL's
query planner handles the multi-tenant pattern well.

### Transaction Counter

PostgreSQL's `INSERT ... ON CONFLICT ... RETURNING` enables atomic tx
allocation in a single query:

```sql
INSERT INTO stg_tx (actor, next_tx) VALUES ($1, 1)
ON CONFLICT (actor) DO UPDATE SET next_tx = stg_tx.next_tx + 1
RETURNING next_tx;
```

This is cleaner than D1's two-query approach (UPDATE then SELECT).

### Driver Selection

```typescript
// src/index.ts — engine middleware
const driver = c.env.STORAGE_DRIVER;
const engine =
  driver === "do"         ? new DOEngine(...)
  : driver === "hyperdrive" ? new HyperdriveEngine(...)
  : driver === "neon"       ? new NeonEngine(...)
  : new D1Engine(...);     // default
```

### When to Use PostgreSQL

- Need unlimited storage per actor (no 10 GB/1 GB caps)
- Cross-actor analytics with JOIN, window functions, GROUP BY
- Existing PostgreSQL infrastructure or team expertise
- Standard SQL ecosystem (pg_dump, psql, pgAdmin)
- Vendor portability (Hyperdrive works with any PostgreSQL)

See `spec/0766_hyperdrive.md`, `spec/0767_neon.md`, and
`spec/0768_benchmark_postgres.md` for detailed architecture and benchmarks.
