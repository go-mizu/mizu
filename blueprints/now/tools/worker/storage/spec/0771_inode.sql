-- Storage Engine v2
-- Inode style metadata model with per actor sharding
--
-- Core idea:
--   node_id is identity
--   path is derived (not primary storage key)
--
-- Why:
--   avoids expensive subtree rewrites on directory move
--   enables stable identity across rename/move
--
-- Tradeoff:
--   full path lookup is no longer trivial → solved via path_cache projection


-- Transaction log per actor shard
CREATE TABLE IF NOT EXISTS transactions_{shard} (
  tx           INTEGER PRIMARY KEY,
  ts           INTEGER NOT NULL,
  msg          TEXT
);

-- Reason:
--   monotonic ordering of mutations per actor
-- Tradeoff:
--   ordering is local to actor, not global


-- Stable identity table
CREATE TABLE IF NOT EXISTS nodes_{shard} (
  node_id       TEXT PRIMARY KEY,
  kind          TEXT NOT NULL CHECK(kind IN ('file','dir')),
  created_at    INTEGER NOT NULL,
  deleted_at    INTEGER
);

CREATE INDEX IF NOT EXISTS idx_nodes_{shard}_kind
  ON nodes_{shard}(kind);

-- Reason:
--   separates identity from naming
-- Tradeoff:
--   requires joins / traversal to resolve paths


-- Namespace (directory structure)
CREATE TABLE IF NOT EXISTS directory_entries_{shard} (
  parent_id     TEXT NOT NULL,
  name          TEXT NOT NULL,
  child_id      TEXT NOT NULL,
  created_tx    INTEGER NOT NULL,
  deleted_tx    INTEGER,
  PRIMARY KEY (parent_id, name)
);

CREATE INDEX IF NOT EXISTS idx_directory_entries_{shard}_child
  ON directory_entries_{shard}(child_id);

CREATE INDEX IF NOT EXISTS idx_directory_entries_{shard}_parent
  ON directory_entries_{shard}(parent_id);

-- Reason:
--   explicit tree instead of string prefix hacks
--   rename = update name
--   move   = update parent_id
--
-- Major benefit:
--   directory move is O(1) in source of truth
--
-- Tradeoff:
--   path resolution requires traversal unless cached


-- File version history (immutable)
CREATE TABLE IF NOT EXISTS file_versions_{shard} (
  node_id        TEXT NOT NULL,
  version        INTEGER NOT NULL,
  content_hash   TEXT NOT NULL,
  size           INTEGER NOT NULL,
  content_type   TEXT,
  created_tx     INTEGER NOT NULL,
  PRIMARY KEY (node_id, version)
);

CREATE INDEX IF NOT EXISTS idx_file_versions_{shard}_hash
  ON file_versions_{shard}(content_hash);

-- Reason:
--   enables full history and snapshot reconstruction
-- Tradeoff:
--   more writes per operation


-- Current file state (fast read path)
CREATE TABLE IF NOT EXISTS file_current_state_{shard} (
  node_id         TEXT PRIMARY KEY,
  content_hash    TEXT NOT NULL,
  size            INTEGER NOT NULL,
  content_type    TEXT,
  version         INTEGER NOT NULL,
  updated_tx      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL
);

-- Reason:
--   avoids scanning version history for reads
-- Tradeoff:
--   must be kept consistent with file_versions


-- Blob reference tracking
CREATE TABLE IF NOT EXISTS blob_references_{shard} (
  content_hash   TEXT PRIMARY KEY,
  size           INTEGER NOT NULL,
  ref_count      INTEGER NOT NULL,
  created_at     INTEGER NOT NULL
);

-- Reason:
--   deduplication via content hash
--   enables safe garbage collection
--
-- Tradeoff:
--   correctness depends on accurate ref_count updates


-- Event log (identity-based)
CREATE TABLE IF NOT EXISTS events_{shard} (
  tx              INTEGER NOT NULL,
  action          TEXT NOT NULL CHECK(
                    action IN (
                      'create_node',
                      'link',
                      'unlink',
                      'rename',
                      'move',
                      'write',
                      'delete_node'
                    )
                  ),
  node_id         TEXT NOT NULL,
  parent_id       TEXT,
  old_parent_id   TEXT,
  name            TEXT,
  old_name        TEXT,
  content_hash    TEXT,
  size            INTEGER,
  content_type    TEXT,
  ts              INTEGER NOT NULL,
  meta            TEXT,
  PRIMARY KEY (tx, node_id, action)
);

CREATE INDEX IF NOT EXISTS idx_events_{shard}_tx
  ON events_{shard}(tx DESC);

CREATE INDEX IF NOT EXISTS idx_events_{shard}_node
  ON events_{shard}(node_id);

-- Reason:
--   tracks history by stable identity instead of path
--   fixes ambiguity after rename
--
-- Tradeoff:
--   less human-readable than path logs unless path is included in meta


-- Path cache (optional projection)
CREATE TABLE IF NOT EXISTS path_cache_{shard} (
  path           TEXT PRIMARY KEY,
  node_id        TEXT NOT NULL,
  parent_id      TEXT NOT NULL,
  name           TEXT NOT NULL,
  updated_tx     INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_path_cache_{shard}_node
  ON path_cache_{shard}(node_id);

CREATE INDEX IF NOT EXISTS idx_path_cache_{shard}_parent
  ON path_cache_{shard}(parent_id);

-- Reason:
--   restores fast lookup by full path
--
-- Critical design point:
--   this is NOT source of truth
--
-- Tradeoff options:
--   eager update:
--     faster reads, expensive directory move
--   lazy rebuild:
--     cheap move, slower first read


-- Name index (optional search projection)
CREATE TABLE IF NOT EXISTS name_index_{shard} (
  node_id        TEXT PRIMARY KEY,
  name           TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_name_index_{shard}_name
  ON name_index_{shard}(name);

-- Reason:
--   avoids inefficient LIKE queries on large datasets
--
-- Tradeoff:
--   extra maintenance cost
--   can be replaced with full text search later


-- Root node requirement
--
-- Each actor must have a root directory:
--   node_id = 'root' (or UUID)
--   kind = 'dir'
--
-- Root has no parent entry
--
-- Reason:
--   simplifies traversal logic


-- Important invariants (enforced in application layer)
--
-- 1. directory_entries.parent_id must refer to a directory node
-- 2. file_current_state only exists for file nodes
-- 3. file_versions only exists for file nodes
-- 4. no cycles in directory graph
-- 5. optional: no hard links (one parent per node)
--
-- Reason:
--   SQLite cannot enforce all graph constraints cleanly


-- Key design takeaway
--
-- Old design:
--   path = identity → simple but rename/move expensive
--
-- New design:
--   node_id = identity → slightly more complex reads,
--                        but correct and scalable namespace operations
--
-- This shifts cost:
--   from write-time subtree rewrites
--   to optional read-time or cache maintenance