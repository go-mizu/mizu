# 0771 — Inode-Based Storage Engine v2

## Status: DRAFT

---

## 1. Problem Statement

The current storage engine (v1) uses **path as identity**. A file's primary key
is `(actor, path)`. This works well for simple CRUD but creates fundamental
problems at scale:

### 1.1 Directory move is O(children)

Moving `projects/old-name/` to `projects/new-name/` requires rewriting every
row whose path starts with `projects/old-name/`:

```sql
-- v1: must update EVERY child file
UPDATE f_{shard} SET path = REPLACE(path, 'projects/old-name/', 'projects/new-name/'),
                     name = ... -- may change for the directory itself
WHERE path LIKE 'projects/old-name/%';

-- Plus: UPDATE events for each file, new event per file
-- Plus: UPDATE path references in share_links, api_keys prefix scopes
```

For a directory with 1,000 files, this is 1,000 row updates in `files`, 1,000
event inserts, and potential breakage of share links and API key prefix scopes.
D1 batch limit is 100 statements. A 1,000-file move requires 10+ batches,
each a separate D1 round-trip. At ~5ms per batch, that's 50ms+ of serialized
D1 writes — and D1 is single-writer, so this blocks all other writes to the
database during execution.

### 1.2 No stable identity across rename

When a file is renamed from `draft.md` to `final.md`, it gets a new primary
key. Any external system that referenced it by path now has a dangling
reference. The event log records a "move" event, but reconstructing "this is
the same entity" requires following the move chain — which is fragile and
expensive.

### 1.3 No true directory objects

Directories are synthetic. `list()` infers them from path prefixes at query
time. There's no way to:
- Store metadata on a directory (description, icon, color)
- Move an empty directory
- Distinguish "directory was explicitly created" from "directory exists because
  a file is inside it"

### 1.4 Path-based search is string matching

`LIKE '%query%'` on full paths conflates directory names with file names. A
search for "readme" matches `readme.md`, `backup/readme.md`, and
`readme-old/notes.txt` (the directory name matched). v1's search scoring tries
to fix this in application code, but the underlying data model doesn't
distinguish "name of this entity" from "path segment of a parent".

---

## 2. Inode Model: Core Idea

Inspired by Unix filesystem design:

```
node_id = identity (stable, never changes)
path    = derived from directory_entries traversal (or cached)
```

A file named `docs/readme.md` is represented as:

```
nodes:             root (dir)
                     │
directory_entries:   root ──"docs"──► node_abc (dir)
                                        │
                                     node_abc ──"readme.md"──► node_xyz (file)

file_current_state: node_xyz → { content_hash: "a1b2...", size: 4096, version: 3 }
```

**Move** = change one row in `directory_entries`. O(1) regardless of subtree size.
**Rename** = change `name` in one `directory_entries` row. O(1).
**Stable reference** = `node_xyz` never changes, even after rename/move.

---

## 3. Schema Design

The SQL in `spec/0771_inode.sql` is a reference. Below is the production-optimized
schema, adapted per driver with performance annotations.

### 3.1 Common Schema (all drivers)

#### `transactions` — Per-actor mutation log

```sql
CREATE TABLE transactions (
  tx    INTEGER PRIMARY KEY,
  ts    INTEGER NOT NULL,         -- unix millis
  msg   TEXT                      -- commit message
);
```

**Why separate from events:** Events can have multiple rows per tx. The
transaction table is the single source of truth for "what happened when"
and enables efficient "list recent transactions" without scanning events.

**Cost:** 1 extra row write per mutation. Negligible.

#### `nodes` — Stable identity

```sql
CREATE TABLE nodes (
  node_id     TEXT PRIMARY KEY,   -- nanoid (21 chars) or 'root'
  kind        TEXT NOT NULL,      -- 'file' | 'dir'
  created_at  INTEGER NOT NULL,
  deleted_at  INTEGER             -- soft-delete timestamp; NULL = alive
);
```

**Why nanoid over UUID:** 21 chars vs 36 chars. 30% shorter keys. A-Za-z0-9_-
character set is URL-safe. Collision probability < UUID v4 for our scale
(~10^-30 at 1B nodes). Every row in `directory_entries`, `file_versions`,
`events`, and `path_cache` stores a `node_id` — smaller keys compound.

**Why soft-delete (`deleted_at`) instead of hard-delete:** Enables undelete
within a retention window. When `deleted_at` is set, the node is invisible to
read operations but still referenced by events for history. Hard purge happens
during GC (events older than retention window).

#### `directory_entries` — Namespace tree

```sql
CREATE TABLE directory_entries (
  parent_id   TEXT NOT NULL,
  name        TEXT NOT NULL,      -- segment name (not full path)
  child_id    TEXT NOT NULL,
  created_tx  INTEGER NOT NULL,
  deleted_tx  INTEGER,            -- NULL = active
  PRIMARY KEY (parent_id, name)
);

CREATE INDEX idx_de_child ON directory_entries(child_id);
```

**Critical design: `(parent_id, name)` as PK.** This enforces uniqueness —
no two entries with the same name in the same directory. The index on
`child_id` enables efficient "find my parent" lookups (for path
reconstruction).

