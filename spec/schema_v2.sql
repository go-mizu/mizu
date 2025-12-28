-- ============================================================
-- GitHome Database Schema v2 (DuckDB-oriented)
--
-- Goals
--   1) Correctness-first: constrain duplicates and invalid states early
--   2) Keep OLTP surface minimal: DuckDB is columnar; avoid index sprawl
--   3) Make ownership and permissions model consistent across user/org
--   4) Prefer idempotent link tables using composite primary keys
--   5) Treat counters as cached/derived, not authoritative
--
-- Notes for DuckDB
--   - Foreign key enforcement may be weaker than Postgres depending on
--     engine version and settings. Keep FKs as documentation, but design
--     service-layer deletes as if FKs might not cascade.
--   - Indexes are intentionally limited. Add more only after profiling.
-- ============================================================

-- ============================================================
-- USERS + AUTH
-- ============================================================

-- Users are never hard-deleted in a Git hosting system.
-- We keep is_active and avoid cascade deletes from users to the world.
-- This preserves repos, issues, PRs, audit logs, and references.
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    username      VARCHAR NOT NULL UNIQUE,
    email         VARCHAR NOT NULL UNIQUE,
    password_hash VARCHAR NOT NULL,

    full_name     VARCHAR DEFAULT '',
    avatar_url    VARCHAR DEFAULT '',
    bio           TEXT DEFAULT '',
    location      VARCHAR DEFAULT '',
    website       VARCHAR DEFAULT '',
    company       VARCHAR DEFAULT '',

    is_admin      BOOLEAN DEFAULT FALSE,
    is_active     BOOLEAN DEFAULT TRUE,

    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions are ephemeral and safe to cascade with user.
