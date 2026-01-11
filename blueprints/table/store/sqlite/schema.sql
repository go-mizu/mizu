-- Table Store Schema for SQLite

-- Enable foreign key support
PRAGMA foreign_keys = ON;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- Workspaces
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    icon TEXT,
    plan TEXT DEFAULT 'free',
    owner_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);
CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_id);

-- Workspace members
CREATE TABLE IF NOT EXISTS workspace_members (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);

-- Bases
CREATE TABLE IF NOT EXISTS bases (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    color TEXT DEFAULT '#2563EB',
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bases_workspace ON bases(workspace_id);

-- Tables
CREATE TABLE IF NOT EXISTS tables (
    id TEXT PRIMARY KEY,
    base_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    position INTEGER DEFAULT 0,
    primary_field_id TEXT,
    auto_number_seq INTEGER DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tables_base ON tables(base_id);
CREATE INDEX IF NOT EXISTS idx_tables_position ON tables(base_id, position);

-- Fields (columns)
CREATE TABLE IF NOT EXISTS fields (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT,
    options TEXT,
    position INTEGER DEFAULT 0,
    is_primary INTEGER DEFAULT 0,
    is_computed INTEGER DEFAULT 0,
    is_hidden INTEGER DEFAULT 0,
    width INTEGER DEFAULT 200,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_fields_table ON fields(table_id);
CREATE INDEX IF NOT EXISTS idx_fields_position ON fields(table_id, position);

-- Select choices (for single_select and multi_select fields)
CREATE TABLE IF NOT EXISTS select_choices (
    id TEXT PRIMARY KEY,
    field_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#6B7280',
    position INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_select_choices_field ON select_choices(field_id);

-- Records (rows with JSON cells)
CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL,
    cells TEXT NOT NULL DEFAULT '{}',
    position INTEGER DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_records_table ON records(table_id);
CREATE INDEX IF NOT EXISTS idx_records_position ON records(table_id, position);
CREATE INDEX IF NOT EXISTS idx_records_created ON records(table_id, created_at DESC);

-- Views
CREATE TABLE IF NOT EXISTS views (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'grid',
    config TEXT,
    filters TEXT DEFAULT '[]',
    sorts TEXT DEFAULT '[]',
    groups TEXT DEFAULT '[]',
    field_config TEXT DEFAULT '[]',
    position INTEGER DEFAULT 0,
    is_default INTEGER DEFAULT 0,
    is_locked INTEGER DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_views_table ON views(table_id);
CREATE INDEX IF NOT EXISTS idx_views_position ON views(table_id, position);

-- Operations log (for history, undo/redo, and time-travel)
CREATE TABLE IF NOT EXISTS operations (
    id TEXT PRIMARY KEY,
    table_id TEXT,
    record_id TEXT,
    field_id TEXT,
    view_id TEXT,
    op_type TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    user_id TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operations_table ON operations(table_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_operations_record ON operations(record_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_operations_user ON operations(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_operations_time ON operations(timestamp DESC);

-- Snapshots (for efficient time-travel queries)
CREATE TABLE IF NOT EXISTS snapshots (
    id TEXT PRIMARY KEY,
    table_id TEXT NOT NULL,
    data BLOB,
    op_cursor TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_snapshots_table ON snapshots(table_id, created_at DESC);

-- Shares
CREATE TABLE IF NOT EXISTS shares (
    id TEXT PRIMARY KEY,
    base_id TEXT NOT NULL,
    table_id TEXT,
    view_id TEXT,
    type TEXT NOT NULL,
    permission TEXT NOT NULL DEFAULT 'read',
    user_id TEXT,
    email TEXT,
    token TEXT UNIQUE,
    expires_at DATETIME,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shares_base ON shares(base_id);
CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token);
CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id);

-- Attachments
CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    record_id TEXT NOT NULL,
    field_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    url TEXT NOT NULL,
    thumbnail_url TEXT,
    width INTEGER,
    height INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_attachments_record ON attachments(record_id, field_id);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY,
    record_id TEXT NOT NULL,
    parent_id TEXT,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    is_resolved INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_record ON comments(record_id, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    base_id TEXT NOT NULL,
    table_id TEXT,
    url TEXT NOT NULL,
    events TEXT NOT NULL,
    secret TEXT,
    is_active INTEGER DEFAULT 1,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhooks_base ON webhooks(base_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_table ON webhooks(table_id);

-- Webhook deliveries
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL,
    event TEXT NOT NULL,
    payload TEXT NOT NULL,
    status_code INTEGER,
    response TEXT,
    duration_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);

-- Linked record relationships (for efficient bi-directional lookup)
CREATE TABLE IF NOT EXISTS record_links (
    id TEXT PRIMARY KEY,
    source_record_id TEXT NOT NULL,
    source_field_id TEXT NOT NULL,
    target_record_id TEXT NOT NULL,
    position INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_record_links_source ON record_links(source_record_id, source_field_id);
CREATE INDEX IF NOT EXISTS idx_record_links_target ON record_links(target_record_id);
