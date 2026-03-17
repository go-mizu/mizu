-- Add email column to actors (nullable for existing key-based users)
ALTER TABLE actors ADD COLUMN email TEXT;

-- Unique index on email (only for non-null values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_actors_email ON actors(email);

-- Magic link tokens
CREATE TABLE IF NOT EXISTS magic_tokens (
  token TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  actor TEXT,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_magic_expires ON magic_tokens(expires_at);
