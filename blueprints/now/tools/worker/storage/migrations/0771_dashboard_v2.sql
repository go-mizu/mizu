-- 0771: Dashboard v2 enhancements
-- Add last_accessed tracking for API keys
ALTER TABLE api_keys ADD COLUMN last_accessed INTEGER;

-- Add view counting for share links
ALTER TABLE share_links ADD COLUMN views INTEGER NOT NULL DEFAULT 0;
