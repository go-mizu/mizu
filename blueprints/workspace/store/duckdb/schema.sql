-- schema.sql
-- Workspace - Notion-style collaborative workspace schema

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
-- Pages
-- ============================================================

CREATE TABLE IF NOT EXISTS pages (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    parent_id    VARCHAR,
    parent_type  VARCHAR DEFAULT 'workspace',
    title        VARCHAR DEFAULT '',
    icon         VARCHAR,
    cover        VARCHAR,
    cover_y      DOUBLE DEFAULT 0.5,
    properties   JSON DEFAULT '{}',
    is_template  BOOLEAN DEFAULT FALSE,
    is_archived  BOOLEAN DEFAULT FALSE,
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by   VARCHAR NOT NULL,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Blocks
-- ============================================================

CREATE TABLE IF NOT EXISTS blocks (
    id         VARCHAR PRIMARY KEY,
    page_id    VARCHAR NOT NULL,
    parent_id  VARCHAR,
    type       VARCHAR NOT NULL,
    content    JSON DEFAULT '{}',
    position   INTEGER DEFAULT 0,
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
    page_id      VARCHAR NOT NULL,
    title        VARCHAR DEFAULT 'Untitled',
    description  JSON DEFAULT '[]',
    icon         VARCHAR,
    cover        VARCHAR,
    is_inline    BOOLEAN DEFAULT FALSE,
    properties   JSON DEFAULT '[]',
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
    type        VARCHAR NOT NULL DEFAULT 'table',
    filter      JSON,
    sorts       JSON DEFAULT '[]',
    properties  JSON DEFAULT '[]',
    group_by    VARCHAR,
    calendar_by VARCHAR,
    position    INTEGER DEFAULT 0,
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Database Rows
-- ============================================================

CREATE TABLE IF NOT EXISTS database_rows (
    id          VARCHAR PRIMARY KEY,
    database_id VARCHAR NOT NULL,
    properties  JSON DEFAULT '{}',
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by  VARCHAR NOT NULL,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR PRIMARY KEY,
    page_id     VARCHAR NOT NULL,
    block_id    VARCHAR,
    parent_id   VARCHAR,
    content     JSON NOT NULL,
    author_id   VARCHAR NOT NULL,
    is_resolved BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Sharing
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id          VARCHAR PRIMARY KEY,
    page_id     VARCHAR NOT NULL,
    type        VARCHAR NOT NULL,
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
-- Indexes
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_pages_workspace ON pages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_pages_parent ON pages(parent_id, parent_type);
CREATE INDEX IF NOT EXISTS idx_blocks_page ON blocks(page_id);
CREATE INDEX IF NOT EXISTS idx_blocks_parent ON blocks(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_page ON comments(page_id);
CREATE INDEX IF NOT EXISTS idx_activities_workspace ON activities(workspace_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id, workspace_id);
CREATE INDEX IF NOT EXISTS idx_views_database ON views(database_id);
CREATE INDEX IF NOT EXISTS idx_database_rows_database ON database_rows(database_id);
CREATE INDEX IF NOT EXISTS idx_database_rows_created ON database_rows(database_id, created_at);
CREATE INDEX IF NOT EXISTS idx_members_workspace ON members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_members_user ON members(user_id);
