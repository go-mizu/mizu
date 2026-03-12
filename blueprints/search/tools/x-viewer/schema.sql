-- Profiles: PK lookup by lowercase username
CREATE TABLE IF NOT EXISTS profiles (
  username TEXT PRIMARY KEY,
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL
);

-- Tweet detail + replies, composite PK for cursor pagination
CREATE TABLE IF NOT EXISTS tweets (
  tweet_id TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL,
  PRIMARY KEY (tweet_id, cursor)
);

-- Articles: body stored separately for direct access
CREATE TABLE IF NOT EXISTS articles (
  tweet_id TEXT PRIMARY KEY,
  tweet_data TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL
);

-- User timelines, composite PK
CREATE TABLE IF NOT EXISTS timelines (
  username TEXT NOT NULL,
  tab TEXT NOT NULL DEFAULT 'tweets',
  cursor TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL,
  PRIMARY KEY (username, tab, cursor)
);

-- Search results, composite PK
CREATE TABLE IF NOT EXISTS searches (
  query TEXT NOT NULL,
  mode TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL,
  PRIMARY KEY (query, mode, cursor)
);

-- List metadata
CREATE TABLE IF NOT EXISTS lists (
  list_id TEXT PRIMARY KEY,
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL
);

-- List content (tweets or members), composite PK
CREATE TABLE IF NOT EXISTS list_content (
  list_id TEXT NOT NULL,
  content_type TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL,
  PRIMARY KEY (list_id, content_type, cursor)
);

-- Follow lists (followers/following), composite PK
CREATE TABLE IF NOT EXISTS follows (
  username TEXT NOT NULL,
  follow_type TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL,
  fetched_at INTEGER NOT NULL DEFAULT 0,
  expires_at INTEGER NOT NULL,
  PRIMARY KEY (username, follow_type, cursor)
);

-- Drop old generic cache table
DROP TABLE IF EXISTS cache;

-- Migration: add fetched_at to existing tables (safe to run multiple times via try/catch in code)
-- ALTER TABLE profiles ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE tweets ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE articles ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE timelines ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE searches ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE lists ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE list_content ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
-- ALTER TABLE follows ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
