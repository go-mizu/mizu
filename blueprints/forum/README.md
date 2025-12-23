# Forum Blueprint

A modern, full-featured discussion forum built with Mizu. Inspired by Reddit and Discourse, this blueprint demonstrates how to build a scalable community platform with nested discussions, voting, moderation, and rich user interactions.

## Features

### Core Functionality
- **Boards** - Topic-based communities (like subreddits or Discourse categories)
- **Threads** - Discussion starters with title, content, and optional media
- **Nested Comments** - Threaded replies with infinite depth (collapsible)
- **Voting System** - Upvotes/downvotes with karma tracking
- **User Accounts** - Registration, profiles, karma, badges

### Social Features
- **Bookmarks** - Save threads and comments for later
- **Following** - Follow boards, users, and threads
- **Notifications** - Replies, mentions, votes, moderation actions
- **User Profiles** - Post history, karma breakdown, badges

### Discovery
- **Search** - Full-text search across threads and comments
- **Tags** - Cross-board topic tagging
- **Trending** - Hot threads, rising posts, top of all time
- **Sorting** - Hot, New, Top (day/week/month/year/all), Controversial

### Moderation
- **Board Moderators** - Per-board moderation teams
- **Post Actions** - Remove, lock, pin, NSFW/spoiler tags
- **User Actions** - Ban (temp/permanent), mute, warn
- **Mod Queue** - Reported content review
- **Mod Log** - Audit trail of all moderation actions

## Architecture

```
forum/
├── cmd/forum/           # CLI entry point
├── cli/                 # Commands (serve, init, seed)
├── app/web/             # HTTP server and handlers
│   ├── server.go        # Server orchestration
│   ├── routes.go        # Route definitions
│   └── handler/         # HTTP handlers by feature
├── feature/             # Domain features
│   ├── accounts/        # User identity & auth
│   ├── boards/          # Board management
│   ├── threads/         # Thread posts
│   ├── comments/        # Nested comments
│   ├── votes/           # Voting system
│   ├── bookmarks/       # Saved content
│   ├── notifications/   # User notifications
│   ├── search/          # Full-text search
│   ├── moderation/      # Mod tools
│   └── tags/            # Topic tagging
├── store/duckdb/        # Database layer
├── assets/              # Embedded static files
│   ├── static/          # CSS, JS, icons
│   └── views/           # HTML templates
└── pkg/                 # Shared utilities
    ├── ulid/            # ID generation
    ├── password/        # Password hashing
    ├── text/            # Text processing
    └── markdown/        # Markdown rendering
```

## Data Models

### Account
```go
type Account struct {
    ID           string    // ULID
    Username     string    // Unique, 3-20 chars, alphanumeric + underscore
    Email        string    // Unique, for notifications
    DisplayName  string    // Optional display name
    Bio          string    // Profile bio (max 500 chars)
    AvatarURL    string    // Profile picture
    Karma        int64     // Total karma (upvotes - downvotes received)
    PostKarma    int64     // Karma from threads
    CommentKarma int64     // Karma from comments
    IsAdmin      bool      // Site-wide admin
    IsSuspended  bool      // Account suspended
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### Board
```go
type Board struct {
    ID          string    // ULID
    Name        string    // URL slug (3-21 chars, lowercase alphanumeric)
    Title       string    // Display title
    Description string    // Board description (max 500 chars)
    Sidebar     string    // Extended description/rules (Markdown)
    IconURL     string    // Board icon
    BannerURL   string    // Board banner
    IsNSFW      bool      // Adult content
    IsPrivate   bool      // Invite-only
    MemberCount int64     // Subscriber count
    CreatedAt   time.Time
    CreatedBy   string    // Account ID of creator
}
```

### Thread
```go
type Thread struct {
    ID           string       // ULID
    BoardID      string       // Parent board
    AuthorID     string       // Account ID
    Title        string       // Thread title (max 300 chars)
    Content      string       // Body text (Markdown, max 40000 chars)
    ContentHTML  string       // Rendered HTML
    URL          string       // Optional link post
    Type         ThreadType   // text, link, image, poll
    Score        int64        // Upvotes - Downvotes
    UpvoteCount  int64
    DownvoteCount int64
    CommentCount int64
    ViewCount    int64
    IsPinned     bool         // Sticky post
    IsLocked     bool         // No new comments
    IsRemoved    bool         // Mod removed
    IsNSFW       bool         // Adult content
    IsSpoiler    bool         // Spoiler tag
    CreatedAt    time.Time
    UpdatedAt    time.Time

    // Relationships
    Author   *Account
    Board    *Board
    Tags     []*Tag

    // Viewer state
    Vote       int  // -1, 0, 1
    IsBookmarked bool
    IsOwner    bool
}

