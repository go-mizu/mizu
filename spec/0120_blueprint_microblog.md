# Spec 0120: Microblog Blueprint

A production-ready microblogging platform combining the best features from X/Twitter, Threads, and Mastodon.

## Overview

The Microblog blueprint implements a full-featured short-form social platform. It demonstrates:

- Asymmetric social graphs (follow model)
- Feed generation and timeline algorithms
- Real-time interactions (likes, reposts, replies)
- Content moderation and visibility controls
- Hashtag and mention parsing
- Media attachments with accessibility
- Notification systems
- Search and discovery

## Feature Analysis

### Features from X/Twitter

| Feature | Included | Notes |
|---------|----------|-------|
| Short posts (280 chars) | Yes | Configurable limit, default 500 |
| Retweets | Yes | Called "reposts" |
| Quote tweets | Yes | Quote posts with commentary |
| Likes | Yes | Heart icon |
| Reply threads | Yes | Threaded conversations |
| Hashtags | Yes | Clickable, searchable |
| @mentions | Yes | Notifications sent |
| Following/followers | Yes | Asymmetric graph |
| Home timeline | Yes | Reverse chronological |
| Trending topics | Yes | Based on hashtag velocity |
| Bookmarks | Yes | Private saves |
| Lists | Yes | Curated groups |
| Verified badges | Yes | Admin-managed |
| Character counter | Yes | Real-time feedback |

### Features from Threads

| Feature | Included | Notes |
|---------|----------|-------|
| 500 char limit | Yes | Default limit |
| Conversation focus | Yes | Thread context UI |
| Clean threading UX | Yes | Visual thread indicators |
| Reposts | Yes | Same as Twitter |
| Quote posts | Yes | Same as Twitter |
| Simple design | Yes | Minimal UI |
| ActivityPub ready | Future | Architecture supports it |

### Features from Mastodon

| Feature | Included | Notes |
|---------|----------|-------|
| Content warnings | Yes | Spoiler text |
| Polls | Yes | With expiration |
| Media attachments | Yes | Up to 4 images |
| Alt text | Yes | Accessibility |
| Visibility controls | Yes | 4 levels |
| Boosts | Yes | Called "reposts" |
| Local timeline | Yes | All instance posts |
| Character limit config | Yes | Per-instance |
| Edit history | Yes | Full revision tracking |
| Mute with expiration | Yes | Temporary mutes |
| Custom profile fields | Yes | Key-value pairs |

## Data Model

### Entity Relationships

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   Account   │──1:N──│    Post     │──1:N──│    Media    │
└─────────────┘       └─────────────┘       └─────────────┘
       │                    │
       │                    │ reply_to_id
       │                    ▼
       │              ┌─────────────┐
       │              │    Post     │ (parent)
       │              └─────────────┘
       │                    │
       │                    │ quote_of_id
       │                    ▼
       │              ┌─────────────┐
       │              │    Post     │ (quoted)
       │              └─────────────┘
       │
       ├──M:N──────── follows ────────┐
       │                              │
       ├──M:N──────── blocks ─────────┤
       │                              │
       ├──M:N──────── mutes ──────────┤
       │                              │
       ├──1:N──────── likes ──────────┼── Post
       │                              │
       ├──1:N──────── reposts ────────┘
       │
       ├──1:N──────── bookmarks ──────── Post
       │
       ├──1:N──────── notifications
       │
       └──1:N──────── lists ──────M:N── Account
```

### Post Visibility Levels

| Level | Description | Appears In |
|-------|-------------|------------|
| `public` | Anyone can see | Home, Local, Profile, Search |
| `unlisted` | Anyone with link | Home, Profile |
| `followers` | Only followers | Home (followers only) |
| `direct` | Only mentioned users | Notifications only |

### Notification Types

| Type | Trigger | Contains |
|------|---------|----------|
| `follow` | New follower | actor_id |
| `like` | Post liked | actor_id, post_id |
| `repost` | Post reposted | actor_id, post_id |
| `mention` | @mentioned in post | actor_id, post_id |
| `reply` | Reply to your post | actor_id, post_id |
| `poll` | Poll you voted in ended | post_id |
| `update` | Post you interacted with edited | post_id |

## Content Processing

### Post Parsing Pipeline

```
Raw Content
    │
    ▼
┌─────────────────┐
│ Length Check    │ ── Reject if > limit
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Mention Extract │ ── @username → account lookup
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Hashtag Extract │ ── #tag → hashtag registry
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ URL Detection   │ ── Auto-link URLs
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Emoji Shortcode │ ── :emoji: → unicode (optional)
└─────────────────┘
    │
    ▼
Processed Post + Entities
```

### Mention Format

```
@username         → Local account lookup
@username@domain  → Remote account (federation, future)
```

### Hashtag Rules

- Case insensitive (`#GoLang` = `#golang`)
- Alphanumeric + underscore only
- Stored lowercase in registry
- Original case preserved in post display

