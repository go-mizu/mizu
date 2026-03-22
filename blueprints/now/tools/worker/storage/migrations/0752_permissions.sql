-- 0752: Comprehensive Permission & Security Model
-- Run with: wrangler d1 execute storage-db --remote --file=migrations/0752_permissions.sql

-- Step 1: Recreate shares table with new roles (viewer, editor, uploader)
-- The old table has CHECK(permission IN ('read','write')), so we must copy+translate in one step
CREATE TABLE IF NOT EXISTS shares_new (
  id         TEXT PRIMARY KEY,
  object_id  TEXT NOT NULL,
  owner      TEXT NOT NULL,
  grantee    TEXT NOT NULL,
  permission TEXT NOT NULL CHECK(permission IN ('viewer','editor','uploader')),
  created_at INTEGER NOT NULL
);
INSERT OR IGNORE INTO shares_new (id, object_id, owner, grantee, permission, created_at)
  SELECT id, object_id, owner, grantee,
    CASE permission WHEN 'read' THEN 'viewer' WHEN 'write' THEN 'editor' ELSE permission END,
    created_at
  FROM shares;
DROP TABLE IF EXISTS shares;
ALTER TABLE shares_new RENAME TO shares;
CREATE INDEX IF NOT EXISTS idx_shares_grantee ON shares(grantee);
CREATE INDEX IF NOT EXISTS idx_shares_object ON shares(object_id);

-- Step 2: Public links
CREATE TABLE IF NOT EXISTS public_links (
  id             TEXT PRIMARY KEY,
  object_id      TEXT NOT NULL,
  owner          TEXT NOT NULL,
  token          TEXT NOT NULL UNIQUE,
  permission     TEXT NOT NULL DEFAULT 'viewer' CHECK(permission IN ('viewer','editor')),
  password_hash  TEXT,
  expires_at     INTEGER,
  max_downloads  INTEGER,
  download_count INTEGER DEFAULT 0,
  created_at     INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_public_links_token ON public_links(token);
CREATE INDEX IF NOT EXISTS idx_public_links_owner ON public_links(owner);

-- Step 3: Scoped API keys
CREATE TABLE IF NOT EXISTS api_keys (
  id           TEXT PRIMARY KEY,
  actor        TEXT NOT NULL,
  token_hash   TEXT NOT NULL UNIQUE,
  name         TEXT NOT NULL,
  scopes       TEXT NOT NULL DEFAULT '*',
  path_prefix  TEXT DEFAULT '',
  expires_at   INTEGER,
  last_used_at INTEGER,
  created_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(token_hash);

-- Step 4: Audit log
CREATE TABLE IF NOT EXISTS audit_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  actor      TEXT,
  action     TEXT NOT NULL,
  resource   TEXT,
  detail     TEXT,
  ip         TEXT,
  ts         INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor, ts);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(ts);

-- Step 5: Rate limiting
CREATE TABLE IF NOT EXISTS rate_limits (
  key        TEXT PRIMARY KEY,
  count      INTEGER NOT NULL DEFAULT 1,
  window     INTEGER NOT NULL
);
