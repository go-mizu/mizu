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

CREATE TABLE IF NOT EXISTS sessions (
    id             VARCHAR PRIMARY KEY,
    user_id        VARCHAR NOT NULL,
    expires_at     TIMESTAMP NOT NULL,
    user_agent     VARCHAR DEFAULT '',
    ip_address     VARCHAR DEFAULT '',
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS ssh_keys (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    public_key   TEXT NOT NULL,
    fingerprint  VARCHAR NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,

    UNIQUE(user_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id ON ssh_keys(user_id);

CREATE TABLE IF NOT EXISTS api_tokens (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    token_hash   VARCHAR NOT NULL,
    expires_at   TIMESTAMP,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,

    UNIQUE(user_id, token_hash)
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);

CREATE TABLE IF NOT EXISTS api_token_scopes (
    token_id   VARCHAR NOT NULL,
    scope      VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(token_id, scope),
    CHECK (scope <> '')
);

-- ============================================================
-- ORGS + ACTORS
-- ============================================================

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

CREATE TABLE IF NOT EXISTS actors (
    id         VARCHAR PRIMARY KEY,
    actor_type VARCHAR NOT NULL,
    user_id    VARCHAR,
    org_id     VARCHAR,
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

CREATE TABLE IF NOT EXISTS org_members (
    org_id      VARCHAR NOT NULL,
    user_id     VARCHAR NOT NULL,
    role        VARCHAR NOT NULL DEFAULT 'member',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (role IN ('member', 'admin', 'owner')),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON org_members(user_id);

-- ============================================================
-- TEAMS
-- ============================================================

CREATE TABLE IF NOT EXISTS teams (
    id          VARCHAR PRIMARY KEY,
    org_id      VARCHAR NOT NULL,
    name        VARCHAR NOT NULL,
    slug        VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    permission  VARCHAR NOT NULL DEFAULT 'read',
    parent_id   VARCHAR,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    UNIQUE(org_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_teams_org_id ON teams(org_id);
CREATE INDEX IF NOT EXISTS idx_teams_parent_id ON teams(parent_id);

CREATE TABLE IF NOT EXISTS team_members (
    team_id    VARCHAR NOT NULL,
    user_id    VARCHAR NOT NULL,
    role       VARCHAR NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (role IN ('member', 'maintainer')),
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);

-- ============================================================
-- REPOSITORIES
-- ============================================================

CREATE TABLE IF NOT EXISTS repositories (
    id                  VARCHAR PRIMARY KEY,

    owner_actor_id       VARCHAR NOT NULL,

    name                VARCHAR NOT NULL,
    slug                VARCHAR NOT NULL,
    description         TEXT DEFAULT '',
    website             VARCHAR DEFAULT '',
    default_branch      VARCHAR DEFAULT 'main',

    is_private          BOOLEAN DEFAULT FALSE,
    is_archived         BOOLEAN DEFAULT FALSE,
    is_template         BOOLEAN DEFAULT FALSE,
    is_fork             BOOLEAN DEFAULT FALSE,
    forked_from_repo_id VARCHAR,

    star_count          INTEGER DEFAULT 0,
    fork_count          INTEGER DEFAULT 0,
    watcher_count       INTEGER DEFAULT 0,
    open_issue_count    INTEGER DEFAULT 0,
    open_pr_count       INTEGER DEFAULT 0,

    size_kb             INTEGER DEFAULT 0,
    license             VARCHAR DEFAULT '',

    language            VARCHAR DEFAULT '',
    language_color      VARCHAR DEFAULT '',

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

CREATE INDEX IF NOT EXISTS idx_repos_owner_actor ON repositories(owner_actor_id);
CREATE INDEX IF NOT EXISTS idx_repos_forked_from ON repositories(forked_from_repo_id);

CREATE TABLE IF NOT EXISTS repo_topics (
    repo_id     VARCHAR NOT NULL,
    topic       VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(repo_id, topic),
    CHECK (topic <> '')
);

CREATE TABLE IF NOT EXISTS repo_storage (
    repo_id          VARCHAR PRIMARY KEY,
    storage_backend  VARCHAR NOT NULL DEFAULT 'fs',
    storage_path     VARCHAR NOT NULL,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (storage_backend IN ('fs', 's3', 'r2', 'other')),
    CHECK (storage_path <> '')
);

CREATE TABLE IF NOT EXISTS collaborators (
    repo_id    VARCHAR NOT NULL,
    user_id    VARCHAR NOT NULL,
    permission VARCHAR NOT NULL DEFAULT 'read',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    PRIMARY KEY (repo_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_collaborators_user_id ON collaborators(user_id);

CREATE TABLE IF NOT EXISTS team_repos (
    team_id    VARCHAR NOT NULL,
    repo_id    VARCHAR NOT NULL,
    permission VARCHAR NOT NULL DEFAULT 'read',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    PRIMARY KEY(team_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_team_repos_repo_id ON team_repos(repo_id);

CREATE TABLE IF NOT EXISTS stars (
    user_id    VARCHAR NOT NULL,
    repo_id    VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(user_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_stars_repo_id ON stars(repo_id);

CREATE TABLE IF NOT EXISTS watches (
    user_id    VARCHAR NOT NULL,
    repo_id    VARCHAR NOT NULL,
    level      VARCHAR NOT NULL DEFAULT 'watching',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (level IN ('watching', 'releases_only', 'ignoring')),
    PRIMARY KEY(user_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_watches_repo_id ON watches(repo_id);

-- ============================================================
-- LABELS + MILESTONES
-- ============================================================

CREATE TABLE IF NOT EXISTS labels (
    id          VARCHAR PRIMARY KEY,
    repo_id     VARCHAR NOT NULL,
    name        VARCHAR NOT NULL,
    color       VARCHAR NOT NULL DEFAULT '0366d6',
    description TEXT DEFAULT '',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(repo_id, name),
    CHECK (name <> ''),
    CHECK (color <> '')
);

CREATE INDEX IF NOT EXISTS idx_labels_repo_id ON labels(repo_id);

CREATE TABLE IF NOT EXISTS milestones (
    id          VARCHAR PRIMARY KEY,
    repo_id     VARCHAR NOT NULL,
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

CREATE TABLE IF NOT EXISTS issues (
    id              VARCHAR PRIMARY KEY,
    repo_id         VARCHAR NOT NULL,
    number          INTEGER NOT NULL,
    title           VARCHAR NOT NULL,
    body            TEXT DEFAULT '',
    author_id       VARCHAR NOT NULL,

    state           VARCHAR NOT NULL DEFAULT 'open',
    state_reason    VARCHAR DEFAULT '',
    is_locked       BOOLEAN DEFAULT FALSE,
    lock_reason     VARCHAR DEFAULT '',

    milestone_id    VARCHAR,

    comment_count   INTEGER DEFAULT 0,
    reactions_count INTEGER DEFAULT 0,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at       TIMESTAMP,
    closed_by_id    VARCHAR,

    CHECK (number > 0),
    CHECK (title <> ''),
    CHECK (state IN ('open', 'closed')),
    UNIQUE(repo_id, number)
);

CREATE INDEX IF NOT EXISTS idx_issues_repo_state_updated ON issues(repo_id, state, updated_at);
CREATE INDEX IF NOT EXISTS idx_issues_author_id ON issues(author_id);

CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id   VARCHAR NOT NULL,
    label_id   VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(issue_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_labels_label_id ON issue_labels(label_id);

CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id   VARCHAR NOT NULL,
    user_id    VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(issue_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_assignees_user_id ON issue_assignees(user_id);

-- ============================================================
-- PULL REQUESTS
-- ============================================================

CREATE TABLE IF NOT EXISTS pull_requests (
    id               VARCHAR PRIMARY KEY,
    repo_id          VARCHAR NOT NULL,
    number           INTEGER NOT NULL,
    title            VARCHAR NOT NULL,
    body             TEXT DEFAULT '',
    author_id        VARCHAR NOT NULL,

    head_repo_id     VARCHAR,
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
    merged_by_id     VARCHAR,

    additions        INTEGER DEFAULT 0,
    deletions        INTEGER DEFAULT 0,
    changed_files    INTEGER DEFAULT 0,

    comment_count    INTEGER DEFAULT 0,
    review_comments  INTEGER DEFAULT 0,
    commits          INTEGER DEFAULT 0,

    milestone_id     VARCHAR,

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

CREATE TABLE IF NOT EXISTS pr_labels (
    pr_id      VARCHAR NOT NULL,
    label_id   VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(pr_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_labels_label_id ON pr_labels(label_id);

CREATE TABLE IF NOT EXISTS pr_assignees (
    pr_id      VARCHAR NOT NULL,
    user_id    VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(pr_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_assignees_user_id ON pr_assignees(user_id);

CREATE TABLE IF NOT EXISTS pr_reviewers (
    pr_id      VARCHAR NOT NULL,
    user_id    VARCHAR NOT NULL,
    state      VARCHAR NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (state IN ('pending', 'reviewed')),
    PRIMARY KEY(pr_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pr_reviewers(user_id);

CREATE TABLE IF NOT EXISTS pr_reviews (
    id           VARCHAR PRIMARY KEY,
    pr_id        VARCHAR NOT NULL,
    user_id      VARCHAR NOT NULL,
    body         TEXT DEFAULT '',
    state        VARCHAR NOT NULL,
    commit_sha   VARCHAR,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP,

    CHECK (state IN ('pending', 'commented', 'approved', 'changes_requested', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_pr_reviews_pr_id ON pr_reviews(pr_id);
CREATE INDEX IF NOT EXISTS idx_pr_reviews_user_id ON pr_reviews(user_id);

CREATE TABLE IF NOT EXISTS review_comments (
    id                VARCHAR PRIMARY KEY,
    review_id         VARCHAR NOT NULL,
    user_id           VARCHAR NOT NULL,
    path              VARCHAR NOT NULL,
    position          INTEGER,
    original_position INTEGER,
    diff_hunk         TEXT,
    line              INTEGER,
    original_line     INTEGER,
    side              VARCHAR DEFAULT 'RIGHT',
    body              TEXT NOT NULL,
    in_reply_to_id    VARCHAR,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_review_comments_review_id ON review_comments(review_id);

-- ============================================================
-- COMMENTS (for issues and PRs)
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR PRIMARY KEY,
    target_type VARCHAR NOT NULL,
    target_id   VARCHAR NOT NULL,
    user_id     VARCHAR NOT NULL,
    body        TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (target_type IN ('issue', 'pull_request'))
);

CREATE INDEX IF NOT EXISTS idx_comments_target ON comments(target_type, target_id);

-- ============================================================
-- RELEASES
-- ============================================================

CREATE TABLE IF NOT EXISTS releases (
    id               VARCHAR PRIMARY KEY,
    repo_id          VARCHAR NOT NULL,
    tag_name         VARCHAR NOT NULL,
    target_commitish VARCHAR NOT NULL DEFAULT 'main',
    name             VARCHAR DEFAULT '',
    body             TEXT DEFAULT '',
    is_draft         BOOLEAN DEFAULT FALSE,
    is_prerelease    BOOLEAN DEFAULT FALSE,
    author_id        VARCHAR NOT NULL,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at     TIMESTAMP,

    UNIQUE(repo_id, tag_name)
);

CREATE INDEX IF NOT EXISTS idx_releases_repo_id ON releases(repo_id);

CREATE TABLE IF NOT EXISTS release_assets (
    id             VARCHAR PRIMARY KEY,
    release_id     VARCHAR NOT NULL,
    name           VARCHAR NOT NULL,
    label          VARCHAR DEFAULT '',
    content_type   VARCHAR NOT NULL,
    size_bytes     INTEGER NOT NULL,
    download_count INTEGER DEFAULT 0,
    uploader_id    VARCHAR NOT NULL,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_release_assets_release_id ON release_assets(release_id);

-- ============================================================
-- WEBHOOKS
-- ============================================================

CREATE TABLE IF NOT EXISTS webhooks (
    id                 VARCHAR PRIMARY KEY,
    repo_id            VARCHAR,
    org_id             VARCHAR,
    url                VARCHAR NOT NULL,
    secret             VARCHAR DEFAULT '',
    content_type       VARCHAR DEFAULT 'json',
    events             VARCHAR NOT NULL DEFAULT 'push',
    active             BOOLEAN DEFAULT TRUE,
    insecure_ssl       BOOLEAN DEFAULT FALSE,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_response_code INTEGER,
    last_response_at   TIMESTAMP,

    CHECK (repo_id IS NOT NULL OR org_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_webhooks_repo_id ON webhooks(repo_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_org_id ON webhooks(org_id);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id               VARCHAR PRIMARY KEY,
    webhook_id       VARCHAR NOT NULL,
    event            VARCHAR NOT NULL,
    guid             VARCHAR NOT NULL,
    payload          TEXT NOT NULL,
    request_headers  TEXT,
    response_headers TEXT,
    response_body    TEXT,
    status_code      INTEGER,
    delivered        BOOLEAN DEFAULT FALSE,
    duration_ms      INTEGER,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);

-- ============================================================
-- NOTIFICATIONS
-- ============================================================

CREATE TABLE IF NOT EXISTS notifications (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    repo_id      VARCHAR,
    type         VARCHAR NOT NULL,
    actor_id     VARCHAR,
    target_type  VARCHAR NOT NULL,
    target_id    VARCHAR NOT NULL,
    title        VARCHAR NOT NULL,
    reason       VARCHAR NOT NULL,
    unread       BOOLEAN DEFAULT TRUE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, unread);

-- ============================================================
-- ACTIVITY FEED
-- ============================================================

CREATE TABLE IF NOT EXISTS activities (
    id          VARCHAR PRIMARY KEY,
    actor_id    VARCHAR NOT NULL,
    event_type  VARCHAR NOT NULL,
    repo_id     VARCHAR,
    target_type VARCHAR,
    target_id   VARCHAR,
    ref         VARCHAR DEFAULT '',
    ref_type    VARCHAR DEFAULT '',
    payload     TEXT DEFAULT '{}',
    is_public   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activities_actor ON activities(actor_id);
CREATE INDEX IF NOT EXISTS idx_activities_repo ON activities(repo_id);
CREATE INDEX IF NOT EXISTS idx_activities_created ON activities(created_at);

-- ============================================================
-- REACTIONS
-- ============================================================

CREATE TABLE IF NOT EXISTS reactions (
    user_id     VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id   VARCHAR NOT NULL,
    content     VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (content IN ('+1', '-1', 'laugh', 'confused', 'heart', 'hooray', 'rocket', 'eyes')),
    PRIMARY KEY(user_id, target_type, target_id, content)
);

CREATE INDEX IF NOT EXISTS idx_reactions_target ON reactions(target_type, target_id);
