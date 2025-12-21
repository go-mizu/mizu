# Microblog

**A modern microblogging platform combining the best of X/Twitter, Threads, and Mastodon.**

Microblog is a self-hosted social platform for short-form content. It takes the best features from the leading platforms: the threading model of X, the conversation focus of Threads, and the federation-ready architecture of Mastodon.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Posts  →  Timelines  →  Interactions  →  Notifications  →  Discovery  │
│  500 chars   home/local    likes/reposts    real-time       trending    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Features

### Core Features (From All Platforms)

| Feature | Origin | Description |
|---------|--------|-------------|
| **Short Posts** | Twitter/Threads | Configurable limit (default 500 chars) |
| **Threading** | Twitter | Reply chains form conversation trees |
| **Reposts** | All | Share posts to your followers |
| **Quote Posts** | Twitter | Repost with your own commentary |
| **Likes** | All | Show appreciation for posts |
| **Bookmarks** | Twitter | Save posts privately |
| **Hashtags** | Twitter/Mastodon | Discoverable topic tags |
| **Mentions** | All | @username notifications |

### Content Features (Mastodon-Inspired)

| Feature | Description |
|---------|-------------|
| **Media Attachments** | Up to 4 images or 1 video per post |
| **Alt Text** | Accessibility descriptions for media |
| **Content Warnings** | Hide sensitive content behind warnings |
| **Polls** | Multi-choice polls with expiration |
| **Visibility Controls** | Public, unlisted, followers-only, direct |
| **Edit History** | Edit posts with full revision history |

### Social Features

| Feature | Description |
|---------|-------------|
| **Following/Followers** | Asymmetric social graph |
| **Lists** | Curated groups of accounts |
| **Mute/Block** | Control your experience |
| **Profile Customization** | Bio, avatar, header, custom fields |
| **Pinned Posts** | Highlight posts on your profile |
| **Verification** | Admin-managed verified badges |

### Timelines

| Timeline | Description |
|----------|-------------|
| **Home** | Posts from accounts you follow (reverse chronological) |
| **Local** | All public posts on the instance |
| **Explore** | Trending posts and hashtags |
| **User** | Single user's posts and replies |
| **Hashtag** | Posts with a specific hashtag |
| **List** | Posts from a curated list |

### Discovery

| Feature | Description |
|---------|-------------|
| **Trending Hashtags** | Popular tags in the last 24h |
| **Trending Posts** | Most interacted posts |
| **User Search** | Find accounts by name/handle |
| **Post Search** | Full-text search in posts |
| **Suggested Accounts** | Recommendations based on graph |

---

## Quick Start

```bash
# 1. Build the CLI
make build

# 2. Initialize the database
microblog init

# 3. Create an admin account
microblog user create admin --admin --password secret

# 4. Start the server
microblog serve

# 5. Open http://localhost:8080
```

---

## Commands Reference

### `microblog init`

Initialize the database and create required tables.

```bash
microblog init                     # Uses default data directory
microblog init --data ~/my-data    # Custom data directory
```

### `microblog serve`

Start the web server.

```bash
microblog serve                    # Default :8080
microblog serve --addr :3000       # Custom port
microblog serve --dev              # Development mode (hot reload)
```

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | HTTP listen address |
| `--data` | `$HOME/data/blueprint/microblog` | Data directory |
| `--dev` | `false` | Enable development mode |

### `microblog user`

Manage user accounts.

```bash
microblog user create alice                    # Create user
microblog user create admin --admin            # Create admin
microblog user list                            # List all users
microblog user verify alice                    # Add verified badge
microblog user suspend bob --reason "spam"     # Suspend account
```

### `microblog import`

Import data from other platforms.

```bash
microblog import twitter archive.zip          # Import Twitter archive
microblog import mastodon export.json         # Import Mastodon export
```

---

## Database Schema

Microblog uses DuckDB for persistence. Here's the complete schema:

### `accounts` - User Accounts

```sql
CREATE TABLE accounts (
  id            VARCHAR PRIMARY KEY,   -- Unique identifier (ULID)
  username      VARCHAR UNIQUE,        -- Handle (lowercase, alphanumeric)
  display_name  VARCHAR,               -- Display name
  email         VARCHAR UNIQUE,        -- Email (for auth)
  password_hash VARCHAR,               -- Argon2id hash
  bio           TEXT,                  -- Profile bio (500 chars max)
  avatar_url    VARCHAR,               -- Profile picture URL
  header_url    VARCHAR,               -- Banner image URL
  fields        JSON,                  -- Custom profile fields [{name, value}]
  verified      BOOLEAN DEFAULT FALSE, -- Verified badge
  admin         BOOLEAN DEFAULT FALSE, -- Admin privileges
  suspended     BOOLEAN DEFAULT FALSE, -- Account suspended
  created_at    TIMESTAMP,             -- Account creation time
  updated_at    TIMESTAMP              -- Last update time
);
```

