-- schema.sql
-- GitHome DuckDB schema - GitHub-compatible data model
-- Implements all feature/* Store interfaces

-- ============================================================
-- Sequences for auto-incrementing IDs
-- ============================================================

CREATE SEQUENCE IF NOT EXISTS seq_users;
CREATE SEQUENCE IF NOT EXISTS seq_organizations;
CREATE SEQUENCE IF NOT EXISTS seq_repositories;
CREATE SEQUENCE IF NOT EXISTS seq_teams;
CREATE SEQUENCE IF NOT EXISTS seq_issues;
CREATE SEQUENCE IF NOT EXISTS seq_issue_events;
CREATE SEQUENCE IF NOT EXISTS seq_labels;
CREATE SEQUENCE IF NOT EXISTS seq_milestones;
CREATE SEQUENCE IF NOT EXISTS seq_issue_comments;
CREATE SEQUENCE IF NOT EXISTS seq_commit_comments;
CREATE SEQUENCE IF NOT EXISTS seq_pull_requests;
CREATE SEQUENCE IF NOT EXISTS seq_pr_reviews;
CREATE SEQUENCE IF NOT EXISTS seq_pr_review_comments;
CREATE SEQUENCE IF NOT EXISTS seq_collaborator_invitations;
CREATE SEQUENCE IF NOT EXISTS seq_reactions;
CREATE SEQUENCE IF NOT EXISTS seq_releases;
CREATE SEQUENCE IF NOT EXISTS seq_release_assets;
CREATE SEQUENCE IF NOT EXISTS seq_webhooks;
CREATE SEQUENCE IF NOT EXISTS seq_webhook_deliveries;

-- ============================================================
-- Users and Authentication
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_users'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    login         VARCHAR NOT NULL,
    name          VARCHAR NOT NULL DEFAULT '',
    email         VARCHAR,
    avatar_url    VARCHAR NOT NULL DEFAULT '',
    gravatar_id   VARCHAR NOT NULL DEFAULT '',
    type          VARCHAR NOT NULL DEFAULT 'User',
    site_admin    BOOLEAN NOT NULL DEFAULT FALSE,
    bio           VARCHAR NOT NULL DEFAULT '',
    blog          VARCHAR NOT NULL DEFAULT '',
    location      VARCHAR NOT NULL DEFAULT '',
    company       VARCHAR NOT NULL DEFAULT '',
    hireable      BOOLEAN NOT NULL DEFAULT FALSE,
    twitter_username VARCHAR NOT NULL DEFAULT '',
    public_repos  INTEGER NOT NULL DEFAULT 0,
    public_gists  INTEGER NOT NULL DEFAULT 0,
    followers     INTEGER NOT NULL DEFAULT 0,
    following     INTEGER NOT NULL DEFAULT 0,
    password_hash VARCHAR NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_follows (
    follower_id   BIGINT NOT NULL,
    followed_id   BIGINT NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, followed_id)
);

-- ============================================================
-- Organizations
-- ============================================================

CREATE TABLE IF NOT EXISTS organizations (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_organizations'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    login         VARCHAR NOT NULL,
    name          VARCHAR NOT NULL DEFAULT '',
    description   VARCHAR NOT NULL DEFAULT '',
    company       VARCHAR NOT NULL DEFAULT '',
    blog          VARCHAR NOT NULL DEFAULT '',
    location      VARCHAR NOT NULL DEFAULT '',
    email         VARCHAR NOT NULL DEFAULT '',
    twitter_username VARCHAR NOT NULL DEFAULT '',
    avatar_url    VARCHAR NOT NULL DEFAULT '',
    is_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    has_organization_projects BOOLEAN NOT NULL DEFAULT TRUE,
    has_repository_projects BOOLEAN NOT NULL DEFAULT TRUE,
    public_repos  INTEGER NOT NULL DEFAULT 0,
    public_gists  INTEGER NOT NULL DEFAULT 0,
    followers     INTEGER NOT NULL DEFAULT 0,
    following     INTEGER NOT NULL DEFAULT 0,
    total_private_repos INTEGER NOT NULL DEFAULT 0,
    owned_private_repos INTEGER NOT NULL DEFAULT 0,
    default_repository_permission VARCHAR NOT NULL DEFAULT 'read',
    members_can_create_repositories BOOLEAN NOT NULL DEFAULT TRUE,
    members_can_create_public_repositories BOOLEAN NOT NULL DEFAULT TRUE,
    members_can_create_private_repositories BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS org_members (
    org_id        BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    role          VARCHAR NOT NULL DEFAULT 'member',
    is_public     BOOLEAN NOT NULL DEFAULT FALSE,
    state         VARCHAR NOT NULL DEFAULT 'active',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (org_id, user_id)
);

-- ============================================================
-- Repositories
-- ============================================================

CREATE TABLE IF NOT EXISTS repositories (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_repositories'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    name          VARCHAR NOT NULL,
    full_name     VARCHAR NOT NULL,
    owner_id      BIGINT NOT NULL,
    owner_type    VARCHAR NOT NULL DEFAULT 'User',
    private       BOOLEAN NOT NULL DEFAULT FALSE,
    description   VARCHAR NOT NULL DEFAULT '',
    fork          BOOLEAN NOT NULL DEFAULT FALSE,
    parent_id     BIGINT,
    homepage      VARCHAR NOT NULL DEFAULT '',
    language      VARCHAR NOT NULL DEFAULT '',
    forks_count   INTEGER NOT NULL DEFAULT 0,
    stargazers_count INTEGER NOT NULL DEFAULT 0,
    watchers_count INTEGER NOT NULL DEFAULT 0,
    size          INTEGER NOT NULL DEFAULT 0,
    default_branch VARCHAR NOT NULL DEFAULT 'main',
    open_issues_count INTEGER NOT NULL DEFAULT 0,
    is_template   BOOLEAN NOT NULL DEFAULT FALSE,
    has_issues    BOOLEAN NOT NULL DEFAULT TRUE,
    has_projects  BOOLEAN NOT NULL DEFAULT TRUE,
    has_wiki      BOOLEAN NOT NULL DEFAULT TRUE,
    has_pages     BOOLEAN NOT NULL DEFAULT FALSE,
    has_downloads BOOLEAN NOT NULL DEFAULT TRUE,
    has_discussions BOOLEAN NOT NULL DEFAULT FALSE,
    archived      BOOLEAN NOT NULL DEFAULT FALSE,
    disabled      BOOLEAN NOT NULL DEFAULT FALSE,
    visibility    VARCHAR NOT NULL DEFAULT 'public',
    pushed_at     TIMESTAMP,
    allow_rebase_merge BOOLEAN NOT NULL DEFAULT TRUE,
    allow_squash_merge BOOLEAN NOT NULL DEFAULT TRUE,
    allow_merge_commit BOOLEAN NOT NULL DEFAULT TRUE,
    allow_auto_merge BOOLEAN NOT NULL DEFAULT FALSE,
    delete_branch_on_merge BOOLEAN NOT NULL DEFAULT FALSE,
    allow_forking BOOLEAN NOT NULL DEFAULT TRUE,
    web_commit_signoff_required BOOLEAN NOT NULL DEFAULT FALSE,
    license_key   VARCHAR NOT NULL DEFAULT '',
    license_name  VARCHAR NOT NULL DEFAULT '',
    license_spdx_id VARCHAR NOT NULL DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS repo_topics (
    repo_id       BIGINT NOT NULL,
    topic         VARCHAR NOT NULL,
    PRIMARY KEY (repo_id, topic)
);

CREATE TABLE IF NOT EXISTS repo_languages (
    repo_id       BIGINT NOT NULL,
    language      VARCHAR NOT NULL,
    bytes         INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (repo_id, language)
);

-- ============================================================
-- Teams
-- ============================================================

CREATE TABLE IF NOT EXISTS teams (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_teams'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    org_id        BIGINT NOT NULL,
    name          VARCHAR NOT NULL,
    slug          VARCHAR NOT NULL,
    description   VARCHAR NOT NULL DEFAULT '',
    privacy       VARCHAR NOT NULL DEFAULT 'closed',
    permission    VARCHAR NOT NULL DEFAULT 'pull',
    parent_id     BIGINT,
    members_count INTEGER NOT NULL DEFAULT 0,
    repos_count   INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id       BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    role          VARCHAR NOT NULL DEFAULT 'member',
    state         VARCHAR NOT NULL DEFAULT 'active',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE IF NOT EXISTS team_repos (
    team_id       BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    permission    VARCHAR NOT NULL DEFAULT 'pull',
    PRIMARY KEY (team_id, repo_id)
);

-- ============================================================
-- Issues
-- ============================================================

CREATE TABLE IF NOT EXISTS issues (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_issues'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    number        INTEGER NOT NULL,
    state         VARCHAR NOT NULL DEFAULT 'open',
    state_reason  VARCHAR NOT NULL DEFAULT '',
    title         VARCHAR NOT NULL,
    body          VARCHAR NOT NULL DEFAULT '',
    creator_id    BIGINT NOT NULL,
    locked        BOOLEAN NOT NULL DEFAULT FALSE,
    active_lock_reason VARCHAR NOT NULL DEFAULT '',
    comments      INTEGER NOT NULL DEFAULT 0,
    closed_at     TIMESTAMP,
    closed_by_id  BIGINT,
    milestone_id  BIGINT,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id      BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    PRIMARY KEY (issue_id, user_id)
);

CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id      BIGINT NOT NULL,
    label_id      BIGINT NOT NULL,
    PRIMARY KEY (issue_id, label_id)
);

CREATE TABLE IF NOT EXISTS issue_events (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_issue_events'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    issue_id      BIGINT NOT NULL,
    actor_id      BIGINT NOT NULL,
    event         VARCHAR NOT NULL,
    commit_id     VARCHAR NOT NULL DEFAULT '',
    commit_url    VARCHAR NOT NULL DEFAULT '',
    label_id      BIGINT,
    assignee_id   BIGINT,
    assigner_id   BIGINT,
    milestone_id  BIGINT,
    rename_from   VARCHAR,
    rename_to     VARCHAR,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Labels
-- ============================================================

CREATE TABLE IF NOT EXISTS labels (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_labels'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    name          VARCHAR NOT NULL,
    description   VARCHAR NOT NULL DEFAULT '',
    color         VARCHAR NOT NULL DEFAULT 'ededed',
    is_default    BOOLEAN NOT NULL DEFAULT FALSE
);

-- ============================================================
-- Milestones
-- ============================================================

CREATE TABLE IF NOT EXISTS milestones (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_milestones'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    number        INTEGER NOT NULL,
    state         VARCHAR NOT NULL DEFAULT 'open',
    title         VARCHAR NOT NULL,
    description   VARCHAR NOT NULL DEFAULT '',
    creator_id    BIGINT NOT NULL,
    open_issues   INTEGER NOT NULL DEFAULT 0,
    closed_issues INTEGER NOT NULL DEFAULT 0,
    closed_at     TIMESTAMP,
    due_on        TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS issue_comments (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_issue_comments'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    issue_id      BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    creator_id    BIGINT NOT NULL,
    body          VARCHAR NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS commit_comments (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_commit_comments'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    creator_id    BIGINT NOT NULL,
    commit_sha    VARCHAR NOT NULL,
    body          VARCHAR NOT NULL,
    path          VARCHAR NOT NULL DEFAULT '',
    position      INTEGER,
    line          INTEGER,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Pull Requests
-- ============================================================

CREATE TABLE IF NOT EXISTS pull_requests (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_pull_requests'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    number        INTEGER NOT NULL,
    state         VARCHAR NOT NULL DEFAULT 'open',
    locked        BOOLEAN NOT NULL DEFAULT FALSE,
    title         VARCHAR NOT NULL,
    body          VARCHAR NOT NULL DEFAULT '',
    creator_id    BIGINT NOT NULL,
    head_ref      VARCHAR NOT NULL,
    head_sha      VARCHAR NOT NULL DEFAULT '',
    head_repo_id  BIGINT,
    head_label    VARCHAR NOT NULL DEFAULT '',
    base_ref      VARCHAR NOT NULL,
    base_sha      VARCHAR NOT NULL DEFAULT '',
    base_label    VARCHAR NOT NULL DEFAULT '',
    draft         BOOLEAN NOT NULL DEFAULT FALSE,
    merged        BOOLEAN NOT NULL DEFAULT FALSE,
    mergeable     BOOLEAN,
    rebaseable    BOOLEAN,
    mergeable_state VARCHAR NOT NULL DEFAULT '',
    merge_commit_sha VARCHAR NOT NULL DEFAULT '',
    merged_at     TIMESTAMP,
    merged_by_id  BIGINT,
    comments      INTEGER NOT NULL DEFAULT 0,
    review_comments INTEGER NOT NULL DEFAULT 0,
    maintainer_can_modify BOOLEAN NOT NULL DEFAULT FALSE,
    commits       INTEGER NOT NULL DEFAULT 0,
    additions     INTEGER NOT NULL DEFAULT 0,
    deletions     INTEGER NOT NULL DEFAULT 0,
    changed_files INTEGER NOT NULL DEFAULT 0,
    closed_at     TIMESTAMP,
    milestone_id  BIGINT,
    active_lock_reason VARCHAR NOT NULL DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_assignees (
    pr_id         BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    PRIMARY KEY (pr_id, user_id)
);

CREATE TABLE IF NOT EXISTS pr_labels (
    pr_id         BIGINT NOT NULL,
    label_id      BIGINT NOT NULL,
    PRIMARY KEY (pr_id, label_id)
);

CREATE TABLE IF NOT EXISTS pr_reviews (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_pr_reviews'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    pr_id         BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    body          VARCHAR NOT NULL DEFAULT '',
    state         VARCHAR NOT NULL DEFAULT 'PENDING',
    commit_id     VARCHAR NOT NULL,
    submitted_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_review_comments (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_pr_review_comments'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    pr_id         BIGINT NOT NULL,
    review_id     BIGINT,
    user_id       BIGINT NOT NULL,
    diff_hunk     VARCHAR NOT NULL DEFAULT '',
    path          VARCHAR NOT NULL,
    position      INTEGER,
    original_position INTEGER,
    commit_id     VARCHAR NOT NULL,
    original_commit_id VARCHAR NOT NULL DEFAULT '',
    in_reply_to_id BIGINT,
    body          VARCHAR NOT NULL,
    line          INTEGER,
    original_line INTEGER,
    start_line    INTEGER,
    original_start_line INTEGER,
    side          VARCHAR NOT NULL DEFAULT 'RIGHT',
    start_side    VARCHAR NOT NULL DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pr_requested_reviewers (
    pr_id         BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    PRIMARY KEY (pr_id, user_id)
);

CREATE TABLE IF NOT EXISTS pr_requested_teams (
    pr_id         BIGINT NOT NULL,
    team_id       BIGINT NOT NULL,
    PRIMARY KEY (pr_id, team_id)
);

-- ============================================================
-- Stars
-- ============================================================

CREATE TABLE IF NOT EXISTS stars (
    user_id       BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, repo_id)
);

-- ============================================================
-- Watches
-- ============================================================

CREATE TABLE IF NOT EXISTS watches (
    user_id       BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    subscribed    BOOLEAN NOT NULL DEFAULT TRUE,
    ignored       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, repo_id)
);

-- ============================================================
-- Collaborators
-- ============================================================

CREATE TABLE IF NOT EXISTS collaborators (
    repo_id       BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    permission    VARCHAR NOT NULL DEFAULT 'pull',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (repo_id, user_id)
);

CREATE TABLE IF NOT EXISTS collaborator_invitations (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_collaborator_invitations'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    invitee_id    BIGINT NOT NULL,
    inviter_id    BIGINT NOT NULL,
    permissions   VARCHAR NOT NULL DEFAULT 'pull',
    expired       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Reactions
-- ============================================================

CREATE TABLE IF NOT EXISTS reactions (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_reactions'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    subject_type  VARCHAR NOT NULL,
    subject_id    BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    content       VARCHAR NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Releases
-- ============================================================

CREATE TABLE IF NOT EXISTS releases (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_releases'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    repo_id       BIGINT NOT NULL,
    tag_name      VARCHAR NOT NULL,
    target_commitish VARCHAR NOT NULL DEFAULT 'main',
    name          VARCHAR NOT NULL DEFAULT '',
    body          VARCHAR NOT NULL DEFAULT '',
    draft         BOOLEAN NOT NULL DEFAULT FALSE,
    prerelease    BOOLEAN NOT NULL DEFAULT FALSE,
    author_id     BIGINT NOT NULL,
    published_at  TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS release_assets (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_release_assets'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    release_id    BIGINT NOT NULL,
    uploader_id   BIGINT NOT NULL,
    name          VARCHAR NOT NULL,
    label         VARCHAR NOT NULL DEFAULT '',
    state         VARCHAR NOT NULL DEFAULT 'uploaded',
    content_type  VARCHAR NOT NULL,
    size          INTEGER NOT NULL DEFAULT 0,
    download_count INTEGER NOT NULL DEFAULT 0,
    storage_path  VARCHAR NOT NULL DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Activities/Events
-- ============================================================

CREATE TABLE IF NOT EXISTS events (
    id            VARCHAR PRIMARY KEY,
    type          VARCHAR NOT NULL,
    actor_id      BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    org_id        BIGINT,
    payload       VARCHAR NOT NULL DEFAULT '{}',
    public        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS feed_subscriptions (
    user_id       BIGINT NOT NULL,
    target_id     BIGINT NOT NULL,
    target_type   VARCHAR NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, target_id, target_type)
);

-- ============================================================
-- Notifications
-- ============================================================

CREATE TABLE IF NOT EXISTS notifications (
    id            VARCHAR PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    unread        BOOLEAN NOT NULL DEFAULT TRUE,
    reason        VARCHAR NOT NULL,
    subject_type  VARCHAR NOT NULL,
    subject_title VARCHAR NOT NULL,
    subject_url   VARCHAR NOT NULL,
    subject_latest_comment_url VARCHAR NOT NULL DEFAULT '',
    last_read_at  TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS thread_subscriptions (
    thread_id     VARCHAR NOT NULL,
    user_id       BIGINT NOT NULL,
    ignored       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (thread_id, user_id)
);

-- ============================================================
-- Webhooks
-- ============================================================

CREATE TABLE IF NOT EXISTS webhooks (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_webhooks'),
    node_id       VARCHAR NOT NULL DEFAULT '',
    owner_id      BIGINT NOT NULL,
    owner_type    VARCHAR NOT NULL,
    name          VARCHAR NOT NULL DEFAULT 'web',
    url           VARCHAR NOT NULL,
    content_type  VARCHAR NOT NULL DEFAULT 'json',
    secret        VARCHAR NOT NULL DEFAULT '',
    insecure_ssl  VARCHAR NOT NULL DEFAULT '0',
    events        VARCHAR NOT NULL DEFAULT '["push"]',
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id            BIGINT PRIMARY KEY DEFAULT nextval('seq_webhook_deliveries'),
    hook_id       BIGINT NOT NULL,
    guid          VARCHAR NOT NULL,
    delivered_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    redelivery    BOOLEAN NOT NULL DEFAULT FALSE,
    duration      DOUBLE NOT NULL DEFAULT 0,
    status        VARCHAR NOT NULL DEFAULT 'pending',
    status_code   INTEGER NOT NULL DEFAULT 0,
    event         VARCHAR NOT NULL,
    action        VARCHAR NOT NULL DEFAULT '',
    request_headers VARCHAR NOT NULL DEFAULT '{}',
    request_payload VARCHAR NOT NULL DEFAULT '{}',
    response_headers VARCHAR NOT NULL DEFAULT '{}',
    response_payload VARCHAR NOT NULL DEFAULT ''
);

-- ============================================================
-- Branch Protection
-- ============================================================

CREATE TABLE IF NOT EXISTS branch_protections (
    repo_id       BIGINT NOT NULL,
    branch        VARCHAR NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    settings_json VARCHAR NOT NULL DEFAULT '{}',
    PRIMARY KEY (repo_id, branch)
);
