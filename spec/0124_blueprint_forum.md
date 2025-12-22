# Spec 0124: Forum Blueprint

A production-ready forum platform combining the best features from Reddit, Discourse, and traditional forums.

## Overview

The Forum blueprint implements a full-featured discussion platform. It demonstrates:

- Multi-level organization (forums → threads → posts)
- Voting and ranking algorithms (hot, top, best, controversial)
- Rich moderation tools and permissions
- Karma and reputation systems
- Threaded discussions with nested replies
- Content discovery and search
- User badges and achievements
- Tag/flair systems
- Cross-forum features

## Feature Analysis

### Features from Reddit

| Feature | Included | Notes |
|---------|----------|-------|
| Subreddits | Yes | Called "forums" |
| Upvote/downvote | Yes | With karma tracking |
| Karma system | Yes | Post + comment karma |
| Hot algorithm | Yes | Time decay + score |
| Controversial sorting | Yes | Upvote/downvote ratio |
| Awards | Yes | Custom badges on posts |
| User flair | Yes | Per-forum customizable |
| Post flair | Yes | Forum-specific tags |
| Saved posts | Yes | Private bookmarks |
| Cross-posting | Yes | Reference original |
| Nested comments | Yes | Tree structure |
| Sticky posts | Yes | Pin to top |
| Locked threads | Yes | Prevent new replies |
| Reports | Yes | User reporting |
| Moderation tools | Yes | Remove, approve, ban |
| Wiki pages | Future | Per-forum wikis |

### Features from Discourse

| Feature | Included | Notes |
|---------|----------|-------|
| Categories | Yes | Hierarchical forums |
| Tags | Yes | Cross-cutting labels |
| Trust levels | Yes | Reputation tiers |
| Badges | Yes | Achievement system |
| Best posts | Yes | Quality highlighting |
| Reading tracking | Yes | Read/unread state |
| Bookmarks | Yes | With notes |
| Topic watching | Yes | Notifications |
| Suggested topics | Yes | Related threads |
| Post revisions | Yes | Full edit history |
| Draft system | Yes | Auto-save |
| Rich formatting | Yes | Markdown + preview |
| @ mentions | Yes | User notifications |
| Moderation queue | Yes | Review flagged content |

### Features from Traditional Forums

| Feature | Included | Notes |
|---------|----------|-------|
| Forums/subforums | Yes | Nested categories |
| Threads/topics | Yes | Discussion containers |
| Posts/replies | Yes | Threaded or linear |
| User signatures | Yes | Auto-appended text |
| Private messages | Future | DM system |
| Polls | Yes | In-thread voting |
| Reactions | Yes | Multiple emoji types |
| View counts | Yes | Thread popularity |
| Sticky/announcement | Yes | Forum-level pins |
| Thread subscription | Yes | Email/notification |

## Data Model

### Entity Relationships

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│   Account    │──1:N──│    Forum     │──1:N──│   Thread     │
└──────────────┘       └──────────────┘       └──────────────┘
       │                      │                       │
       │                      │                       │
       │                      │                       ▼
       │                      │                ┌──────────────┐
       │                      │                │     Post     │
       │                      │                └──────────────┘
       │                      │                       │
       │                      ├──M:N── members ───────┤
       │                      │                       │
       │                      ├──M:N── moderators ────┤
       │                      │                       │
       ├──1:N──── votes ──────┴───────────────────────┘
       │
       ├──1:N──── badges
       │
       ├──1:N──── forum_flair ──── Forum
       │
       ├──M:N──── subscriptions ─── Thread
       │
       └──1:N──── saved ──────────── Post/Thread
```

### Forum Hierarchy

Forums can be organized hierarchically:

```
General Discussion (parent)
├── Announcements
├── Feedback
└── Off-topic

Technology (parent)
├── Programming
│   ├── Go
│   ├── Python
│   └── JavaScript
├── DevOps
└── Data Science
```

### Thread Types

| Type | Description | Features |
|------|-------------|----------|
| `discussion` | Normal thread | All features enabled |
| `question` | Q&A format | Best answer marking |
| `poll` | Poll thread | Voting on options |
| `announcement` | Official notice | Comments optional |
| `sticky` | Pinned thread | Always on top |

### Post Structure

Posts form a tree via `parent_id`:

```
Post 1 (depth 0)
├── Post 2 (depth 1, parent = 1)
│   ├── Post 4 (depth 2, parent = 2)
│   └── Post 5 (depth 2, parent = 2)
└── Post 3 (depth 1, parent = 1)
    └── Post 6 (depth 2, parent = 3)
        └── Post 7 (depth 3, parent = 6)
