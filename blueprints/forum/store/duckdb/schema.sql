-- Forum Database Schema

-- Accounts: User accounts with karma tracking
CREATE TABLE IF NOT EXISTS accounts (
  id            VARCHAR PRIMARY KEY,
  username      VARCHAR UNIQUE NOT NULL,
  display_name  VARCHAR,
  email         VARCHAR UNIQUE,
  password_hash VARCHAR,
  bio           TEXT,
  avatar_url    VARCHAR,
  header_url    VARCHAR,
  signature     TEXT,
  post_karma    INTEGER DEFAULT 0,
  comment_karma INTEGER DEFAULT 0,
  total_karma   INTEGER DEFAULT 0,
  trust_level   INTEGER DEFAULT 0,
  verified      BOOLEAN DEFAULT FALSE,
  admin         BOOLEAN DEFAULT FALSE,
  suspended     BOOLEAN DEFAULT FALSE,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username);
CREATE INDEX IF NOT EXISTS idx_accounts_email ON accounts(email);
CREATE INDEX IF NOT EXISTS idx_accounts_karma ON accounts(total_karma DESC);

-- Sessions: Authentication sessions
CREATE TABLE IF NOT EXISTS sessions (
  token      VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_account ON sessions(account_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Forums: Discussion categories and subcategories
CREATE TABLE IF NOT EXISTS forums (
  id           VARCHAR PRIMARY KEY,
  parent_id    VARCHAR,
  name         VARCHAR NOT NULL,
  slug         VARCHAR UNIQUE NOT NULL,
  description  TEXT,
  icon         VARCHAR,
  banner       VARCHAR,
  type         VARCHAR DEFAULT 'public',
  nsfw         BOOLEAN DEFAULT FALSE,
  archived     BOOLEAN DEFAULT FALSE,
  thread_count INTEGER DEFAULT 0,
  post_count   INTEGER DEFAULT 0,
  member_count INTEGER DEFAULT 0,
  position     INTEGER DEFAULT 0,
  settings     JSON,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (parent_id) REFERENCES forums(id)
);

CREATE INDEX IF NOT EXISTS idx_forums_parent ON forums(parent_id);
CREATE INDEX IF NOT EXISTS idx_forums_slug ON forums(slug);
CREATE INDEX IF NOT EXISTS idx_forums_type ON forums(type);
CREATE INDEX IF NOT EXISTS idx_forums_position ON forums(position);

-- Forum members: Membership tracking
CREATE TABLE IF NOT EXISTS forum_members (
  id         VARCHAR PRIMARY KEY,
  forum_id   VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  role       VARCHAR DEFAULT 'member',
  joined_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(forum_id, account_id),
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_forum_members_forum ON forum_members(forum_id);
CREATE INDEX IF NOT EXISTS idx_forum_members_account ON forum_members(account_id);
CREATE INDEX IF NOT EXISTS idx_forum_members_role ON forum_members(forum_id, role);

-- Forum rules: Per-forum rules
CREATE TABLE IF NOT EXISTS forum_rules (
  id          VARCHAR PRIMARY KEY,
  forum_id    VARCHAR NOT NULL,
  title       VARCHAR NOT NULL,
  description TEXT,
  position    INTEGER DEFAULT 0,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (forum_id) REFERENCES forums(id)
);

CREATE INDEX IF NOT EXISTS idx_forum_rules_forum ON forum_rules(forum_id, position);

-- Forum tags: Tags/flair for categorization
CREATE TABLE IF NOT EXISTS forum_tags (
  id       VARCHAR PRIMARY KEY,
  forum_id VARCHAR NOT NULL,
  name     VARCHAR NOT NULL,
  color    VARCHAR,
  UNIQUE(forum_id, name),
  FOREIGN KEY (forum_id) REFERENCES forums(id)
);

CREATE INDEX IF NOT EXISTS idx_forum_tags_forum ON forum_tags(forum_id);

-- Threads: Discussion topics
CREATE TABLE IF NOT EXISTS threads (
  id                  VARCHAR PRIMARY KEY,
  forum_id            VARCHAR NOT NULL,
  account_id          VARCHAR NOT NULL,
  type                VARCHAR DEFAULT 'discussion',
  title               VARCHAR NOT NULL,
  content             TEXT,
  slug                VARCHAR,
  sticky              BOOLEAN DEFAULT FALSE,
  locked              BOOLEAN DEFAULT FALSE,
  nsfw                BOOLEAN DEFAULT FALSE,
  spoiler             BOOLEAN DEFAULT FALSE,
  state               VARCHAR DEFAULT 'open',
  view_count          INTEGER DEFAULT 0,
  score               INTEGER DEFAULT 0,
  upvotes             INTEGER DEFAULT 0,
  downvotes           INTEGER DEFAULT 0,
  post_count          INTEGER DEFAULT 0,
  best_post_id        VARCHAR,
  hot_score           DOUBLE DEFAULT 0,
  best_score          DOUBLE DEFAULT 0,
  controversial_score DOUBLE DEFAULT 0,
  last_post_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  edited_at           TIMESTAMP,
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_threads_forum ON threads(forum_id);
CREATE INDEX IF NOT EXISTS idx_threads_account ON threads(account_id);
CREATE INDEX IF NOT EXISTS idx_threads_slug ON threads(forum_id, slug);
CREATE INDEX IF NOT EXISTS idx_threads_created ON threads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_hot ON threads(forum_id, hot_score DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_top ON threads(forum_id, score DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_best ON threads(forum_id, best_score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_new ON threads(forum_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_controversial ON threads(forum_id, controversial_score DESC);
CREATE INDEX IF NOT EXISTS idx_threads_sticky ON threads(forum_id, sticky DESC, created_at DESC);

-- Thread tags: Many-to-many relationship
CREATE TABLE IF NOT EXISTS thread_tags (
  thread_id VARCHAR NOT NULL,
  tag_id    VARCHAR NOT NULL,
  UNIQUE(thread_id, tag_id),
  FOREIGN KEY (thread_id) REFERENCES threads(id),
  FOREIGN KEY (tag_id) REFERENCES forum_tags(id)
);

CREATE INDEX IF NOT EXISTS idx_thread_tags_thread ON thread_tags(thread_id);
CREATE INDEX IF NOT EXISTS idx_thread_tags_tag ON thread_tags(tag_id);

-- Posts: Replies in threads (tree structure)
CREATE TABLE IF NOT EXISTS posts (
  id         VARCHAR PRIMARY KEY,
  thread_id  VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  parent_id  VARCHAR,
  content    TEXT NOT NULL,
  depth      INTEGER DEFAULT 0,
  score      INTEGER DEFAULT 0,
  upvotes    INTEGER DEFAULT 0,
  downvotes  INTEGER DEFAULT 0,
  is_best    BOOLEAN DEFAULT FALSE,
  type       VARCHAR DEFAULT 'comment',
  path       VARCHAR,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  edited_at  TIMESTAMP,
  FOREIGN KEY (thread_id) REFERENCES threads(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (parent_id) REFERENCES posts(id)
);

CREATE INDEX IF NOT EXISTS idx_posts_thread ON posts(thread_id);
CREATE INDEX IF NOT EXISTS idx_posts_account ON posts(account_id);
CREATE INDEX IF NOT EXISTS idx_posts_parent ON posts(parent_id);
CREATE INDEX IF NOT EXISTS idx_posts_thread_parent ON posts(thread_id, parent_id, created_at);
CREATE INDEX IF NOT EXISTS idx_posts_thread_depth ON posts(thread_id, depth, score DESC);
CREATE INDEX IF NOT EXISTS idx_posts_thread_score ON posts(thread_id, score DESC, created_at);
CREATE INDEX IF NOT EXISTS idx_posts_path ON posts(thread_id, path);

-- Edit history: Track all edits
CREATE TABLE IF NOT EXISTS edit_history (
  id          VARCHAR PRIMARY KEY,
  target_type VARCHAR NOT NULL,
  target_id   VARCHAR NOT NULL,
  editor_id   VARCHAR NOT NULL,
  content     TEXT NOT NULL,
  reason      VARCHAR,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (editor_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_edit_history_target ON edit_history(target_type, target_id, created_at DESC);

-- Votes: Upvotes/downvotes on threads and posts
CREATE TABLE IF NOT EXISTS votes (
  id          VARCHAR PRIMARY KEY,
  account_id  VARCHAR NOT NULL,
  target_type VARCHAR NOT NULL,
  target_id   VARCHAR NOT NULL,
  value       INTEGER NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, target_type, target_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_votes_target ON votes(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_votes_account ON votes(account_id);
CREATE INDEX IF NOT EXISTS idx_votes_account_target ON votes(account_id, target_type, target_id);

-- Saved: Bookmarked threads and posts
CREATE TABLE IF NOT EXISTS saved (
  id          VARCHAR PRIMARY KEY,
  account_id  VARCHAR NOT NULL,
  target_type VARCHAR NOT NULL,
  target_id   VARCHAR NOT NULL,
  note        TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, target_type, target_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_saved_account ON saved(account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_saved_target ON saved(target_type, target_id);

-- Subscriptions: Thread watching
CREATE TABLE IF NOT EXISTS subscriptions (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  thread_id  VARCHAR NOT NULL,
  type       VARCHAR DEFAULT 'all',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, thread_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (thread_id) REFERENCES threads(id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_account ON subscriptions(account_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_thread ON subscriptions(thread_id);

-- Notifications: User notifications
CREATE TABLE IF NOT EXISTS notifications (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  actor_id   VARCHAR,
  type       VARCHAR NOT NULL,
  thread_id  VARCHAR,
  post_id    VARCHAR,
  badge_id   VARCHAR,
  read       BOOLEAN DEFAULT FALSE,
  dismissed  BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (actor_id) REFERENCES accounts(id),
  FOREIGN KEY (thread_id) REFERENCES threads(id),
  FOREIGN KEY (post_id) REFERENCES posts(id)
);

CREATE INDEX IF NOT EXISTS idx_notifications_account ON notifications(account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(account_id, read, created_at DESC);

-- Badges: Achievement definitions
CREATE TABLE IF NOT EXISTS badges (
  id          VARCHAR PRIMARY KEY,
  name        VARCHAR NOT NULL,
  description TEXT,
  icon        VARCHAR,
  tier        VARCHAR DEFAULT 'bronze',
  criteria    JSON,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Account badges: Earned badges
CREATE TABLE IF NOT EXISTS account_badges (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  badge_id   VARCHAR NOT NULL,
  reason     VARCHAR,
  granted_by VARCHAR,
  granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, badge_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (badge_id) REFERENCES badges(id),
  FOREIGN KEY (granted_by) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_account_badges_account ON account_badges(account_id);
CREATE INDEX IF NOT EXISTS idx_account_badges_badge ON account_badges(badge_id);

-- Awards: Post awards
CREATE TABLE IF NOT EXISTS awards (
  id          VARCHAR PRIMARY KEY,
  name        VARCHAR NOT NULL,
  description TEXT,
  icon        VARCHAR,
  cost        INTEGER DEFAULT 0,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Post awards: Given awards
CREATE TABLE IF NOT EXISTS post_awards (
  id        VARCHAR PRIMARY KEY,
  post_id   VARCHAR NOT NULL,
  award_id  VARCHAR NOT NULL,
  given_by  VARCHAR NOT NULL,
  anonymous BOOLEAN DEFAULT FALSE,
  message   TEXT,
  given_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_id) REFERENCES posts(id),
  FOREIGN KEY (award_id) REFERENCES awards(id),
  FOREIGN KEY (given_by) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_post_awards_post ON post_awards(post_id);
CREATE INDEX IF NOT EXISTS idx_post_awards_giver ON post_awards(given_by);

-- Reports: Content reports
CREATE TABLE IF NOT EXISTS reports (
  id          VARCHAR PRIMARY KEY,
  reporter_id VARCHAR NOT NULL,
  target_type VARCHAR NOT NULL,
  target_id   VARCHAR NOT NULL,
  reason      VARCHAR NOT NULL,
  details     TEXT,
  status      VARCHAR DEFAULT 'pending',
  reviewed_by VARCHAR,
  reviewed_at TIMESTAMP,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (reporter_id) REFERENCES accounts(id),
  FOREIGN KEY (reviewed_by) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_target ON reports(target_type, target_id);

-- Mod actions: Moderation log
CREATE TABLE IF NOT EXISTS mod_actions (
  id           VARCHAR PRIMARY KEY,
  forum_id     VARCHAR,
  moderator_id VARCHAR NOT NULL,
  action       VARCHAR NOT NULL,
  target_type  VARCHAR NOT NULL,
  target_id    VARCHAR NOT NULL,
  reason       VARCHAR,
  details      JSON,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (moderator_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_mod_actions_forum ON mod_actions(forum_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mod_actions_moderator ON mod_actions(moderator_id);
CREATE INDEX IF NOT EXISTS idx_mod_actions_target ON mod_actions(target_type, target_id);

-- Bans: User bans
CREATE TABLE IF NOT EXISTS bans (
  id         VARCHAR PRIMARY KEY,
  forum_id   VARCHAR,
  account_id VARCHAR NOT NULL,
  banned_by  VARCHAR NOT NULL,
  reason     VARCHAR,
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(forum_id, account_id),
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (banned_by) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_bans_account ON bans(account_id);
CREATE INDEX IF NOT EXISTS idx_bans_forum ON bans(forum_id, account_id);
CREATE INDEX IF NOT EXISTS idx_bans_expires ON bans(expires_at);

-- Mutes: Temporary restrictions
CREATE TABLE IF NOT EXISTS mutes (
  id         VARCHAR PRIMARY KEY,
  forum_id   VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  muted_by   VARCHAR NOT NULL,
  reason     VARCHAR,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(forum_id, account_id),
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (muted_by) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_mutes_account ON mutes(account_id);
CREATE INDEX IF NOT EXISTS idx_mutes_forum ON mutes(forum_id, account_id);
CREATE INDEX IF NOT EXISTS idx_mutes_expires ON mutes(expires_at);

-- User flair: Per-forum user flair
CREATE TABLE IF NOT EXISTS user_flair (
  id         VARCHAR PRIMARY KEY,
  forum_id   VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  text       VARCHAR,
  text_color VARCHAR,
  background VARCHAR,
  emoji      VARCHAR,
  UNIQUE(forum_id, account_id),
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_user_flair_forum ON user_flair(forum_id);
CREATE INDEX IF NOT EXISTS idx_user_flair_account ON user_flair(account_id);

-- Polls: Thread polls
CREATE TABLE IF NOT EXISTS polls (
  id           VARCHAR PRIMARY KEY,
  thread_id    VARCHAR UNIQUE NOT NULL,
  options      JSON NOT NULL,
  multiple     BOOLEAN DEFAULT FALSE,
  expires_at   TIMESTAMP,
  voters_count INTEGER DEFAULT 0,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (thread_id) REFERENCES threads(id)
);

-- Poll votes
CREATE TABLE IF NOT EXISTS poll_votes (
  id         VARCHAR PRIMARY KEY,
  poll_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  choice     INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(poll_id, account_id, choice),
  FOREIGN KEY (poll_id) REFERENCES polls(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_poll_votes_poll ON poll_votes(poll_id);
CREATE INDEX IF NOT EXISTS idx_poll_votes_account ON poll_votes(account_id);

-- Drafts: Auto-saved work
CREATE TABLE IF NOT EXISTS drafts (
  id          VARCHAR PRIMARY KEY,
  account_id  VARCHAR NOT NULL,
  draft_type  VARCHAR NOT NULL,
  forum_id    VARCHAR,
  thread_id   VARCHAR,
  parent_id   VARCHAR,
  title       VARCHAR,
  content     TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (forum_id) REFERENCES forums(id),
  FOREIGN KEY (thread_id) REFERENCES threads(id)
);

CREATE INDEX IF NOT EXISTS idx_drafts_account ON drafts(account_id, updated_at DESC);

-- View tracking: Thread views (simplified)
CREATE TABLE IF NOT EXISTS thread_views (
  thread_id  VARCHAR NOT NULL,
  account_id VARCHAR,
  ip_hash    VARCHAR,
  viewed_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(thread_id, account_id, ip_hash),
  FOREIGN KEY (thread_id) REFERENCES threads(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_thread_views_thread ON thread_views(thread_id);
