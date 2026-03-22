CREATE TABLE IF NOT EXISTS bot_cache (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bot_cache_expires ON bot_cache(expires_at);
