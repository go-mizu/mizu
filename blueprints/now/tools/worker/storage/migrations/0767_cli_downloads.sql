-- Track CLI binary downloads for analytics and monitoring
CREATE TABLE IF NOT EXISTS cli_downloads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version TEXT NOT NULL,
  filename TEXT NOT NULL,
  os TEXT,
  arch TEXT,
  ip TEXT,
  country TEXT,
  user_agent TEXT,
  referrer TEXT,
  ts INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cli_dl_ts ON cli_downloads(ts);
CREATE INDEX IF NOT EXISTS idx_cli_dl_version ON cli_downloads(version);