**Soft-delete via `deleted_tx`:** When a file is moved or deleted, the old
directory entry gets `deleted_tx` set rather than being removed. This enables:
- History reconstruction ("what was in this directory at tx=50?")
- Undelete within retention window
- Simpler move semantics (unlink old, link new — both in one tx)

**No hard links (constraint):** Each `child_id` appears at most once in active
entries (where `deleted_tx IS NULL`). Enforced in application code, not SQL.
This keeps path resolution unambiguous — every node has exactly one active
parent path.

#### `file_versions` — Immutable version history

```sql
CREATE TABLE file_versions (
  node_id       TEXT NOT NULL,
  version       INTEGER NOT NULL, -- monotonic per node_id
  content_hash  TEXT NOT NULL,    -- SHA-256 hex
  size          INTEGER NOT NULL,
  content_type  TEXT,
  created_tx    INTEGER NOT NULL,
  PRIMARY KEY (node_id, version)
);

CREATE INDEX idx_fv_hash ON file_versions(content_hash);
```

**Why per-node version counter:** Not global tx. A file with versions 1,2,3
may span tx 5, 12, 89. The version number is a compact, node-local sequence.
Clients can request "version 2 of this file" without knowing the global tx.

**Why index on `content_hash`:** Enables efficient "what files share this
content?" queries for dedup analysis and GC.

#### `file_current_state` — Fast read path

```sql
CREATE TABLE file_current_state (
  node_id       TEXT PRIMARY KEY,
  content_hash  TEXT NOT NULL,
  size          INTEGER NOT NULL,
  content_type  TEXT,
  version       INTEGER NOT NULL,
  updated_tx    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL   -- unix millis
);
```

**Why a separate table instead of a view:** `SELECT * FROM file_versions
WHERE (node_id, version) IN (SELECT node_id, MAX(version) ...)` is expensive
on SQLite — it requires scanning or an anti-join. A materialized "current
state" table turns every read into a single PK lookup. The trade-off is one
extra write per file mutation (upsert into `file_current_state`), but writes
already touch 4+ tables so one more is marginal.

**Consistency invariant:** `file_current_state.version` always equals
`MAX(file_versions.version)` for the same `node_id`. Enforced by updating
both in the same transaction.

#### `blob_references` — Content dedup + GC

```sql
CREATE TABLE blob_references (
  content_hash  TEXT PRIMARY KEY,
  size          INTEGER NOT NULL,
  ref_count     INTEGER NOT NULL,  -- number of active file_versions pointing here
  created_at    INTEGER NOT NULL
);
```

**ref_count semantics change from v1:** In v1, ref_count tracks `files` rows.
In v2, ref_count tracks `file_versions` rows (since old versions still
reference blobs). A blob is GC-eligible only when `ref_count = 0` AND no
`file_versions` row references it. This is stricter than v1 but correct —
it prevents deleting blobs that old versions still need for snapshot reads.

**GC strategy:** Periodic sweep (cron) deletes R2 objects where:
```sql
SELECT content_hash FROM blob_references
WHERE ref_count <= 0
AND created_at < (now - 24h)  -- grace period
```

#### `events` — Identity-based audit log

```sql
CREATE TABLE events (
  tx            INTEGER NOT NULL,
  action        TEXT NOT NULL,     -- create_node, link, unlink, rename, move, write, delete_node
  node_id       TEXT NOT NULL,
  parent_id     TEXT,              -- for link/unlink/move
  old_parent_id TEXT,              -- for move (source parent)
  name          TEXT,              -- segment name
  old_name      TEXT,              -- for rename (source name)
  content_hash  TEXT,              -- for write
  size          INTEGER,
  content_type  TEXT,
  ts            INTEGER NOT NULL,
  meta          TEXT,              -- JSON for extensibility
  PRIMARY KEY (tx, node_id, action)
);

CREATE INDEX idx_events_tx ON events(tx DESC);
CREATE INDEX idx_events_node ON events(node_id);
```

**Why 7 actions instead of 3:** v1 had `write | move | delete`. v2 expands to
capture the structural operations precisely:

| Action        | Meaning                              | Changed tables                    |
|---------------|--------------------------------------|-----------------------------------|
| `create_node` | New node (file or dir) created       | `nodes`                          |
| `link`        | Node linked into a directory         | `directory_entries`              |
| `unlink`      | Node unlinked from a directory       | `directory_entries`              |
| `rename`      | Name changed within same parent      | `directory_entries`              |
| `move`        | Moved to different parent            | `directory_entries`              |
| `write`       | File content updated                 | `file_versions`, `file_current_state`, `blob_references` |
| `delete_node` | Node soft-deleted                    | `nodes`                          |

This granularity enables precise replay: "at tx=50, node_xyz was unlinked from
docs/ and linked into archive/". v1 can only say "move docs/readme.md →
archive/readme.md" — which breaks if docs/ itself was renamed later.

#### `path_cache` — Optional read-time projection

