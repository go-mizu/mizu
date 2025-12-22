# Forum Blueprint

A production-ready forum platform built with Mizu, combining the best features from Reddit, Discourse, and traditional forums.

## Features

### Organization
- **Hierarchical forums** - Nested categories and subcategories
- **Multiple thread types** - Discussions, Q&A, polls, announcements
- **Tagging system** - Cross-cutting labels and forum-specific flair
- **Thread states** - Open, locked, archived, sticky

### Engagement
- **Voting system** - Upvote/downvote with karma tracking
- **Nested comments** - Tree-structured replies with depth limits
- **Awards & badges** - Give awards to posts, earn achievement badges
- **User flair** - Customizable per-forum identity
- **Subscriptions** - Follow threads and get notifications

### Sorting & Discovery
- **Multiple algorithms** - Hot, Top, Best, New, Rising, Controversial
- **Full-text search** - Search threads, posts, forums, and users
- **Trending content** - Discover popular forums and tags
- **Personalization** - Saved posts, custom feeds

### Moderation
- **Role-based permissions** - Owner, admin, moderator roles
- **Comprehensive tools** - Remove, approve, lock, sticky, ban, mute
- **Report system** - User reporting with moderation queue
- **Audit log** - Track all moderator actions
- **Auto-moderation** - Karma and age requirements

### Content
- **Rich formatting** - Markdown with syntax highlighting
- **Media support** - Images, videos, embeds
- **Polls** - In-thread voting
- **Edit history** - Full revision tracking
- **Draft system** - Auto-save work in progress

### Reputation
- **Karma system** - Post karma + comment karma
- **Trust levels** - 5 tiers unlocking progressive permissions
- **Badges** - Achievement system for milestones
- **User profiles** - Stats, history, badges, cake day

## Quick Start

### Installation

```bash
# Using the Mizu CLI
mizu new myForum --template forum
cd myForum

# Or clone from blueprints
cp -r blueprints/forum myForum
cd myForum
go mod tidy
```

### Initialize Database

```bash
# Create database and run migrations
go run ./cmd/forum init

# Optionally seed with sample data
go run ./cmd/forum seed
```

### Run Server

```bash
# Start the web server
go run ./cmd/forum serve

# Or build and run
go build -o forum ./cmd/forum
./forum serve
```

Visit http://localhost:8080

### Create Admin User

```bash
# Create user via CLI
go run ./cmd/forum user create --username admin --email admin@example.com --admin

# Or register via web and promote
go run ./cmd/forum user promote admin
```

## Configuration

Configuration via `config.yaml` or environment variables:

```yaml
server:
  host: localhost
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  path: forum.db
  driver: duckdb

forum:
  site_name: My Forum
  site_description: A community forum
  max_upload_size: 10485760  # 10 MB
  posts_per_page: 25
  comments_per_page: 50

registration:
  enabled: true
  require_email: true
  min_username_length: 3
  max_username_length: 20

moderation:
  min_karma_to_post: 0
  min_account_age: 0
  auto_approve: true

rate_limits:
  threads_per_hour: 10
  posts_per_hour: 50
  votes_per_hour: 200
```

## Architecture

### Clean Architecture

The forum follows clean architecture principles:

```
┌─────────────────────────────────────┐
│   Handlers (HTTP/Web)               │
│   - Parse requests                  │
│   - Call services                   │
│   - Render responses                │
└─────────────────┬───────────────────┘
                  │
┌─────────────────▼───────────────────┐
│   Services (Business Logic)         │
│   - Validation                      │
│   - Business rules                  │
│   - Orchestration                   │
└─────────────────┬───────────────────┘
                  │
┌─────────────────▼───────────────────┐
│   Store (Data Access)               │
│   - SQL queries                     │
│   - Transactions                    │
│   - Data mapping                    │
└─────────────────────────────────────┘
```

### Project Structure

```
blueprints/forum/
├── cmd/forum/          # CLI entry point
├── cli/                # CLI commands (serve, init, user, seed)
├── app/web/            # HTTP server & handlers
│   ├── handler/        # Request handlers
│   └── middleware.go   # Auth, logging, rate limiting
├── feature/            # Business logic by feature
│   ├── accounts/       # User management
│   ├── forums/         # Forum CRUD
│   ├── threads/        # Thread operations
│   ├── posts/          # Post/comment management
│   ├── votes/          # Voting & karma
│   ├── moderation/     # Mod tools
│   ├── badges/         # Achievements
│   ├── subscriptions/  # Thread watching
│   ├── search/         # Search & filtering
│   └── trending/       # Discovery algorithms
├── store/duckdb/       # DuckDB implementation
├── pkg/                # Shared utilities
│   ├── markdown/       # Markdown parsing
│   ├── password/       # Password hashing
│   ├── ranking/        # Sorting algorithms
│   └── ulid/           # ID generation
└── assets/             # Templates & static files
    ├── static/css/
    ├── static/js/
    └── views/
```

