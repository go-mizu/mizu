-- Migration for spec 0745: new auth model
-- Adds new tables and columns to existing schema

-- New tables
CREATE TABLE IF NOT EXISTS challenges (
  id TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  nonce TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_challenges_expires ON challenges(expires_at);

CREATE TABLE IF NOT EXISTS sessions (
  token TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Add type column to actors (default 'human' for existing rows)
ALTER TABLE actors ADD COLUMN type TEXT NOT NULL DEFAULT 'human';

-- Add role column to members (default 'member' for existing rows)
ALTER TABLE members ADD COLUMN role TEXT NOT NULL DEFAULT 'member';

-- Add client_id column to messages
ALTER TABLE messages ADD COLUMN client_id TEXT;
CREATE INDEX IF NOT EXISTS idx_messages_client ON messages(client_id);

-- Add index on members.actor
CREATE INDEX IF NOT EXISTS idx_members_actor ON members(actor);
