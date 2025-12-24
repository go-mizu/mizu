-- Chat Application Database Schema
-- Inspired by Discord and Slack

-- Users: User accounts and profiles
CREATE TABLE IF NOT EXISTS users (
  id              VARCHAR PRIMARY KEY,
  username        VARCHAR UNIQUE NOT NULL,
  discriminator   VARCHAR(4) NOT NULL DEFAULT '0001',
  display_name    VARCHAR,
  email           VARCHAR UNIQUE,
  password_hash   VARCHAR,
  avatar_url      VARCHAR,
  banner_url      VARCHAR,
  bio             TEXT,
  status          VARCHAR DEFAULT 'offline',
  custom_status   VARCHAR,
  locale          VARCHAR DEFAULT 'en-US',
  is_bot          BOOLEAN DEFAULT FALSE,
  is_verified     BOOLEAN DEFAULT FALSE,
  is_admin        BOOLEAN DEFAULT FALSE,
  mfa_enabled     BOOLEAN DEFAULT FALSE,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_disc ON users(username, discriminator);

-- Sessions: Auth sessions
CREATE TABLE IF NOT EXISTS sessions (
  id          VARCHAR PRIMARY KEY,
  user_id     VARCHAR NOT NULL,
  token       VARCHAR UNIQUE NOT NULL,
  user_agent  VARCHAR,
  ip_address  VARCHAR,
  device_type VARCHAR,
  expires_at  TIMESTAMP NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Servers: Discord-style guilds/Slack-style workspaces
CREATE TABLE IF NOT EXISTS servers (
  id              VARCHAR PRIMARY KEY,
  name            VARCHAR NOT NULL,
  description     TEXT,
  icon_url        VARCHAR,
  banner_url      VARCHAR,
  owner_id        VARCHAR NOT NULL,
  is_public       BOOLEAN DEFAULT FALSE,
  is_verified     BOOLEAN DEFAULT FALSE,
  invite_code     VARCHAR UNIQUE,
  default_channel VARCHAR,
  member_count    INTEGER DEFAULT 0,
  max_members     INTEGER DEFAULT 500000,
  features        JSON,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_servers_owner_id ON servers(owner_id);
CREATE INDEX IF NOT EXISTS idx_servers_invite_code ON servers(invite_code);
CREATE INDEX IF NOT EXISTS idx_servers_is_public ON servers(is_public);

-- Categories: Channel groupings within servers
CREATE TABLE IF NOT EXISTS categories (
  id          VARCHAR PRIMARY KEY,
  server_id   VARCHAR NOT NULL,
  name        VARCHAR NOT NULL,
  position    INTEGER DEFAULT 0,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX IF NOT EXISTS idx_categories_server_id ON categories(server_id);

-- Channels: Text, voice, DM, group DM, thread
CREATE TABLE IF NOT EXISTS channels (
  id              VARCHAR PRIMARY KEY,
  server_id       VARCHAR,
  category_id     VARCHAR,
  type            VARCHAR NOT NULL DEFAULT 'text',
  name            VARCHAR,
  topic           VARCHAR,
  position        INTEGER DEFAULT 0,
  is_private      BOOLEAN DEFAULT FALSE,
  is_nsfw         BOOLEAN DEFAULT FALSE,
  slow_mode_delay INTEGER DEFAULT 0,
  bitrate         INTEGER,
  user_limit      INTEGER,
  last_message_id VARCHAR,
  last_message_at TIMESTAMP,
  message_count   BIGINT DEFAULT 0,
  icon_url        VARCHAR,
  owner_id        VARCHAR,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (server_id) REFERENCES servers(id),
  FOREIGN KEY (category_id) REFERENCES categories(id),
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_channels_server_id ON channels(server_id);
CREATE INDEX IF NOT EXISTS idx_channels_category_id ON channels(category_id);
CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type);

-- Channel recipients: For DMs and group DMs
CREATE TABLE IF NOT EXISTS channel_recipients (
  channel_id  VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (channel_id, user_id),
  FOREIGN KEY (channel_id) REFERENCES channels(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Roles: Permission groups within servers
CREATE TABLE IF NOT EXISTS roles (
  id            VARCHAR PRIMARY KEY,
  server_id     VARCHAR NOT NULL,
  name          VARCHAR NOT NULL,
  color         INTEGER DEFAULT 0,
  position      INTEGER DEFAULT 0,
  permissions   BIGINT DEFAULT 0,
  is_default    BOOLEAN DEFAULT FALSE,
  is_hoisted    BOOLEAN DEFAULT FALSE,
  is_mentionable BOOLEAN DEFAULT FALSE,
  icon_url      VARCHAR,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX IF NOT EXISTS idx_roles_server_id ON roles(server_id);

-- Members: Server membership
CREATE TABLE IF NOT EXISTS members (
  server_id   VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  nickname    VARCHAR,
  avatar_url  VARCHAR,
  is_muted    BOOLEAN DEFAULT FALSE,
  is_deafened BOOLEAN DEFAULT FALSE,
  joined_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (server_id, user_id),
  FOREIGN KEY (server_id) REFERENCES servers(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_members_user_id ON members(user_id);

-- Member roles: Role assignments
CREATE TABLE IF NOT EXISTS member_roles (
  server_id  VARCHAR NOT NULL,
  user_id    VARCHAR NOT NULL,
  role_id    VARCHAR NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (server_id, user_id, role_id),
  FOREIGN KEY (server_id) REFERENCES servers(id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- Channel permission overrides
CREATE TABLE IF NOT EXISTS channel_permissions (
  channel_id  VARCHAR NOT NULL,
  target_id   VARCHAR NOT NULL,
  target_type VARCHAR NOT NULL,
  allow       BIGINT DEFAULT 0,
  deny        BIGINT DEFAULT 0,
  PRIMARY KEY (channel_id, target_id),
  FOREIGN KEY (channel_id) REFERENCES channels(id)
);

-- Messages: All messages
CREATE TABLE IF NOT EXISTS messages (
  id                VARCHAR PRIMARY KEY,
  channel_id        VARCHAR NOT NULL,
  author_id         VARCHAR NOT NULL,
  content           TEXT,
  content_html      TEXT,
  type              VARCHAR DEFAULT 'default',
  reply_to_id       VARCHAR,
  thread_id         VARCHAR,
  flags             INTEGER DEFAULT 0,
  mention_everyone  BOOLEAN DEFAULT FALSE,
  is_pinned         BOOLEAN DEFAULT FALSE,
  is_edited         BOOLEAN DEFAULT FALSE,
  edited_at         TIMESTAMP,
  created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (channel_id) REFERENCES channels(id),
  FOREIGN KEY (author_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_messages_author_id ON messages(author_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_channel_created ON messages(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_reply_to ON messages(reply_to_id);
CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id);

-- Message mentions: User mentions in messages
CREATE TABLE IF NOT EXISTS message_mentions (
  message_id  VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  PRIMARY KEY (message_id, user_id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Message role mentions
CREATE TABLE IF NOT EXISTS message_role_mentions (
  message_id  VARCHAR NOT NULL,
  role_id     VARCHAR NOT NULL,
  PRIMARY KEY (message_id, role_id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- Attachments: File attachments
CREATE TABLE IF NOT EXISTS attachments (
  id            VARCHAR PRIMARY KEY,
  message_id    VARCHAR NOT NULL,
  filename      VARCHAR NOT NULL,
  content_type  VARCHAR,
  size          BIGINT NOT NULL,
  url           VARCHAR NOT NULL,
  proxy_url     VARCHAR,
  width         INTEGER,
  height        INTEGER,
  is_spoiler    BOOLEAN DEFAULT FALSE,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (message_id) REFERENCES messages(id)
);

CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

-- Embeds: Rich embeds in messages
CREATE TABLE IF NOT EXISTS embeds (
  id          VARCHAR PRIMARY KEY,
  message_id  VARCHAR NOT NULL,
  type        VARCHAR DEFAULT 'rich',
  title       VARCHAR,
  description TEXT,
  url         VARCHAR,
  color       INTEGER,
  image_url   VARCHAR,
  video_url   VARCHAR,
  thumbnail   VARCHAR,
  footer      VARCHAR,
  author_name VARCHAR,
  author_url  VARCHAR,
  fields      JSON,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (message_id) REFERENCES messages(id)
);

CREATE INDEX IF NOT EXISTS idx_embeds_message_id ON embeds(message_id);

-- Reactions: Message reactions
CREATE TABLE IF NOT EXISTS reactions (
  message_id  VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  emoji       VARCHAR NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (message_id, user_id, emoji),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_reactions_message_id ON reactions(message_id);

-- Pins: Pinned messages
CREATE TABLE IF NOT EXISTS pins (
  channel_id  VARCHAR NOT NULL,
  message_id  VARCHAR NOT NULL,
  pinned_by   VARCHAR NOT NULL,
  pinned_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (channel_id, message_id),
  FOREIGN KEY (channel_id) REFERENCES channels(id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (pinned_by) REFERENCES users(id)
);

-- Read states: Track read position per user per channel
CREATE TABLE IF NOT EXISTS read_states (
  user_id         VARCHAR NOT NULL,
  channel_id      VARCHAR NOT NULL,
  last_read_id    VARCHAR,
  mention_count   INTEGER DEFAULT 0,
  last_pin_at     TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, channel_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (channel_id) REFERENCES channels(id)
);

-- Invites: Server invites
CREATE TABLE IF NOT EXISTS invites (
  code        VARCHAR PRIMARY KEY,
  server_id   VARCHAR NOT NULL,
  channel_id  VARCHAR NOT NULL,
  inviter_id  VARCHAR NOT NULL,
  max_uses    INTEGER DEFAULT 0,
  uses        INTEGER DEFAULT 0,
  max_age     INTEGER DEFAULT 86400,
  is_temporary BOOLEAN DEFAULT FALSE,
  expires_at  TIMESTAMP,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (server_id) REFERENCES servers(id),
  FOREIGN KEY (channel_id) REFERENCES channels(id),
  FOREIGN KEY (inviter_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_invites_server_id ON invites(server_id);

-- Bans: Server bans
CREATE TABLE IF NOT EXISTS bans (
  server_id   VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  reason      TEXT,
  banned_by   VARCHAR NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (server_id, user_id),
  FOREIGN KEY (server_id) REFERENCES servers(id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (banned_by) REFERENCES users(id)
);

-- Threads: Thread metadata
CREATE TABLE IF NOT EXISTS threads (
  channel_id        VARCHAR PRIMARY KEY,
  parent_channel_id VARCHAR NOT NULL,
  parent_message_id VARCHAR NOT NULL,
  owner_id          VARCHAR NOT NULL,
  message_count     INTEGER DEFAULT 0,
  member_count      INTEGER DEFAULT 0,
  archived          BOOLEAN DEFAULT FALSE,
  auto_archive_mins INTEGER DEFAULT 1440,
  locked            BOOLEAN DEFAULT FALSE,
  archive_at        TIMESTAMP,
  created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (channel_id) REFERENCES channels(id),
  FOREIGN KEY (parent_channel_id) REFERENCES channels(id),
  FOREIGN KEY (parent_message_id) REFERENCES messages(id),
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- Thread members
CREATE TABLE IF NOT EXISTS thread_members (
  thread_id   VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  joined_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (thread_id, user_id),
  FOREIGN KEY (thread_id) REFERENCES channels(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Presence: User presence state
CREATE TABLE IF NOT EXISTS presence (
  user_id       VARCHAR PRIMARY KEY,
  status        VARCHAR DEFAULT 'offline',
  custom_status VARCHAR,
  activities    JSON,
  client_status JSON,
  last_seen_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Notifications: User notifications
CREATE TABLE IF NOT EXISTS notifications (
  id          VARCHAR PRIMARY KEY,
  user_id     VARCHAR NOT NULL,
  type        VARCHAR NOT NULL,
  server_id   VARCHAR,
  channel_id  VARCHAR,
  message_id  VARCHAR,
  actor_id    VARCHAR,
  data        JSON,
  is_read     BOOLEAN DEFAULT FALSE,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

-- User settings: Per-user settings
CREATE TABLE IF NOT EXISTS user_settings (
  user_id             VARCHAR PRIMARY KEY,
  theme               VARCHAR DEFAULT 'dark',
  message_display     VARCHAR DEFAULT 'cozy',
  locale              VARCHAR DEFAULT 'en-US',
  timezone            VARCHAR,
  developer_mode      BOOLEAN DEFAULT FALSE,
  compact_mode        BOOLEAN DEFAULT FALSE,
  animate_emoji       BOOLEAN DEFAULT TRUE,
  enable_tts          BOOLEAN DEFAULT TRUE,
  render_embeds       BOOLEAN DEFAULT TRUE,
  render_reactions    BOOLEAN DEFAULT TRUE,
  notification_settings JSON,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Server settings: Per-server per-user settings
CREATE TABLE IF NOT EXISTS server_settings (
  user_id               VARCHAR NOT NULL,
  server_id             VARCHAR NOT NULL,
  muted                 BOOLEAN DEFAULT FALSE,
  suppress_everyone     BOOLEAN DEFAULT FALSE,
  suppress_roles        BOOLEAN DEFAULT FALSE,
  notification_level    VARCHAR DEFAULT 'all',
  mobile_push           BOOLEAN DEFAULT TRUE,
  PRIMARY KEY (user_id, server_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (server_id) REFERENCES servers(id)
);

-- Channel settings: Per-channel per-user settings
CREATE TABLE IF NOT EXISTS channel_settings (
  user_id             VARCHAR NOT NULL,
  channel_id          VARCHAR NOT NULL,
  muted               BOOLEAN DEFAULT FALSE,
  notification_level  VARCHAR DEFAULT 'inherit',
  collapsed           BOOLEAN DEFAULT FALSE,
  PRIMARY KEY (user_id, channel_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (channel_id) REFERENCES channels(id)
);

-- Relationships: Friends, blocks, etc.
CREATE TABLE IF NOT EXISTS relationships (
  user_id     VARCHAR NOT NULL,
  target_id   VARCHAR NOT NULL,
  type        VARCHAR NOT NULL,
  nickname    VARCHAR,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, target_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (target_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_relationships_user_id ON relationships(user_id);

-- Audit log: Server audit events
CREATE TABLE IF NOT EXISTS audit_log (
  id          VARCHAR PRIMARY KEY,
  server_id   VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  target_id   VARCHAR,
  action      VARCHAR NOT NULL,
  changes     JSON,
  reason      TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (server_id) REFERENCES servers(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_audit_log_server_id ON audit_log(server_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at DESC);

-- Meta: Application metadata
CREATE TABLE IF NOT EXISTS meta (
  k VARCHAR PRIMARY KEY,
  v VARCHAR NOT NULL
);