```sql
CREATE TABLE path_cache (
  path       TEXT PRIMARY KEY,
  node_id    TEXT NOT NULL,
  parent_id  TEXT NOT NULL,
  name       TEXT NOT NULL,
  updated_tx INTEGER NOT NULL
);

CREATE INDEX idx_pc_node ON path_cache(node_id);
CREATE INDEX idx_pc_parent ON path_cache(parent_id);
```

**This is NOT source of truth.** It's a denormalized projection rebuilt from
`directory_entries` traversal. Two maintenance strategies:

| Strategy        | Write cost      | Read cost (cold) | Read cost (warm) | Move dir cost    |
|-----------------|-----------------|------------------|------------------|------------------|
| **Eager update** | O(subtree_size) on dir move | O(1) always | O(1)           | O(subtree_size)  |
| **Lazy rebuild** | O(1) always    | O(depth) first time | O(1) cached   | O(1)             |

**Recommendation: lazy rebuild with eager invalidation.**

On directory move/rename:
1. Delete all `path_cache` rows where `path LIKE old_prefix || '%'`
2. Don't rebuild immediately

On path lookup:
1. Check `path_cache` for exact match
2. If miss: walk `directory_entries` from root, populate cache on the way down
3. Return result

This gives O(1) writes for moves (just invalidate) and amortized O(1) reads
(populated on first access). Worst case is a cold cache after a directory
move — the next `list()` call does one walk from root. For a depth-5 path,
that's 5 SQLite lookups (~0.5ms total on DO, ~2.5ms on D1).

---

## 4. Driver-Specific Adaptations

### 4.1 D1 v2 Driver (`d1_v2_driver.ts`)

**Sharding:** Same as v1 — all tables get `_{shard}` suffix. Registry in
`shards` table maps actor → shard.

```
nodes_{shard}
directory_entries_{shard}
file_versions_{shard}
file_current_state_{shard}
blob_references_{shard}
events_{shard}
transactions_{shard}
path_cache_{shard}
```

**Table creation:** Lazy, on first `ensureShard()` call. Same pattern as v1
but with 8 CREATE TABLE statements + indexes instead of 3.

**Batch limits:** D1 `db.batch()` is limited to 100 statements per call.
A single file write needs ~8-10 statements (depending on whether it's new or
overwrite). This leaves room for small batch operations. For bulk operations
(>10 files), split into multiple batches.

**D1 limitation — no RETURNING:** Transaction allocation still requires the
two-step UPDATE + SELECT pattern from v1.

**Path cache invalidation on dir move:**
```sql
-- Single batch:
DELETE FROM path_cache_{shard} WHERE path LIKE ? || '%'
```
This is a single statement regardless of subtree size. D1 handles the
`LIKE` predicate efficiently on the primary key index.

**Performance profile:**

| Operation           | D1 v1 (ms) | D1 v2 (ms) | Notes                              |
|---------------------|-------------|-------------|-------------------------------------|
| write (new file)    | ~8          | ~12         | +2 tables (versions, current_state) + node creation |
| write (overwrite)   | ~6          | ~8          | No new node; version insert + state upsert |
| move (single file)  | ~5          | ~4          | Simpler: 1 directory_entries update vs 3 file ops |
| move (dir, 100 files)| ~50        | **~4**      | O(1) directory_entries + cache invalidation |
| move (dir, 1000 files)| ~500+     | **~4**      | Same O(1) — this is the killer feature |
| read (cache hit)    | ~3          | ~3          | Same: path_cache lookup → R2 get |
| read (cache miss)   | ~3          | ~5          | Walk directory_entries (depth 3-5) + R2 get |
| list (prefix)       | ~4          | ~5          | Join path_cache with file_current_state |
| delete (single)     | ~5          | ~7          | Soft-delete node + unlink + ref_count update |

**Net:** Single-file writes are ~50% slower due to more tables. Directory
moves go from O(n) to O(1) — a categorical improvement. Read performance
is equivalent when path_cache is warm.

### 4.2 DO v2 Driver (`do_v2_driver.ts`)

**No sharding.** Each actor's DO has local SQLite with plain table names:

```sql
nodes, directory_entries, file_versions, file_current_state,
blob_references, events, transactions, path_cache
```

**Synchronous SQL.** All metadata operations run as `transactionSync()` — a
single synchronous SQLite transaction. No network hops for metadata. This is
the DO's killer advantage.

**Performance profile:**

| Operation             | DO v1 (ms) | DO v2 (ms) | Notes                           |
|-----------------------|------------|------------|----------------------------------|
| write (new file)      | ~4         | ~6         | +2 tables, still synchronous    |
| write (overwrite)     | ~3         | ~4         | Version insert + state upsert   |
| move (single file)    | ~2         | ~1.5       | Single directory_entries update  |
| move (dir, 100 files) | ~25        | **~1.5**   | O(1) — all synchronous          |
| move (dir, 1000 files)| ~250+      | **~1.5**   | Same O(1)                        |
| read (cache hit)      | ~1 + R2    | ~1 + R2    | Local SQLite lookup + R2 get    |
| list (prefix)         | ~2         | ~3         | Join across tables              |

**DO has the best story for v2:** synchronous transactions make the inode model's
multi-table writes cheap (no network per statement), and the O(1) move is
purely a SQLite operation with zero RPC overhead.

