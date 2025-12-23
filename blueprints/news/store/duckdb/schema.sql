-- News Schema

-- Users
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    about TEXT,
    karma BIGINT DEFAULT 0,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(LOWER(username));
CREATE INDEX IF NOT EXISTS idx_users_karma ON users(karma DESC);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    token VARCHAR UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    description TEXT,
    color VARCHAR DEFAULT '#666666',
    story_count BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(LOWER(name));

-- Stories
CREATE TABLE IF NOT EXISTS stories (
    id VARCHAR PRIMARY KEY,
    author_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    url VARCHAR,
    domain VARCHAR,
    text TEXT,
    text_html TEXT,
    score BIGINT DEFAULT 1,
    comment_count BIGINT DEFAULT 0,
    hot_score DOUBLE DEFAULT 0,
    is_removed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stories_hot ON stories(hot_score DESC);
CREATE INDEX IF NOT EXISTS idx_stories_new ON stories(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_stories_top ON stories(score DESC);
CREATE INDEX IF NOT EXISTS idx_stories_author ON stories(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_stories_domain ON stories(domain);

-- Story tags (many-to-many)
CREATE TABLE IF NOT EXISTS story_tags (
    story_id VARCHAR NOT NULL,
    tag_id VARCHAR NOT NULL,
    PRIMARY KEY (story_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_story_tags_tag ON story_tags(tag_id);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    story_id VARCHAR NOT NULL,
    parent_id VARCHAR,
    author_id VARCHAR NOT NULL,
    text TEXT NOT NULL,
    text_html TEXT,
    score BIGINT DEFAULT 1,
    depth INT DEFAULT 0,
    path VARCHAR NOT NULL,
    child_count BIGINT DEFAULT 0,
    is_removed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_story ON comments(story_id, path);
CREATE INDEX IF NOT EXISTS idx_comments_author ON comments(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);

-- Votes
CREATE TABLE IF NOT EXISTS votes (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    value INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique ON votes(user_id, target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_votes_target ON votes(target_type, target_id);

-- Seed mappings (for HN import)
CREATE TABLE IF NOT EXISTS seed_mappings (
    source VARCHAR NOT NULL,
    entity_type VARCHAR NOT NULL,
    external_id VARCHAR NOT NULL,
    local_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source, entity_type, external_id)
);

CREATE INDEX IF NOT EXISTS idx_seed_mappings_local ON seed_mappings(local_id);