CREATE TABLE IF NOT EXISTS sessions (
    id             VARCHAR PRIMARY KEY,
    user_id        VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at     TIMESTAMP NOT NULL,
    user_agent     VARCHAR DEFAULT '',
    ip_address     VARCHAR DEFAULT '',
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Minimal indexes: session lookups and expiration sweeps.
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- SSH keys are user-scoped; fingerprint is unique per user to prevent duplicates.
-- Optionally add UNIQUE(fingerprint) if you want global uniqueness.
CREATE TABLE IF NOT EXISTS ssh_keys (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR NOT NULL,
    public_key   TEXT NOT NULL,
    fingerprint  VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,

    UNIQUE(user_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id ON ssh_keys(user_id);

-- API tokens are user-scoped; store token_hash only (never store raw token).
CREATE TABLE IF NOT EXISTS api_tokens (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR NOT NULL,
    token_hash   VARCHAR NOT NULL,
    expires_at   TIMESTAMP,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,

    UNIQUE(user_id, token_hash)
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);

-- Token scopes are normalized instead of storing CSV in api_tokens.
-- Decision: use (token_id, scope) composite PK for idempotent insert.
CREATE TABLE IF NOT EXISTS api_token_scopes (
    token_id   VARCHAR NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
    scope      VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(token_id, scope),
    CHECK (scope <> '')
);

-- ============================================================
-- ORGS + ACTORS
-- ============================================================

-- Organizations are separate from users.
CREATE TABLE IF NOT EXISTS organizations (
    id           VARCHAR PRIMARY KEY,
    name         VARCHAR NOT NULL UNIQUE,
    slug         VARCHAR NOT NULL UNIQUE,
    display_name VARCHAR DEFAULT '',
    description  TEXT DEFAULT '',
    avatar_url   VARCHAR DEFAULT '',
    location     VARCHAR DEFAULT '',
    website      VARCHAR DEFAULT '',
    email        VARCHAR DEFAULT '',
    is_verified  BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Actors unify "owner" across users and orgs.
-- Decision: repos/activities reference actors so they can FK cleanly.
-- Deletion decision:
--   - If you hard-delete users/orgs, you likely DO NOT want repos deleted.
--   - Prefer soft-deleting users (is_active=false). Therefore, avoid
--     cascade from users/orgs to actors (use RESTRICT / NO ACTION).
--
-- DuckDB does not always enforce ON DELETE RESTRICT explicitly, but leaving
-- it unspecified is safer than CASCADE for hosting metadata.
CREATE TABLE IF NOT EXISTS actors (
    id         VARCHAR PRIMARY KEY,
    actor_type VARCHAR NOT NULL,
    user_id    VARCHAR REFERENCES users(id),
    org_id     VARCHAR REFERENCES organizations(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (actor_type IN ('user', 'org')),
    CHECK (
        (actor_type = 'user' AND user_id IS NOT NULL AND org_id IS NULL) OR
        (actor_type = 'org'  AND org_id  IS NOT NULL AND user_id IS NULL)
    ),

    UNIQUE(user_id),
    UNIQUE(org_id)
);

CREATE INDEX IF NOT EXISTS idx_actors_user_id ON actors(user_id);
CREATE INDEX IF NOT EXISTS idx_actors_org_id ON actors(org_id);

-- Org membership is link-table with composite PK to prevent duplicates.
CREATE TABLE IF NOT EXISTS org_members (
    org_id      VARCHAR NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        VARCHAR NOT NULL DEFAULT 'member',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (role IN ('member', 'admin', 'owner')),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON org_members(user_id);

-- ============================================================
-- TEAMS
-- ============================================================

-- Teams are scoped to an org.
-- Decision: UNIQUE(org_id, slug) to support routing like /orgs/:org/teams/:slug
CREATE TABLE IF NOT EXISTS teams (
    id          VARCHAR PRIMARY KEY,
    org_id      VARCHAR NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        VARCHAR NOT NULL,
    slug        VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    permission  VARCHAR NOT NULL DEFAULT 'read',
    parent_id   VARCHAR REFERENCES teams(id) ON DELETE SET NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    UNIQUE(org_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_teams_org_id ON teams(org_id);
CREATE INDEX IF NOT EXISTS idx_teams_parent_id ON teams(parent_id);

-- Team members composite PK for idempotent add/remove.
CREATE TABLE IF NOT EXISTS team_members (
    team_id    VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (role IN ('member', 'maintainer')),
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);

-- ============================================================
-- REPOSITORIES
-- ============================================================

-- Repositories belong to an actor (user or org).
-- Decision: repo name and slug uniqueness is per owner, not global.
-- We enforce UNIQUE(owner_actor_id, slug). Optionally also UNIQUE(owner_actor_id, name)
-- if you want case-sensitive "name" to be the canonical identifier.
CREATE TABLE IF NOT EXISTS repositories (
    id                  VARCHAR PRIMARY KEY,

    owner_actor_id       VARCHAR NOT NULL REFERENCES actors(id),

    name                VARCHAR NOT NULL,
    slug                VARCHAR NOT NULL,
    description         TEXT DEFAULT '',
    website             VARCHAR DEFAULT '',
    default_branch      VARCHAR DEFAULT 'main',

    is_private          BOOLEAN DEFAULT FALSE,
    is_archived         BOOLEAN DEFAULT FALSE,
    is_template         BOOLEAN DEFAULT FALSE,
    is_fork             BOOLEAN DEFAULT FALSE,
    forked_from_repo_id VARCHAR REFERENCES repositories(id) ON DELETE SET NULL,

    -- Cached counters only: treat as non-authoritative.
    -- Decision: keep for UI speed, but recompute periodically from source tables.
    star_count          INTEGER DEFAULT 0,
    fork_count          INTEGER DEFAULT 0,
    watcher_count       INTEGER DEFAULT 0,
    open_issue_count    INTEGER DEFAULT 0,
    open_pr_count       INTEGER DEFAULT 0,

    size_kb             INTEGER DEFAULT 0,
    license             VARCHAR DEFAULT '',

    has_issues          BOOLEAN DEFAULT TRUE,
    has_wiki            BOOLEAN DEFAULT FALSE,
    has_projects        BOOLEAN DEFAULT FALSE,

    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pushed_at           TIMESTAMP,

    CHECK (name <> ''),
    CHECK (slug <> ''),
    UNIQUE(owner_actor_id, slug)
);

-- Minimal indexes for owner listing and fork graph.
CREATE INDEX IF NOT EXISTS idx_repos_owner_actor ON repositories(owner_actor_id);
CREATE INDEX IF NOT EXISTS idx_repos_forked_from ON repositories(forked_from_repo_id);

-- Repo topics are normalized.
-- Decision: composite PK avoids duplicates and supports idempotent upsert logic.
CREATE TABLE IF NOT EXISTS repo_topics (
    repo_id     VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    topic       VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(repo_id, topic),
    CHECK (topic <> '')
);

-- Repo storage maps repo metadata to where git objects live.
-- Decision: keep git data out of DuckDB; store a pointer to FS/S3-like backend.
CREATE TABLE IF NOT EXISTS repo_storage (
    repo_id          VARCHAR PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    storage_backend  VARCHAR NOT NULL DEFAULT 'fs',
    storage_path     VARCHAR NOT NULL,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (storage_backend IN ('fs', 's3', 'r2', 'other')),
    CHECK (storage_path <> '')
);

-- Collaborators represent explicit user access to a repo.
CREATE TABLE IF NOT EXISTS collaborators (
    repo_id    VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR NOT NULL DEFAULT 'read',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    PRIMARY KEY (repo_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_collaborators_user_id ON collaborators(user_id);

-- Team access to repos.
CREATE TABLE IF NOT EXISTS team_repos (
    team_id    VARCHAR NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    repo_id    VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    permission VARCHAR NOT NULL DEFAULT 'read',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    PRIMARY KEY(team_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_team_repos_repo_id ON team_repos(repo_id);

-- Stars and watches are canonical source-of-truth tables.
CREATE TABLE IF NOT EXISTS stars (
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id    VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(user_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_stars_repo_id ON stars(repo_id);

CREATE TABLE IF NOT EXISTS watches (
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id    VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    level      VARCHAR NOT NULL DEFAULT 'watching',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (level IN ('watching', 'releases_only', 'ignore')),
    PRIMARY KEY(user_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_watches_repo_id ON watches(repo_id);

-- ============================================================
-- LABELS + MILESTONES
-- ============================================================

-- Labels are per-repo; UNIQUE(repo_id, name) enforces GitHub semantics.
CREATE TABLE IF NOT EXISTS labels (
    id          VARCHAR PRIMARY KEY,
    repo_id     VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    name        VARCHAR NOT NULL,
    color       VARCHAR NOT NULL DEFAULT '0366d6',
    description TEXT DEFAULT '',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(repo_id, name),
    CHECK (name <> ''),
    CHECK (color <> '')
);

CREATE INDEX IF NOT EXISTS idx_labels_repo_id ON labels(repo_id);

-- Milestones are per-repo and numbered.
CREATE TABLE IF NOT EXISTS milestones (
    id          VARCHAR PRIMARY KEY,
    repo_id     VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number      INTEGER NOT NULL,
    title       VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    state       VARCHAR NOT NULL DEFAULT 'open',
    due_date    TIMESTAMP,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at   TIMESTAMP,

    CHECK (number > 0),
    CHECK (title <> ''),
    CHECK (state IN ('open', 'closed')),
    UNIQUE(repo_id, number)
);

CREATE INDEX IF NOT EXISTS idx_milestones_repo_id ON milestones(repo_id);

-- ============================================================
-- ISSUES
-- ============================================================

-- Issues are numbered per repo.
-- Decision: keep milestone_id FK, but enforce "milestone must belong to same repo"
-- in service layer (cross-table constraint).
CREATE TABLE IF NOT EXISTS issues (
    id              VARCHAR PRIMARY KEY,
    repo_id         VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number          INTEGER NOT NULL,
    title           VARCHAR NOT NULL,
    body            TEXT DEFAULT '',
    author_id       VARCHAR NOT NULL REFERENCES users(id),

    state           VARCHAR NOT NULL DEFAULT 'open',
    state_reason    VARCHAR DEFAULT '',
    is_locked       BOOLEAN DEFAULT FALSE,
    lock_reason     VARCHAR DEFAULT '',

    milestone_id    VARCHAR REFERENCES milestones(id) ON DELETE SET NULL,

    -- Cached counters
    comment_count   INTEGER DEFAULT 0,
    reactions_count INTEGER DEFAULT 0,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at       TIMESTAMP,
    closed_by_id    VARCHAR REFERENCES users(id) ON DELETE SET NULL,

    CHECK (number > 0),
    CHECK (title <> ''),
    CHECK (state IN ('open', 'closed')),
    UNIQUE(repo_id, number)
);

CREATE INDEX IF NOT EXISTS idx_issues_repo_state_updated ON issues(repo_id, state, updated_at);
CREATE INDEX IF NOT EXISTS idx_issues_author_id ON issues(author_id);

-- Many-to-many issue labels: composite PK prevents duplicates.
CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id   VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id   VARCHAR NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(issue_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_labels_label_id ON issue_labels(label_id);

-- Issue assignees: composite PK prevents duplicates.
CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id   VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(issue_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_assignees_user_id ON issue_assignees(user_id);

-- ============================================================
-- PULL REQUESTS
-- ============================================================

-- Pull requests are numbered per repo.
-- Decision: include lock_reason for symmetry with issues and to support API.
-- Decision: merge_method stored but mergeability is still computed in service
-- (conflicts + branch rules + statuses). mergeable fields are effectively cached.
CREATE TABLE IF NOT EXISTS pull_requests (
    id               VARCHAR PRIMARY KEY,
    repo_id          VARCHAR NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number           INTEGER NOT NULL,
    title            VARCHAR NOT NULL,
    body             TEXT DEFAULT '',
    author_id        VARCHAR NOT NULL REFERENCES users(id),

    head_repo_id     VARCHAR REFERENCES repositories(id) ON DELETE SET NULL,
    head_branch      VARCHAR NOT NULL,
    head_sha         VARCHAR NOT NULL,
    base_branch      VARCHAR NOT NULL,
    base_sha         VARCHAR NOT NULL,

    state            VARCHAR NOT NULL DEFAULT 'open',
    is_draft         BOOLEAN DEFAULT FALSE,
    is_locked        BOOLEAN DEFAULT FALSE,
    lock_reason      VARCHAR DEFAULT '',

    mergeable        BOOLEAN DEFAULT TRUE,
    mergeable_state  VARCHAR DEFAULT '',
    merge_method     VARCHAR DEFAULT '',
    merge_commit_sha VARCHAR DEFAULT '',
    merge_message    TEXT DEFAULT '',

    merged_at        TIMESTAMP,
    merged_by_id     VARCHAR REFERENCES users(id) ON DELETE SET NULL,

    additions        INTEGER DEFAULT 0,
    deletions        INTEGER DEFAULT 0,
    changed_files    INTEGER DEFAULT 0,

    comment_count    INTEGER DEFAULT 0,
    review_comments  INTEGER DEFAULT 0,
    commits          INTEGER DEFAULT 0,

    milestone_id     VARCHAR REFERENCES milestones(id) ON DELETE SET NULL,

    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at        TIMESTAMP,

    CHECK (number > 0),
    CHECK (title <> ''),
    CHECK (state IN ('open', 'closed', 'merged')),
    CHECK (merge_method IN ('', 'merge', 'squash', 'rebase')),
    UNIQUE(repo_id, number)
);

CREATE INDEX IF NOT EXISTS idx_prs_repo_state_updated ON pull_requests(repo_id, state, updated_at);
CREATE INDEX IF NOT EXISTS idx_prs_author_id ON pull_requests(author_id);

-- PR labels: composite PK for idempotent label add/remove.
CREATE TABLE IF NOT EXISTS pr_labels (
    pr_id      VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    label_id   VARCHAR NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(pr_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_labels_label_id ON pr_labels(label_id);

-- PR assignees: composite PK for idempotent assignee add/remove.
CREATE TABLE IF NOT EXISTS pr_assignees (
    pr_id      VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(pr_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_assignees_user_id ON pr_assignees(user_id);

-- PR reviewers represent active review requests.
-- Decision: only keep active requests; remove means DELETE row.
-- Therefore, "state" can remain simple as 'pending' or you can drop it.
-- We keep it to support future transitions like 'reviewed' if you want.
CREATE TABLE IF NOT EXISTS pr_reviewers (
    pr_id      VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id    VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state      VARCHAR NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (state IN ('pending', 'reviewed')),
    PRIMARY KEY(pr_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pr_reviewers(user_id);

-- PR reviews are immutable events with optional submission time.
CREATE TABLE IF NOT EXISTS pr_reviews (
    id           VARCHAR PRIMARY KEY,
    pr_id        VARCHAR NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id      VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body         TEXT DEFAULT '',
    state        VARCHAR NOT NULL,
    commit_sha   VARCHAR,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP,

    CHECK (state IN ('pending', 'commented', 'approved', 'changes_requested', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_pr_reviews_pr_id ON pr_reviews(pr_id);
CREATE INDEX IF NOT EXISTS idx_pr_reviews_user_id ON pr_reviews(user_
