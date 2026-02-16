-- Specialized tables replacing the generic kv table.
-- The kv table stays for backward compatibility.

-- Accounts with proper columns
CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  email_provider TEXT NOT NULL,
  email_password_enc TEXT NOT NULL DEFAULT '',
  session_csrf TEXT NOT NULL,
  session_cookies TEXT NOT NULL,
  session_created_at TEXT NOT NULL,
  pro_queries INTEGER NOT NULL DEFAULT 5,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  last_used_at TEXT NOT NULL,
  disabled_at TEXT,
  disable_reason TEXT,
  total_queries_used INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts (status);

-- Singleton round-robin counter
CREATE TABLE IF NOT EXISTS account_robin (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  value INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO account_robin (id, value) VALUES (1, 0);

-- Append-only registration logs
CREATE TABLE IF NOT EXISTS account_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp TEXT NOT NULL,
  event TEXT NOT NULL,
  message TEXT NOT NULL,
  provider TEXT,
  email TEXT,
  account_id TEXT,
  duration_ms INTEGER,
  error TEXT
);

-- Singleton registration lock
CREATE TABLE IF NOT EXISTS account_lock (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  locked_at TEXT,
  expires_at INTEGER
);

INSERT OR IGNORE INTO account_lock (id) VALUES (1);

-- Thread metadata
CREATE TABLE IF NOT EXISTS threads (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  mode TEXT NOT NULL,
  model TEXT NOT NULL,
  message_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_threads_updated ON threads (updated_at DESC);

-- Normalized thread messages
CREATE TABLE IF NOT EXISTS thread_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  thread_id TEXT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  citations TEXT,
  web_results TEXT,
  related_queries TEXT,
  images TEXT,
  videos TEXT,
  thinking_steps TEXT,
  backend_uuid TEXT,
  model TEXT,
  duration_ms INTEGER,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_thread_messages_thread ON thread_messages (thread_id, seq);

-- Session pool
CREATE TABLE IF NOT EXISTS sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  csrf_token TEXT NOT NULL,
  cookies TEXT NOT NULL,
  created_at TEXT NOT NULL,
  is_legacy INTEGER NOT NULL DEFAULT 0
);

-- OG metadata cache
CREATE TABLE IF NOT EXISTS og_cache (
  url TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  site_name TEXT NOT NULL DEFAULT '',
  expires_at INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
