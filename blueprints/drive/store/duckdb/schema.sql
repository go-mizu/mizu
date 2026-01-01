-- schema.sql
-- Drive file storage system database schema.
-- Supports full cloud storage functionality like Google Drive, Box, Dropbox.

-- ============================================================
-- Users and authentication
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id             VARCHAR PRIMARY KEY,
    email          VARCHAR UNIQUE NOT NULL,
    name           VARCHAR NOT NULL,
    password_hash  VARCHAR NOT NULL,
    avatar_url     VARCHAR,
    storage_quota  BIGINT NOT NULL DEFAULT 10737418240,  -- 10GB default
    storage_used   BIGINT NOT NULL DEFAULT 0,
    is_admin       BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id             VARCHAR PRIMARY KEY,
    user_id        VARCHAR NOT NULL,
    token_hash     VARCHAR NOT NULL,
    ip_address     VARCHAR,
    user_agent     VARCHAR,
    last_active_at TIMESTAMP,
    expires_at     TIMESTAMP NOT NULL,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Folders
-- ============================================================

CREATE TABLE IF NOT EXISTS folders (
    id          VARCHAR PRIMARY KEY,
    user_id     VARCHAR NOT NULL,
    parent_id   VARCHAR,
    name        VARCHAR NOT NULL,
    description VARCHAR,
    color       VARCHAR,
    is_starred  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    trashed_at  TIMESTAMP
);

-- ============================================================
-- Files
-- ============================================================

CREATE TABLE IF NOT EXISTS files (
    id          VARCHAR PRIMARY KEY,
    user_id     VARCHAR NOT NULL,
    parent_id   VARCHAR,
    name        VARCHAR NOT NULL,
    mime_type   VARCHAR NOT NULL,
    size        BIGINT NOT NULL,
    storage_key VARCHAR NOT NULL,
    checksum    VARCHAR,
    description VARCHAR,
    is_starred  BOOLEAN NOT NULL DEFAULT FALSE,
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    trashed_at  TIMESTAMP
);

-- ============================================================
-- File Versions
-- ============================================================

CREATE TABLE IF NOT EXISTS file_versions (
    id          VARCHAR PRIMARY KEY,
    file_id     VARCHAR NOT NULL,
    version     INTEGER NOT NULL,
    size        BIGINT NOT NULL,
    storage_key VARCHAR NOT NULL,
    checksum    VARCHAR,
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Shares
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id                 VARCHAR PRIMARY KEY,
    resource_type      VARCHAR NOT NULL,
    resource_id        VARCHAR NOT NULL,
    owner_id           VARCHAR NOT NULL,
    shared_with_id     VARCHAR,
    permission         VARCHAR NOT NULL DEFAULT 'viewer',
    link_token         VARCHAR,
    link_password_hash VARCHAR,
    expires_at         TIMESTAMP,
    download_limit     INTEGER,
    download_count     INTEGER NOT NULL DEFAULT 0,
    prevent_download   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR PRIMARY KEY,
    file_id     VARCHAR NOT NULL,
    user_id     VARCHAR NOT NULL,
    parent_id   VARCHAR,
    content     VARCHAR NOT NULL,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Activities (Audit Log)
-- ============================================================

CREATE TABLE IF NOT EXISTS activities (
    id            VARCHAR PRIMARY KEY,
    user_id       VARCHAR NOT NULL,
    action        VARCHAR NOT NULL,
    resource_type VARCHAR NOT NULL,
    resource_id   VARCHAR NOT NULL,
    resource_name VARCHAR,
    details       VARCHAR,
    ip_address    VARCHAR,
    user_agent    VARCHAR,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- User Settings
-- ============================================================

CREATE TABLE IF NOT EXISTS settings (
    user_id               VARCHAR PRIMARY KEY,
    theme                 VARCHAR NOT NULL DEFAULT 'system',
    language              VARCHAR NOT NULL DEFAULT 'en',
    timezone              VARCHAR NOT NULL DEFAULT 'UTC',
    list_view             VARCHAR NOT NULL DEFAULT 'list',
    sort_by               VARCHAR NOT NULL DEFAULT 'name',
    sort_order            VARCHAR NOT NULL DEFAULT 'asc',
    notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    email_notifications   BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
