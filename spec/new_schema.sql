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
--
-- ------------------------------------------------------------
-- Concept mapping to popular products (for migration/import)
-- ------------------------------------------------------------
--
-- Workspace
--   Trello        : Workspace
--   monday.com    : Workspace
--   Linear        : Workspace (Organization)
--   Jira          : Site / Organization
--   GitHub        : Owner (Org/User)
--
-- Team (REQUIRED)
--   Linear        : Team (primary organizing unit)
--   Jira          : Often implicit (groups/roles), not always explicit
--   Trello        : Not explicit (can be inferred)
--   monday.com    : Often implicit (user groups)
--
-- Project (Board)
--   Trello        : Board
--   monday.com    : Board
--   Linear        : Project
--   Jira          : Project
--   GitHub        : Project
--
-- Column
--   Trello        : List
--   monday.com    : Group / Status grouping
--   Linear        : Workflow State / Board Column
--   Jira          : Status column
--   GitHub        : Board column (grouping by a status field)
--
-- Issue (Card)
--   Trello        : Card
--   monday.com    : Item (row)
--   Linear        : Issue
--   Jira          : Issue
--   GitHub        : Item referencing Issue/PR/Draft
--
-- Cycle
--   Linear        : Cycle
--   Plane.so      : Cycle
--   Jira          : Sprint (closest equivalent)
--   GitHub        : Iteration (field type)
--   monday.com    : Often modeled via groups/dates; not always first-class
--
-- Field / Value
--   monday.com    : Column / Cell
--   GitHub        : Field / FieldValue
--   Jira          : Custom Field / Value
--
-- ------------------------------------------------------------
-- Design principles
-- ------------------------------------------------------------
-- - Minimal required columns only
-- - Everything optional becomes a Field + Value
-- - Typed values are AI/analytics-friendly and avoid parsing text
-- - No secondary indexes (DuckDB columnar scans + zone maps)
--
-- Invariants enforced by application code:
-- - projects.issue_counter allocates issue numbers and stable keys.
-- - issues.position orders issues within a column for drag and drop.
-- - Only one values.value_* column is set per row according to fields.kind.

PRAGMA foreign_keys = true;

-- ============================================================
-- Users and authentication
-- ============================================================

-- Users: global accounts (identity)
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR UNIQUE NOT NULL,
    username      VARCHAR UNIQUE NOT NULL,
    display_name  VARCHAR NOT NULL,
    password_hash VARCHAR NOT NULL
);

-- Sessions: auth session storage (cookie/token implementation detail)
CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR PRIMARY KEY,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Workspaces and membership
-- ============================================================

-- Workspace: tenant boundary (org/company)
CREATE TABLE IF NOT EXISTS workspaces (
    id   VARCHAR PRIMARY KEY,
    slug VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL
);

-- Workspace members: coarse access control
CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         VARCHAR NOT NULL DEFAULT 'member', -- owner, admin, member, guest
    joined_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id)
);

-- ============================================================
-- Teams (ESSENTIAL)
-- ============================================================

-- Team: primary organizing unit inside a workspace (Linear-like)
CREATE TABLE IF NOT EXISTS teams (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          VARCHAR NOT NULL, -- short code like "ENG"
    name         VARCHAR NOT NULL,
    UNIQUE (workspace_id, key),
    UNIQUE (workspace_id, name)
);

-- Team members: team-scoped membership and roles
CREATE TABLE IF NOT EXISTS team_members (
    team_id   VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id   VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      VARCHAR NOT NULL DEFAULT 'member', -- lead, member
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id)
);

-- ============================================================
-- Projects (Boards)
-- ============================================================

-- Project: board container under a team
CREATE TABLE IF NOT EXISTS projects (
    id            VARCHAR PRIMARY KEY,
    team_id       VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    key           VARCHAR NOT NULL,  -- optional prefix for issue keys ("PROJ")
    name          VARCHAR NOT NULL,
    issue_counter INTEGER NOT NULL DEFAULT 0, -- allocate sequential issue numbers per project
    UNIQUE (team_id, key)
);

