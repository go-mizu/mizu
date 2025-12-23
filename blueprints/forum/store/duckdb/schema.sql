-- Forum Schema

-- Accounts
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    display_name VARCHAR,
    bio TEXT,
    avatar_url VARCHAR,
    banner_url VARCHAR,
    karma BIGINT DEFAULT 0,
    post_karma BIGINT DEFAULT 0,
    comment_karma BIGINT DEFAULT 0,
    is_admin BOOLEAN DEFAULT FALSE,
    is_suspended BOOLEAN DEFAULT FALSE,
    suspend_reason VARCHAR,
    suspend_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(LOWER(username));
CREATE INDEX IF NOT EXISTS idx_accounts_email ON accounts(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_accounts_karma ON accounts(karma DESC);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    token VARCHAR UNIQUE NOT NULL,
    user_agent VARCHAR,
    ip VARCHAR,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_account ON sessions(account_id);

-- Boards
CREATE TABLE IF NOT EXISTS boards (
    id VARCHAR PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    title VARCHAR NOT NULL,
    description TEXT,
    sidebar TEXT,
    sidebar_html TEXT,
    icon_url VARCHAR,
    banner_url VARCHAR,
    primary_color VARCHAR,
    is_nsfw BOOLEAN DEFAULT FALSE,
    is_private BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    member_count BIGINT DEFAULT 0,
    thread_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_boards_name ON boards(LOWER(name));
CREATE INDEX IF NOT EXISTS idx_boards_members ON boards(member_count DESC);
CREATE INDEX IF NOT EXISTS idx_boards_created ON boards(created_at DESC);

-- Board members
CREATE TABLE IF NOT EXISTS board_members (
    board_id VARCHAR NOT NULL,
    account_id VARCHAR NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (board_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_board_members_account ON board_members(account_id);

-- Board moderators
CREATE TABLE IF NOT EXISTS board_moderators (
    board_id VARCHAR NOT NULL,
    account_id VARCHAR NOT NULL,
    permissions VARCHAR,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    added_by VARCHAR,
    PRIMARY KEY (board_id, account_id)
);

-- Threads
CREATE TABLE IF NOT EXISTS threads (
    id VARCHAR PRIMARY KEY,
    board_id VARCHAR NOT NULL,
    author_id VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    content TEXT,
    content_html TEXT,
    url VARCHAR,
    domain VARCHAR,
    thumbnail_url VARCHAR,
    type VARCHAR DEFAULT 'text',
    score BIGINT DEFAULT 0,
    upvote_count BIGINT DEFAULT 0,
    downvote_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    view_count BIGINT DEFAULT 0,
    hot_score DOUBLE DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_locked BOOLEAN DEFAULT FALSE,
    is_removed BOOLEAN DEFAULT FALSE,
    is_nsfw BOOLEAN DEFAULT FALSE,
    is_spoiler BOOLEAN DEFAULT FALSE,
    is_oc BOOLEAN DEFAULT FALSE,
    remove_reason VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_threads_board_hot ON threads(board_id, is_pinned DESC, hot_score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_board_new ON threads(board_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_board_top ON threads(board_id, score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_author ON threads(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_hot ON threads(hot_score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_new ON threads(created_at DESC);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    thread_id VARCHAR NOT NULL,
    parent_id VARCHAR,
    author_id VARCHAR NOT NULL,
    content TEXT NOT NULL,
    content_html TEXT,
    score BIGINT DEFAULT 0,
    upvote_count BIGINT DEFAULT 0,
    downvote_count BIGINT DEFAULT 0,
    depth INT DEFAULT 0,
    path VARCHAR NOT NULL,
    child_count BIGINT DEFAULT 0,
    is_removed BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    remove_reason VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_thread ON comments(thread_id, path);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_author ON comments(author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_path ON comments(path);

-- Votes
CREATE TABLE IF NOT EXISTS votes (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique ON votes(account_id, target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_votes_target ON votes(target_type, target_id);

-- Bookmarks
CREATE TABLE IF NOT EXISTS bookmarks (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bookmarks_unique ON bookmarks(account_id, target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_account ON bookmarks(account_id, created_at DESC);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    actor_id VARCHAR,
    board_id VARCHAR,
    thread_id VARCHAR,
    comment_id VARCHAR,
    message TEXT,
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_account ON notifications(account_id, read, created_at DESC);

-- Mod actions
CREATE TABLE IF NOT EXISTS mod_actions (
    id VARCHAR PRIMARY KEY,
    board_id VARCHAR NOT NULL,
    moderator_id VARCHAR NOT NULL,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    action VARCHAR NOT NULL,
    reason TEXT,
    details VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mod_actions_board ON mod_actions(board_id, created_at DESC);

-- Bans
CREATE TABLE IF NOT EXISTS bans (
    id VARCHAR PRIMARY KEY,
    board_id VARCHAR,
    account_id VARCHAR NOT NULL,
    reason TEXT,
    message TEXT,
    mod_id VARCHAR,
    is_permanent BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bans_board_account ON bans(board_id, account_id);
CREATE INDEX IF NOT EXISTS idx_bans_account ON bans(account_id);

-- Reports
CREATE TABLE IF NOT EXISTS reports (
    id VARCHAR PRIMARY KEY,
    reporter_id VARCHAR NOT NULL,
    board_id VARCHAR,
    target_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    reason VARCHAR NOT NULL,
    details TEXT,
    status VARCHAR DEFAULT 'pending',
    resolved_by VARCHAR,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reports_board_status ON reports(board_id, status, created_at DESC);