### Interface-Based Design

All features expose clean interfaces:

```go
// Feature API interface
type ForumsAPI interface {
    Create(ctx, accountID, *CreateIn) (*Forum, error)
    GetByID(ctx, id) (*Forum, error)
    List(ctx, *ListFilters) ([]*Forum, error)
    Update(ctx, id, accountID, *UpdateIn) (*Forum, error)
    Delete(ctx, id, accountID) error
}

// Storage interface
type ForumsStore interface {
    Insert(ctx, *Forum) error
    GetByID(ctx, id) (*Forum, error)
    // ...
}

// Service implements API, depends on Store
type Service struct {
    store ForumsStore
}
```

## Data Model

### Core Entities

#### Forums
Organizational containers for threads:

```go
type Forum struct {
    ID          string
    ParentID    string  // For nested forums
    Name        string
    Slug        string
    Description string
    Type        string  // public, restricted, private, archived
    NSFW        bool
    Settings    ForumSettings
    Rules       []ForumRule

    // Counts
    ThreadCount int
    MemberCount int

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

#### Threads
Discussion topics:

```go
type Thread struct {
    ID        string
    ForumID   string
    AccountID string
    Type      string  // discussion, question, poll, announcement
    Title     string
    Content   string
    Sticky    bool
    Locked    bool
    NSFW      bool

    // Engagement
    Score      int
    Upvotes    int
    Downvotes  int
    ViewCount  int
    PostCount  int

    // Computed scores
    HotScore          float64
    BestScore         float64
    ControversialScore float64

    CreatedAt time.Time
    EditedAt  *time.Time
}
```

#### Posts
Replies in threads:

```go
type Post struct {
    ID        string
    ThreadID  string
    AccountID string
    ParentID  string  // NULL for top-level
    Content   string
    Depth     int

    // Engagement
    Score     int
    Upvotes   int
    Downvotes int
    IsBest    bool    // Marked as best answer

    CreatedAt time.Time
    EditedAt  *time.Time
}
```

#### Votes
User votes on content:

```go
type Vote struct {
    AccountID  string
    TargetType string  // thread, post
    TargetID   string
    Value      int     // -1, 0, 1
    CreatedAt  time.Time
}
```

## Algorithms

### Hot Ranking

Based on Reddit's algorithm:

```go
func HotScore(score int, createdAt time.Time) float64 {
    order := math.Log10(math.Max(math.Abs(float64(score)), 1))
    sign := 0.0
    if score > 0 {
        sign = 1
    } else if score < 0 {
        sign = -1
    }

    epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    seconds := createdAt.Sub(epoch).Seconds()

    return sign*order + seconds/45000
}
```

### Best Ranking

Wilson score confidence interval:

```go
func BestScore(upvotes, downvotes int) float64 {
    n := float64(upvotes + downvotes)
    if n == 0 {
        return 0
    }

    z := 1.96  // 95% confidence
    phat := float64(upvotes) / n

    return (phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)) / (1 + z*z/n)
}
```

### Controversial Ranking

Balances upvotes and downvotes:

```go
func ControversialScore(upvotes, downvotes int) float64 {
    total := float64(upvotes + downvotes)
    if total == 0 {
        return 0
    }

    balance := math.Min(float64(upvotes), float64(downvotes))
    magnitude := float64(upvotes + downvotes)

    return balance * magnitude
}
```

## API Endpoints

### Forums

```
GET    /api/v1/forums                 - List all forums
POST   /api/v1/forums                 - Create forum (admin)
GET    /api/v1/forums/:id             - Get forum details
PATCH  /api/v1/forums/:id             - Update forum (admin)
DELETE /api/v1/forums/:id             - Delete forum (admin)
POST   /api/v1/forums/:id/join        - Join forum
POST   /api/v1/forums/:id/leave       - Leave forum
GET    /api/v1/forums/:id/moderators  - List moderators
POST   /api/v1/forums/:id/moderators  - Add moderator (owner)
```

### Threads

```
GET    /api/v1/forums/:id/threads        - List threads
POST   /api/v1/forums/:id/threads        - Create thread
GET    /api/v1/threads/:id               - Get thread
PATCH  /api/v1/threads/:id               - Update thread
DELETE /api/v1/threads/:id               - Delete thread
POST   /api/v1/threads/:id/vote          - Vote on thread
POST   /api/v1/threads/:id/subscribe     - Subscribe to thread
DELETE /api/v1/threads/:id/subscribe     - Unsubscribe
POST   /api/v1/threads/:id/lock          - Lock thread (mod)
POST   /api/v1/threads/:id/sticky        - Sticky thread (mod)
```

### Posts

```
GET    /api/v1/threads/:id/posts      - List posts
POST   /api/v1/threads/:id/posts      - Create post
GET    /api/v1/posts/:id              - Get post
PATCH  /api/v1/posts/:id              - Update post
DELETE /api/v1/posts/:id              - Delete post
POST   /api/v1/posts/:id/vote         - Vote on post
POST   /api/v1/posts/:id/award        - Give award (costs karma)
POST   /api/v1/posts/:id/best         - Mark as best answer (thread author)
```

### Moderation

```
GET    /api/v1/forums/:id/queue       - Moderation queue
POST   /api/v1/posts/:id/approve      - Approve post (mod)
POST   /api/v1/posts/:id/remove       - Remove post (mod)
GET    /api/v1/forums/:id/reports     - List reports (mod)
POST   /api/v1/reports                - Create report
POST   /api/v1/forums/:id/ban         - Ban user (mod)
GET    /api/v1/forums/:id/logs        - Mod action log (mod)
```

### Search

```
GET    /api/v1/search?q=query&type=threads
GET    /api/v1/search?q=query&type=posts&forum=:id
GET    /api/v1/trending/forums
GET    /api/v1/trending/tags
```

## Web Routes

```
GET    /                          - Home (trending threads)
GET    /f/:slug                   - Forum page
GET    /f/:slug/new               - Create thread
GET    /f/:slug/t/:id             - Thread page
GET    /search                    - Search page
GET    /u/:username               - User profile
GET    /u/:username/posts         - User's posts
GET    /u/:username/threads       - User's threads
GET    /settings                  - User settings
GET    /settings/profile          - Profile settings
GET    /login                     - Login page
GET    /register                  - Registration
```

## Moderation

### Roles

- **Owner**: Forum creator, can add/remove mods
- **Admin**: Full moderation + settings
- **Moderator**: Remove, approve, ban users

### Tools

- **Mod Queue**: Review flagged/new content
- **Mod Log**: Audit trail of all actions
- **Ban/Mute**: Temporary or permanent restrictions
- **Lock/Sticky**: Thread management
- **Remove/Approve**: Content moderation

### Auto-moderation

Configurable requirements:
- Minimum karma to post
- Minimum account age
- Auto-approve vs manual review
- Rate limits per user

## Testing

```bash
# Run all tests
go test ./...