-- ============================================================
-- Columns (Kanban columns)
-- ============================================================

-- Columns: per-project board columns (Trello lists, Jira/Linear workflow columns)
CREATE TABLE IF NOT EXISTS columns (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR NOT NULL,            -- "Todo", "Doing", "Done"
    position    INTEGER NOT NULL DEFAULT 0,  -- column order
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (project_id, name)
);

-- ============================================================
-- Cycles (planning periods)
-- ============================================================

-- Cycles: team-scoped time boxes (Linear cycles, Jira sprint equivalent).
-- A card may optionally belong to one cycle; cycle membership is not required to use the board.
CREATE TABLE IF NOT EXISTS cycles (
    id         VARCHAR PRIMARY KEY,
    team_id    VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,

    number     INTEGER NOT NULL, -- sequential per team
    name       VARCHAR NOT NULL, -- "Cycle 12" or custom name
    status     VARCHAR NOT NULL DEFAULT 'planning', -- planning, active, completed

    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (team_id, number)
);

-- ============================================================
-- Issues (Cards)
-- ============================================================

-- Issues: cards within a project board.
-- Minimal card model: title + column + ordering.
-- Everything else (description, priority, due date, labels, type, estimate, etc.) should be Fields.
CREATE TABLE IF NOT EXISTS issues (
    id         VARCHAR PRIMARY KEY,
    project_id VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    number     INTEGER NOT NULL, -- sequential per project
    key        VARCHAR NOT NULL, -- e.g. "PROJ-123"
    title      VARCHAR NOT NULL,

    column_id  VARCHAR NOT NULL REFERENCES columns(id) ON DELETE RESTRICT,
    position   INTEGER NOT NULL DEFAULT 0, -- ordering within a column for drag/drop

    creator_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    cycle_id   VARCHAR REFERENCES cycles(id) ON DELETE SET NULL, -- optional planning attachment

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (project_id, number),
    UNIQUE (project_id, key)
);

-- Assignees: common collaboration primitive (Trello members, Jira/Linear assignees).
-- Keeping this as a table avoids having to model it as a custom field early.
CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id  VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, user_id)
);

-- ============================================================
-- Comments (Discussion)
-- ============================================================

-- Comments: markdown text updates on a card.
CREATE TABLE IF NOT EXISTS comments (
    id         VARCHAR PRIMARY KEY,
    issue_id   VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author_id  VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    VARCHAR NOT NULL,
    edited_at  TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Fields and Values (Custom columns, monday/GitHub style)
-- ============================================================

-- Fields: project-scoped custom columns.
-- Add Jira/Linear-style attributes without schema changes:
--   description (text)
--   priority (select)
--   due_date (date)
--   estimate (number)
--   type (select)
--   labels (json array)
--   links (json) or later a dedicated table
--   AI metadata (ai_summary, ai_priority, embedding_ref, etc.)
CREATE TABLE IF NOT EXISTS fields (
    id          VARCHAR PRIMARY KEY,
    project_id  VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    key         VARCHAR NOT NULL, -- stable identifier: "priority", "due_date"
    name        VARCHAR NOT NULL, -- display label
    kind        VARCHAR NOT NULL, -- text, number, bool, date, ts, select, user, json

    position    INTEGER NOT NULL DEFAULT 0,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,

    settings_json VARCHAR,        -- JSON: options, formatting, select choices

    UNIQUE (project_id, key),
    UNIQUE (project_id, name)
);

-- Values: typed values for fields on a per-card basis.
-- Convention: only one value_* column is set, according to fields.kind.
CREATE TABLE IF NOT EXISTS values (
    issue_id  VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    field_id  VARCHAR NOT NULL REFERENCES fields(id) ON DELETE CASCADE,

    value_text VARCHAR,
    value_num  DOUBLE,
    value_bool BOOLEAN,
    value_date DATE,
    value_ts   TIMESTAMP,
    value_ref  VARCHAR,  -- referenced id (commonly users.id for kind=user)
    value_json VARCHAR,  -- arrays / complex objects (multi-select, structured data)

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (issue_id, field_id)
);
