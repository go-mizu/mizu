-- schema.sql
-- Minimal Kanban schema with one consistent vocabulary across all layers (UI/API/service/store/DB).
-- Includes Cycles (planning periods) and a monday/GitHub-style Fields + Values system.
--
-- Core mental model (what users learn):
--   Workspace → Team → Project (Board) → Columns → Cards
-- Planning:
--   Team → Cycles → Cards (optional attachment)
-- Extensibility:
--   Cards can have Fields, and each card stores Values for those fields.

-- ============================================================
-- Users and authentication
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR UNIQUE NOT NULL,
    username      VARCHAR UNIQUE NOT NULL,
    display_name  VARCHAR NOT NULL,
    password_hash VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR PRIMARY KEY,
    user_id    VARCHAR NOT NULL REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Workspaces and membership
-- ============================================================

CREATE TABLE IF NOT EXISTS workspaces (
    id   VARCHAR PRIMARY KEY,
    slug VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
    user_id      VARCHAR NOT NULL REFERENCES users(id),
    role         VARCHAR NOT NULL DEFAULT 'member',
    joined_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id)
);

-- ============================================================
-- Teams
-- ============================================================

CREATE TABLE IF NOT EXISTS teams (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id),
    key          VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    UNIQUE (workspace_id, key),
    UNIQUE (workspace_id, name)
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id   VARCHAR NOT NULL REFERENCES teams(id),
    user_id   VARCHAR NOT NULL REFERENCES users(id),
    role      VARCHAR NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id)
);

-- ============================================================
-- Projects (Boards)
-- ============================================================

CREATE TABLE IF NOT EXISTS projects (
    id            VARCHAR PRIMARY KEY,
    team_id       VARCHAR NOT NULL REFERENCES teams(id),
    key           VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    issue_counter INTEGER NOT NULL DEFAULT 0,
    UNIQUE (team_id, key)
);

-- ============================================================
-- Columns (Kanban columns)
-- ============================================================

CREATE TABLE IF NOT EXISTS columns (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL REFERENCES projects(id),
    name        VARCHAR NOT NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (project_id, name)
);

-- ============================================================
-- Cycles (planning periods)
-- ============================================================

CREATE TABLE IF NOT EXISTS cycles (
    id         VARCHAR PRIMARY KEY,
    team_id    VARCHAR NOT NULL REFERENCES teams(id),
    number     INTEGER NOT NULL,
    name       VARCHAR NOT NULL,
    status     VARCHAR NOT NULL DEFAULT 'planning',
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (team_id, number)
);

-- ============================================================
-- Issues (Cards)
-- ============================================================

CREATE TABLE IF NOT EXISTS issues (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL REFERENCES projects(id),
    number      INTEGER NOT NULL,
    key         VARCHAR NOT NULL,
    title       VARCHAR NOT NULL,
    description VARCHAR DEFAULT '',
    column_id   VARCHAR NOT NULL REFERENCES columns(id),
    position    INTEGER NOT NULL DEFAULT 0,
    priority    INTEGER NOT NULL DEFAULT 0,
    creator_id  VARCHAR NOT NULL REFERENCES users(id),
    cycle_id    VARCHAR REFERENCES cycles(id),
    due_date    DATE,
    start_date  DATE,
    end_date    DATE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number),
    UNIQUE (project_id, key)
);

CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id VARCHAR NOT NULL REFERENCES issues(id),
    user_id  VARCHAR NOT NULL REFERENCES users(id),
    PRIMARY KEY (issue_id, user_id)
);

-- ============================================================
-- Comments (Discussion)
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id         VARCHAR PRIMARY KEY,
    issue_id   VARCHAR NOT NULL REFERENCES issues(id),
    author_id  VARCHAR NOT NULL REFERENCES users(id),
    content    VARCHAR NOT NULL,
    edited_at  TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Fields and Values (Custom columns)
-- ============================================================

CREATE TABLE IF NOT EXISTS fields (
    id            VARCHAR PRIMARY KEY,
    project_id    VARCHAR NOT NULL REFERENCES projects(id),
    key           VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,
    kind          VARCHAR NOT NULL,
    position      INTEGER NOT NULL DEFAULT 0,
    is_required   BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived   BOOLEAN NOT NULL DEFAULT FALSE,
    settings_json VARCHAR,
    UNIQUE (project_id, key),
    UNIQUE (project_id, name)
);

CREATE TABLE IF NOT EXISTS field_values (
    issue_id   VARCHAR NOT NULL REFERENCES issues(id),
    field_id   VARCHAR NOT NULL REFERENCES fields(id),
    value_text VARCHAR,
    value_num  DOUBLE,
    value_bool BOOLEAN,
    value_date DATE,
    value_ts   TIMESTAMP,
    value_ref  VARCHAR,
    value_json VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (issue_id, field_id)
);