## Timeline Algorithms

### Home Timeline

Pull-based, reverse chronological:

```sql
SELECT p.*, a.username, a.display_name, a.avatar_url
FROM posts p
JOIN accounts a ON a.id = p.account_id
WHERE p.account_id IN (
    SELECT following_id FROM follows WHERE follower_id = :me
)
  AND p.account_id NOT IN (
    SELECT target_id FROM blocks WHERE account_id = :me
)
  AND p.account_id NOT IN (
    SELECT target_id FROM mutes WHERE account_id = :me
      AND (expires_at IS NULL OR expires_at > NOW())
)
  AND p.visibility IN ('public', 'unlisted', 'followers')
ORDER BY p.created_at DESC
LIMIT :limit
```

### Including Reposts

```sql
-- Union own posts with reposts from followed accounts
SELECT p.*, r.account_id as reposter_id, r.created_at as repost_time
FROM posts p
JOIN reposts r ON r.post_id = p.id
WHERE r.account_id IN (
    SELECT following_id FROM follows WHERE follower_id = :me
)
UNION ALL
SELECT p.*, NULL, NULL FROM posts p
WHERE p.account_id IN (
    SELECT following_id FROM follows WHERE follower_id = :me
)
ORDER BY COALESCE(repost_time, created_at) DESC
```

### Local Timeline

All public posts on the instance:

```sql
SELECT p.*, a.username, a.display_name, a.avatar_url
FROM posts p
JOIN accounts a ON a.id = p.account_id
WHERE p.visibility = 'public'
  AND a.suspended = FALSE
ORDER BY p.created_at DESC
LIMIT :limit
```

### Hashtag Timeline

```sql
SELECT p.*, a.username, a.display_name
FROM posts p
JOIN post_hashtags ph ON ph.post_id = p.id
JOIN hashtags h ON h.id = ph.hashtag_id
JOIN accounts a ON a.id = p.account_id
WHERE h.name = LOWER(:tag)
  AND p.visibility IN ('public', 'unlisted')
ORDER BY p.created_at DESC
LIMIT :limit
```

## Trending Algorithm

### Hashtag Trending

Score based on velocity (usage in time window):

```sql
WITH recent_usage AS (
    SELECT h.id, h.name, COUNT(*) as count_24h
    FROM hashtags h
    JOIN post_hashtags ph ON ph.hashtag_id = h.id
    JOIN posts p ON p.id = ph.post_id
    WHERE p.created_at > NOW() - INTERVAL '24 hours'
    GROUP BY h.id, h.name
),
previous_usage AS (
    SELECT h.id, COUNT(*) as count_prev
    FROM hashtags h
    JOIN post_hashtags ph ON ph.hashtag_id = h.id
    JOIN posts p ON p.id = ph.post_id
    WHERE p.created_at BETWEEN NOW() - INTERVAL '48 hours'
                           AND NOW() - INTERVAL '24 hours'
    GROUP BY h.id
)
SELECT
    r.name,
    r.count_24h,
    COALESCE(p.count_prev, 1) as count_prev,
    r.count_24h::float / COALESCE(p.count_prev, 1) as velocity
FROM recent_usage r
LEFT JOIN previous_usage p ON p.id = r.id
WHERE r.count_24h >= 3  -- Minimum threshold
ORDER BY velocity DESC, r.count_24h DESC
LIMIT 10
```

### Post Trending

Weighted engagement score:

```sql
SELECT p.*,
    (p.likes_count * 1.0 +
     p.reposts_count * 2.0 +
     p.replies_count * 1.5) as engagement,
    -- Decay factor: newer posts score higher
    (p.likes_count * 1.0 + p.reposts_count * 2.0 + p.replies_count * 1.5) /
    POWER(EXTRACT(EPOCH FROM NOW() - p.created_at) / 3600 + 2, 1.5) as score
FROM posts p
WHERE p.visibility = 'public'
  AND p.created_at > NOW() - INTERVAL '24 hours'
ORDER BY score DESC
LIMIT 20
```

## Threading Model

### Thread Structure

Posts form a tree via `reply_to_id`:

```
Post A (thread_id = A)
├── Post B (reply_to_id = A, thread_id = A)
│   ├── Post D (reply_to_id = B, thread_id = A)
│   └── Post E (reply_to_id = B, thread_id = A)
└── Post C (reply_to_id = A, thread_id = A)
    └── Post F (reply_to_id = C, thread_id = A)
```

### Thread Context Query

Get ancestors and descendants for a post:

```go
type ThreadContext struct {
    Ancestors   []*Post // Parent chain up to root
    Post        *Post   // The requested post
    Descendants []*Post // All replies (flattened or nested)
}
```