**RPC boundary:** Same as v1 — DOEngine (Worker) calls StorageDO (Durable
Object) via RPC for metadata, handles R2 directly. The RPC payloads change
slightly (node_id instead of path for some operations) but the split stays
the same.

### 4.3 PostgreSQL v2 Base (`pg_v2_base.ts`)

**Schema prefix:** `stg_` prefix on all tables (as in v1). PostgreSQL uses
`owner` column + indexes instead of table sharding.

```sql
CREATE TABLE stg_nodes (
  owner       TEXT NOT NULL,
  node_id     TEXT NOT NULL,
  kind        TEXT NOT NULL,
  created_at  BIGINT NOT NULL,
  deleted_at  BIGINT,
  PRIMARY KEY (owner, node_id)
);

CREATE TABLE stg_directory_entries (
  owner       TEXT NOT NULL,
  parent_id   TEXT NOT NULL,
  name        TEXT NOT NULL,
  child_id    TEXT NOT NULL,
  created_tx  INTEGER NOT NULL,
  deleted_tx  INTEGER,
  PRIMARY KEY (owner, parent_id, name)
);
CREATE INDEX idx_stg_de_child ON stg_directory_entries(owner, child_id);

CREATE TABLE stg_file_versions (
  owner         TEXT NOT NULL,
  node_id       TEXT NOT NULL,
  version       INTEGER NOT NULL,
  content_hash  TEXT NOT NULL,
  size          BIGINT NOT NULL,
  content_type  TEXT,
  created_tx    INTEGER NOT NULL,
  PRIMARY KEY (owner, node_id, version)
);

CREATE TABLE stg_file_current_state (
  owner         TEXT NOT NULL,
  node_id       TEXT NOT NULL,
  content_hash  TEXT NOT NULL,
  size          BIGINT NOT NULL,
  content_type  TEXT,
  version       INTEGER NOT NULL,
  updated_tx    INTEGER NOT NULL,
  updated_at    BIGINT NOT NULL,
  PRIMARY KEY (owner, node_id)
);

CREATE TABLE stg_blob_references (
  owner         TEXT NOT NULL,
  content_hash  TEXT PRIMARY KEY,
  size          BIGINT NOT NULL,
  ref_count     INTEGER NOT NULL,
  created_at    BIGINT NOT NULL,
  PRIMARY KEY (owner, content_hash)
);

CREATE TABLE stg_events (
  id            BIGSERIAL PRIMARY KEY,
  tx            INTEGER NOT NULL,
  owner         TEXT NOT NULL,
  action        TEXT NOT NULL,
  node_id       TEXT NOT NULL,
  parent_id     TEXT,
  old_parent_id TEXT,
  name          TEXT,
  old_name      TEXT,
  content_hash  TEXT,
  size          BIGINT,
  content_type  TEXT,
  ts            BIGINT NOT NULL,
  meta          TEXT
);
CREATE INDEX idx_stg_events_owner_tx ON stg_events(owner, tx DESC);
CREATE INDEX idx_stg_events_node ON stg_events(owner, node_id);

CREATE TABLE stg_transactions (
  owner   TEXT NOT NULL,
  tx      INTEGER NOT NULL,
  ts      BIGINT NOT NULL,
  msg     TEXT,
  PRIMARY KEY (owner, tx)
);

CREATE TABLE stg_path_cache (
  owner      TEXT NOT NULL,
  path       TEXT NOT NULL,
  node_id    TEXT NOT NULL,
  parent_id  TEXT NOT NULL,
  name       TEXT NOT NULL,
  updated_tx INTEGER NOT NULL,
  PRIMARY KEY (owner, path)
);
CREATE INDEX idx_stg_pc_node ON stg_path_cache(owner, node_id);
```

**PostgreSQL advantages over D1/DO for inode model:**

1. **Real `RETURNING`:** Transaction allocation is one statement:
   ```sql
   INSERT INTO stg_tx (owner, next_tx) VALUES ($1, 1)
   ON CONFLICT (owner) DO UPDATE SET next_tx = stg_tx.next_tx + 1
   RETURNING next_tx
   ```

2. **Recursive CTEs for path resolution:**
   ```sql
   WITH RECURSIVE ancestors AS (
     SELECT parent_id, name, child_id, 0 AS depth
     FROM stg_directory_entries
     WHERE owner = $1 AND child_id = $2 AND deleted_tx IS NULL
     UNION ALL
     SELECT de.parent_id, de.name, de.child_id, a.depth + 1
     FROM stg_directory_entries de
     JOIN ancestors a ON de.child_id = a.parent_id AND de.owner = $1
     WHERE de.deleted_tx IS NULL AND a.depth < 20
   )
   SELECT * FROM ancestors ORDER BY depth DESC;
   ```
   This resolves a full path in one query. D1's SQLite supports CTEs too,
   but PostgreSQL's query planner handles them more efficiently with proper
   index usage.