### `posts` - All Posts

```sql
CREATE TABLE posts (
  id            VARCHAR PRIMARY KEY,    -- Unique identifier (ULID)
  account_id    VARCHAR NOT NULL,       -- Author account
  content       TEXT,                   -- Post content (500 chars)
  content_warning TEXT,                 -- Spoiler/CW text
  visibility    VARCHAR DEFAULT 'public', -- public/unlisted/followers/direct
  reply_to_id   VARCHAR,                -- Parent post (for replies)
  thread_id     VARCHAR,                -- Root of thread
  quote_of_id   VARCHAR,                -- Quoted post (for quote posts)
  poll_id       VARCHAR,                -- Associated poll
  language      VARCHAR,                -- ISO language code
  sensitive     BOOLEAN DEFAULT FALSE,  -- Contains sensitive media
  edited_at     TIMESTAMP,              -- Last edit time
  created_at    TIMESTAMP,              -- Post creation time

  -- Denormalized counters (updated by triggers/jobs)
  likes_count   INTEGER DEFAULT 0,
  reposts_count INTEGER DEFAULT 0,
  replies_count INTEGER DEFAULT 0,

  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (reply_to_id) REFERENCES posts(id),
  FOREIGN KEY (quote_of_id) REFERENCES posts(id)
);
```

### `media` - Media Attachments

```sql
CREATE TABLE media (
  id          VARCHAR PRIMARY KEY,  -- Unique identifier
  post_id     VARCHAR NOT NULL,     -- Parent post
  type        VARCHAR NOT NULL,     -- image/video/audio
  url         VARCHAR NOT NULL,     -- Media URL
  preview_url VARCHAR,              -- Thumbnail URL
  alt_text    TEXT,                 -- Accessibility description
  width       INTEGER,              -- Pixel width
  height      INTEGER,              -- Pixel height
  position    INTEGER DEFAULT 0,    -- Order in post (0-3)
  created_at  TIMESTAMP,

  FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

### `polls` - Polls

```sql
CREATE TABLE polls (
  id          VARCHAR PRIMARY KEY,
  post_id     VARCHAR UNIQUE,       -- One poll per post
  options     JSON NOT NULL,        -- [{title, votes_count}]
  multiple    BOOLEAN DEFAULT FALSE,-- Allow multiple choices
  expires_at  TIMESTAMP,            -- Poll expiration
  voters_count INTEGER DEFAULT 0,
  created_at  TIMESTAMP,

  FOREIGN KEY (post_id) REFERENCES posts(id)
);

