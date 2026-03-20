-- Storage API Protocol v1 — minimal schema
-- R2 is source of truth for file bytes. D1 stores auth state + search index.

-- Users / agents
CREATE TABLE IF NOT EXISTS actors (
  actor      TEXT PRIMARY KEY,
  type       TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')),
  public_key TEXT,
  email      TEXT UNIQUE,
  created_at INTEGER NOT NULL
);

-- Ed25519 challenge/response auth
CREATE TABLE IF NOT EXISTS challenges (
  id         TEXT PRIMARY KEY,
  actor      TEXT NOT NULL,
  nonce      TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_challenges_expires ON challenges(expires_at);

-- Session tokens (from /auth/verify)
CREATE TABLE IF NOT EXISTS sessions (
  token      TEXT PRIMARY KEY,
  actor      TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor, expires_at);

-- API keys (SHA-256 hashed, optional path prefix restriction)
CREATE TABLE IF NOT EXISTS api_keys (
  id         TEXT PRIMARY KEY,
  actor      TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL DEFAULT '',
  prefix     TEXT NOT NULL DEFAULT '',
  expires_at INTEGER,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor);

-- File search index (kept in sync with R2 on write/delete)
-- NOT the source of truth — R2 is. Used for fast listing and fuzzy search.
CREATE TABLE IF NOT EXISTS files (
  owner      TEXT NOT NULL,
  path       TEXT NOT NULL,
  name       TEXT NOT NULL,
  size       INTEGER NOT NULL DEFAULT 0,
  type       TEXT NOT NULL DEFAULT 'application/octet-stream',
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (owner, path)
);
CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE);

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
  id     INTEGER PRIMARY KEY AUTOINCREMENT,
  actor  TEXT,
  action TEXT NOT NULL,
  path   TEXT,
  ip     TEXT,
  ts     INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(actor, ts);

-- Share links (opaque, revocable)
CREATE TABLE IF NOT EXISTS share_links (
  token      TEXT PRIMARY KEY,
  actor      TEXT NOT NULL,
  path       TEXT NOT NULL,
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_share_links_actor ON share_links(actor, created_at);
CREATE INDEX IF NOT EXISTS idx_share_links_expires ON share_links(expires_at);

-- Magic link tokens (passwordless email sign-in)
CREATE TABLE IF NOT EXISTS magic_tokens (
  token      TEXT PRIMARY KEY,
  email      TEXT NOT NULL,
  actor      TEXT,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_magic_tokens_email ON magic_tokens(email);
CREATE INDEX IF NOT EXISTS idx_magic_tokens_expires ON magic_tokens(expires_at);

-- OAuth clients (dynamic registration)
CREATE TABLE IF NOT EXISTS oauth_clients (
  client_id                    TEXT PRIMARY KEY,
  redirect_uris                TEXT NOT NULL DEFAULT '[]',
  client_name                  TEXT NOT NULL DEFAULT '',
  token_endpoint_auth_method   TEXT NOT NULL DEFAULT 'none',
  created_at                   INTEGER NOT NULL
);

-- OAuth authorization codes (single-use, short-lived)
CREATE TABLE IF NOT EXISTS oauth_codes (
  code                   TEXT PRIMARY KEY,
  actor                  TEXT NOT NULL,
  client_id              TEXT NOT NULL,
  redirect_uri           TEXT NOT NULL,
  scope                  TEXT NOT NULL DEFAULT '*',
  code_challenge         TEXT NOT NULL,
  code_challenge_method  TEXT NOT NULL DEFAULT 'S256',
  expires_at             INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_codes(expires_at);
