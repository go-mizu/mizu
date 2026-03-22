-- Event-sourced storage engine (spec/0762)
-- Events: append-only mutation log for write/move/delete
-- Blobs:  ref-counted content-addressed objects in R2
-- tx_counter: per-actor monotonic transaction numbers

-- Per-actor transaction counter
CREATE TABLE IF NOT EXISTS tx_counter (
  actor   TEXT PRIMARY KEY,
  next_tx INTEGER NOT NULL DEFAULT 1
);

-- Mutation event log (data events only — audit stays separate)
CREATE TABLE IF NOT EXISTS events (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  tx      INTEGER NOT NULL,
  actor   TEXT    NOT NULL,
  action  TEXT    NOT NULL CHECK(action IN ('write','move','delete')),
  path    TEXT    NOT NULL,
  addr    TEXT,                        -- content address (SHA-256), NULL for delete
  size    INTEGER NOT NULL DEFAULT 0,
  type    TEXT,                        -- MIME type
  meta    TEXT,                        -- JSON metadata (e.g. {"from":"old/path"} for moves)
  msg     TEXT,                        -- commit message
  ts      INTEGER NOT NULL             -- unix epoch millis
);
CREATE INDEX IF NOT EXISTS idx_events_actor_tx   ON events(actor, tx);
CREATE INDEX IF NOT EXISTS idx_events_actor_path ON events(actor, path, tx);

-- Blob reference tracking (per-actor, for GC)
CREATE TABLE IF NOT EXISTS blobs (
  addr       TEXT    NOT NULL,
  actor      TEXT    NOT NULL,
  size       INTEGER NOT NULL,
  ref_count  INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (addr, actor)
);

-- Add version columns to files table
ALTER TABLE files ADD COLUMN addr    TEXT;
ALTER TABLE files ADD COLUMN tx      INTEGER;
ALTER TABLE files ADD COLUMN tx_time INTEGER;

-- Rename audit_log → audit
ALTER TABLE audit_log RENAME TO audit;

-- Re-create index on new table name
CREATE INDEX IF NOT EXISTS idx_audit_actor_ts ON audit(actor, ts);
