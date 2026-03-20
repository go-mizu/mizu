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

-- Buckets (top-level containers)
CREATE TABLE IF NOT EXISTS buckets (
  id                 TEXT PRIMARY KEY,
  owner              TEXT NOT NULL,
  name               TEXT NOT NULL,
  public             INTEGER NOT NULL DEFAULT 0,
  file_size_limit    INTEGER DEFAULT NULL,
  allowed_mime_types TEXT DEFAULT NULL,
  created_at         INTEGER NOT NULL,
  updated_at         INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_buckets_owner_name ON buckets(owner, name);

-- Objects (files within buckets)
CREATE TABLE IF NOT EXISTS objects (
  id           TEXT PRIMARY KEY,
  owner        TEXT NOT NULL,
  bucket_id    TEXT NOT NULL,
  path         TEXT NOT NULL,
  name         TEXT NOT NULL,
  content_type TEXT DEFAULT '',
  size         INTEGER DEFAULT 0,
  r2_key       TEXT DEFAULT '',
  metadata     TEXT DEFAULT '{}',
  accessed_at  INTEGER DEFAULT NULL,
  created_at   INTEGER NOT NULL,
  updated_at   INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_bucket_path ON objects(bucket_id, path);
CREATE INDEX IF NOT EXISTS idx_objects_owner ON objects(owner);
CREATE INDEX IF NOT EXISTS idx_objects_bucket ON objects(bucket_id);

-- Signed URLs (time-limited access tokens)
CREATE TABLE IF NOT EXISTS signed_urls (
  id         TEXT PRIMARY KEY,
  owner      TEXT NOT NULL,
  bucket_id  TEXT NOT NULL REFERENCES buckets(id),
  path       TEXT NOT NULL,
  token      TEXT NOT NULL UNIQUE,
  type       TEXT NOT NULL CHECK(type IN ('download','upload')),
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_signed_urls_token ON signed_urls(token);
CREATE INDEX IF NOT EXISTS idx_signed_urls_expires ON signed_urls(expires_at);

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

-- OAuth: dynamically registered clients
CREATE TABLE IF NOT EXISTS oauth_clients (
  client_id TEXT PRIMARY KEY,
  redirect_uris TEXT NOT NULL,
  client_name TEXT DEFAULT '',
  token_endpoint_auth_method TEXT DEFAULT 'none',
  created_at INTEGER NOT NULL
);

-- OAuth: authorization codes (short-lived, single-use)
CREATE TABLE IF NOT EXISTS oauth_codes (
  code TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  client_id TEXT NOT NULL,
  redirect_uri TEXT NOT NULL,
  scope TEXT DEFAULT '*',
  code_challenge TEXT NOT NULL,
  code_challenge_method TEXT DEFAULT 'S256',
  expires_at INTEGER NOT NULL
);

-- TUS resumable uploads (in-progress chunked uploads)
CREATE TABLE IF NOT EXISTS tus_uploads (
  id            TEXT PRIMARY KEY,
  owner         TEXT NOT NULL,
  bucket_id     TEXT NOT NULL,
  path          TEXT NOT NULL,
  upload_length INTEGER NOT NULL,
  upload_offset INTEGER NOT NULL DEFAULT 0,
  part_count    INTEGER NOT NULL DEFAULT 0,
  content_type  TEXT DEFAULT '',
  metadata      TEXT DEFAULT '{}',
  upsert        INTEGER NOT NULL DEFAULT 0,
  expires_at    INTEGER NOT NULL,
  created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tus_uploads_owner ON tus_uploads(owner);
CREATE INDEX IF NOT EXISTS idx_tus_uploads_expires ON tus_uploads(expires_at);

-- Rate limiting (sliding window counters)
CREATE TABLE IF NOT EXISTS rate_limits (
  key    TEXT PRIMARY KEY,
  count  INTEGER NOT NULL DEFAULT 1,
  window INTEGER NOT NULL
);
