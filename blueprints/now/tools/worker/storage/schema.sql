-- Actors
CREATE TABLE IF NOT EXISTS actors (
  actor      TEXT PRIMARY KEY,
  type       TEXT NOT NULL CHECK(type IN ('human','agent')),
  public_key TEXT,
  email      TEXT UNIQUE,
  bio        TEXT DEFAULT '',
  created_at INTEGER NOT NULL
);

-- Auth: challenges
CREATE TABLE IF NOT EXISTS challenges (
  id         TEXT PRIMARY KEY,
  actor      TEXT NOT NULL,
  nonce      TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_challenges_expires ON challenges(expires_at);

-- Auth: sessions
CREATE TABLE IF NOT EXISTS sessions (
  token      TEXT PRIMARY KEY,
  actor      TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor, expires_at);

-- Auth: magic tokens
CREATE TABLE IF NOT EXISTS magic_tokens (
  token      TEXT PRIMARY KEY,
  email      TEXT NOT NULL,
  actor      TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);

-- File/folder metadata
CREATE TABLE IF NOT EXISTS objects (
  id           TEXT PRIMARY KEY,
  owner        TEXT NOT NULL,
  path         TEXT NOT NULL,
  name         TEXT NOT NULL,
  is_folder    INTEGER NOT NULL DEFAULT 0,
  content_type TEXT DEFAULT '',
  size         INTEGER DEFAULT 0,
  r2_key       TEXT DEFAULT '',
  starred      INTEGER DEFAULT 0,
  trashed_at   INTEGER DEFAULT NULL,
  accessed_at  INTEGER DEFAULT NULL,
  description  TEXT DEFAULT '',
  created_at   INTEGER NOT NULL,
  updated_at   INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_owner_path ON objects(owner, path);

-- Shares (viewer, editor, uploader roles)
CREATE TABLE IF NOT EXISTS shares (
  id         TEXT PRIMARY KEY,
  object_id  TEXT NOT NULL,
  owner      TEXT NOT NULL,
  grantee    TEXT NOT NULL,
  permission TEXT NOT NULL CHECK(permission IN ('viewer','editor','uploader')),
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_shares_grantee ON shares(grantee);
CREATE INDEX IF NOT EXISTS idx_shares_object ON shares(object_id);

-- Public links (token-based unauthenticated access)
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

-- Scoped API keys (SHA-256 hashed tokens)
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

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
  id       INTEGER PRIMARY KEY AUTOINCREMENT,
  actor    TEXT,
  action   TEXT NOT NULL,
  resource TEXT,
  detail   TEXT,
  ip       TEXT,
  ts       INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor, ts);

-- Rate limiting (sliding window counters)
CREATE TABLE IF NOT EXISTS rate_limits (
  key    TEXT PRIMARY KEY,
  count  INTEGER NOT NULL DEFAULT 1,
  window INTEGER NOT NULL
);