```

### Voting System

Each account can vote once per post/thread:

| Vote Type | Value | Effect |
|-----------|-------|--------|
| Upvote | +1 | Increases score, adds karma |
| Downvote | -1 | Decreases score, removes karma |
| No vote | 0 | Neutral |

Score calculation:
```
score = upvotes - downvotes
```

Karma calculation:
```
post_karma = sum(post_scores) for all posts by user
comment_karma = sum(comment_scores) for all comments by user
total_karma = post_karma + comment_karma
```

### Sorting Algorithms

#### Hot (Default)

Based on Reddit's algorithm with time decay:

```go
func HotScore(score int, createdAt time.Time) float64 {
    // Score direction: positive = upvotes, negative = downvotes
    order := math.Log10(math.Max(math.Abs(float64(score)), 1))

    // Sign: 1 for positive score, -1 for negative, 0 for zero
    sign := 0.0
    if score > 0 {
        sign = 1
    } else if score < 0 {
        sign = -1
    }

    // Seconds since epoch - older posts decay
    epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    seconds := createdAt.Sub(epoch).Seconds()

    return sign*order + seconds/45000
}
```

#### Top

Simply by score in time window:

```sql
SELECT * FROM threads
WHERE created_at > NOW() - INTERVAL :window
ORDER BY score DESC, created_at DESC
```

#### Best

Wilson score confidence interval (balances score and vote count):

```go
func BestScore(upvotes, downvotes int) float64 {
    n := float64(upvotes + downvotes)
    if n == 0 {
        return 0
    }

    z := 1.96 // 95% confidence
    phat := float64(upvotes) / n

    return (phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)) / (1 + z*z/n)
}
```

#### Controversial

Balances upvotes and downvotes (high engagement, divisive):

```go
func ControversialScore(upvotes, downvotes int) float64 {
    total := float64(upvotes + downvotes)
    if total == 0 {
        return 0
    }

    // Posts with near 50/50 split score highest
    balance := math.Min(float64(upvotes), float64(downvotes))
    magnitude := float64(upvotes + downvotes)

    return balance * magnitude
}
```

#### New

Simple chronological:

```sql
SELECT * FROM threads
ORDER BY created_at DESC
```

#### Rising

New posts with growing scores:

```sql
SELECT *,
    score / POWER(EXTRACT(EPOCH FROM NOW() - created_at) / 3600 + 2, 1.5) as rising_score
