-- schema.sql
-- Workspace - Notion-style collaborative workspace schema (DuckDB oriented)

-- ============================================================
-- Users and authentication
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR UNIQUE NOT NULL,
    name          VARCHAR NOT NULL,
    avatar_url    VARCHAR,
    password_hash VARCHAR NOT NULL,
    settings      JSON DEFAULT '{}',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR PRIMARY KEY,
    user_id    VARCHAR NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Workspaces and membership
-- ============================================================

CREATE TABLE IF NOT EXISTS workspaces (
    id         VARCHAR PRIMARY KEY,
    name       VARCHAR NOT NULL,
    slug       VARCHAR UNIQUE NOT NULL,
    icon       VARCHAR,
    domain     VARCHAR,
    plan       VARCHAR DEFAULT 'free',
    settings   JSON DEFAULT '{}',
    owner_id   VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS members (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    user_id      VARCHAR NOT NULL,
    role         VARCHAR NOT NULL DEFAULT 'member',
    joined_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    invited_by   VARCHAR,
    UNIQUE(workspace_id, user_id)
);

CREATE TABLE IF NOT EXISTS invites (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    email        VARCHAR NOT NULL,
    role         VARCHAR NOT NULL DEFAULT 'member',
    token        VARCHAR UNIQUE NOT NULL,
    expires_at   TIMESTAMP NOT NULL,
    created_by   VARCHAR,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Pages (also used as Database Rows)
-- ============================================================
-- A database row is a page with pages.database_id set.
-- pages.properties stores either:
-- - page properties (for normal pages)
-- - row properties (for database rows), keyed by database schema properties

CREATE TABLE IF NOT EXISTS pages (
    id            VARCHAR PRIMARY KEY,
    workspace_id  VARCHAR NOT NULL,

    -- hierarchy
    parent_id     VARCHAR,
    parent_type   VARCHAR NOT NULL DEFAULT 'workspace', -- workspace | page | database

    -- database row support (Notion-style)
    database_id   VARCHAR, -- nullable; when set, this page is a database row
    row_position  BIGINT DEFAULT 0, -- ordering inside a database (or view default)

    -- display
    title         VARCHAR DEFAULT '',
    icon          VARCHAR,
    cover         VARCHAR,
    cover_y       DOUBLE DEFAULT 0.5,

    -- content + metadata
    properties    JSON DEFAULT '{}',
    is_template   BOOLEAN DEFAULT FALSE,
    is_archived   BOOLEAN DEFAULT FALSE,

    created_by    VARCHAR NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by    VARCHAR NOT NULL,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Blocks (page content)
-- ============================================================

CREATE TABLE IF NOT EXISTS blocks (
    id         VARCHAR PRIMARY KEY,
    page_id    VARCHAR NOT NULL,
    parent_id  VARCHAR,               -- nesting within blocks
    type       VARCHAR NOT NULL,       -- paragraph, heading, todo, etc.
    content    JSON DEFAULT '{}',
    position   BIGINT DEFAULT 0,       -- ordering within a parent
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Databases
-- ============================================================

CREATE TABLE IF NOT EXISTS databases (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    page_id      VARCHAR NOT NULL,      -- the page that hosts this database
    title        VARCHAR DEFAULT 'Untitled',
    description  JSON DEFAULT '[]',
    icon         VARCHAR,
    cover        VARCHAR,
    is_inline    BOOLEAN DEFAULT FALSE,
    properties   JSON DEFAULT '[]',     -- schema: list of property definitions
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by   VARCHAR NOT NULL,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Views
-- ============================================================

CREATE TABLE IF NOT EXISTS views (
    id          VARCHAR PRIMARY KEY,
    database_id VARCHAR NOT NULL,
    name        VARCHAR NOT NULL,
    type        VARCHAR NOT NULL DEFAULT 'table', -- table | board | calendar | list | gallery
    filter      JSON,
    sorts       JSON DEFAULT '[]',
    properties  JSON DEFAULT '[]',
    group_by    VARCHAR,
    calendar_by VARCHAR,
    position    BIGINT DEFAULT 0,
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Comments (polymorphic)
-- ============================================================
-- target_type: page | block | database_row
-- target_id:   pages.id (for page/database_row) or blocks.id (for block)

CREATE TABLE IF NOT EXISTS comments (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    target_type  VARCHAR NOT NULL,
    target_id    VARCHAR NOT NULL,
    parent_id    VARCHAR,              -- reply threading
    content      JSON NOT NULL,
    author_id    VARCHAR NOT NULL,
    is_resolved  BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Sharing
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id          VARCHAR PRIMARY KEY,
    page_id     VARCHAR NOT NULL,
    type        VARCHAR NOT NULL,            -- public_link | workspace | user
    permission  VARCHAR NOT NULL DEFAULT 'read',
    user_id     VARCHAR,
    token       VARCHAR UNIQUE,
    password    VARCHAR,
    expires_at  TIMESTAMP,
    domain      VARCHAR,
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Favorites
-- ============================================================

CREATE TABLE IF NOT EXISTS favorites (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    page_id      VARCHAR NOT NULL,
    workspace_id VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, page_id)
);

-- ============================================================
-- History
-- ============================================================

CREATE TABLE IF NOT EXISTS revisions (
    id         VARCHAR PRIMARY KEY,
    page_id    VARCHAR NOT NULL,
    version    INTEGER NOT NULL,
    title      VARCHAR,
    blocks     JSON,
    properties JSON,
    author_id  VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS activities (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    page_id      VARCHAR,
    block_id     VARCHAR,
    actor_id     VARCHAR NOT NULL,
    action       VARCHAR NOT NULL,
    details      JSON,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Notifications
-- ============================================================

CREATE TABLE IF NOT EXISTS notifications (
    id         VARCHAR PRIMARY KEY,
    user_id    VARCHAR NOT NULL,
    type       VARCHAR NOT NULL,
    title      VARCHAR NOT NULL,
    body       VARCHAR,
    page_id    VARCHAR,
    actor_id   VARCHAR,
    is_read    BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Templates
-- ============================================================

CREATE TABLE IF NOT EXISTS templates (
    id           VARCHAR PRIMARY KEY,
    name         VARCHAR NOT NULL,
    description  VARCHAR,
    category     VARCHAR,
    preview      VARCHAR,
    page_id      VARCHAR NOT NULL,
    is_system    BOOLEAN DEFAULT FALSE,
    workspace_id VARCHAR,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Search (page access tracking)
-- ============================================================

CREATE TABLE IF NOT EXISTS page_access (
    user_id     VARCHAR NOT NULL,
    page_id     VARCHAR NOT NULL,
    accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(user_id, page_id)
);

-- ============================================================
-- Indexes (trimmed for DuckDB)
-- ============================================================
-- Keep only the ones that align with very common access paths:
-- - workspace scoping
-- - hierarchy navigation
-- - ordering retrieval
-- - auth/session lookups
-- - notification inbox

CREATE INDEX IF NOT EXISTS idx_sessions_user_expires
    ON sessions(user_id, expires_at);

CREATE INDEX IF NOT EXISTS idx_members_workspace_user
    ON members(workspace_id, user_id);

CREATE INDEX IF NOT EXISTS idx_pages_workspace_parent
    ON pages(workspace_id, parent_type, parent_id);

CREATE INDEX IF NOT EXISTS idx_pages_database_order
    ON pages(database_id, row_position);

CREATE INDEX IF NOT EXISTS idx_blocks_page_parent_order
    ON blocks(page_id, parent_id, position);

CREATE INDEX IF NOT EXISTS idx_views_database_order
    ON views(database_id, position);

CREATE INDEX IF NOT EXISTS idx_comments_target
    ON comments(workspace_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_notifications_inbox
    ON notifications(user_id, is_read, created_at);

CREATE INDEX IF NOT EXISTS idx_activities_workspace_time
    ON activities(workspace_id, created_at);

CREATE INDEX IF NOT EXISTS idx_shares_token
    ON shares(token);