3. **Partial indexes for active entries:**
   ```sql
   CREATE INDEX idx_stg_de_active ON stg_directory_entries(owner, parent_id, name)
   WHERE deleted_tx IS NULL;
   ```
   Postgres partial indexes skip soft-deleted rows entirely. D1/SQLite has
   partial indexes too (since 3.8.0) but they're less commonly used.

4. **LISTEN/NOTIFY for cache invalidation:** Future optimization —
   PostgreSQL can push invalidation events to connected clients. Not
   applicable to Cloudflare Workers (no persistent connections) but
   valuable for self-hosted deployments.

**Sharding strategy for PostgreSQL:** The `owner` column with composite
primary keys is the right approach for PostgreSQL. Unlike D1 (which has
limited query planner), Postgres handles `WHERE owner = $1 AND ...`
efficiently with composite indexes. No table-per-actor sharding needed.

For truly large scale (>10M files per actor, >100 actors), consider
PostgreSQL table partitioning by `owner`:

```sql
CREATE TABLE stg_directory_entries (
  owner TEXT NOT NULL,
  ...
) PARTITION BY HASH (owner);

CREATE TABLE stg_de_p0 PARTITION OF stg_directory_entries
  FOR VALUES WITH (MODULUS 16, REMAINDER 0);
-- ... p1 through p15
```

This is a future optimization. The unpartitioned schema handles millions of
files without issues.

---

## 5. Engine Interface Changes

### 5.1 New types

```typescript
export interface NodeInfo {
  node_id: string;
  kind: "file" | "dir";
  created_at: number;
}

export interface DirEntry {
  name: string;
  node_id: string;
  kind: "file" | "dir";
  size?: number;          // from file_current_state (files only)
  content_type?: string;
  updated_at?: number;
  version?: number;
  tx?: number;
}

export interface FileVersion {
  version: number;
  content_hash: string;
  size: number;
  content_type: string | null;
  created_tx: number;
}
```

### 5.2 Interface additions

```typescript
export interface StorageEngine {
  // ... existing methods stay for backward compatibility ...

  /** Create a directory (explicit). */
  mkdir(actor: string, path: string, msg?: string): Promise<MutationResult>;

  /** Get file version history. */
  versions(
    actor: string,
    path: string,
    opts?: { limit?: number },
  ): Promise<FileVersion[]>;

  /** Read a specific version of a file. */
  readVersion(
    actor: string,
    path: string,
    version: number,
  ): Promise<ReadResult | null>;

  /** Resolve a path to a node_id (for stable references). */
  resolve(actor: string, path: string): Promise<NodeInfo | null>;
}
```

### 5.3 Backward compatibility

**The existing `StorageEngine` interface does NOT change for v1 methods.**
All existing routes continue to work. The v2 drivers implement both v1 methods
(write, move, delete, read, list, etc.) and new v2 methods. Internally, v1
methods delegate to inode operations:

```typescript
// write("docs/readme.md", body) internally:
//   1. Resolve "docs/" → find or create dir chain
//   2. Find or create file node in "docs/"
//   3. Write new version
//   4. Update file_current_state
//   5. Return { tx, time, size }
```

Routes don't need to change. The inode model is an internal implementation
detail — the API contract (path-based operations) remains the same.

---

## 6. Path Resolution: The Critical Path

Every path-based operation (read, write, list, move) must resolve a string
path like `docs/projects/readme.md` to a node_id. This is the most
performance-sensitive part of the inode model.

### 6.1 Resolution algorithm

```
resolve("docs/projects/readme.md"):
  1. Check path_cache for "docs/projects/readme.md"
     → hit: return node_id
     → miss: continue

  2. Start at root node
     segments = ["docs", "projects", "readme.md"]

  3. For each segment:
     SELECT child_id FROM directory_entries
     WHERE parent_id = ? AND name = ? AND deleted_tx IS NULL

  4. Cache the full path: INSERT INTO path_cache (path, node_id, parent_id, name, updated_tx)
     Also cache intermediate paths: "docs/", "docs/projects/"

  5. Return final node_id
```

**Worst case:** 3 SELECT queries for a depth-3 path (+ 3 cache inserts).
**Best case:** 1 SELECT on path_cache.
**Amortized:** O(1) after first access — paths are accessed repeatedly.

### 6.2 Cache invalidation

On directory move (e.g., `projects/` → `archive/`):

```sql
DELETE FROM path_cache WHERE path LIKE 'projects/%' OR path = 'projects/'
```

This invalidates all cached paths under the moved directory. Next access
to any path under `archive/` triggers a fresh walk.

**Why not update in place?** Updating would require knowing all the old paths,
which is the same O(subtree_size) operation we're trying to avoid. Deletion
is O(matched rows) but doesn't require knowing the new paths — they'll be
populated lazily.

### 6.3 PostgreSQL optimization: batch resolve

For `list()` operations that return many entries, PostgreSQL can resolve all
children of a directory in a single query:

