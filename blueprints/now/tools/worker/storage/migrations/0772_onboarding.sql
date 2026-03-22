-- Add username (user-chosen, immutable) and display_name to actors
ALTER TABLE actors ADD COLUMN username TEXT;
ALTER TABLE actors ADD COLUMN display_name TEXT;
CREATE UNIQUE INDEX idx_actors_username ON actors(username) WHERE username IS NOT NULL;
