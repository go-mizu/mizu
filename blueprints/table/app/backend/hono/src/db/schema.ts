/**
 * Database schema SQL for the table blueprint
 */

export const SCHEMA_SQL = `
-- Users and authentication
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    avatar_url    TEXT,
    password_hash TEXT NOT NULL,
    settings      TEXT DEFAULT '{}',
    created_at    TEXT DEFAULT (datetime('now')),
    updated_at    TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Workspaces (organizations)
CREATE TABLE IF NOT EXISTS workspaces (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT UNIQUE NOT NULL,
    icon       TEXT,
    plan       TEXT DEFAULT 'free',
    settings   TEXT DEFAULT '{}',
    owner_id   TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workspace_members (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    role         TEXT NOT NULL DEFAULT 'member',
    joined_at    TEXT DEFAULT (datetime('now')),
    invited_by   TEXT,
    UNIQUE(workspace_id, user_id),
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Bases (projects/databases)
CREATE TABLE IF NOT EXISTS bases (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT,
    icon         TEXT,
    color        TEXT DEFAULT '#2D7FF9',
    settings     TEXT DEFAULT '{}',
    is_template  INTEGER DEFAULT 0,
    created_by   TEXT NOT NULL,
    created_at   TEXT DEFAULT (datetime('now')),
    updated_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Tables
CREATE TABLE IF NOT EXISTS tables (
    id               TEXT PRIMARY KEY,
    base_id          TEXT NOT NULL,
    name             TEXT NOT NULL,
    description      TEXT,
    icon             TEXT,
    position         INTEGER DEFAULT 0,
    primary_field_id TEXT,
    settings         TEXT DEFAULT '{}',
    created_by       TEXT NOT NULL,
    created_at       TEXT DEFAULT (datetime('now')),
    updated_at       TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (base_id) REFERENCES bases(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Fields (columns)
CREATE TABLE IF NOT EXISTS fields (
    id           TEXT PRIMARY KEY,
    table_id     TEXT NOT NULL,
    name         TEXT NOT NULL,
    type         TEXT NOT NULL,
    description  TEXT,
    options      TEXT DEFAULT '{}',
    position     INTEGER DEFAULT 0,
    is_primary   INTEGER DEFAULT 0,
    is_computed  INTEGER DEFAULT 0,
    is_hidden    INTEGER DEFAULT 0,
    width        INTEGER DEFAULT 200,
    created_by   TEXT NOT NULL,
    created_at   TEXT DEFAULT (datetime('now')),
    updated_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Select options (for single/multi-select fields)
CREATE TABLE IF NOT EXISTS select_options (
    id       TEXT PRIMARY KEY,
    field_id TEXT NOT NULL,
    name     TEXT NOT NULL,
    color    TEXT NOT NULL DEFAULT '#CFDFFF',
    position INTEGER DEFAULT 0,
    FOREIGN KEY (field_id) REFERENCES fields(id) ON DELETE CASCADE
);

-- Records (rows)
CREATE TABLE IF NOT EXISTS records (
    id           TEXT PRIMARY KEY,
    table_id     TEXT NOT NULL,
    position     INTEGER DEFAULT 0,
    created_by   TEXT NOT NULL,
    created_at   TEXT DEFAULT (datetime('now')),
    updated_by   TEXT NOT NULL,
    updated_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    FOREIGN KEY (updated_by) REFERENCES users(id)
);

-- Cell values (sparse storage)
CREATE TABLE IF NOT EXISTS cell_values (
    id           TEXT PRIMARY KEY,
    record_id    TEXT NOT NULL,
    field_id     TEXT NOT NULL,
    value        TEXT,
    text_value   TEXT,
    number_value REAL,
    date_value   TEXT,
    updated_at   TEXT DEFAULT (datetime('now')),
    UNIQUE(record_id, field_id),
    FOREIGN KEY (record_id) REFERENCES records(id) ON DELETE CASCADE,
    FOREIGN KEY (field_id) REFERENCES fields(id) ON DELETE CASCADE
);

-- Linked records
CREATE TABLE IF NOT EXISTS linked_records (
    id               TEXT PRIMARY KEY,
    field_id         TEXT NOT NULL,
    source_record_id TEXT NOT NULL,
    target_record_id TEXT NOT NULL,
    position         INTEGER DEFAULT 0,
    created_at       TEXT DEFAULT (datetime('now')),
    UNIQUE(field_id, source_record_id, target_record_id),
    FOREIGN KEY (field_id) REFERENCES fields(id) ON DELETE CASCADE,
    FOREIGN KEY (source_record_id) REFERENCES records(id) ON DELETE CASCADE,
    FOREIGN KEY (target_record_id) REFERENCES records(id) ON DELETE CASCADE
);

-- Attachments
CREATE TABLE IF NOT EXISTS attachments (
    id            TEXT PRIMARY KEY,
    record_id     TEXT NOT NULL,
    field_id      TEXT NOT NULL,
    filename      TEXT NOT NULL,
    size          INTEGER NOT NULL,
    mime_type     TEXT NOT NULL,
    url           TEXT NOT NULL,
    thumbnail_url TEXT,
    width         INTEGER,
    height        INTEGER,
    position      INTEGER DEFAULT 0,
    uploaded_by   TEXT NOT NULL,
    created_at    TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (record_id) REFERENCES records(id) ON DELETE CASCADE,
    FOREIGN KEY (field_id) REFERENCES fields(id) ON DELETE CASCADE,
    FOREIGN KEY (uploaded_by) REFERENCES users(id)
);

-- Views
CREATE TABLE IF NOT EXISTS views (
    id           TEXT PRIMARY KEY,
    table_id     TEXT NOT NULL,
    name         TEXT NOT NULL,
    type         TEXT NOT NULL DEFAULT 'grid',
    filters      TEXT DEFAULT '[]',
    sorts        TEXT DEFAULT '[]',
    groups       TEXT DEFAULT '[]',
    field_config TEXT DEFAULT '[]',
    settings     TEXT DEFAULT '{}',
    position     INTEGER DEFAULT 0,
    is_default   INTEGER DEFAULT 0,
    is_locked    INTEGER DEFAULT 0,
    created_by   TEXT NOT NULL,
    created_at   TEXT DEFAULT (datetime('now')),
    updated_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id           TEXT PRIMARY KEY,
    record_id    TEXT NOT NULL,
    parent_id    TEXT,
    author_id    TEXT NOT NULL,
    content      TEXT NOT NULL,
    is_resolved  INTEGER DEFAULT 0,
    created_at   TEXT DEFAULT (datetime('now')),
    updated_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (record_id) REFERENCES records(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES users(id)
);

-- Shares
CREATE TABLE IF NOT EXISTS shares (
    id           TEXT PRIMARY KEY,
    base_id      TEXT NOT NULL,
    type         TEXT NOT NULL,
    permission   TEXT NOT NULL DEFAULT 'read',
    user_id      TEXT,
    email        TEXT,
    token        TEXT UNIQUE,
    password     TEXT,
    expires_at   TEXT,
    created_by   TEXT NOT NULL,
    created_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (base_id) REFERENCES bases(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Activities
CREATE TABLE IF NOT EXISTS activities (
    id           TEXT PRIMARY KEY,
    base_id      TEXT NOT NULL,
    table_id     TEXT,
    record_id    TEXT,
    field_id     TEXT,
    actor_id     TEXT NOT NULL,
    action       TEXT NOT NULL,
    old_value    TEXT,
    new_value    TEXT,
    created_at   TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (base_id) REFERENCES bases(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES users(id)
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    type       TEXT NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT,
    base_id    TEXT,
    record_id  TEXT,
    actor_id   TEXT,
    is_read    INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Favorites
CREATE TABLE IF NOT EXISTS favorites (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    base_id      TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    created_at   TEXT DEFAULT (datetime('now')),
    UNIQUE(user_id, base_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (base_id) REFERENCES bases(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sessions_user_expires ON sessions(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);
CREATE INDEX IF NOT EXISTS idx_bases_workspace ON bases(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tables_base_position ON tables(base_id, position);
CREATE INDEX IF NOT EXISTS idx_fields_table_position ON fields(table_id, position);
CREATE INDEX IF NOT EXISTS idx_select_options_field ON select_options(field_id, position);
CREATE INDEX IF NOT EXISTS idx_records_table_position ON records(table_id, position);
CREATE INDEX IF NOT EXISTS idx_cell_values_record ON cell_values(record_id);
CREATE INDEX IF NOT EXISTS idx_cell_values_field ON cell_values(field_id);
CREATE INDEX IF NOT EXISTS idx_cell_values_text ON cell_values(text_value);
CREATE INDEX IF NOT EXISTS idx_linked_records_source ON linked_records(source_record_id);
CREATE INDEX IF NOT EXISTS idx_linked_records_target ON linked_records(target_record_id);
CREATE INDEX IF NOT EXISTS idx_attachments_record_field ON attachments(record_id, field_id);
CREATE INDEX IF NOT EXISTS idx_views_table_position ON views(table_id, position);
CREATE INDEX IF NOT EXISTS idx_comments_record ON comments(record_id);
CREATE INDEX IF NOT EXISTS idx_shares_base ON shares(base_id);
CREATE INDEX IF NOT EXISTS idx_shares_token ON shares(token);
CREATE INDEX IF NOT EXISTS idx_activities_base_time ON activities(base_id, created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read, created_at);
CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id);
`;
