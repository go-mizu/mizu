-- Key-value store for account data
CREATE TABLE IF NOT EXISTS kv (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER -- unix timestamp ms, NULL = no expiry
);

CREATE INDEX IF NOT EXISTS idx_kv_prefix ON kv (key);
CREATE INDEX IF NOT EXISTS idx_kv_expires ON kv (expires_at) WHERE expires_at IS NOT NULL;
