-- Drive Database Schema

-- =============================================================================
-- ACCOUNTS & AUTHENTICATION
-- =============================================================================

CREATE TABLE IF NOT EXISTS accounts (
    id            VARCHAR PRIMARY KEY,
    username      VARCHAR(50) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(255),
    avatar_url    VARCHAR(500),
    storage_quota BIGINT DEFAULT 16106127360,
    storage_used  BIGINT DEFAULT 0,
    is_admin      BOOLEAN DEFAULT FALSE,
    is_suspended  BOOLEAN DEFAULT FALSE,
    preferences   JSON,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username);
CREATE INDEX IF NOT EXISTS idx_accounts_email ON accounts(email);

CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    token      VARCHAR(255) UNIQUE NOT NULL,
    user_agent VARCHAR(500),
    ip_address VARCHAR(45),
    last_used  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_account_id ON sessions(account_id);

-- =============================================================================
-- FOLDERS
-- =============================================================================

CREATE TABLE IF NOT EXISTS folders (
    id         VARCHAR PRIMARY KEY,
    owner_id   VARCHAR NOT NULL,
    parent_id  VARCHAR,
    name       VARCHAR(255) NOT NULL,
    path       VARCHAR(4096) NOT NULL,
    depth      INTEGER DEFAULT 0,
    color      VARCHAR(7),
    is_root    BOOLEAN DEFAULT FALSE,
    starred    BOOLEAN DEFAULT FALSE,
    trashed    BOOLEAN DEFAULT FALSE,
    trashed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_folders_owner_id ON folders(owner_id);
CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);
CREATE INDEX IF NOT EXISTS idx_folders_trashed ON folders(trashed);
CREATE INDEX IF NOT EXISTS idx_folders_starred ON folders(starred);
CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_unique_name ON folders(owner_id, parent_id, name);

-- =============================================================================
-- FILES
-- =============================================================================

