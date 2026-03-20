-- Magic link tokens for passwordless email sign-in
CREATE TABLE IF NOT EXISTS magic_tokens (
  token      TEXT PRIMARY KEY,
  email      TEXT NOT NULL,
  actor      TEXT,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_magic_tokens_email ON magic_tokens(email);
CREATE INDEX IF NOT EXISTS idx_magic_tokens_expires ON magic_tokens(expires_at);