```sql
SELECT de.name, de.child_id, n.kind,
       fcs.size, fcs.content_type, fcs.updated_at, fcs.version, fcs.updated_tx
FROM stg_directory_entries de
JOIN stg_nodes n ON n.owner = de.owner AND n.node_id = de.child_id
LEFT JOIN stg_file_current_state fcs
  ON fcs.owner = de.owner AND fcs.node_id = de.child_id
WHERE de.owner = $1 AND de.parent_id = $2 AND de.deleted_tx IS NULL
  AND n.deleted_at IS NULL
ORDER BY n.kind DESC, de.name  -- dirs first, then files
LIMIT $3 OFFSET $4
```

This replaces the v1 `LIKE` query with a single-parent lookup — more efficient
because it uses the primary key index exactly.

---

## 7. Write Path v2

```
write(actor, "docs/readme.md", body, contentType, msg)
  │
  ├─ 1. Buffer + SHA-256 hash → content_hash
  │
  ├─ 2. R2 dedup check + upload (same as v1)
  │
  ├─ 3. Resolve parent: ensure "docs/" directory exists
  │     └─ If not: create root → "docs" chain (create_node + link events)
  │
  ├─ 4. Check if file exists in parent directory
  │     SELECT child_id FROM directory_entries
  │     WHERE parent_id = ? AND name = 'readme.md' AND deleted_tx IS NULL
  │
  │     ┌─ File exists (node_id found):
  │     │   ├─ Get current version number
  │     │   ├─ Insert file_versions (version + 1)
  │     │   ├─ Upsert file_current_state
  │     │   ├─ Update blob_references
  │     │   └─ Insert event (action = 'write')
  │     │
  │     └─ File doesn't exist:
  │         ├─ Insert node (kind = 'file')
  │         ├─ Insert directory_entry (parent, 'readme.md', new_node)
  │         ├─ Insert file_versions (version = 1)
  │         ├─ Insert file_current_state
  │         ├─ Insert blob_references
  │         ├─ Insert event (create_node)
  │         ├─ Insert event (link)
  │         └─ Insert event (write)
  │
  ├─ 5. Insert transaction record
  │
  ├─ 6. Upsert path_cache for "docs/readme.md"
  │
  └─ Return { tx, time, size }
```

**Statement count:** New file = ~10 statements. Overwrite = ~6 statements.
Within D1's 100-statement batch limit.

---

## 8. Move Path v2

```
move(actor, "projects/old/", "archive/old/", msg)
  │
  ├─ 1. Resolve "projects/old/" → node_id (must be dir)
  │
  ├─ 2. Resolve target parent "archive/" → parent_node_id
  │     └─ If not exists: create directory chain
  │
  ├─ 3. One transaction:
  │     a. UPDATE directory_entries SET deleted_tx = tx
  │        WHERE parent_id = old_parent AND name = 'old'
  │     b. INSERT directory_entries (parent_id = new_parent, name = 'old', child_id = same)
  │     c. INSERT event (action = 'move', node_id, parent_id = new, old_parent_id = old)
  │     d. INSERT transaction
  │     e. DELETE FROM path_cache WHERE path LIKE 'projects/old/%'
  │     f. INSERT/UPDATE path_cache for 'archive/old/'
  │
  └─ Return { tx, time }
```

**Total cost:** ~6 statements. **Same regardless of subtree size.**

Compare v1: 1000 files under `projects/old/` = 1000+ UPDATE + 1000+ INSERT
event = 2000+ statements = 20+ D1 batches.

**v2 wins by 2-3 orders of magnitude on directory moves.**

---

## 9. Cost Analysis

### 9.1 D1 pricing impact

| Resource          | v1 per write | v2 per write (new file) | v2 per write (overwrite) |
|-------------------|-------------|-------------------------|--------------------------|
| D1 row reads      | 2           | 3-4                     | 2-3                      |
| D1 row writes     | 3-4         | 8-10                    | 5-7                      |
| R2 HEAD           | 1           | 1                       | 1                        |
| R2 PUT            | 0-1         | 0-1                     | 0-1                      |

D1 pricing: $0.75 per million row writes, $0.001 per million row reads.

**Per-write cost increase:** ~5 more row writes = $0.00000375 more per write.
At 1M writes/month, that's $3.75 more. Negligible.

**Per-move cost decrease (large dirs):**

| Dir size | v1 D1 writes  | v2 D1 writes | v1 cost    | v2 cost     |
|----------|---------------|-------------|------------|-------------|
| 10 files | 30            | 6           | $0.0000225 | $0.0000045  |
| 100      | 300           | 6           | $0.000225  | $0.0000045  |
| 1,000    | 3,000         | 6           | $0.00225   | $0.0000045  |
| 10,000   | 30,000        | 6           | $0.0225    | $0.0000045  |

**Break-even:** If >20% of mutations are directory moves (or renames of
directories with children), v2 is cheaper overall despite the per-write
increase.

### 9.2 Storage overhead

| Table              | v1 row size (est.) | v2 row size (est.) | v2 overhead |
|--------------------|-------------------|-------------------|-------------|
| files / file_current_state | ~150 bytes    | ~130 bytes        | -13%        |
| events             | ~120 bytes        | ~160 bytes (more columns) | +33%  |
| nodes              | N/A               | ~60 bytes         | new table   |
| directory_entries   | N/A               | ~80 bytes         | new table   |
| file_versions      | N/A               | ~100 bytes        | new table   |
| path_cache         | N/A               | ~120 bytes        | new table   |