CREATE TABLE IF NOT EXISTS files (
    id              VARCHAR PRIMARY KEY,
    owner_id        VARCHAR NOT NULL,
    folder_id       VARCHAR,
    name            VARCHAR(255) NOT NULL,
    path            VARCHAR(4096) NOT NULL,
    size            BIGINT NOT NULL DEFAULT 0,
    mime_type       VARCHAR(255),
    extension       VARCHAR(50),
    storage_path    VARCHAR(1024) NOT NULL,
    checksum_sha256 VARCHAR(64),
    has_thumbnail   BOOLEAN DEFAULT FALSE,
    thumbnail_path  VARCHAR(1024),
    starred         BOOLEAN DEFAULT FALSE,
    trashed         BOOLEAN DEFAULT FALSE,
    trashed_at      TIMESTAMP,
    locked          BOOLEAN DEFAULT FALSE,
    locked_by       VARCHAR,
    locked_at       TIMESTAMP,
    lock_expires_at TIMESTAMP,
    version_count   INTEGER DEFAULT 1,
    current_version INTEGER DEFAULT 1,
    description     TEXT,
    metadata        JSON,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    accessed_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_files_owner_id ON files(owner_id);
CREATE INDEX IF NOT EXISTS idx_files_folder_id ON files(folder_id);
CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
CREATE INDEX IF NOT EXISTS idx_files_mime_type ON files(mime_type);
CREATE INDEX IF NOT EXISTS idx_files_trashed ON files(trashed);
CREATE INDEX IF NOT EXISTS idx_files_starred ON files(starred);
CREATE INDEX IF NOT EXISTS idx_files_accessed_at ON files(accessed_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_unique_name ON files(owner_id, folder_id, name);

-- =============================================================================
-- FILE VERSIONS
-- =============================================================================

CREATE TABLE IF NOT EXISTS file_versions (
    id              VARCHAR PRIMARY KEY,
    file_id         VARCHAR NOT NULL,
    version_number  INTEGER NOT NULL,
    size            BIGINT NOT NULL,
    storage_path    VARCHAR(1024) NOT NULL,
    checksum_sha256 VARCHAR(64),
    uploaded_by     VARCHAR NOT NULL,
    comment         TEXT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_file_versions_file_id ON file_versions(file_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_file_versions_unique ON file_versions(file_id, version_number);

-- =============================================================================
-- CHUNKED UPLOADS
-- =============================================================================

CREATE TABLE IF NOT EXISTS chunked_uploads (
    id           VARCHAR PRIMARY KEY,
    account_id   VARCHAR NOT NULL,
    folder_id    VARCHAR,
    filename     VARCHAR(255) NOT NULL,
    total_size   BIGINT NOT NULL,
    chunk_size   INTEGER NOT NULL,
    total_chunks INTEGER NOT NULL,
    mime_type    VARCHAR(255),
    status       VARCHAR(20) DEFAULT 'pending',
    temp_path    VARCHAR(1024),
    expires_at   TIMESTAMP NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chunked_uploads_account_id ON chunked_uploads(account_id);
CREATE INDEX IF NOT EXISTS idx_chunked_uploads_status ON chunked_uploads(status);

CREATE TABLE IF NOT EXISTS upload_chunks (
    upload_id    VARCHAR NOT NULL,
    chunk_index  INTEGER NOT NULL,
    size         INTEGER NOT NULL,
    checksum     VARCHAR(64),
    storage_path VARCHAR(1024) NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(upload_id, chunk_index)
);

-- =============================================================================
-- SHARING & PERMISSIONS
-- =============================================================================

CREATE TABLE IF NOT EXISTS shares (
    id          VARCHAR PRIMARY KEY,
    item_id     VARCHAR NOT NULL,
    item_type   VARCHAR(10) NOT NULL,
    owner_id    VARCHAR NOT NULL,
    shared_with VARCHAR,
    permission  VARCHAR(20) NOT NULL,
    notify      BOOLEAN DEFAULT TRUE,
    message     TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shares_item ON shares(item_id, item_type);
CREATE INDEX IF NOT EXISTS idx_shares_shared_with ON shares(shared_with);
CREATE INDEX IF NOT EXISTS idx_shares_owner_id ON shares(owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_shares_unique ON shares(item_id, item_type, shared_with);

CREATE TABLE IF NOT EXISTS share_links (
    id             VARCHAR PRIMARY KEY,
    item_id        VARCHAR NOT NULL,
    item_type      VARCHAR(10) NOT NULL,
    owner_id       VARCHAR NOT NULL,
    token          VARCHAR(64) UNIQUE NOT NULL,
    permission     VARCHAR(20) NOT NULL DEFAULT 'viewer',
    password_hash  VARCHAR(255),
    expires_at     TIMESTAMP,
    download_limit INTEGER,
    download_count INTEGER DEFAULT 0,
    allow_download BOOLEAN DEFAULT TRUE,
    disabled       BOOLEAN DEFAULT FALSE,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    accessed_at    TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_share_links_token ON share_links(token);
CREATE INDEX IF NOT EXISTS idx_share_links_item ON share_links(item_id, item_type);

-- =============================================================================
-- ORGANIZATION: TAGS
-- =============================================================================

CREATE TABLE IF NOT EXISTS tags (
    id         VARCHAR PRIMARY KEY,
    owner_id   VARCHAR NOT NULL,
    name       VARCHAR(50) NOT NULL,
    color      VARCHAR(7),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tags_owner_id ON tags(owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_unique ON tags(owner_id, name);

CREATE TABLE IF NOT EXISTS item_tags (
    item_id   VARCHAR NOT NULL,
    item_type VARCHAR(10) NOT NULL,
    tag_id    VARCHAR NOT NULL,
    added_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(item_id, item_type, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_item_tags_tag_id ON item_tags(tag_id);

-- =============================================================================
-- COMMENTS
-- =============================================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR PRIMARY KEY,
    file_id     VARCHAR NOT NULL,
    author_id   VARCHAR NOT NULL,
    parent_id   VARCHAR,
    content     TEXT NOT NULL,
    resolved    BOOLEAN DEFAULT FALSE,
    resolved_by VARCHAR,
    resolved_at TIMESTAMP,
    edited      BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_file_id ON comments(file_id);
CREATE INDEX IF NOT EXISTS idx_comments_author_id ON comments(author_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

-- =============================================================================
-- ACTIVITY LOG
-- =============================================================================

CREATE TABLE IF NOT EXISTS activities (
    id          VARCHAR PRIMARY KEY,
    account_id  VARCHAR NOT NULL,
    action      VARCHAR(50) NOT NULL,
    item_id     VARCHAR,
    item_type   VARCHAR(10),
    item_name   VARCHAR(255),
    target_id   VARCHAR,
    target_name VARCHAR(255),
    details     JSON,
    ip_address  VARCHAR(45),
    user_agent  VARCHAR(500),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activities_account_id ON activities(account_id);
CREATE INDEX IF NOT EXISTS idx_activities_item_id ON activities(item_id);
CREATE INDEX IF NOT EXISTS idx_activities_action ON activities(action);
CREATE INDEX IF NOT EXISTS idx_activities_created_at ON activities(created_at DESC);

-- =============================================================================
-- NOTIFICATIONS
-- =============================================================================

CREATE TABLE IF NOT EXISTS notifications (
    id         VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    type       VARCHAR(50) NOT NULL,
    actor_id   VARCHAR,
    item_id    VARCHAR,
    item_type  VARCHAR(10),
    item_name  VARCHAR(255),
    message    TEXT,
    read       BOOLEAN DEFAULT FALSE,
    read_at    TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_account_id ON notifications(account_id);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

-- =============================================================================
-- METADATA
-- =============================================================================

CREATE TABLE IF NOT EXISTS meta (
    k VARCHAR PRIMARY KEY,
    v VARCHAR NOT NULL
);

INSERT INTO meta (k, v) VALUES ('schema_version', '1') ON CONFLICT DO NOTHING;
