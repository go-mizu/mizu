CREATE TABLE IF NOT EXISTS chats (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  creator TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'public',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS members (
  chat_id TEXT NOT NULL,
  actor TEXT NOT NULL,
  joined_at INTEGER NOT NULL,
  PRIMARY KEY (chat_id, actor),
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  chat_id TEXT NOT NULL,
  actor TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_members_chat ON members(chat_id);

CREATE TABLE IF NOT EXISTS actors (
  actor TEXT PRIMARY KEY,
  public_key TEXT NOT NULL,
  recovery_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  created_ip_hash TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_actors_ip ON actors(created_ip_hash, created_at);