FROM threads
WHERE created_at > NOW() - INTERVAL '12 hours'
ORDER BY rising_score DESC
```

## Forum Features

### Forum Types

| Type | Description | Membership |
|------|-------------|------------|
| `public` | Anyone can view/post | Open |
| `restricted` | Anyone can view, approved posters | Request to post |
| `private` | Only members see/post | Invite only |
| `archived` | Read-only | No new posts |

### Forum Settings

```go
type ForumSettings struct {
    AllowPolls        bool
    AllowImages       bool
    AllowVideos       bool
    RequireApproval   bool // Posts need mod approval
    MinKarmaToPost    int
    MinAgeToPost      time.Duration
    NSFW              bool
    Restricted18Plus  bool
    RateLimitPosts    int // Posts per day
    RateLimitComments int // Comments per day
}
```

### Forum Rules

Each forum can define rules:

```go
type ForumRule struct {
    ID          string
    ForumID     string
    Title       string
    Description string
    Position    int
    CreatedAt   time.Time
}
```

Displayed on sidebar and referenced in reports.

## User System

### Account Profile

Extended from microblog:

```go
type Account struct {
    ID           string
    Username     string
    DisplayName  string
    Email        string
    Bio          string
    AvatarURL    string
    HeaderURL    string
    Signature    string // Auto-appended to posts
    PostKarma    int
    CommentKarma int
    TotalKarma   int
    TrustLevel   int // 0-4 (new, basic, member, regular, leader)
    CakeDay      time.Time // Join anniversary
    Verified     bool
    Admin        bool
    Suspended    bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### Trust Levels

Inspired by Discourse:

| Level | Name | Requirements | Permissions |
|-------|------|--------------|-------------|
| 0 | New User | Just joined | Basic posting |
| 1 | Basic | 5 posts, 10 minutes read | Create threads |
| 2 | Member | 50 posts, 100 karma | Edit wiki, flag posts |
| 3 | Regular | 500 posts, 1000 karma | Pin/close own threads |
| 4 | Leader | Granted by admins | Global moderation |

### Badges

Achievement system:

```go
type Badge struct {
    ID          string
    Name        string
    Description string
    Icon        string
    Tier        string // bronze, silver, gold, platinum
    Criteria    string // JSON criteria for auto-grant
}

type AccountBadge struct {
    ID        string
    AccountID string
    BadgeID   string
    Reason    string // Custom award reason
    GrantedBy string // Account ID or "system"
    GrantedAt time.Time
}
```

Example badges:
- First Post
- Helpful (10 best answers)
- Popular Post (100+ upvotes)
- Contributor (100 posts)
- Veteran (1 year member)
- Philanthropist (100 awards given)

### User Flair

Per-forum custom flair:

```go
type ForumFlair struct {
    ID          string
    ForumID     string
    Text        string
    TextColor   string
    Background  string
    Emoji       string
    ModOnly     bool
}

type AccountFlair struct {
    AccountID string
    ForumID   string
    FlairID   string
}
```

## Thread Features

### Thread States

| State | Description | Effect |
|-------|-------------|--------|
| `open` | Active discussion | Replies allowed |
| `locked` | Closed for comments | Read-only |
| `archived` | Auto-locked after time | Read-only |
| `removed` | Deleted by mods | Hidden from listing |
| `pending` | Awaiting approval | Visible to author only |

### Thread Attributes

```go
type Thread struct {
    ID          string
    ForumID     string
    AccountID   string
    Type        string // discussion, question, poll, announcement
    Title       string
    Content     string
    FlairID     string
    Sticky      bool
    Locked      bool
    NSFW        bool
    Spoiler     bool
    State       string
    ViewCount   int
    Score       int
    Upvotes     int
    Downvotes   int
    PostCount   int // Reply count
    BestPostID  string // Marked best answer
    LastPostAt  time.Time
    CreatedAt   time.Time
    EditedAt    *time.Time

    // Computed fields
    HotScore          float64
    BestScore         float64
    ControversialScore float64
}
```

### Thread Tags

Forums can define tags:

```go
type ForumTag struct {
    ID      string
    ForumID string
    Name    string
    Color   string
}

type ThreadTag struct {
    ThreadID string
    TagID    string
}
```

## Post Features

### Post Types

| Type | Description | Use Case |
|------|-------------|----------|
| `comment` | Normal reply | Discussion |
| `best_answer` | Marked solution | Q&A threads |
| `mod_note` | Moderator comment | Official statements |

### Post Attributes

```go
type Post struct {
    ID        string
    ThreadID  string
    AccountID string
    ParentID  string // NULL for top-level, otherwise parent post ID
    Content   string
    Depth     int
    Score     int
    Upvotes   int
    Downvotes int
    IsBest    bool
    Type      string
    EditedAt  *time.Time
    CreatedAt time.Time

    // Relationships
    Account  *Account
    Children []*Post // Nested replies
    Awards   []*PostAward

    // Current user state
    UserVote    int  // -1, 0, 1
    IsSaved     bool
    IsOwner     bool
}
```

### Post Formatting

Support Markdown with extensions:

- Headers (h1-h6)
- Bold, italic, strikethrough
- Lists (ordered, unordered)
- Links (auto-detect, explicit)
- Code blocks with syntax highlighting
- Blockquotes
- Tables
- Spoiler tags
- User mentions (@username)
- Thread mentions (/r/forum)

### Post Awards

Users can give awards to posts:

```go
type Award struct {
    ID          string
    Name        string
    Description string
    Icon        string
    Cost        int // Virtual currency cost
}

type PostAward struct {
    ID        string
    PostID    string
    AwardID   string
    GivenBy   string
    Anonymous bool
    Message   string // Optional message
    GivenAt   time.Time
}
```

## Moderation

### Moderator Roles

| Role | Permissions |
|------|-------------|
| `moderator` | Remove posts, ban users, edit flair |
| `admin` | All moderator + forum settings |
| `owner` | Creator, can add/remove mods |

### Moderator Actions

```go
type ModAction struct {
    ID          string
    ForumID     string
    ModeratorID string
    Action      string
    TargetType  string // thread, post, account
    TargetID    string
    Reason      string
    Details     string // JSON details
    CreatedAt   time.Time
}
```

Action types:
- `approve_post`
- `remove_post`
- `lock_thread`
- `sticky_thread`
- `ban_user`
- `mute_user`
- `add_flair`
- `edit_settings`

### Reports

User reporting system:

```go
type Report struct {
    ID         string
    ReporterID string
    TargetType string // thread, post, account
    TargetID   string
    Reason     string
    Details    string
    Status     string // pending, reviewed, dismissed
    ReviewedBy string
    ReviewedAt *time.Time
    CreatedAt  time.Time
}
```

Report reasons:
- Spam
- Harassment
- Hate speech
- Violence
- NSFW (not tagged)
- Misinformation
- Off-topic
- Other (custom)

### Bans

```go
type Ban struct {
    ID        string
    ForumID   string // NULL for site-wide
    AccountID string
    BannedBy  string
    Reason    string
    ExpiresAt *time.Time // NULL for permanent
    CreatedAt time.Time
}
```

### Mutes

Temporary restrictions:

```go
type Mute struct {
    ID        string
    ForumID   string
    AccountID string
    MutedBy   string
    Reason    string
    ExpiresAt time.Time
    CreatedAt time.Time
}
```

## Subscriptions & Notifications

### Thread Subscriptions

```go
type Subscription struct {
    ID        string
    AccountID string
    ThreadID  string
    Type      string // all, mentions, watching
    CreatedAt time.Time
}
```

Subscription types:
- `all`: Notify on all new posts
- `mentions`: Only when mentioned
- `watching`: Default for created/replied threads

### Notification Types

| Type | Trigger | Contains |
|------|---------|----------|
| `reply` | Reply to your post | actor_id, post_id |
| `mention` | @username in post | actor_id, post_id |
| `quote` | Your post quoted | actor_id, post_id |
| `best_answer` | Your answer marked best | thread_id, post_id |
| `badge` | Badge earned | badge_id |
| `mod_action` | Moderated content | action_id |
| `thread_reply` | Subscribed thread activity | thread_id, post_id |

## Search & Discovery

### Search Types

| Type | Scope | Indexed Fields |
|------|-------|----------------|
| Threads | Title, content | Full-text |
| Posts | Content | Full-text |
| Forums | Name, description | Prefix |
| Users | Username, display name | Prefix |
| Tags | Tag name | Exact |

### Search Filters

```go
type SearchFilters struct {
    ForumID    string
    AuthorID   string
    Tags       []string
    TimeRange  string // hour, day, week, month, year, all
    MinScore   int
    HasImages  bool
    HasVideos  bool
    NSFW       *bool
}
```

### Trending Content

#### Trending Forums

Based on activity velocity:

```sql
WITH recent_activity AS (
    SELECT forum_id, COUNT(*) as activity_24h
    FROM threads
    WHERE created_at > NOW() - INTERVAL '24 hours'
    GROUP BY forum_id
),
previous_activity AS (
    SELECT forum_id, COUNT(*) as activity_prev
    FROM threads
    WHERE created_at BETWEEN NOW() - INTERVAL '48 hours'
                         AND NOW() - INTERVAL '24 hours'
    GROUP BY forum_id
)
SELECT f.*,
    r.activity_24h,
    COALESCE(p.activity_prev, 1) as activity_prev,
    r.activity_24h::float / COALESCE(p.activity_prev, 1) as velocity
FROM forums f
JOIN recent_activity r ON r.forum_id = f.id
LEFT JOIN previous_activity p ON p.forum_id = f.id
WHERE r.activity_24h >= 5
ORDER BY velocity DESC, r.activity_24h DESC
LIMIT 10
```

#### Trending Tags

Similar to hashtag trending in microblog:

```sql
WITH recent_usage AS (
    SELECT tag_id, COUNT(*) as count_24h
    FROM thread_tags tt
    JOIN threads t ON t.id = tt.thread_id
    WHERE t.created_at > NOW() - INTERVAL '24 hours'
    GROUP BY tag_id
),
previous_usage AS (
    SELECT tag_id, COUNT(*) as count_prev
    FROM thread_tags tt
    JOIN threads t ON t.id = tt.thread_id
    WHERE t.created_at BETWEEN NOW() - INTERVAL '48 hours'
                           AND NOW() - INTERVAL '24 hours'
    GROUP BY tag_id
)
SELECT ft.name,
    r.count_24h,
    r.count_24h::float / COALESCE(p.count_prev, 1) as velocity
FROM forum_tags ft
JOIN recent_usage r ON r.tag_id = ft.id
LEFT JOIN previous_usage p ON p.tag_id = ft.id
WHERE r.count_24h >= 3
ORDER BY velocity DESC
LIMIT 10
```

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
        "message": "Thread title too short",
        "details": {
            "min_length": 5,
            "actual_length": 2
        }
    }
}
```

### Pagination

Cursor-based for threads:

```
GET /api/v1/forums/{id}/threads?limit=25&sort=hot
GET /api/v1/forums/{id}/threads?limit=25&sort=hot&after=01HXYZ
```

Offset-based for posts (in thread):

```
GET /api/v1/threads/{id}/posts?limit=50&depth=2
```

### Rate Limiting

| Endpoint | Limit |
|----------|-------|
| POST /threads | 10/hour |
| POST /posts | 50/hour |
| POST /votes | 200/hour |
| GET /forums/* | 300/hour |
| GET /search | 60/hour |

## Directory Structure

```
blueprints/forum/
├── cmd/forum/
│   └── main.go
├── cli/
│   ├── root.go
│   ├── serve.go
│   ├── init.go
│   ├── user.go
│   └── seed.go
├── app/web/
│   ├── server.go
│   ├── config.go
│   ├── middleware.go
│   └── handler/
│       ├── auth.go
│       ├── forums.go
│       ├── threads.go
│       ├── posts.go
│       ├── votes.go
│       ├── moderation.go
│       ├── search.go
│       ├── profiles.go
│       └── pages.go
├── feature/
│   ├── accounts/
│   │   ├── api.go
│   │   └── service.go
│   ├── forums/
│   │   ├── api.go
│   │   └── service.go
│   ├── threads/
│   │   ├── api.go
│   │   └── service.go
│   ├── posts/
│   │   ├── api.go
│   │   └── service.go
│   ├── votes/
│   │   ├── api.go
│   │   └── service.go
│   ├── moderation/
│   │   ├── api.go
│   │   └── service.go
│   ├── badges/
│   │   ├── api.go
│   │   └── service.go
│   ├── subscriptions/
│   │   ├── api.go
│   │   └── service.go
│   ├── search/
│   │   ├── api.go
│   │   └── service.go
│   └── trending/
│       ├── api.go
│       └── service.go
├── store/duckdb/
│   ├── store.go
│   ├── schema.sql
│   ├── accounts_store.go
│   ├── forums_store.go
│   ├── threads_store.go
│   ├── posts_store.go
│   ├── votes_store.go
│   ├── moderation_store.go
│   ├── badges_store.go
│   ├── subscriptions_store.go
│   ├── search_store.go
│   └── trending_store.go
├── pkg/
│   ├── ulid/
│   │   └── ulid.go
│   ├── markdown/
│   │   ├── parser.go
│   │   └── sanitizer.go
│   ├── password/
│   │   └── argon2.go
│   └── ranking/
│       ├── hot.go
│       ├── best.go
│       └── controversial.go
├── assets/
│   ├── assets.go
│   ├── static/
│   │   ├── css/
│   │   │   └── app.css
│   │   └── js/
│   │       └── app.js
│   └── views/
│       ├── layouts/
│       │   └── default.html
│       ├── pages/
│       │   ├── home.html
│       │   ├── forum.html
│       │   ├── thread.html
│       │   ├── profile.html
│       │   ├── search.html
│       │   ├── mod_queue.html
│       │   └── settings.html
│       └── components/
│           ├── forum_card.html
│           ├── thread_card.html
│           ├── post_card.html
│           ├── vote_buttons.html
│           ├── user_badge.html
│           └── sidebar.html
├── go.mod
├── go.sum
└── README.md
```

## Implementation Phases

### Phase 1: Core Foundation

- Account registration/login
- Forum CRUD
- Thread creation (basic)
- Post creation (flat, no nesting)
- Basic voting (upvote only)

### Phase 2: Hierarchy & Structure

- Nested forums (parent/child)
- Threaded posts (tree structure)
- Thread types (discussion, question, poll)
- Post depth limits
- Forum membership

### Phase 3: Voting & Ranking

- Full voting (up/down)
- Karma tracking
- Sorting algorithms (hot, top, best, new, controversial, rising)
- Score calculations
- Trust levels

### Phase 4: Moderation

- Moderator roles
- Remove/approve posts
- Lock/sticky threads
- Ban/mute users
- Mod queue
- Mod log

### Phase 5: Discovery & Search

- Search (threads, posts, forums, users)
- Trending forums
- Trending tags
- Popular threads
- Suggested content

### Phase 6: Engagement

- Thread subscriptions
- Notifications
- Saved posts
- Post awards
- Badges/achievements
- User flair
- Thread flair/tags

### Phase 7: Polish

- Edit history
- Draft system
- Markdown preview
- Rich embeds
- User signatures
- Profile customization
- Forum rules
- Settings pages

## Security Considerations

### Input Validation

- Sanitize all Markdown/HTML input
- Rate limit by IP and account
- Validate forum membership before posting
- Check moderator permissions
- CSRF protection on all forms
- XSS prevention in user content

### Privacy

- Private forums not in search
- Deleted content truly removed
- Moderator actions logged
- Reports are anonymous
- User IPs not exposed

### Spam Prevention

- Karma threshold for posting
- Account age requirements
- Rate limiting per forum
- Moderator queue for low-trust users
- Automatic spam detection (future)

### Voting Integrity

- One vote per account per item
- Detect vote brigading (future)
- Prevent self-voting manipulation
- Hide scores for new posts (anti-bandwagon)

## Performance Considerations

### Caching Strategy

```go
// Cache hot threads per forum (5 min TTL)
key := fmt.Sprintf("forum:%s:threads:hot", forumID)