For an actor with 10,000 files and 50,000 events:

| Component          | v1 size  | v2 size  |
|--------------------|----------|----------|
| files/state        | 1.5 MB   | 1.3 MB   |
| events             | 6.0 MB   | 8.0 MB   |
| nodes              | —        | 0.6 MB   |
| directory_entries   | —        | 0.8 MB   |
| file_versions      | —        | 1.0 MB   |
| path_cache         | —        | 1.2 MB   |
| **Total**          | **7.5 MB** | **12.9 MB** |

**~72% storage increase.** This is meaningful for D1 (10 GB limit shared
across all actors). At 12.9 MB per actor, D1 supports ~775 actors at 10K
files each. v1 supports ~1,333. Both are acceptable — D1's 10 GB limit is
unlikely to be hit before other scaling concerns arise.

For DO (1 GB per actor), 12.9 MB for 10K files means one DO supports ~77K
files — more than enough for any single user.

For PostgreSQL, storage is effectively unlimited. The increase is negligible.

### 9.3 Durable Object billing impact

DO pricing: $0.15 per million requests + $12.50 per million GB-seconds of
wall-clock duration.

v2 adds no extra RPC calls — the DOEngine adapter still makes one RPC call per
storage operation. The RPC payload is slightly larger (more data returned) but
this has negligible billing impact.

SQLite operations within the DO are free (no per-query billing). The main
cost driver is request count and duration, both unchanged.

---

## 10. Migration Strategy

### 10.1 Approach: parallel drivers, not in-place migration

**Do NOT migrate v1 tables in place.** Instead:

1. Create v2 tables alongside v1 tables (different names)
2. Add a `STORAGE_DRIVER` env flag: `"d1"` (v1, default), `"d1v2"`, `"dov2"`, `"pgv2"`
3. New actors automatically use v2
4. Existing actors stay on v1 until manually migrated
5. Background migration job converts actors from v1 to v2

### 10.2 Per-actor migration

```
migrate_actor(actor):
  1. Read all files from v1 (f_{shard} table)
  2. Build directory tree from paths:
     "docs/readme.md" → root/docs/ → docs/readme.md
  3. Create nodes for every file and every unique directory
  4. Create directory_entries for the tree structure
  5. Copy file_versions (v1 only has version 1 for all files)
  6. Copy file_current_state from v1 files table
  7. Copy blob_references
  8. Copy events (rewrite from path-based to node-based)
  9. Build path_cache from the tree
  10. Mark actor as v2 in shards table (add `driver_version` column)
```

**Risk:** The migration must be atomic per actor. If it fails halfway, the
actor must remain on v1. Use a transaction (or D1 batch) for steps 2-9.

**Batch size:** For an actor with 10K files, migration produces ~30K row
inserts. This requires ~300 D1 batches (100 per batch). At 5ms per batch,
total migration time is ~1.5 seconds. Acceptable as a background job.

### 10.3 Dual-read during migration

While migration is in progress, some actors are v1 and some are v2. The
engine middleware in `index.ts` must select the correct driver per-actor:

```typescript
// Engine middleware (simplified)
const driver_version = await getActorDriverVersion(actor);
if (driver_version === 2) {
  c.set("engine", new D1V2Engine(config));
} else {
  c.set("engine", new D1Engine(config));
}
```

This per-request lookup adds one D1 read. Cache it per-request or per-isolate
to avoid the overhead.

---

## 11. Scalability Analysis

### 11.1 D1 scaling limits

| Metric                 | v1 limit      | v2 limit       | Bottleneck             |
|------------------------|---------------|----------------|------------------------|
| Max files per actor    | ~500K         | ~300K          | D1 10 GB shared limit  |
| Max actors (10K files) | ~1,333        | ~775           | D1 10 GB shared limit  |
| Write throughput       | ~100/sec      | ~80/sec        | D1 single-writer       |
| Read throughput        | ~1000/sec     | ~800/sec       | D1 edge replicas       |
| Batch size limit       | 100 stmts     | 100 stmts      | D1 API limit           |
| Dir move (1K files)    | ~500ms        | **~4ms**       | D1 single-writer time  |

v2 trades some per-operation overhead for dramatically better directory
operations. The D1 single-writer bottleneck is the same for both versions.

### 11.2 DO scaling limits

| Metric                 | v1 limit      | v2 limit       | Bottleneck             |
|------------------------|---------------|----------------|------------------------|
| Max files per actor    | ~2M           | ~1.2M          | DO 1 GB SQLite limit   |
| Write throughput       | ~500/sec      | ~400/sec       | DO CPU (synchronous)   |
| Read throughput        | ~2000/sec     | ~1500/sec      | DO CPU + R2            |
| Dir move (1K files)    | ~250ms        | **~1.5ms**     | Local SQLite           |

DO is the best fit for v2 — local synchronous SQL makes multi-table
transactions cheap, and the per-actor isolation means no cross-actor
contention.

### 11.3 PostgreSQL scaling limits