type ThreadType string
const (
    ThreadTypeText  ThreadType = "text"
    ThreadTypeLink  ThreadType = "link"
    ThreadTypeImage ThreadType = "image"
    ThreadTypePoll  ThreadType = "poll"
)
```

### Comment
```go
type Comment struct {
    ID            string    // ULID
    ThreadID      string    // Parent thread
    ParentID      string    // Parent comment (empty for top-level)
    AuthorID      string    // Account ID
    Content       string    // Comment text (Markdown, max 10000 chars)
    ContentHTML   string    // Rendered HTML
    Score         int64     // Upvotes - Downvotes
    UpvoteCount   int64
    DownvoteCount int64
    Depth         int       // Nesting level (0 = top-level)
    Path          string    // Materialized path for tree queries
    IsRemoved     bool      // Mod removed
    IsDeleted     bool      // User deleted (content hidden)
    CreatedAt     time.Time
    UpdatedAt     time.Time

    // Relationships
    Author   *Account
    Thread   *Thread
    Parent   *Comment
    Children []*Comment

    // Viewer state
    Vote       int
    IsBookmarked bool
    IsOwner    bool
    IsCollapsed bool  // For deeply nested threads
}
```

### Vote
```go
type Vote struct {
    ID         string    // ULID
    AccountID  string
    TargetType string    // "thread" or "comment"
    TargetID   string    // Thread or Comment ID
    Value      int       // -1 (downvote) or 1 (upvote)
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### Notification
```go
type Notification struct {
    ID         string           // ULID
    AccountID  string           // Recipient
    Type       NotificationType
    ActorID    string           // Who triggered it
    ThreadID   string           // Related thread
    CommentID  string           // Related comment
    BoardID    string           // Related board
    Read       bool
    CreatedAt  time.Time

    // Relationships
    Actor   *Account
    Thread  *Thread
    Comment *Comment
    Board   *Board
}

type NotificationType string
const (
    NotificationReply        NotificationType = "reply"
    NotificationMention      NotificationType = "mention"
    NotificationThreadVote   NotificationType = "thread_vote"
    NotificationCommentVote  NotificationType = "comment_vote"
    NotificationFollow       NotificationType = "follow"
    NotificationModAction    NotificationType = "mod_action"
)
```

## API Endpoints

### Authentication
```
POST   /api/auth/register     # Create account
POST   /api/auth/login        # Login
POST   /api/auth/logout       # Logout
GET    /api/auth/me           # Current user
```

### Boards
```
GET    /api/boards            # List boards
POST   /api/boards            # Create board
GET    /api/boards/:name      # Get board
PUT    /api/boards/:name      # Update board
DELETE /api/boards/:name      # Delete board
POST   /api/boards/:name/join # Join board
DELETE /api/boards/:name/join # Leave board
GET    /api/boards/:name/moderators  # List mods
POST   /api/boards/:name/moderators  # Add mod
```

### Threads
```
GET    /api/threads                    # List (with filters)
POST   /api/boards/:name/threads       # Create thread
GET    /api/threads/:id                # Get thread
PUT    /api/threads/:id                # Update thread
DELETE /api/threads/:id                # Delete thread
POST   /api/threads/:id/vote           # Vote on thread
DELETE /api/threads/:id/vote           # Remove vote
POST   /api/threads/:id/bookmark       # Bookmark
DELETE /api/threads/:id/bookmark       # Remove bookmark
```

### Comments
```
GET    /api/threads/:id/comments       # List comments
POST   /api/threads/:id/comments       # Create comment
GET    /api/comments/:id               # Get comment
PUT    /api/comments/:id               # Update comment
DELETE /api/comments/:id               # Delete comment
POST   /api/comments/:id/vote          # Vote
DELETE /api/comments/:id/vote          # Remove vote
POST   /api/comments/:id/bookmark      # Bookmark
```

### Users
```
GET    /api/users/:username            # Get profile
GET    /api/users/:username/threads    # User's threads
GET    /api/users/:username/comments   # User's comments
POST   /api/users/:username/follow     # Follow user
DELETE /api/users/:username/follow     # Unfollow
```

### Search
```
GET    /api/search?q=...&type=threads  # Search
GET    /api/search?q=...&type=comments
GET    /api/search?q=...&type=boards
GET    /api/search?q=...&type=users
```

### Moderation
```
POST   /api/threads/:id/remove         # Remove thread
POST   /api/threads/:id/approve        # Approve thread
POST   /api/threads/:id/lock           # Lock thread
POST   /api/threads/:id/pin            # Pin thread
POST   /api/comments/:id/remove        # Remove comment
POST   /api/boards/:name/ban           # Ban user from board
GET    /api/boards/:name/modqueue      # Mod queue
GET    /api/boards/:name/modlog        # Mod log
```

## Sorting Algorithms

### Hot Score (Reddit-style)
```go
func HotScore(ups, downs int64, createdAt time.Time) float64 {
    score := float64(ups - downs)
    order := math.Log10(math.Max(math.Abs(score), 1))

    sign := 0.0
    if score > 0 {
        sign = 1
    } else if score < 0 {
        sign = -1
    }

    seconds := createdAt.Unix() - 1134028003 // Reddit epoch
    return sign*order + float64(seconds)/45000
}
```

### Controversial Score
```go
func ControversialScore(ups, downs int64) float64 {
    if ups <= 0 || downs <= 0 {
        return 0
    }
    magnitude := float64(ups + downs)
    balance := float64(min(ups, downs)) / float64(max(ups, downs))
    return magnitude * balance
}
```

### Wilson Score (for "Best" comments)
```go
func WilsonScore(ups, downs int64, confidence float64) float64 {
    n := float64(ups + downs)
    if n == 0 {
        return 0
    }
    z := 1.96 // 95% confidence
    phat := float64(ups) / n
    return (phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)) / (1 + z*z/n)
}
```

## UI Pages

### Public Pages
- `/` - Home feed (popular across all boards)
- `/all` - All posts (chronological)
- `/b/:name` - Board page with threads
- `/b/:name/submit` - Create thread form
- `/b/:name/:id/:slug` - Thread with comments
- `/u/:username` - User profile
- `/search` - Search results

### Authenticated Pages
- `/home` - Personal feed (subscribed boards)
- `/bookmarks` - Saved threads/comments
- `/notifications` - Notification center
- `/settings` - Account settings

### Moderation Pages
- `/b/:name/mod` - Board moderation dashboard
- `/b/:name/mod/queue` - Reported content queue
- `/b/:name/mod/log` - Moderation log
- `/b/:name/mod/banned` - Banned users
- `/b/:name/mod/settings` - Board settings

## Quick Start

```bash
# Build the forum
cd blueprints/forum
go build -o forum ./cmd/forum

# Initialize database
./forum init

# Seed sample data (optional)
./forum seed

# Start server
./forum serve --addr :8080

# Or use make from repo root
make -C blueprints/forum run
```

## Configuration

```go
type Config struct {
    Addr    string // Server address (default: ":8080")
    DataDir string // Data directory (default: "~/.forum")
    Dev     bool   // Development mode
}
```

Environment variables:
- `FORUM_ADDR` - Server address
- `FORUM_DATA_DIR` - Data directory
- `FORUM_DEV` - Enable development mode

## Database Schema

Uses DuckDB with the following tables:
- `accounts` - User accounts
- `sessions` - Auth sessions
- `boards` - Board definitions
- `board_members` - Board subscriptions
- `board_moderators` - Moderator assignments
- `threads` - Thread posts
- `comments` - Nested comments
- `votes` - Vote records
- `bookmarks` - Saved content
- `notifications` - User notifications
- `tags` - Tag definitions
- `thread_tags` - Thread-tag associations
- `mod_actions` - Moderation audit log
- `bans` - User bans

## Design Philosophy

1. **Performance First** - Efficient queries, materialized paths for comments
2. **Progressive Enhancement** - Works without JavaScript
3. **Mobile Responsive** - Touch-friendly, readable on all devices
4. **Accessible** - WCAG 2.1 AA compliant
5. **Modular** - Easy to add/remove features
6. **Secure** - CSRF protection, rate limiting, input validation

## License

MIT License - Same as Mizu framework
