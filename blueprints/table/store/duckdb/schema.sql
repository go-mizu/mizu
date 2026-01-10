-- schema.sql
-- Table Blueprint - Airtable-style database schema (DuckDB oriented)

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
-- Workspaces (organizations)
-- ============================================================

CREATE TABLE IF NOT EXISTS workspaces (
    id         VARCHAR PRIMARY KEY,
    name       VARCHAR NOT NULL,
    slug       VARCHAR UNIQUE NOT NULL,
    icon       VARCHAR,
    plan       VARCHAR DEFAULT 'free',
    settings   JSON DEFAULT '{}',
    owner_id   VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workspace_members (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    user_id      VARCHAR NOT NULL,
    role         VARCHAR NOT NULL DEFAULT 'member',
    joined_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    invited_by   VARCHAR,
    UNIQUE(workspace_id, user_id)
);

-- ============================================================
-- Bases (projects/databases)
-- ============================================================

CREATE TABLE IF NOT EXISTS bases (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    description  VARCHAR,
    icon         VARCHAR,
    color        VARCHAR DEFAULT '#2D7FF9',
    settings     JSON DEFAULT '{}',
    is_template  BOOLEAN DEFAULT FALSE,
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Tables
-- ============================================================

CREATE TABLE IF NOT EXISTS tables (
    id               VARCHAR PRIMARY KEY,
    base_id          VARCHAR NOT NULL,
    name             VARCHAR NOT NULL,
    description      VARCHAR,
    icon             VARCHAR,
    position         INTEGER DEFAULT 0,
    primary_field_id VARCHAR,
    settings         JSON DEFAULT '{}',
    created_by       VARCHAR NOT NULL,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Fields (columns)
-- ============================================================
-- Types: text, long_text, number, currency, percent, single_select,
--        multi_select, date, checkbox, rating, duration, phone, email,
--        url, attachment, user, created_time, modified_time, created_by,
--        modified_by, autonumber, barcode, formula, rollup, count, lookup, link

CREATE TABLE IF NOT EXISTS fields (
    id           VARCHAR PRIMARY KEY,
    table_id     VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    type         VARCHAR NOT NULL,
    description  VARCHAR,
    options      JSON DEFAULT '{}',
    position     INTEGER DEFAULT 0,
    is_primary   BOOLEAN DEFAULT FALSE,
    is_computed  BOOLEAN DEFAULT FALSE,
    is_hidden    BOOLEAN DEFAULT FALSE,
    width        INTEGER DEFAULT 200,
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Select options (for single/multi-select fields)
-- ============================================================

CREATE TABLE IF NOT EXISTS select_options (
    id       VARCHAR PRIMARY KEY,
    field_id VARCHAR NOT NULL,
    name     VARCHAR NOT NULL,
    color    VARCHAR NOT NULL DEFAULT '#CFDFFF',
    position INTEGER DEFAULT 0
);

-- ============================================================
-- Records (rows)
-- ============================================================

CREATE TABLE IF NOT EXISTS records (
    id           VARCHAR PRIMARY KEY,
    table_id     VARCHAR NOT NULL,
    position     BIGINT DEFAULT 0,
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by   VARCHAR NOT NULL,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Cell values (sparse storage - only non-empty cells)
-- ============================================================

CREATE TABLE IF NOT EXISTS cell_values (
    id           VARCHAR PRIMARY KEY,
    record_id    VARCHAR NOT NULL,
    field_id     VARCHAR NOT NULL,
    value        JSON,
    text_value   VARCHAR,     -- For text search indexing
    number_value DOUBLE,      -- For numeric sorting/filtering
    date_value   TIMESTAMP,   -- For date operations
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(record_id, field_id)
);

-- ============================================================
-- Linked records (relationships between records)
-- ============================================================

CREATE TABLE IF NOT EXISTS linked_records (
    id               VARCHAR PRIMARY KEY,
    field_id         VARCHAR NOT NULL,
    source_record_id VARCHAR NOT NULL,
    target_record_id VARCHAR NOT NULL,
    position         INTEGER DEFAULT 0,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(field_id, source_record_id, target_record_id)
);

-- ============================================================
-- Attachments
-- ============================================================

CREATE TABLE IF NOT EXISTS attachments (
    id            VARCHAR PRIMARY KEY,
    record_id     VARCHAR NOT NULL,
    field_id      VARCHAR NOT NULL,
    filename      VARCHAR NOT NULL,
    size          BIGINT NOT NULL,
    mime_type     VARCHAR NOT NULL,
    url           VARCHAR NOT NULL,
    thumbnail_url VARCHAR,
    width         INTEGER,
    height        INTEGER,
    position      INTEGER DEFAULT 0,
    uploaded_by   VARCHAR NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Views
-- ============================================================
-- Types: grid, kanban, calendar, gallery, form, timeline, list

CREATE TABLE IF NOT EXISTS views (
    id           VARCHAR PRIMARY KEY,
    table_id     VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    type         VARCHAR NOT NULL DEFAULT 'grid',
    filters      JSON DEFAULT '[]',
    sorts        JSON DEFAULT '[]',
    groups       JSON DEFAULT '[]',
    field_config JSON DEFAULT '[]',
    settings     JSON DEFAULT '{}',
    position     INTEGER DEFAULT 0,
    is_default   BOOLEAN DEFAULT FALSE,
    is_locked    BOOLEAN DEFAULT FALSE,
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id           VARCHAR PRIMARY KEY,
    record_id    VARCHAR NOT NULL,
    parent_id    VARCHAR,
    author_id    VARCHAR NOT NULL,
    content      JSON NOT NULL,
    is_resolved  BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Sharing
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id           VARCHAR PRIMARY KEY,
    base_id      VARCHAR NOT NULL,
    type         VARCHAR NOT NULL,            -- invite, public_link
    permission   VARCHAR NOT NULL DEFAULT 'read',
    user_id      VARCHAR,
    email        VARCHAR,
    token        VARCHAR UNIQUE,
    password     VARCHAR,
    expires_at   TIMESTAMP,
    created_by   VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Activity log
-- ============================================================

CREATE TABLE IF NOT EXISTS activities (
    id           VARCHAR PRIMARY KEY,
    base_id      VARCHAR NOT NULL,
    table_id     VARCHAR,
    record_id    VARCHAR,
    field_id     VARCHAR,
    actor_id     VARCHAR NOT NULL,
    action       VARCHAR NOT NULL,
    old_value    JSON,
    new_value    JSON,
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
    base_id    VARCHAR,
    record_id  VARCHAR,
    actor_id   VARCHAR,
    is_read    BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Favorites
-- ============================================================

CREATE TABLE IF NOT EXISTS favorites (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    base_id      VARCHAR NOT NULL,
    workspace_id VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, base_id)
);

-- ============================================================
-- Indexes
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_sessions_user_expires
    ON sessions(user_id, expires_at);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace
    ON workspace_members(workspace_id);

CREATE INDEX IF NOT EXISTS idx_workspace_members_user
    ON workspace_members(user_id);

CREATE INDEX IF NOT EXISTS idx_bases_workspace
    ON bases(workspace_id);

CREATE INDEX IF NOT EXISTS idx_tables_base_position
    ON tables(base_id, position);

CREATE INDEX IF NOT EXISTS idx_fields_table_position
    ON fields(table_id, position);

CREATE INDEX IF NOT EXISTS idx_select_options_field
    ON select_options(field_id, position);

CREATE INDEX IF NOT EXISTS idx_records_table_position
    ON records(table_id, position);

CREATE INDEX IF NOT EXISTS idx_cell_values_record
    ON cell_values(record_id);

CREATE INDEX IF NOT EXISTS idx_cell_values_field
    ON cell_values(field_id);

CREATE INDEX IF NOT EXISTS idx_cell_values_text
    ON cell_values(text_value);

CREATE INDEX IF NOT EXISTS idx_cell_values_number
    ON cell_values(number_value);

CREATE INDEX IF NOT EXISTS idx_cell_values_date
    ON cell_values(date_value);

CREATE INDEX IF NOT EXISTS idx_linked_records_source
    ON linked_records(source_record_id);

CREATE INDEX IF NOT EXISTS idx_linked_records_target
    ON linked_records(target_record_id);

CREATE INDEX IF NOT EXISTS idx_linked_records_field
    ON linked_records(field_id);

CREATE INDEX IF NOT EXISTS idx_attachments_record_field
    ON attachments(record_id, field_id);

CREATE INDEX IF NOT EXISTS idx_views_table_position
    ON views(table_id, position);

CREATE INDEX IF NOT EXISTS idx_comments_record
    ON comments(record_id);

CREATE INDEX IF NOT EXISTS idx_shares_base
    ON shares(base_id);

CREATE INDEX IF NOT EXISTS idx_shares_token
    ON shares(token);

CREATE INDEX IF NOT EXISTS idx_activities_base_time
    ON activities(base_id, created_at);

CREATE INDEX IF NOT EXISTS idx_activities_record
    ON activities(record_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_read
    ON notifications(user_id, is_read, created_at);

CREATE INDEX IF NOT EXISTS idx_favorites_user
    ON favorites(user_id);
