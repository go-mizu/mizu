-- TUS resumable uploads: tracks in-progress chunked uploads
CREATE TABLE IF NOT EXISTS tus_uploads (
  id            TEXT PRIMARY KEY,
  owner         TEXT NOT NULL,
  bucket_id     TEXT NOT NULL,
  path          TEXT NOT NULL,
  upload_length INTEGER NOT NULL,
  upload_offset INTEGER NOT NULL DEFAULT 0,
  content_type  TEXT DEFAULT '',
  metadata      TEXT DEFAULT '{}',
  upsert        INTEGER NOT NULL DEFAULT 0,
  expires_at    INTEGER NOT NULL,
  created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tus_uploads_owner ON tus_uploads(owner);
CREATE INDEX IF NOT EXISTS idx_tus_uploads_expires ON tus_uploads(expires_at);