// Cache forum stats (1 min TTL)
key := fmt.Sprintf("forum:%s:stats", forumID)

// Cache user karma (5 min TTL)
key := fmt.Sprintf("account:%s:karma", accountID)

// Cache thread with posts (30 sec TTL)
key := fmt.Sprintf("thread:%s:posts", threadID)
```

### Database Indexes

Critical indexes for performance:

```sql
-- Thread sorting
CREATE INDEX idx_threads_hot ON threads(forum_id, hot_score DESC, created_at DESC);
CREATE INDEX idx_threads_top ON threads(forum_id, score DESC, created_at DESC);
CREATE INDEX idx_threads_new ON threads(forum_id, created_at DESC);

-- Post hierarchy
CREATE INDEX idx_posts_thread_parent ON posts(thread_id, parent_id, created_at);
CREATE INDEX idx_posts_depth ON posts(thread_id, depth, score DESC);

-- Voting
CREATE INDEX idx_votes_account_target ON votes(account_id, target_type, target_id);

-- Moderation
CREATE INDEX idx_reports_status ON reports(status, created_at DESC);
CREATE INDEX idx_mod_actions_forum ON mod_actions(forum_id, created_at DESC);
```

### Denormalization

Pre-computed fields for performance:

- `threads.score` (upvotes - downvotes)
- `threads.hot_score` (computed ranking)
- `threads.post_count` (reply count)
- `accounts.total_karma` (sum of all karma)
- `forums.thread_count` (total threads)
- `forums.member_count` (total members)

## Future Enhancements

- **Multi-reddit**: Custom forum groups
- **Wiki system**: Per-forum wikis
- **Live chat**: Real-time chat rooms
- **Private messaging**: DM system
- **Post scheduling**: Schedule thread posts
- **Content filters**: User-defined filters
- **RES features**: Tag users, filter keywords
- **Mobile apps**: Native iOS/Android
- **Federation**: ActivityPub support
- **Webhooks**: Integration events
- **API v2**: GraphQL endpoint
- **AI moderation**: Auto-detect spam/abuse
- **Advanced analytics**: Forum insights