# Run feature tests
go test ./feature/...

# Run with coverage
go test -cover ./...

# Run specific feature
go test -v ./feature/threads
```

## Deployment

### Build

```bash
# Build optimized binary
CGO_ENABLED=1 go build -ldflags="-s -w" -o forum ./cmd/forum

# The binary embeds all assets
./forum serve
```

### Docker

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 go build -o forum ./cmd/forum

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/forum /usr/local/bin/
EXPOSE 8080
CMD ["forum", "serve"]
```

### Environment Variables

```bash
FORUM_HOST=0.0.0.0
FORUM_PORT=8080
FORUM_DB_PATH=/data/forum.db
FORUM_SITE_NAME="My Forum"
FORUM_SESSION_SECRET="random-secret-key"
```

## Contributing

See [spec/0124_blueprint_forum.md](../../spec/0124_blueprint_forum.md) for detailed specifications.

### Development

1. Clone the repository
2. Run `go mod download`
3. Initialize database: `go run ./cmd/forum init`
4. Seed sample data: `go run ./cmd/forum seed`
5. Start server: `go run ./cmd/forum serve`
6. Make changes and test
7. Run tests: `go test ./...`

## License

MIT License - see LICENSE file for details.

## Acknowledgments

Inspired by:
- **Reddit** - Voting, karma, subreddits
- **Discourse** - Trust levels, badges, modern UX
- **Stack Overflow** - Q&A format, best answers
- **phpBB/vBulletin** - Traditional forum structure