```sql
-- Ancestors (recursive CTE)
WITH RECURSIVE ancestors AS (
    SELECT * FROM posts WHERE id = :post_id
    UNION ALL
    SELECT p.* FROM posts p
    JOIN ancestors a ON p.id = a.reply_to_id
)
SELECT * FROM ancestors WHERE id != :post_id
ORDER BY created_at ASC;

-- Descendants (recursive CTE)
WITH RECURSIVE descendants AS (
    SELECT * FROM posts WHERE reply_to_id = :post_id
    UNION ALL
    SELECT p.* FROM posts p
    JOIN descendants d ON p.reply_to_id = d.id
)
SELECT * FROM descendants
ORDER BY created_at ASC;
```

## Interaction Rules

### Like

- One like per account per post
- Idempotent (re-liking is no-op)
- Increments `likes_count` on post
- Creates notification for post author

### Repost

- One repost per account per post
- Can unrepost
- Increments `reposts_count` on post
- Creates notification for post author
- Appears in reposter's profile timeline
- Appears in followers' home timelines

### Quote Post

- Creates new post with `quote_of_id` reference
- Original post author notified
- Original post's `reposts_count` incremented
- Quoted post displayed inline

### Bookmark

- Private (not visible to anyone else)
- No notification to post author
- No counter increment
- Unlimited bookmarks

## Media Handling

### Upload Flow

```
Client Upload
    │
    ▼
┌─────────────────┐
│ Validate Type   │ ── image/jpeg, image/png, image/gif, video/mp4
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Check Size      │ ── Max 10MB images, 40MB video
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Generate ID     │ ── ULID for filename
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Store Original  │ ── media/posts/{id}/original.{ext}
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Generate Thumb  │ ── Async job
└─────────────────┘
    │
    ▼
┌─────────────────┐
│ Extract Meta    │ ── Dimensions, duration
└─────────────────┘
    │
    ▼
Media Record Created
```

### Supported Formats

| Type | Formats | Max Size | Max Dimensions |
|------|---------|----------|----------------|
| Image | JPEG, PNG, GIF, WebP | 10 MB | 4096x4096 |
| Video | MP4, WebM | 40 MB | 1920x1080 |
| Audio | MP3, OGG, WAV | 10 MB | N/A |

### Thumbnail Generation

- Images: 400x400 max, JPEG quality 85
- Videos: First frame or 1s mark, 400x225
- Stored in `preview_url`

## Poll Implementation

### Poll Creation

```go
type CreatePollIn struct {
    Options   []string      // 2-4 options, max 50 chars each
    Multiple  bool          // Allow multiple selections
    ExpiresIn time.Duration // 5min to 7 days
}
```

### Voting

- One vote per account (or multiple if `multiple=true`)
- Cannot change vote after casting
- Cannot vote after expiration
- Results visible after voting or expiration

### Poll Results

```go
type PollResult struct {
    Options []struct {
        Title      string
        VotesCount int
        Percentage float64
    }
    VotersCount int
    Expired     bool
    ExpiresAt   time.Time
    Voted       bool      // Current user voted
    OwnVotes    []int     // Indices the user voted for
}
```

## Search Implementation

### Search Types

| Type | Query | Index |
|------|-------|-------|
| Posts | Full-text content search | DuckDB FTS |
| Accounts | Username, display name | Prefix + fuzzy |
| Hashtags | Tag name | Prefix match |

### Search Query

```sql
-- Combined search endpoint
SELECT 'post' as type, id, content as text, NULL as username
FROM posts
WHERE content ILIKE '%' || :query || '%'
  AND visibility = 'public'
UNION ALL
SELECT 'account', id, display_name, username
FROM accounts
WHERE username ILIKE :query || '%'
   OR display_name ILIKE '%' || :query || '%'
UNION ALL
SELECT 'hashtag', id, name, NULL
FROM hashtags
WHERE name ILIKE :query || '%'
ORDER BY type, text
LIMIT 25
```

## Authentication

### Session Management

- Token-based (JWT or opaque tokens)
- Stored in `sessions` table
- HTTP-only cookies for web
- Bearer tokens for API

### Password Hashing

- Argon2id (recommended)
- Time cost: 1 iteration
- Memory cost: 64 MB
- Parallelism: 4 threads

## API Design

### Response Format

Success:
```json
{
    "data": { ... }
}
```

Error:
```json
{
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Post content too long",
        "details": {
            "max_length": 500,
            "actual_length": 523
        }
    }
}
```

### Pagination

Cursor-based using `max_id` and `since_id`:

```
GET /api/v1/timelines/home?limit=20
GET /api/v1/timelines/home?max_id=01HXYZ&limit=20  (older)
GET /api/v1/timelines/home?since_id=01HABC&limit=20  (newer)
```