| Metric                 | v1 limit      | v2 limit       | Bottleneck             |
|------------------------|---------------|----------------|------------------------|
| Max files per actor    | ~10M+         | ~10M+          | Postgres capacity      |
| Max total files        | ~100M+        | ~100M+         | Disk + connections     |
| Write throughput       | ~1000/sec     | ~800/sec       | Transaction round-trip |
| Read throughput        | ~5000/sec     | ~4000/sec      | Connection pool        |
| Dir move (1K files)    | ~300ms        | **~5ms**       | Single transaction     |

PostgreSQL has the highest absolute capacity but higher per-query latency
(network hop to database). Recursive CTEs make path resolution efficient.

---

## 12. Trade-off Summary

### What we gain

1. **O(1) directory move/rename** — the primary motivation. Goes from seconds
   to milliseconds for large directories.
2. **Stable identity** — node_id survives rename/move. External systems can
   reference files reliably.
3. **Version history** — built-in, not bolted on. Each file has a version
   chain; old versions are readable until GC.
4. **True directories** — first-class objects with their own identity. Can
   store metadata, be explicitly created, survive being emptied.
5. **Precise event log** — identity-based events are unambiguous. "node_xyz
   was moved from A to B" is clearer than "A/file.md was moved to B/file.md"
   when A itself was also renamed.
6. **Foundation for features** — hard links, symlinks, snapshots, branching
   all become possible (though not planned for v2).

### What we pay

1. **~50% more D1 writes per file operation** — more tables to update. Costs
   an extra $0.000004 per write. Insignificant.
2. **~72% more SQLite storage** — new tables for nodes, directory entries,
   versions, path cache. Reduces max actors per D1 database by ~40%.
3. **Path resolution overhead** — cold cache reads require walking the tree.
   Amortized O(1) with path_cache, but first access after dir move is slower.
4. **Complexity** — more tables, more indexes, more code paths. The driver
   implementation is ~2x larger than v1. More surface area for bugs.
5. **Migration effort** — existing actors must be migrated. Per-actor
   migration is ~1.5 seconds but must be orchestrated carefully.

### When to use v2

- Workloads with frequent directory reorganization (AI agents that restructure
  projects, CI/CD pipelines that move build outputs)
- Applications that need stable file references (shareable links that survive
  rename, integrations that track files by ID)
- Users who need version history (writers, developers, anyone who says "I
  want the old version back")

### When v1 is still fine

- Simple flat-file storage (no deep directory hierarchy)
- Write-heavy, read-light workloads where the per-write overhead matters
- Actors with very few files (<100) where directory moves are rare

---

## 13. Implementation Order

1. **`engine_v2.ts`** — New types (NodeInfo, DirEntry, FileVersion) + extended
   interface. Keep backward-compatible with v1 interface.

2. **`do_v2_driver.ts`** — Start with DO because synchronous SQL makes
   debugging easiest. No sharding complexity. Implement all v2 operations +
   v1 compatibility layer.

3. **`d1_v2_driver.ts`** — Port from DO v2, adding shard suffix to all table
   names and converting synchronous SQL to D1 batch calls.

4. **`pg_v2_base.ts`** — Port from DO v2, adapting for PostgreSQL syntax
   (BIGINT, RETURNING, partial indexes, recursive CTEs).

5. **Migration tooling** — Background worker that converts v1 actors to v2.
   Can be run as a Cloudflare Cron Trigger or manually via admin endpoint.

6. **Route updates** — Add new endpoints for `versions()`, `readVersion()`,
   `mkdir()`, `resolve()`. Existing routes unchanged.

---

## 14. Open Questions

1. **Path cache: eager or lazy for `list()`?** Lazy invalidation works for
   single-path lookups. But `list()` needs all children of a directory —
   should it go through path_cache or query directory_entries directly?
   **Recommendation:** `list()` queries `directory_entries` directly (it
   already knows the parent_id). path_cache is only for path→node_id lookups.

2. **Version retention:** Keep all versions forever, or cap at N versions per
   file? Unbounded versions mean unbounded `file_versions` growth. A cap of
   100 versions per file + the 90-day event retention from v1 seems
   reasonable.
   **Recommendation:** 100 versions per file. Oldest version auto-pruned when
   101st is written (only if older than 24h — prevent rapid-fire overwrites
   from losing all history).

3. **Symbolic links:** The inode model naturally supports symlinks
   (directory_entries where child_id points to a node in a different
   subtree). Should we support this in v2?
   **Recommendation:** No. Keep the "no hard links" constraint (one active
   parent per node). Symlinks add complexity to path resolution (cycle
   detection, maximum follow depth) for little user benefit. Revisit if
   requested.

4. **node_id in API responses:** Should the public API expose node_id?
   **Recommendation:** Yes, as an opaque identifier. Include `id` in file
   metadata responses. Don't promise any format (nanoid today, could change).
   This enables clients to build stable references.

5. **Empty directory listing:** v1 infers directories from file paths. v2 has
   explicit directory nodes. Should `list("docs/")` return empty if docs/ is
   an explicit directory with no children?
   **Recommendation:** Yes. Return the directory itself in the parent listing
   but return `[]` entries when listed. This matches filesystem semantics.