CREATE TABLE poll_votes (
  id         VARCHAR PRIMARY KEY,
  poll_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  choice     INTEGER NOT NULL,       -- Option index
  created_at TIMESTAMP,

  UNIQUE(poll_id, account_id, choice),
  FOREIGN KEY (poll_id) REFERENCES polls(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);
```

### `follows` - Social Graph

```sql
CREATE TABLE follows (
  id          VARCHAR PRIMARY KEY,
  follower_id VARCHAR NOT NULL,     -- Who is following
  following_id VARCHAR NOT NULL,    -- Who is being followed
  created_at  TIMESTAMP,

  UNIQUE(follower_id, following_id),
  FOREIGN KEY (follower_id) REFERENCES accounts(id),
  FOREIGN KEY (following_id) REFERENCES accounts(id)
);
```

### `likes` - Post Likes

```sql
CREATE TABLE likes (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  post_id    VARCHAR NOT NULL,
  created_at TIMESTAMP,

  UNIQUE(account_id, post_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

### `reposts` - Reposts/Boosts

```sql
CREATE TABLE reposts (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  post_id    VARCHAR NOT NULL,
  created_at TIMESTAMP,

  UNIQUE(account_id, post_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

### `bookmarks` - Saved Posts

```sql
CREATE TABLE bookmarks (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,
  post_id    VARCHAR NOT NULL,
  created_at TIMESTAMP,

  UNIQUE(account_id, post_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

### `hashtags` - Tag Registry

```sql
CREATE TABLE hashtags (
  id         VARCHAR PRIMARY KEY,
  name       VARCHAR UNIQUE,        -- Lowercase tag name
  posts_count INTEGER DEFAULT 0,    -- Usage count
  last_used_at TIMESTAMP,           -- For trending
  created_at TIMESTAMP
);

CREATE TABLE post_hashtags (
  post_id    VARCHAR NOT NULL,
  hashtag_id VARCHAR NOT NULL,

  PRIMARY KEY(post_id, hashtag_id),
  FOREIGN KEY (post_id) REFERENCES posts(id),
  FOREIGN KEY (hashtag_id) REFERENCES hashtags(id)
);
```

### `mentions` - @mentions

```sql
CREATE TABLE mentions (
  id         VARCHAR PRIMARY KEY,
  post_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,      -- Mentioned account
  created_at TIMESTAMP,

  FOREIGN KEY (post_id) REFERENCES posts(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);
```

### `notifications` - User Notifications

```sql
CREATE TABLE notifications (
  id          VARCHAR PRIMARY KEY,
  account_id  VARCHAR NOT NULL,      -- Recipient
  type        VARCHAR NOT NULL,      -- follow/like/repost/mention/reply/poll
  actor_id    VARCHAR,               -- Who triggered it
  post_id     VARCHAR,               -- Related post
  read        BOOLEAN DEFAULT FALSE,
  created_at  TIMESTAMP,

  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (actor_id) REFERENCES accounts(id),
  FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

### `lists` - Curated Account Lists

```sql
CREATE TABLE lists (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,       -- List owner
  title      VARCHAR NOT NULL,       -- List name
  replies_policy VARCHAR DEFAULT 'list', -- list/followed/none
  created_at TIMESTAMP,

  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE list_members (
  list_id    VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,       -- Member account
  created_at TIMESTAMP,

  PRIMARY KEY(list_id, account_id),
  FOREIGN KEY (list_id) REFERENCES lists(id),
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);
```

### `blocks` and `mutes` - User Controls

```sql
CREATE TABLE blocks (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,       -- Who is blocking
  target_id  VARCHAR NOT NULL,       -- Who is blocked
  created_at TIMESTAMP,

  UNIQUE(account_id, target_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (target_id) REFERENCES accounts(id)
);

CREATE TABLE mutes (
  id         VARCHAR PRIMARY KEY,
  account_id VARCHAR NOT NULL,       -- Who is muting
  target_id  VARCHAR NOT NULL,       -- Who is muted
  hide_notifications BOOLEAN DEFAULT TRUE,
  expires_at TIMESTAMP,              -- Optional expiration
  created_at TIMESTAMP,

  UNIQUE(account_id, target_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id),
  FOREIGN KEY (target_id) REFERENCES accounts(id)
);
```

### `edit_history` - Post Revisions

```sql
CREATE TABLE edit_history (
  id         VARCHAR PRIMARY KEY,
  post_id    VARCHAR NOT NULL,
  content    TEXT,
  content_warning TEXT,
  sensitive  BOOLEAN,
  created_at TIMESTAMP,              -- When this version was created

  FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

---

## Data Directory Structure

```
$HOME/data/blueprint/microblog/
├── microblog.duckdb          # Main database
├── media/                    # Uploaded media files
│   ├── avatars/              # Profile pictures
│   ├── headers/              # Banner images
│   └── posts/                # Post attachments
└── cache/                    # Temporary files
    └── thumbnails/           # Generated previews
```

---

## Architecture

```
microblog/
├── cmd/microblog/           # CLI entry point
├── cli/                     # Command implementations
│   ├── root.go              #   → microblog (help)
│   ├── serve.go             #   → microblog serve
│   ├── init.go              #   → microblog init
│   ├── user.go              #   → microblog user *
│   └── import.go            #   → microblog import *
├── app/web/                 # HTTP handlers & routing
│   ├── server.go            # Server setup
│   ├── routes.go            # Route definitions
│   ├── auth.go              # Authentication middleware
│   ├── handlers.go          # Page handlers
│   └── views/               # HTML templates
│       ├── layouts/
│       ├── pages/
│       └── components/
├── feature/                 # Business logic
│   ├── accounts/            # Account management
│   ├── posts/               # Post CRUD & threading
│   ├── timelines/           # Feed generation
│   ├── interactions/        # Likes, reposts, bookmarks
│   ├── relationships/       # Follows, blocks, mutes
│   ├── notifications/       # Notification delivery
│   ├── search/              # Full-text search
│   ├── trending/            # Trending calculation
│   └── media/               # Media upload/processing
├── store/duckdb/            # Data persistence
│   ├── schema.sql           # Table definitions
│   ├── store.go             # Store interface
│   └── queries/             # SQL queries
└── pkg/                     # Shared utilities
    ├── ulid/                # ID generation
    ├── text/                # Mention/hashtag parsing
    └── media/               # Image processing
```

### Feature Layer Design

Each feature follows a consistent pattern:

```go
// feature/posts/service.go
type Service struct {
    store  *store.Store
    events chan Event  // For notifications
}

func (s *Service) Create(ctx context.Context, in *CreateIn) (*Post, error)
func (s *Service) Get(ctx context.Context, id string) (*Post, error)
func (s *Service) Delete(ctx context.Context, id string) error
func (s *Service) Like(ctx context.Context, postID, accountID string) error
func (s *Service) Unlike(ctx context.Context, postID, accountID string) error
```

### Timeline Generation

Timelines use a pull-based model for simplicity:

```go
// Home timeline: posts from followed accounts
SELECT p.* FROM posts p
JOIN follows f ON f.following_id = p.account_id
WHERE f.follower_id = :account_id
  AND p.visibility IN ('public', 'unlisted', 'followers')
ORDER BY p.created_at DESC
LIMIT :limit OFFSET :offset

// Local timeline: all public posts
SELECT * FROM posts
WHERE visibility = 'public'
ORDER BY created_at DESC
LIMIT :limit OFFSET :offset
```

---

## API Endpoints

### Posts

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/posts` | Create a post |
| `GET` | `/api/v1/posts/:id` | Get a post |
| `PUT` | `/api/v1/posts/:id` | Edit a post |
| `DELETE` | `/api/v1/posts/:id` | Delete a post |
| `GET` | `/api/v1/posts/:id/context` | Get thread context |
| `POST` | `/api/v1/posts/:id/like` | Like a post |
| `DELETE` | `/api/v1/posts/:id/like` | Unlike a post |
| `POST` | `/api/v1/posts/:id/repost` | Repost |
| `DELETE` | `/api/v1/posts/:id/repost` | Unrepost |
| `POST` | `/api/v1/posts/:id/bookmark` | Bookmark |
| `DELETE` | `/api/v1/posts/:id/bookmark` | Unbookmark |

### Timelines

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/timelines/home` | Home timeline |
| `GET` | `/api/v1/timelines/local` | Local timeline |
| `GET` | `/api/v1/timelines/tag/:tag` | Hashtag timeline |
| `GET` | `/api/v1/timelines/list/:id` | List timeline |

### Accounts

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/accounts/:id` | Get account |
| `PATCH` | `/api/v1/accounts/update_credentials` | Update profile |
| `GET` | `/api/v1/accounts/:id/posts` | Account's posts |
| `GET` | `/api/v1/accounts/:id/followers` | Followers |
| `GET` | `/api/v1/accounts/:id/following` | Following |
| `POST` | `/api/v1/accounts/:id/follow` | Follow |
| `POST` | `/api/v1/accounts/:id/unfollow` | Unfollow |
| `POST` | `/api/v1/accounts/:id/block` | Block |
| `POST` | `/api/v1/accounts/:id/mute` | Mute |

### Notifications

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/notifications` | Get notifications |
| `POST` | `/api/v1/notifications/clear` | Mark all read |
| `POST` | `/api/v1/notifications/:id/dismiss` | Dismiss one |

### Search

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/search` | Search posts/accounts/hashtags |
| `GET` | `/api/v1/trends/tags` | Trending hashtags |
| `GET` | `/api/v1/trends/posts` | Trending posts |

---

## Development

```bash
# Run with auto-reload
make dev

# Run tests
make test

# Lint
make lint

# Build binary
make build
```

---

## Performance Considerations

| Operation | Strategy |
|-----------|----------|
| **Timeline queries** | Denormalized counters, proper indexes |
| **Trending calculation** | Background job, cached results |
| **Search** | DuckDB full-text search or external (Meilisearch) |
| **Media** | Async processing, CDN for delivery |
| **Notifications** | Event-driven, batched delivery |

---

## Future Enhancements

- **ActivityPub Federation**: Interact with Mastodon, Pleroma, etc.
- **Real-time Updates**: WebSocket for live timeline updates
- **Direct Messages**: Private conversations
- **Scheduled Posts**: Queue posts for later
- **Analytics**: Post performance metrics

---

## License

MIT
