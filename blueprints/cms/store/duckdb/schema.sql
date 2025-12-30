-- schema.sql
-- Modern CMS schema inspired by WordPress and Ghost.
-- Clean, elegant data model for content management.
--
-- Core mental model:
--   Users → Posts/Pages → Categories/Tags → Comments
-- Content:
--   Posts (blog), Pages (static), Collections (custom types)
-- Media:
--   Media library with metadata and relationships
-- Extensibility:
--   Custom collections with dynamic fields

-- ============================================================
-- Users and authentication
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    slug          VARCHAR UNIQUE NOT NULL,
    bio           TEXT,
    avatar_url    VARCHAR,
    role          VARCHAR NOT NULL DEFAULT 'author',
    status        VARCHAR NOT NULL DEFAULT 'active',
    meta          VARCHAR,
    last_login_at TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id            VARCHAR PRIMARY KEY,
    user_id       VARCHAR NOT NULL,
    refresh_token VARCHAR,
    user_agent    TEXT,
    ip_address    VARCHAR,
    expires_at    TIMESTAMP NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Posts (blog content)
-- ============================================================

CREATE TABLE IF NOT EXISTS posts (
    id                VARCHAR PRIMARY KEY,
    author_id         VARCHAR NOT NULL,
    title             VARCHAR NOT NULL,
    slug              VARCHAR UNIQUE NOT NULL,
    excerpt           TEXT,
    content           TEXT,
    content_format    VARCHAR DEFAULT 'markdown',
    featured_image_id VARCHAR,
    status            VARCHAR NOT NULL DEFAULT 'draft',
    visibility        VARCHAR NOT NULL DEFAULT 'public',
    password          VARCHAR,
    published_at      TIMESTAMP,
    scheduled_at      TIMESTAMP,
    meta              VARCHAR,
    reading_time      INTEGER,
    word_count        INTEGER,
    allow_comments    BOOLEAN DEFAULT true,
    is_featured       BOOLEAN DEFAULT false,
    is_sticky         BOOLEAN DEFAULT false,
    sort_order        INTEGER DEFAULT 0,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Pages (static content)
-- ============================================================

CREATE TABLE IF NOT EXISTS pages (
    id                VARCHAR PRIMARY KEY,
    author_id         VARCHAR NOT NULL,
    parent_id         VARCHAR,
    title             VARCHAR NOT NULL,
    slug              VARCHAR NOT NULL,
    content           TEXT,
    content_format    VARCHAR DEFAULT 'markdown',
    featured_image_id VARCHAR,
    template          VARCHAR,
    status            VARCHAR NOT NULL DEFAULT 'draft',
    visibility        VARCHAR NOT NULL DEFAULT 'public',
    meta              VARCHAR,
    sort_order        INTEGER DEFAULT 0,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Revisions (version history)
-- ============================================================

CREATE TABLE IF NOT EXISTS revisions (
    id              VARCHAR PRIMARY KEY,
    entity_type     VARCHAR NOT NULL,
    entity_id       VARCHAR NOT NULL,
    author_id       VARCHAR NOT NULL,
    title           VARCHAR,
    content         TEXT,
    meta            VARCHAR,
    revision_number INTEGER NOT NULL,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Categories (hierarchical taxonomy)
-- ============================================================

CREATE TABLE IF NOT EXISTS categories (
    id                VARCHAR PRIMARY KEY,
    parent_id         VARCHAR,
    name              VARCHAR NOT NULL,
    slug              VARCHAR UNIQUE NOT NULL,
    description       TEXT,
    featured_image_id VARCHAR,
    meta              VARCHAR,
    sort_order        INTEGER DEFAULT 0,
    post_count        INTEGER DEFAULT 0,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Tags (flat taxonomy)
-- ============================================================

CREATE TABLE IF NOT EXISTS tags (
    id                VARCHAR PRIMARY KEY,
    name              VARCHAR NOT NULL,
    slug              VARCHAR UNIQUE NOT NULL,
    description       TEXT,
    featured_image_id VARCHAR,
    meta              VARCHAR,
    post_count        INTEGER DEFAULT 0,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Post-Category relationship
-- ============================================================

CREATE TABLE IF NOT EXISTS post_categories (
    post_id     VARCHAR NOT NULL,
    category_id VARCHAR NOT NULL,
    sort_order  INTEGER DEFAULT 0,
    PRIMARY KEY (post_id, category_id)
);

-- ============================================================
-- Post-Tag relationship
-- ============================================================

CREATE TABLE IF NOT EXISTS post_tags (
    post_id VARCHAR NOT NULL,
    tag_id  VARCHAR NOT NULL,
    PRIMARY KEY (post_id, tag_id)
);

-- ============================================================
-- Media library
-- ============================================================

CREATE TABLE IF NOT EXISTS media (
    id                VARCHAR PRIMARY KEY,
    uploader_id       VARCHAR NOT NULL,
    filename          VARCHAR NOT NULL,
    original_filename VARCHAR NOT NULL,
    mime_type         VARCHAR NOT NULL,
    file_size         BIGINT NOT NULL,
    storage_path      VARCHAR NOT NULL,
    storage_provider  VARCHAR DEFAULT 'local',
    url               VARCHAR NOT NULL,
    alt_text          VARCHAR,
    caption           TEXT,
    title             VARCHAR,
    description       TEXT,
    width             INTEGER,
    height            INTEGER,
    duration          INTEGER,
    meta              VARCHAR,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Post-Media relationship
-- ============================================================

CREATE TABLE IF NOT EXISTS post_media (
    post_id    VARCHAR NOT NULL,
    media_id   VARCHAR NOT NULL,
    sort_order INTEGER DEFAULT 0,
    PRIMARY KEY (post_id, media_id)
);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id           VARCHAR PRIMARY KEY,
    post_id      VARCHAR NOT NULL,
    parent_id    VARCHAR,
    author_id    VARCHAR,
    author_name  VARCHAR,
    author_email VARCHAR,
    author_url   VARCHAR,
    content      TEXT NOT NULL,
    status       VARCHAR NOT NULL DEFAULT 'pending',
    ip_address   VARCHAR,
    user_agent   TEXT,
    likes_count  INTEGER DEFAULT 0,
    meta         VARCHAR,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Collections (custom content types)
-- ============================================================

CREATE TABLE IF NOT EXISTS collections (
    id             VARCHAR PRIMARY KEY,
    name           VARCHAR NOT NULL,
    slug           VARCHAR UNIQUE NOT NULL,
    description    TEXT,
    icon           VARCHAR,
    singular_label VARCHAR,
    plural_label   VARCHAR,
    supports       VARCHAR,
    meta           VARCHAR,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Collection fields (schema definition)
-- ============================================================

CREATE TABLE IF NOT EXISTS collection_fields (
    id            VARCHAR PRIMARY KEY,
    collection_id VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    slug          VARCHAR NOT NULL,
    field_type    VARCHAR NOT NULL,
    description   TEXT,
    placeholder   VARCHAR,
    default_value TEXT,
    options       VARCHAR,
    validation    VARCHAR,
    sort_order    INTEGER DEFAULT 0,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Collection items (custom content)
-- ============================================================

CREATE TABLE IF NOT EXISTS collection_items (
    id            VARCHAR PRIMARY KEY,
    collection_id VARCHAR NOT NULL,
    author_id     VARCHAR NOT NULL,
    title         VARCHAR,
    slug          VARCHAR,
    status        VARCHAR NOT NULL DEFAULT 'draft',
    data          VARCHAR NOT NULL,
    meta          VARCHAR,
    published_at  TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Settings (key-value store)
-- ============================================================

CREATE TABLE IF NOT EXISTS settings (
    id          VARCHAR PRIMARY KEY,
    key         VARCHAR UNIQUE NOT NULL,
    value       TEXT,
    value_type  VARCHAR DEFAULT 'string',
    group_name  VARCHAR,
    description TEXT,
    is_public   BOOLEAN DEFAULT false,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Navigation menus
-- ============================================================

CREATE TABLE IF NOT EXISTS menus (
    id         VARCHAR PRIMARY KEY,
    name       VARCHAR NOT NULL,
    slug       VARCHAR UNIQUE NOT NULL,
    location   VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Menu items
-- ============================================================

CREATE TABLE IF NOT EXISTS menu_items (
    id         VARCHAR PRIMARY KEY,
    menu_id    VARCHAR NOT NULL,
    parent_id  VARCHAR,
    title      VARCHAR NOT NULL,
    url        VARCHAR,
    target     VARCHAR DEFAULT '_self',
    link_type  VARCHAR,
    link_id    VARCHAR,
    css_class  VARCHAR,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Webhooks
-- ============================================================

CREATE TABLE IF NOT EXISTS webhooks (
    id                VARCHAR PRIMARY KEY,
    name              VARCHAR NOT NULL,
    url               VARCHAR NOT NULL,
    secret            VARCHAR,
    events            VARCHAR NOT NULL,
    status            VARCHAR DEFAULT 'active',
    last_triggered_at TIMESTAMP,
    failure_count     INTEGER DEFAULT 0,
    meta              VARCHAR,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- API keys
-- ============================================================

CREATE TABLE IF NOT EXISTS api_keys (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR,
    name         VARCHAR NOT NULL,
    key_hash     VARCHAR NOT NULL,
    key_prefix   VARCHAR NOT NULL,
    permissions  VARCHAR,
    last_used_at TIMESTAMP,
    expires_at   TIMESTAMP,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
