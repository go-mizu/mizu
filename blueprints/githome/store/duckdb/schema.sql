-- GitHome Database Schema

-- Users and Authentication
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    full_name VARCHAR DEFAULT '',
    avatar_url VARCHAR DEFAULT '',
    bio TEXT DEFAULT '',
    location VARCHAR DEFAULT '',
    website VARCHAR DEFAULT '',
    company VARCHAR DEFAULT '',
    is_admin BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    user_agent VARCHAR DEFAULT '',
    ip_address VARCHAR DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ssh_keys (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    public_key TEXT NOT NULL,
    fingerprint VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    token_hash VARCHAR NOT NULL,
    scopes VARCHAR DEFAULT '',
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP
);

-- Organizations and Teams
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    slug VARCHAR UNIQUE NOT NULL,
    display_name VARCHAR DEFAULT '',
    description TEXT DEFAULT '',
    avatar_url VARCHAR DEFAULT '',
    location VARCHAR DEFAULT '',
    website VARCHAR DEFAULT '',
    email VARCHAR DEFAULT '',
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS org_members (
    id VARCHAR PRIMARY KEY,
    org_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    role VARCHAR NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS teams (
    id VARCHAR PRIMARY KEY,
    org_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    slug VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    permission VARCHAR NOT NULL DEFAULT 'read',
    parent_id VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_members (
    id VARCHAR PRIMARY KEY,
    team_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    role VARCHAR NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_repos (
    id VARCHAR PRIMARY KEY,
    team_id VARCHAR NOT NULL,
    repo_id VARCHAR NOT NULL,
    permission VARCHAR NOT NULL DEFAULT 'read',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Repositories
CREATE TABLE IF NOT EXISTS repositories (
    id VARCHAR PRIMARY KEY,
    owner_id VARCHAR NOT NULL,
    owner_type VARCHAR NOT NULL DEFAULT 'user',
    name VARCHAR NOT NULL,
    slug VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    website VARCHAR DEFAULT '',
    default_branch VARCHAR DEFAULT 'main',
    is_private BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    is_fork BOOLEAN DEFAULT FALSE,
    forked_from_id VARCHAR,
    star_count INTEGER DEFAULT 0,
    fork_count INTEGER DEFAULT 0,
    watcher_count INTEGER DEFAULT 0,
    open_issue_count INTEGER DEFAULT 0,
    open_pr_count INTEGER DEFAULT 0,
    size_kb INTEGER DEFAULT 0,
    topics VARCHAR DEFAULT '',
    license VARCHAR DEFAULT '',
    has_issues BOOLEAN DEFAULT TRUE,
    has_wiki BOOLEAN DEFAULT FALSE,
    has_projects BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pushed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS collaborators (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    permission VARCHAR NOT NULL DEFAULT 'read',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stars (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    repo_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS watches (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    repo_id VARCHAR NOT NULL,
    level VARCHAR NOT NULL DEFAULT 'watching',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Labels and Milestones
CREATE TABLE IF NOT EXISTS labels (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    color VARCHAR NOT NULL DEFAULT '0366d6',
    description TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS milestones (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL,
    number INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    description TEXT DEFAULT '',
    state VARCHAR NOT NULL DEFAULT 'open',
    due_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP
);

-- Issues
CREATE TABLE IF NOT EXISTS issues (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL,
    number INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    body TEXT DEFAULT '',
    author_id VARCHAR NOT NULL,
    assignee_id VARCHAR,
    state VARCHAR NOT NULL DEFAULT 'open',
    state_reason VARCHAR DEFAULT '',
    is_locked BOOLEAN DEFAULT FALSE,
    lock_reason VARCHAR DEFAULT '',
    milestone_id VARCHAR,
    comment_count INTEGER DEFAULT 0,
    reactions_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    closed_by_id VARCHAR
);

CREATE TABLE IF NOT EXISTS issue_labels (
    id VARCHAR PRIMARY KEY,
    issue_id VARCHAR NOT NULL,
    label_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS issue_assignees (
    id VARCHAR PRIMARY KEY,
    issue_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Pull Requests
CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL,
    number INTEGER NOT NULL,
    title VARCHAR NOT NULL,
    body TEXT DEFAULT '',
    author_id VARCHAR NOT NULL,
    head_repo_id VARCHAR,
    head_branch VARCHAR NOT NULL,
    head_sha VARCHAR NOT NULL,
    base_branch VARCHAR NOT NULL,
    base_sha VARCHAR NOT NULL,
    state VARCHAR NOT NULL DEFAULT 'open',
    is_draft BOOLEAN DEFAULT FALSE,
    is_locked BOOLEAN DEFAULT FALSE,
    mergeable BOOLEAN DEFAULT TRUE,
    mergeable_state VARCHAR DEFAULT '',
    merge_commit_sha VARCHAR DEFAULT '',
    merged_at TIMESTAMP,
    merged_by_id VARCHAR,
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    changed_files INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    review_comments INTEGER DEFAULT 0,
    commits INTEGER DEFAULT 0,
    milestone_id VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_labels (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL,
    label_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_assignees (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_reviewers (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    state VARCHAR DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_reviews (
    id VARCHAR PRIMARY KEY,
    pr_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    body TEXT DEFAULT '',
    state VARCHAR NOT NULL,
    commit_sha VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS review_comments (
    id VARCHAR PRIMARY KEY,
    review_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    path VARCHAR NOT NULL,
    position INTEGER,
    original_position INTEGER,
    diff_hunk TEXT,
    line INTEGER,
    original_line INTEGER,
    side VARCHAR DEFAULT 'RIGHT',
    body TEXT NOT NULL,
    in_reply_to_id VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Comments (for issues and PRs)
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Releases
CREATE TABLE IF NOT EXISTS releases (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR NOT NULL,
    tag_name VARCHAR NOT NULL,
    target_commitish VARCHAR NOT NULL DEFAULT 'main',
    name VARCHAR DEFAULT '',
    body TEXT DEFAULT '',
    is_draft BOOLEAN DEFAULT FALSE,
    is_prerelease BOOLEAN DEFAULT FALSE,
    author_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS release_assets (
    id VARCHAR PRIMARY KEY,
    release_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    label VARCHAR DEFAULT '',
    content_type VARCHAR NOT NULL,
    size_bytes INTEGER NOT NULL,
    download_count INTEGER DEFAULT 0,
    uploader_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR PRIMARY KEY,
    repo_id VARCHAR,
    org_id VARCHAR,
    url VARCHAR NOT NULL,
    secret VARCHAR DEFAULT '',
    content_type VARCHAR DEFAULT 'json',
    events VARCHAR NOT NULL DEFAULT 'push',
    active BOOLEAN DEFAULT TRUE,
    insecure_ssl BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_response_code INTEGER,
    last_response_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR PRIMARY KEY,
    webhook_id VARCHAR NOT NULL,
    event VARCHAR NOT NULL,
    guid VARCHAR NOT NULL,
    payload TEXT NOT NULL,
    request_headers TEXT,
    response_headers TEXT,
    response_body TEXT,
    status_code INTEGER,
    delivered BOOLEAN DEFAULT FALSE,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    repo_id VARCHAR,
    type VARCHAR NOT NULL,
    actor_id VARCHAR,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    reason VARCHAR NOT NULL,
    unread BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMP
);

-- Activity Feed
CREATE TABLE IF NOT EXISTS activities (
    id VARCHAR PRIMARY KEY,
    actor_id VARCHAR NOT NULL,
    event_type VARCHAR NOT NULL,
    repo_id VARCHAR,
    target_type VARCHAR,
    target_id VARCHAR,
    ref VARCHAR DEFAULT '',
    ref_type VARCHAR DEFAULT '',
    payload TEXT DEFAULT '{}',
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reactions
CREATE TABLE IF NOT EXISTS reactions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    content VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_repos_owner ON repositories(owner_id, owner_type);
CREATE INDEX IF NOT EXISTS idx_repos_slug ON repositories(slug);
CREATE INDEX IF NOT EXISTS idx_issues_repo ON issues(repo_id);
CREATE INDEX IF NOT EXISTS idx_issues_state ON issues(repo_id, state);
CREATE INDEX IF NOT EXISTS idx_issues_author ON issues(author_id);
CREATE INDEX IF NOT EXISTS idx_prs_repo ON pull_requests(repo_id);
CREATE INDEX IF NOT EXISTS idx_prs_state ON pull_requests(repo_id, state);
CREATE INDEX IF NOT EXISTS idx_prs_author ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_comments_target ON comments(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_activities_actor ON activities(actor_id);
CREATE INDEX IF NOT EXISTS idx_activities_repo ON activities(repo_id);
CREATE INDEX IF NOT EXISTS idx_activities_created ON activities(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, unread);
CREATE INDEX IF NOT EXISTS idx_stars_user ON stars(user_id);
CREATE INDEX IF NOT EXISTS idx_stars_repo ON stars(repo_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_repo ON collaborators(repo_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_user ON collaborators(user_id);
