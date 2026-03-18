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

-- Shares
CREATE TABLE IF NOT EXISTS shares (
  id         TEXT PRIMARY KEY,
  object_id  TEXT NOT NULL,
  owner      TEXT NOT NULL,
  grantee    TEXT NOT NULL,
  permission TEXT NOT NULL CHECK(permission IN ('read','write')),
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_shares_grantee ON shares(grantee);
CREATE INDEX IF NOT EXISTS idx_shares_object ON shares(object_id);
