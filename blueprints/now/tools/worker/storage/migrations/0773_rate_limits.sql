-- Bot protection: rate limiting table
CREATE TABLE IF NOT EXISTS rate_limits (
  endpoint TEXT NOT NULL,
  key      TEXT NOT NULL,
  ts       INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rl_lookup ON rate_limits(endpoint, key, ts);
