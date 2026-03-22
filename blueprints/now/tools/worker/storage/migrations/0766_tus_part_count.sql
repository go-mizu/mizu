-- Add part_count to tus_uploads so PATCH/DELETE don't need R2.list()
ALTER TABLE tus_uploads ADD COLUMN part_count INTEGER NOT NULL DEFAULT 0;
