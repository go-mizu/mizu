CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'running',
  config TEXT NOT NULL,
  total INTEGER NOT NULL DEFAULT 0,
  finished INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL,
  url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued',
  http_status INTEGER NOT NULL DEFAULT 0,
  title TEXT NOT NULL DEFAULT '',
  html TEXT,
  markdown TEXT,
  depth INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE INDEX IF NOT EXISTS idx_pages_job_id ON pages(job_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pages_job_url ON pages(job_id, url);

-- Cache for single-URL rendering endpoints
-- PK (url, endpoint, params_hash) covers parameterized variants
-- params_hash is '' for simple URL-only endpoints, 16-char hex for parameterized ones
CREATE TABLE IF NOT EXISTS page_cache (
  url          TEXT    NOT NULL,
  endpoint     TEXT    NOT NULL,
  params_hash  TEXT    NOT NULL DEFAULT '',
  html         TEXT,
  markdown     TEXT,
  result       TEXT,
  title        TEXT,
  created_at   INTEGER NOT NULL,

  PRIMARY KEY (url, endpoint, params_hash)
);

-- Fast lookup by URL (invalidation, cross-endpoint queries)
CREATE INDEX IF NOT EXISTS idx_page_cache_url ON page_cache(url);

-- TTL sweeping by age
CREATE INDEX IF NOT EXISTS idx_page_cache_created ON page_cache(created_at);
