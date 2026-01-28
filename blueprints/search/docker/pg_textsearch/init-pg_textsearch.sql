-- Initialization script for pg_textsearch extension
-- This runs automatically when the container starts

-- Enable the pg_textsearch extension
CREATE EXTENSION IF NOT EXISTS pg_textsearch;

-- Log that initialization is complete
DO $$
BEGIN
    RAISE NOTICE 'pg_textsearch extension enabled successfully';
END $$;
