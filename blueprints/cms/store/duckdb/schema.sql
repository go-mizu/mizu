-- CMS Core Schema
-- This schema provides the foundation tables. Collection-specific tables are created dynamically.

-- Users table (for auth-enabled collections)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(26) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255),
    salt VARCHAR(32),

    -- Profile fields
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    roles TEXT, -- JSON array of roles

    -- Auth fields
    login_attempts INTEGER DEFAULT 0,
    lock_until TIMESTAMP,
    reset_password_token VARCHAR(255),
    reset_password_expiration TIMESTAMP,
    verification_token VARCHAR(255),
    verified BOOLEAN DEFAULT FALSE,

    -- API Key
    api_key VARCHAR(255),
    api_key_index VARCHAR(64), -- Hashed for lookup

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_api_key_index ON users(api_key_index);

-- Sessions table
CREATE TABLE IF NOT EXISTS _sessions (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    collection VARCHAR(255) NOT NULL,
    token TEXT NOT NULL,
    refresh_token TEXT,
    user_agent TEXT,
    ip VARCHAR(45),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON _sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON _sessions(token);

-- Globals table
CREATE TABLE IF NOT EXISTS _globals (
    id VARCHAR(26) PRIMARY KEY,
    slug VARCHAR(255) NOT NULL UNIQUE,
    data TEXT NOT NULL, -- JSON blob
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_globals_slug ON _globals(slug);

-- Global versions
CREATE TABLE IF NOT EXISTS _globals_versions (
    id VARCHAR(26) PRIMARY KEY,
    global_slug VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL,
    snapshot TEXT NOT NULL, -- JSON blob
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(26)
);

CREATE INDEX IF NOT EXISTS idx_globals_versions_slug ON _globals_versions(global_slug);

-- User preferences
CREATE TABLE IF NOT EXISTS _preferences (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL, -- JSON
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_preferences_user_key ON _preferences(user_id, key);

-- Media/Uploads table
CREATE TABLE IF NOT EXISTS media (
    id VARCHAR(26) PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    filesize INTEGER NOT NULL,
    width INTEGER,
    height INTEGER,
    focal_x DOUBLE,
    focal_y DOUBLE,
    alt VARCHAR(255),
    caption TEXT,

    -- Image sizes (JSON map of size name -> {width, height, path})
    sizes TEXT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_filename ON media(filename);
CREATE INDEX IF NOT EXISTS idx_media_mime_type ON media(mime_type);

-- Document locks
CREATE TABLE IF NOT EXISTS _locks (
    id VARCHAR(26) PRIMARY KEY,
    collection VARCHAR(255) NOT NULL,
    document_id VARCHAR(26) NOT NULL,
    user_id VARCHAR(26) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    UNIQUE(collection, document_id)
);

CREATE INDEX IF NOT EXISTS idx_locks_collection_doc ON _locks(collection, document_id);

-- Migrations tracking
CREATE TABLE IF NOT EXISTS _migrations (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    batch INTEGER NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Pages collection (example - normally created dynamically)
CREATE TABLE IF NOT EXISTS pages (
    id VARCHAR(26) PRIMARY KEY,
    title VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    content TEXT, -- Rich text as JSON
    parent VARCHAR(26),
    featured_image VARCHAR(26),
    meta TEXT, -- JSON for SEO group
    status VARCHAR(50) DEFAULT 'draft',
    published_at TIMESTAMP,

    -- Version tracking
    _status VARCHAR(20) DEFAULT 'draft',
    _version INTEGER DEFAULT 1,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pages_slug ON pages(slug);
CREATE INDEX IF NOT EXISTS idx_pages_status ON pages(status);
CREATE INDEX IF NOT EXISTS idx_pages_parent ON pages(parent);

-- Pages versions
CREATE TABLE IF NOT EXISTS pages_versions (
    id VARCHAR(26) PRIMARY KEY,
    parent VARCHAR(26) NOT NULL,
    version INTEGER NOT NULL,
    snapshot TEXT NOT NULL,
    published BOOLEAN DEFAULT FALSE,
    autosave BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(26)
);

CREATE INDEX IF NOT EXISTS idx_pages_versions_parent ON pages_versions(parent);

-- Posts collection
CREATE TABLE IF NOT EXISTS posts (
    id VARCHAR(26) PRIMARY KEY,
    title VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    content TEXT,
    excerpt TEXT,
    author VARCHAR(26),
    categories TEXT, -- JSON array of IDs
    tags TEXT, -- JSON array of IDs
    featured_image VARCHAR(26),
    meta TEXT, -- JSON for SEO group
    status VARCHAR(50) DEFAULT 'draft',
    published_at TIMESTAMP,

    -- Version tracking
    _status VARCHAR(20) DEFAULT 'draft',
    _version INTEGER DEFAULT 1,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug);
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author);
CREATE INDEX IF NOT EXISTS idx_posts_published_at ON posts(published_at);

-- Posts versions
CREATE TABLE IF NOT EXISTS posts_versions (
    id VARCHAR(26) PRIMARY KEY,
    parent VARCHAR(26) NOT NULL,
    version INTEGER NOT NULL,
    snapshot TEXT NOT NULL,
    published BOOLEAN DEFAULT FALSE,
    autosave BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(26)
);

CREATE INDEX IF NOT EXISTS idx_posts_versions_parent ON posts_versions(parent);

-- Categories collection
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    parent VARCHAR(26),

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent);

-- Tags collection
CREATE TABLE IF NOT EXISTS tags (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tags_slug ON tags(slug);
