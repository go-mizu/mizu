-- Messaging Application Database Schema
-- Inspired by WhatsApp and Telegram

-- ============================================
-- CORE ENTITIES
-- ============================================

-- Users: Account and profile information
CREATE TABLE IF NOT EXISTS users (
  id              VARCHAR PRIMARY KEY,
  phone           VARCHAR UNIQUE,
  email           VARCHAR UNIQUE,
  username        VARCHAR UNIQUE,
  display_name    VARCHAR NOT NULL,
  bio             TEXT,
  avatar_url      VARCHAR,
  password_hash   VARCHAR NOT NULL,
  status          VARCHAR DEFAULT 'Hey there! I am using Messaging',
  last_seen_at    TIMESTAMP,
  is_online       BOOLEAN DEFAULT FALSE,
  e2e_public_key  TEXT,
  two_fa_enabled  BOOLEAN DEFAULT FALSE,
  two_fa_secret   VARCHAR,
  privacy_last_seen     VARCHAR DEFAULT 'everyone',
  privacy_profile_photo VARCHAR DEFAULT 'everyone',
  privacy_about         VARCHAR DEFAULT 'everyone',
  privacy_groups        VARCHAR DEFAULT 'everyone',
  privacy_read_receipts BOOLEAN DEFAULT TRUE,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Sessions: Authentication sessions
CREATE TABLE IF NOT EXISTS sessions (
  id          VARCHAR PRIMARY KEY,
  user_id     VARCHAR NOT NULL,
  token       VARCHAR UNIQUE NOT NULL,
  device_name VARCHAR,
  device_type VARCHAR,
  push_token  VARCHAR,
  ip_address  VARCHAR,
  user_agent  VARCHAR,
  expires_at  TIMESTAMP NOT NULL,
  last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- ============================================
-- CONTACTS
-- ============================================

-- Contacts: User's contact list
CREATE TABLE IF NOT EXISTS contacts (
  user_id         VARCHAR NOT NULL,
  contact_user_id VARCHAR NOT NULL,
  display_name    VARCHAR,
  is_blocked      BOOLEAN DEFAULT FALSE,
  is_favorite     BOOLEAN DEFAULT FALSE,
  blocked_at      TIMESTAMP,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, contact_user_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (contact_user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_blocked ON contacts(user_id, is_blocked);

-- ============================================
-- FRIEND CODES (QR Code Friend Feature)
-- ============================================

-- Friend codes: Shareable codes for adding friends via QR
CREATE TABLE IF NOT EXISTS friend_codes (
  id          VARCHAR PRIMARY KEY,
  user_id     VARCHAR NOT NULL,
  code        VARCHAR UNIQUE NOT NULL,
  expires_at  TIMESTAMP NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_friend_codes_user_id ON friend_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_friend_codes_code ON friend_codes(code);
CREATE INDEX IF NOT EXISTS idx_friend_codes_expires ON friend_codes(expires_at);

-- ============================================
-- CHATS & CONVERSATIONS
-- ============================================

-- Chats: Conversation containers (direct, group, broadcast)
CREATE TABLE IF NOT EXISTS chats (
  id              VARCHAR PRIMARY KEY,
  type            VARCHAR NOT NULL DEFAULT 'direct',
  name            VARCHAR,
  description     TEXT,
  icon_url        VARCHAR,
  owner_id        VARCHAR,
  last_message_id VARCHAR,
  last_message_at TIMESTAMP,
  message_count   BIGINT DEFAULT 0,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chats_type ON chats(type);
CREATE INDEX IF NOT EXISTS idx_chats_owner_id ON chats(owner_id);
CREATE INDEX IF NOT EXISTS idx_chats_last_message_at ON chats(last_message_at DESC);

-- Chat participants
CREATE TABLE IF NOT EXISTS chat_participants (
  chat_id             VARCHAR NOT NULL,
  user_id             VARCHAR NOT NULL,
  role                VARCHAR DEFAULT 'member',
  joined_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  is_muted            BOOLEAN DEFAULT FALSE,
  mute_until          TIMESTAMP,
  unread_count        INTEGER DEFAULT 0,
  last_read_message_id VARCHAR,
  last_read_at        TIMESTAMP,
  notification_level  VARCHAR DEFAULT 'all',
  PRIMARY KEY (chat_id, user_id),
  FOREIGN KEY (chat_id) REFERENCES chats(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_chat_participants_user_id ON chat_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_participants_unread ON chat_participants(user_id, unread_count);

-- ============================================
-- GROUPS
-- ============================================

-- Groups: Group-specific metadata
CREATE TABLE IF NOT EXISTS groups (
  chat_id                 VARCHAR PRIMARY KEY,
  invite_link             VARCHAR UNIQUE,
  invite_link_expires_at  TIMESTAMP,
  invite_link_created_by  VARCHAR,
  member_count            INTEGER DEFAULT 0,
  max_members             INTEGER DEFAULT 1024,
  only_admins_can_send    BOOLEAN DEFAULT FALSE,
  only_admins_can_edit    BOOLEAN DEFAULT FALSE,
  disappearing_messages_ttl INTEGER,
  created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

-- Group invites
CREATE TABLE IF NOT EXISTS group_invites (
  code        VARCHAR PRIMARY KEY,
  chat_id     VARCHAR NOT NULL,
  created_by  VARCHAR NOT NULL,
  max_uses    INTEGER DEFAULT 0,
  uses        INTEGER DEFAULT 0,
  expires_at  TIMESTAMP,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (chat_id) REFERENCES chats(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_group_invites_chat_id ON group_invites(chat_id);

-- ============================================
-- MESSAGES
-- ============================================

-- Messages: All message types
CREATE TABLE IF NOT EXISTS messages (
  id              VARCHAR PRIMARY KEY,
  chat_id         VARCHAR NOT NULL,
  sender_id       VARCHAR NOT NULL,
  type            VARCHAR NOT NULL DEFAULT 'text',
  content         TEXT,
  content_html    TEXT,
  reply_to_id     VARCHAR,
  forward_from_id VARCHAR,
  forward_from_chat_id VARCHAR,
  forward_from_sender_name VARCHAR,
  is_forwarded    BOOLEAN DEFAULT FALSE,
  is_edited       BOOLEAN DEFAULT FALSE,
  edited_at       TIMESTAMP,
  is_deleted      BOOLEAN DEFAULT FALSE,
  deleted_at      TIMESTAMP,
  deleted_for_everyone BOOLEAN DEFAULT FALSE,
  expires_at      TIMESTAMP,
  mention_everyone BOOLEAN DEFAULT FALSE,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (chat_id) REFERENCES chats(id),
  FOREIGN KEY (sender_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_chat_created ON messages(chat_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_reply_to ON messages(reply_to_id);

-- Message media (attachments)
CREATE TABLE IF NOT EXISTS message_media (
  id            VARCHAR PRIMARY KEY,
  message_id    VARCHAR NOT NULL,
  type          VARCHAR NOT NULL,
  filename      VARCHAR,
  content_type  VARCHAR,
  size          BIGINT NOT NULL DEFAULT 0,
  url           VARCHAR NOT NULL,
  thumbnail_url VARCHAR,
  duration      INTEGER,
  width         INTEGER,
  height        INTEGER,
  waveform      TEXT,
  is_voice_note BOOLEAN DEFAULT FALSE,
  is_view_once  BOOLEAN DEFAULT FALSE,
  view_count    INTEGER DEFAULT 0,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (message_id) REFERENCES messages(id)
);

CREATE INDEX IF NOT EXISTS idx_message_media_message_id ON message_media(message_id);

-- Standalone media uploads (files uploaded but not yet attached to messages)
CREATE TABLE IF NOT EXISTS media (
  id              VARCHAR PRIMARY KEY,
  user_id         VARCHAR NOT NULL,
  message_id      VARCHAR,
  type            VARCHAR NOT NULL,
  filename        VARCHAR NOT NULL,
  original_filename VARCHAR NOT NULL,
  content_type    VARCHAR NOT NULL,
  size            BIGINT NOT NULL DEFAULT 0,
  url             VARCHAR NOT NULL,
  thumbnail_url   VARCHAR,
  width           INTEGER,
  height          INTEGER,
  duration        INTEGER,
  waveform        TEXT,
  blurhash        VARCHAR,
  is_view_once    BOOLEAN DEFAULT FALSE,
  view_count      INTEGER DEFAULT 0,
  viewed_at       TIMESTAMP,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP,
  deleted_at      TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (message_id) REFERENCES messages(id)
);

CREATE INDEX IF NOT EXISTS idx_media_user_id ON media(user_id);
CREATE INDEX IF NOT EXISTS idx_media_message_id ON media(message_id);
CREATE INDEX IF NOT EXISTS idx_media_type ON media(type);
CREATE INDEX IF NOT EXISTS idx_media_created_at ON media(created_at);

-- Message recipients (for delivery/read tracking in groups)
CREATE TABLE IF NOT EXISTS message_recipients (
  message_id    VARCHAR NOT NULL,
  user_id       VARCHAR NOT NULL,
  status        VARCHAR DEFAULT 'sent',
  delivered_at  TIMESTAMP,
  read_at       TIMESTAMP,
  PRIMARY KEY (message_id, user_id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_message_recipients_status ON message_recipients(message_id, status);

-- Message mentions
CREATE TABLE IF NOT EXISTS message_mentions (
  message_id  VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  PRIMARY KEY (message_id, user_id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Message reactions
CREATE TABLE IF NOT EXISTS message_reactions (
  message_id  VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  emoji       VARCHAR NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (message_id, user_id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_message_reactions_message_id ON message_reactions(message_id);

-- Starred messages
CREATE TABLE IF NOT EXISTS starred_messages (
  user_id     VARCHAR NOT NULL,
  message_id  VARCHAR NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, message_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (message_id) REFERENCES messages(id)
);

CREATE INDEX IF NOT EXISTS idx_starred_messages_user_id ON starred_messages(user_id);

-- Pinned messages in chats
CREATE TABLE IF NOT EXISTS pinned_messages (
  chat_id     VARCHAR NOT NULL,
  message_id  VARCHAR NOT NULL,
  pinned_by   VARCHAR NOT NULL,
  pinned_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (chat_id, message_id),
  FOREIGN KEY (chat_id) REFERENCES chats(id),
  FOREIGN KEY (message_id) REFERENCES messages(id),
  FOREIGN KEY (pinned_by) REFERENCES users(id)
);

-- ============================================
-- STORIES / STATUS
-- ============================================

-- Stories
CREATE TABLE IF NOT EXISTS stories (
  id              VARCHAR PRIMARY KEY,
  user_id         VARCHAR NOT NULL,
  type            VARCHAR NOT NULL DEFAULT 'image',
  content         TEXT,
  media_url       VARCHAR,
  thumbnail_url   VARCHAR,
  background_color VARCHAR,
  text_style      VARCHAR,
  duration        INTEGER DEFAULT 5,
  view_count      INTEGER DEFAULT 0,
  privacy         VARCHAR DEFAULT 'contacts',
  is_highlight    BOOLEAN DEFAULT FALSE,
  expires_at      TIMESTAMP NOT NULL,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_stories_user_id ON stories(user_id);
CREATE INDEX IF NOT EXISTS idx_stories_expires_at ON stories(expires_at);
CREATE INDEX IF NOT EXISTS idx_stories_user_expires ON stories(user_id, expires_at DESC);

-- Story views
CREATE TABLE IF NOT EXISTS story_views (
  story_id    VARCHAR NOT NULL,
  viewer_id   VARCHAR NOT NULL,
  viewed_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (story_id, viewer_id),
  FOREIGN KEY (story_id) REFERENCES stories(id),
  FOREIGN KEY (viewer_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_story_views_story_id ON story_views(story_id);

-- Story privacy exceptions (include/exclude specific users)
CREATE TABLE IF NOT EXISTS story_privacy (
  story_id    VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  is_allowed  BOOLEAN NOT NULL,
  PRIMARY KEY (story_id, user_id),
  FOREIGN KEY (story_id) REFERENCES stories(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Story mutes (users whose stories are muted)
CREATE TABLE IF NOT EXISTS story_mutes (
  user_id       VARCHAR NOT NULL,
  muted_user_id VARCHAR NOT NULL,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, muted_user_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (muted_user_id) REFERENCES users(id)
);

-- ============================================
-- CALLS
-- ============================================

-- Calls
CREATE TABLE IF NOT EXISTS calls (
  id          VARCHAR PRIMARY KEY,
  chat_id     VARCHAR,
  caller_id   VARCHAR NOT NULL,
  type        VARCHAR NOT NULL DEFAULT 'voice',
  status      VARCHAR DEFAULT 'initiated',
  started_at  TIMESTAMP,
  ended_at    TIMESTAMP,
  duration    INTEGER,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (caller_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_calls_caller_id ON calls(caller_id);
CREATE INDEX IF NOT EXISTS idx_calls_created_at ON calls(created_at DESC);

-- Call participants
CREATE TABLE IF NOT EXISTS call_participants (
  call_id     VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  status      VARCHAR DEFAULT 'pending',
  joined_at   TIMESTAMP,
  left_at     TIMESTAMP,
  is_muted    BOOLEAN DEFAULT FALSE,
  is_video_off BOOLEAN DEFAULT TRUE,
  PRIMARY KEY (call_id, user_id),
  FOREIGN KEY (call_id) REFERENCES calls(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================
-- USER ORGANIZATION
-- ============================================

-- Archived chats
CREATE TABLE IF NOT EXISTS archived_chats (
  user_id     VARCHAR NOT NULL,
  chat_id     VARCHAR NOT NULL,
  archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, chat_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

-- Pinned chats
CREATE TABLE IF NOT EXISTS pinned_chats (
  user_id     VARCHAR NOT NULL,
  chat_id     VARCHAR NOT NULL,
  position    INTEGER DEFAULT 0,
  pinned_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, chat_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

-- ============================================
-- BROADCAST LISTS
-- ============================================

-- Broadcast lists
CREATE TABLE IF NOT EXISTS broadcast_lists (
  id              VARCHAR PRIMARY KEY,
  user_id         VARCHAR NOT NULL,
  name            VARCHAR NOT NULL,
  recipient_count INTEGER DEFAULT 0,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_broadcast_lists_user_id ON broadcast_lists(user_id);

-- Broadcast recipients
CREATE TABLE IF NOT EXISTS broadcast_recipients (
  list_id     VARCHAR NOT NULL,
  user_id     VARCHAR NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (list_id, user_id),
  FOREIGN KEY (list_id) REFERENCES broadcast_lists(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================
-- SETTINGS
-- ============================================

-- User settings
CREATE TABLE IF NOT EXISTS user_settings (
  user_id             VARCHAR PRIMARY KEY,
  theme               VARCHAR DEFAULT 'system',
  font_size           VARCHAR DEFAULT 'medium',
  notification_sound  VARCHAR DEFAULT 'default',
  message_preview     BOOLEAN DEFAULT TRUE,
  enter_to_send       BOOLEAN DEFAULT TRUE,
  media_auto_download VARCHAR DEFAULT 'wifi',
  two_column_layout   BOOLEAN DEFAULT FALSE,
  language            VARCHAR DEFAULT 'en',
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================
-- E2E ENCRYPTION
-- ============================================

-- E2E key bundles (for Signal protocol)
CREATE TABLE IF NOT EXISTS key_bundles (
  user_id         VARCHAR PRIMARY KEY,
  identity_key    TEXT NOT NULL,
  signed_prekey   TEXT NOT NULL,
  prekey_signature TEXT NOT NULL,
  one_time_prekeys TEXT,
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================
-- PRESENCE
-- ============================================

-- Presence: Real-time user status
CREATE TABLE IF NOT EXISTS presence (
  user_id       VARCHAR PRIMARY KEY,
  status        VARCHAR DEFAULT 'offline',
  custom_status VARCHAR,
  last_seen_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================
-- NOTIFICATIONS
-- ============================================

-- Push notification tokens
CREATE TABLE IF NOT EXISTS push_tokens (
  id          VARCHAR PRIMARY KEY,
  user_id     VARCHAR NOT NULL,
  token       VARCHAR NOT NULL,
  platform    VARCHAR NOT NULL,
  device_id   VARCHAR,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_push_tokens_user_id ON push_tokens(user_id);

-- ============================================
-- METADATA
-- ============================================

-- Meta: Application metadata
CREATE TABLE IF NOT EXISTS meta (
  k VARCHAR PRIMARY KEY,
  v VARCHAR NOT NULL
);
