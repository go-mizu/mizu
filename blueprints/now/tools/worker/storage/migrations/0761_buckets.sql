-- 0761: Supabase-inspired API redesign
-- Adds buckets, signed_urls; removes shares, public_links, spaces tables

-- ── New: Buckets ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS buckets (
  id                 TEXT PRIMARY KEY,
  owner              TEXT NOT NULL,
  name               TEXT NOT NULL,
  public             INTEGER NOT NULL DEFAULT 0,
  file_size_limit    INTEGER DEFAULT NULL,
  allowed_mime_types TEXT DEFAULT NULL,   -- JSON array, e.g. '["image/png","image/jpeg"]'
  created_at         INTEGER NOT NULL,
  updated_at         INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_buckets_owner_name ON buckets(owner, name);

-- ── New: Signed URLs (replaces public_links + presign) ──────────────────
CREATE TABLE IF NOT EXISTS signed_urls (
  id         TEXT PRIMARY KEY,
  owner      TEXT NOT NULL,
  bucket_id  TEXT NOT NULL REFERENCES buckets(id),
  path       TEXT NOT NULL,
  token      TEXT NOT NULL UNIQUE,
  type       TEXT NOT NULL CHECK(type IN ('download','upload')),
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_signed_urls_token ON signed_urls(token);
CREATE INDEX IF NOT EXISTS idx_signed_urls_expires ON signed_urls(expires_at);

-- ── Migrate objects: add bucket_id ──────────────────────────────────────
-- Step 1: Add column
ALTER TABLE objects ADD COLUMN bucket_id TEXT DEFAULT NULL;

-- Step 2: Recreate objects table without deprecated columns
-- (SQLite < 3.35 doesn't support DROP COLUMN; we recreate)
CREATE TABLE objects_v2 (
  id           TEXT PRIMARY KEY,
  owner        TEXT NOT NULL,
  bucket_id    TEXT NOT NULL,
  path         TEXT NOT NULL,
  name         TEXT NOT NULL,
  content_type TEXT DEFAULT '',
  size         INTEGER DEFAULT 0,
  r2_key       TEXT DEFAULT '',
  metadata     TEXT DEFAULT '{}',     -- JSON key-value pairs
  accessed_at  INTEGER DEFAULT NULL,
  created_at   INTEGER NOT NULL,
  updated_at   INTEGER NOT NULL
);

-- Note: migration script should INSERT INTO objects_v2 SELECT ... FROM objects
-- after creating default buckets per owner. Run as part of deploy script.

-- ── Drop deprecated tables ──────────────────────────────────────────────
DROP TABLE IF EXISTS shares;
DROP TABLE IF EXISTS public_links;
DROP TABLE IF EXISTS spaces;
DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS space_sections;
DROP TABLE IF EXISTS space_items;
DROP TABLE IF EXISTS space_activity;

-- ── Indexes for objects_v2 ──────────────────────────────────────────────
CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_v2_bucket_path ON objects_v2(bucket_id, path);
CREATE INDEX IF NOT EXISTS idx_objects_v2_owner ON objects_v2(owner);
CREATE INDEX IF NOT EXISTS idx_objects_v2_bucket ON objects_v2(bucket_id);
