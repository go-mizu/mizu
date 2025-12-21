-- Microblog Database Schema

-- Accounts: User accounts
CREATE TABLE IF NOT EXISTS accounts (
  id            VARCHAR PRIMARY KEY,
  username      VARCHAR UNIQUE NOT NULL,
  display_name  VARCHAR,
  email         VARCHAR UNIQUE,
  password_hash VARCHAR,
  bio           TEXT,
  avatar_url    VARCHAR,
  header_url    VARCHAR,
  fields        JSON,
  verified      BOOLEAN DEFAULT FALSE,
  admin         BOOLEAN DEFAULT FALSE,
  suspended     BOOLEAN DEFAULT FALSE,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username);
CREATE INDEX IF NOT EXISTS idx_accounts_email ON accounts(email);

-- Posts: All posts/toots/tweets
CREATE TABLE IF NOT EXISTS posts (
  id              VARCHAR PRIMARY KEY,
  account_id      VARCHAR NOT NULL,
  content         TEXT,
  content_warning TEXT,
  visibility      VARCHAR DEFAULT 'public',
  reply_to_id     VARCHAR,
  thread_id       VARCHAR,
  quote_of_id     VARCHAR,
  poll_id         VARCHAR,
  language        VARCHAR,
  sensitive       BOOLEAN DEFAULT FALSE,
  edited_at       TIMESTAMP,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  likes_count     INTEGER DEFAULT 0,
  reposts_count   INTEGER DEFAULT 0,
  replies_count   INTEGER DEFAULT 0,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_posts_account_id ON posts(account_id);
CREATE INDEX IF NOT EXISTS idx_posts_reply_to_id ON posts(reply_to_id);
CREATE INDEX IF NOT EXISTS idx_posts_thread_id ON posts(thread_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility);

-- Media: Attachments for posts
CREATE TABLE IF NOT EXISTS media (
  id          VARCHAR PRIMARY KEY,
  post_id     VARCHAR NOT NULL,
  type        VARCHAR NOT NULL,
  url         VARCHAR NOT NULL,
  preview_url VARCHAR,
  alt_text    TEXT,
  width       INTEGER,
  height      INTEGER,
  position    INTEGER DEFAULT 0,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_media_post_id ON media(post_id);

-- Polls: Polls attached to posts
CREATE TABLE IF NOT EXISTS polls (
  id           VARCHAR PRIMARY KEY,
  post_id      VARCHAR UNIQUE,
  options      JSON NOT NULL,
  multiple     BOOLEAN DEFAULT FALSE,
  expires_at   TIMESTAMP,
  voters_count INTEGER DEFAULT 0,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- Poll votes
CREATE TABLE IF NOT EXISTS poll_votes (
  id         VARCHAR PRIMARY KEY,
  poll_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  choice     INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(poll_id, account_id, choice),
  FOREIGN KEY (poll_id) REFERENCES polls(id) ON DELETE CASCADE,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

-- Follows: Social graph
CREATE TABLE IF NOT EXISTS follows (
  id           VARCHAR PRIMARY KEY,
  follower_id  VARCHAR NOT NULL,
  following_id VARCHAR NOT NULL,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(follower_id, following_id),
  FOREIGN KEY (follower_id) REFERENCES accounts(id),
  FOREIGN KEY (following_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_follows_follower_id ON follows(follower_id);
CREATE INDEX IF NOT EXISTS idx_follows_following_id ON follows(following_id);

-- Likes: Post likes
CREATE TABLE IF NOT EXISTS likes (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  post_id    VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, post_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_likes_account_id ON likes(account_id);
CREATE INDEX IF NOT EXISTS idx_likes_post_id ON likes(post_id);

-- Reposts: Boosts/retweets
CREATE TABLE IF NOT EXISTS reposts (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  post_id    VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, post_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reposts_account_id ON reposts(account_id);
CREATE INDEX IF NOT EXISTS idx_reposts_post_id ON reposts(post_id);

-- Bookmarks: Private saves
CREATE TABLE IF NOT EXISTS bookmarks (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  post_id    VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, post_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_account_id ON bookmarks(account_id);

-- Hashtags: Tag registry
CREATE TABLE IF NOT EXISTS hashtags (
  id           VARCHAR PRIMARY KEY,
  name         VARCHAR UNIQUE NOT NULL,
  posts_count  INTEGER DEFAULT 0,
  last_used_at TIMESTAMP,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hashtags_name ON hashtags(name);

-- Post hashtags: Many-to-many
CREATE TABLE IF NOT EXISTS post_hashtags (
  post_id    VARCHAR NOT NULL,
  hashtag_id VARCHAR NOT NULL,
  PRIMARY KEY(post_id, hashtag_id),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (hashtag_id) REFERENCES hashtags(id)
);

-- Mentions: @mentions in posts
CREATE TABLE IF NOT EXISTS mentions (
  id         VARCHAR PRIMARY KEY,
  post_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_mentions_post_id ON mentions(post_id);
CREATE INDEX IF NOT EXISTS idx_mentions_account_id ON mentions(account_id);

-- Notifications: User notifications
CREATE TABLE IF NOT EXISTS notifications (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  type       VARCHAR NOT NULL,
  actor_id   VARCHAR,
  post_id    VARCHAR,
  read       BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (actor_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notifications_account_id ON notifications(account_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

-- Lists: Curated account lists
CREATE TABLE IF NOT EXISTS lists (
  id             VARCHAR PRIMARY KEY,
  account_id     VARCHAR NOT NULL,
  title          VARCHAR NOT NULL,
  replies_policy VARCHAR DEFAULT 'list',
  created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_lists_account_id ON lists(account_id);

-- List members
CREATE TABLE IF NOT EXISTS list_members (
  list_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY(list_id, account_id),
  FOREIGN KEY (list_id) REFERENCES lists(id) ON DELETE CASCADE,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

-- Blocks: Blocked accounts
CREATE TABLE IF NOT EXISTS blocks (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  target_id  VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, target_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (target_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_account_id ON blocks(account_id);

-- Mutes: Muted accounts
CREATE TABLE IF NOT EXISTS mutes (
  id                 VARCHAR PRIMARY KEY,
  account_id         VARCHAR NOT NULL,
  target_id          VARCHAR NOT NULL,
  hide_notifications BOOLEAN DEFAULT TRUE,
  expires_at         TIMESTAMP,
  created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(account_id, target_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (target_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_mutes_account_id ON mutes(account_id);

-- Edit history: Post revisions
CREATE TABLE IF NOT EXISTS edit_history (
  id              VARCHAR PRIMARY KEY,
  post_id         VARCHAR NOT NULL,
  content         TEXT,
  content_warning TEXT,
  sensitive       BOOLEAN,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_edit_history_post_id ON edit_history(post_id);

-- Sessions: Auth sessions
CREATE TABLE IF NOT EXISTS sessions (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  token      VARCHAR UNIQUE NOT NULL,
  user_agent VARCHAR,
  ip_address VARCHAR,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_account_id ON sessions(account_id);

-- Meta: Store metadata
CREATE TABLE IF NOT EXISTS meta (
  k VARCHAR PRIMARY KEY,
  v VARCHAR NOT NULL
);
