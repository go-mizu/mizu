-- Table Store Schema for DuckDB

-- Users
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR PRIMARY KEY,
    email VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    password_hash VARCHAR NOT NULL,
    avatar_url VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- Workspaces
CREATE TABLE IF NOT EXISTS workspaces (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    slug VARCHAR UNIQUE NOT NULL,
    icon VARCHAR,
    plan VARCHAR DEFAULT 'free',
    owner_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);
CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_id);

-- Workspace members
CREATE TABLE IF NOT EXISTS workspace_members (
    id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);

-- Bases
CREATE TABLE IF NOT EXISTS bases (
    id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    description TEXT,
    icon VARCHAR,
    color VARCHAR DEFAULT '#2563EB',
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bases_workspace ON bases(workspace_id);

-- Tables
CREATE TABLE IF NOT EXISTS tables (
    id VARCHAR PRIMARY KEY,
    base_id VARCHAR NOT NULL REFERENCES bases(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    description TEXT,
    icon VARCHAR,
    position INTEGER DEFAULT 0,
    primary_field_id VARCHAR,
    auto_number_seq BIGINT DEFAULT 0,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tables_base ON tables(base_id);
CREATE INDEX IF NOT EXISTS idx_tables_position ON tables(base_id, position);

-- Fields (columns)
CREATE TABLE IF NOT EXISTS fields (
    id VARCHAR PRIMARY KEY,
    table_id VARCHAR NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    description TEXT,
    options JSON,
    position INTEGER DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,
    is_computed BOOLEAN DEFAULT FALSE,
    is_hidden BOOLEAN DEFAULT FALSE,
    width INTEGER DEFAULT 200,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_fields_table ON fields(table_id);
CREATE INDEX IF NOT EXISTS idx_fields_position ON fields(table_id, position);

-- Select choices (for single_select and multi_select fields)
CREATE TABLE IF NOT EXISTS select_choices (
    id VARCHAR PRIMARY KEY,
    field_id VARCHAR NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    color VARCHAR DEFAULT '#6B7280',
    position INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_select_choices_field ON select_choices(field_id);

-- Records (rows with JSON cells)
CREATE TABLE IF NOT EXISTS records (
    id VARCHAR PRIMARY KEY,
    table_id VARCHAR NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    cells JSON NOT NULL DEFAULT '{}',
    position INTEGER DEFAULT 0,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR
);

CREATE INDEX IF NOT EXISTS idx_records_table ON records(table_id);
CREATE INDEX IF NOT EXISTS idx_records_position ON records(table_id, position);
CREATE INDEX IF NOT EXISTS idx_records_created ON records(table_id, created_at DESC);

-- Views
CREATE TABLE IF NOT EXISTS views (
    id VARCHAR PRIMARY KEY,
    table_id VARCHAR NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    type VARCHAR NOT NULL DEFAULT 'grid',
    config JSON,
    filters JSON DEFAULT '[]',
    sorts JSON DEFAULT '[]',
    groups JSON DEFAULT '[]',
    field_config JSON DEFAULT '[]',
    position INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    is_locked BOOLEAN DEFAULT FALSE,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_views_table ON views(table_id);
CREATE INDEX IF NOT EXISTS idx_views_position ON views(table_id, position);

-- Operations log (for history, undo/redo, and time-travel)
CREATE TABLE IF NOT EXISTS operations (
    id VARCHAR PRIMARY KEY,
    table_id VARCHAR REFERENCES tables(id) ON DELETE SET NULL,
    record_id VARCHAR REFERENCES records(id) ON DELETE SET NULL,
    field_id VARCHAR REFERENCES fields(id) ON DELETE SET NULL,
    view_id VARCHAR REFERENCES views(id) ON DELETE SET NULL,
    op_type VARCHAR NOT NULL,
    old_value JSON,
    new_value JSON,
    user_id VARCHAR NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operations_table ON operations(table_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_operations_record ON operations(record_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_operations_user ON operations(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_operations_time ON operations(timestamp DESC);

-- Snapshots (for efficient time-travel queries)
CREATE TABLE IF NOT EXISTS snapshots (
    id VARCHAR PRIMARY KEY,
    table_id VARCHAR NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    data BLOB,
    op_cursor VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_snapshots_table ON snapshots(table_id, created_at DESC);

-- Shares
CREATE TABLE IF NOT EXISTS shares (
    id VARCHAR PRIMARY KEY,
    base_id VARCHAR NOT NULL REFERENCES bases(id) ON DELETE CASCADE,
    table_id VARCHAR REFERENCES tables(id) ON DELETE CASCADE,
    view_id VARCHAR REFERENCES views(id) ON DELETE CASCADE,
    type VARCHAR NOT NULL,
    permission VARCHAR NOT NULL DEFAULT 'read',
    user_id VARCHAR REFERENCES users(id) ON DELETE SET NULL,
    email VARCHAR,
    token VARCHAR UNIQUE,
    expires_at TIMESTAMP,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shares_base ON shares(base_id);
CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token);
CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id);

-- Attachments
CREATE TABLE IF NOT EXISTS attachments (
    id VARCHAR PRIMARY KEY,
    record_id VARCHAR NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    field_id VARCHAR NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    filename VARCHAR NOT NULL,
    size BIGINT NOT NULL,
    mime_type VARCHAR NOT NULL,
    url VARCHAR NOT NULL,
    thumbnail_url VARCHAR,
    width INTEGER,
    height INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_attachments_record ON attachments(record_id, field_id);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    record_id VARCHAR NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    parent_id VARCHAR REFERENCES comments(id) ON DELETE SET NULL,
    user_id VARCHAR NOT NULL,
    content TEXT NOT NULL,
    is_resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_record ON comments(record_id, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR PRIMARY KEY,
    base_id VARCHAR NOT NULL REFERENCES bases(id) ON DELETE CASCADE,
    table_id VARCHAR REFERENCES tables(id) ON DELETE CASCADE,
    url VARCHAR NOT NULL,
    events JSON NOT NULL,
    secret VARCHAR,
    is_active BOOLEAN DEFAULT TRUE,
    created_by VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhooks_base ON webhooks(base_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_table ON webhooks(table_id);

-- Webhook deliveries
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR PRIMARY KEY,
    webhook_id VARCHAR NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event VARCHAR NOT NULL,
    payload TEXT NOT NULL,
    status_code INTEGER,
    response TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);

-- Linked record relationships (for efficient bi-directional lookup)
CREATE TABLE IF NOT EXISTS record_links (
    id VARCHAR PRIMARY KEY,
    source_record_id VARCHAR NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    source_field_id VARCHAR NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    target_record_id VARCHAR NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    position INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_record_links_source ON record_links(source_record_id, source_field_id);
CREATE INDEX IF NOT EXISTS idx_record_links_target ON record_links(target_record_id);