### Rate Limiting

| Endpoint | Limit |
|----------|-------|
| POST /posts | 30/hour |
| POST /likes | 100/hour |
| POST /follows | 50/hour |
| GET /timelines/* | 300/hour |
| GET /search | 60/hour |

## Directory Structure

```
blueprints/microblog/
├── cmd/microblog/
│   └── main.go                 # CLI entry point
├── cli/
│   ├── root.go                 # Cobra root command
│   ├── serve.go                # serve command
│   ├── init.go                 # init command
│   ├── user.go                 # user subcommands
│   └── import.go               # import subcommands
├── app/
│   └── web/
│       ├── server.go           # HTTP server setup
│       ├── config.go           # Configuration
│       ├── routes.go           # Route definitions
│       ├── middleware.go       # Auth, logging, etc.
│       ├── handlers/
│       │   ├── home.go         # Web pages
│       │   ├── posts.go
│       │   ├── accounts.go
│       │   ├── timelines.go
│       │   ├── notifications.go
│       │   └── search.go
│       ├── api/
│       │   ├── posts.go        # API handlers
│       │   ├── accounts.go
│       │   ├── timelines.go
│       │   └── search.go
│       └── views/
│           ├── layouts/
│           │   └── default.html
│           ├── pages/
│           │   ├── home.html
│           │   ├── login.html
│           │   ├── register.html
│           │   ├── profile.html
│           │   ├── post.html
│           │   ├── settings.html
│           │   └── notifications.html
│           └── components/
│               ├── post_card.html
│               ├── account_card.html
│               ├── timeline.html
│               ├── compose.html
│               └── poll.html
├── feature/
│   ├── accounts/
│   │   ├── service.go          # Account business logic
│   │   ├── types.go            # Account types
│   │   └── validation.go       # Input validation
│   ├── posts/
│   │   ├── service.go          # Post CRUD
│   │   ├── types.go            # Post types
│   │   ├── parser.go           # Content parsing
│   │   └── threading.go        # Thread operations
│   ├── timelines/
│   │   ├── service.go          # Feed generation
│   │   └── types.go
│   ├── interactions/
│   │   ├── service.go          # Likes, reposts, bookmarks
│   │   └── types.go
│   ├── relationships/
│   │   ├── service.go          # Follows, blocks, mutes
│   │   └── types.go
│   ├── notifications/
│   │   ├── service.go          # Notification delivery
│   │   └── types.go
│   ├── search/
│   │   ├── service.go          # Search logic
│   │   └── types.go
│   ├── trending/
│   │   ├── service.go          # Trending calculation
│   │   └── types.go
│   ├── media/
│   │   ├── service.go          # Upload handling
│   │   ├── processor.go        # Image/video processing
│   │   └── types.go
│   └── polls/
│       ├── service.go          # Poll CRUD
│       └── types.go
├── store/
│   └── duckdb/
│       ├── store.go            # Store interface
│       ├── schema.sql          # DDL
│       ├── accounts.go         # Account queries
│       ├── posts.go            # Post queries
│       ├── timelines.go        # Timeline queries
│       ├── interactions.go     # Interaction queries
│       ├── relationships.go    # Relationship queries
│       └── notifications.go    # Notification queries
├── pkg/
│   ├── ulid/
│   │   └── ulid.go             # ID generation
│   ├── text/
│   │   ├── parser.go           # Mention/hashtag parsing
│   │   └── parser_test.go
│   ├── password/
│   │   └── argon2.go           # Password hashing
│   └── media/
│       ├── image.go            # Image processing
│       └── video.go            # Video processing
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Implementation Phases

### Phase 1: Core

- Account registration/login
- Post creation (text only)
- Home timeline (basic)
- Follow/unfollow
- Like/unlike

### Phase 2: Content

- Reply threading
- Quote posts
- Reposts
- Hashtag extraction
- Mention extraction

### Phase 3: Features

- Media attachments
- Polls
- Content warnings
- Visibility controls
- Edit history

### Phase 4: Discovery

- Local timeline
- Trending hashtags
- Trending posts
- Search (posts, accounts, hashtags)

### Phase 5: Social

- Lists
- Mute/block
- Notifications
- Bookmarks
- Suggested accounts

### Phase 6: Polish

- Profile customization
- Settings page
- Import/export
- Admin tools
- API documentation

## Security Considerations

### Input Validation

- Sanitize HTML in all user input
- Validate file types by magic bytes, not extension
- Rate limit all write endpoints
- CSRF protection on all forms

### Privacy

- Followers-only posts not in search
- Direct posts only visible to mentioned
- Block prevents all interaction
- Mute is invisible to target

### Moderation

- Report system for posts and accounts
- Admin dashboard for review
- Suspension with public notice
- Appeal process (future)
