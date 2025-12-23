# Spec 0129: Reddit Seed Implementation

## Overview

Implement `pkg/seed` and `pkg/seed/reddit` packages for seeding real data from Reddit to the forum. The seeding must be idempotent (safe to run multiple times without duplicating data).

## Goals

1. Define a `store.Store` interface for abstraction (allowing different storage backends)
2. Create a reusable `pkg/seed` base package with common seeding infrastructure
3. Implement `pkg/seed/reddit` for fetching and importing Reddit data
4. Ensure idempotent seeding using Reddit IDs as external references
5. Full test coverage with real data from `/r/programming` and `/r/golang`

## Architecture

### Store Interface (`store/store.go`)

```go
// Store provides access to all feature stores
type Store interface {
    Accounts() accounts.Store
    Boards() boards.Store
    Threads() threads.Store
    Comments() comments.Store
    Votes() votes.Store
    Bookmarks() bookmarks.Store
    Notifications() notifications.Store
    Close() error
}
```

### Seed Package Structure

```
pkg/seed/
├── seed.go           # Core seeding infrastructure
├── source.go         # Source interface for data providers
└── reddit/
    ├── client.go     # Reddit API client
    ├── types.go      # Reddit JSON response types
    ├── seeder.go     # Reddit-to-Forum seeder
    └── seeder_test.go
```

### Core Interfaces

```go
// pkg/seed/source.go

// Source represents a data source for seeding
type Source interface {
    Name() string
    // FetchBoards returns boards to seed
    FetchBoards(ctx context.Context, opts FetchOpts) ([]BoardData, error)
    // FetchThreads returns threads for a board
    FetchThreads(ctx context.Context, boardName string, opts FetchOpts) ([]ThreadData, error)
    // FetchComments returns comments for a thread
    FetchComments(ctx context.Context, threadID string, opts FetchOpts) ([]CommentData, error)
}

// BoardData represents board data from any source
type BoardData struct {
    ExternalID  string
    Name        string
    Title       string
    Description string
}

// ThreadData represents thread data from any source
type ThreadData struct {
    ExternalID   string
    Title        string
    Content      string
    URL          string
    Author       string
    Score        int64
    CommentCount int64
    CreatedAt    time.Time
    IsNSFW       bool
    IsSpoiler    bool
}

// CommentData represents comment data from any source
type CommentData struct {
    ExternalID       string
    ExternalParentID string // Empty for top-level
    Author           string
    Content          string
    Score            int64
    CreatedAt        time.Time
    Depth            int
}
```

### Seeder Implementation

```go
// pkg/seed/seed.go

// Seeder handles idempotent seeding from external sources
type Seeder struct {
    accounts accounts.API
    boards   boards.API
    threads  threads.API
    comments comments.API

    // Track external ID mappings for idempotency
    // Uses a simple metadata table or naming convention
}

// SeedResult contains statistics from a seed operation
type SeedResult struct {
    BoardsCreated   int
    BoardsSkipped   int
    ThreadsCreated  int
    ThreadsSkipped  int
    CommentsCreated int
    CommentsSkipped int
    UsersCreated    int
    UsersSkipped    int
}

func (s *Seeder) SeedFromSource(ctx context.Context, source Source, opts SeedOpts) (*SeedResult, error)
```

### Reddit Client

```go
// pkg/seed/reddit/client.go

// Client fetches data from Reddit's JSON API
type Client struct {
    httpClient *http.Client
    userAgent  string
    rateLimit  time.Duration
}

// FetchSubreddit fetches posts from a subreddit
func (c *Client) FetchSubreddit(ctx context.Context, name string, opts FetchOpts) (*SubredditResponse, error)

// FetchComments fetches comments for a post
func (c *Client) FetchComments(ctx context.Context, subreddit, postID string) (*CommentsResponse, error)
```

## Idempotency Strategy

1. **Username-based deduplication**: Reddit usernames become forum usernames with `reddit_` prefix
   - Example: Reddit user `spez` → Forum user `reddit_spez`
   - On re-seed, existing users are looked up by username

2. **Board name matching**: Subreddits map directly to board names
   - Example: `/r/golang` → board `golang`
   - On re-seed, existing boards are looked up by name

3. **External ID tracking**: Threads and comments use Reddit's ID stored in metadata
   - Store Reddit post ID (e.g., `1pbdqkn`) in a way that allows lookup
   - Options:
     a. Add `external_id` column to threads/comments tables
     b. Use a separate `seed_mappings` table
     c. Store in thread URL field for link posts
   - On re-seed, skip if external ID already exists

For simplicity, we'll use option (b): a `seed_mappings` table that maps `(source, external_id)` → `local_id`.

### Schema Addition

```sql
CREATE TABLE IF NOT EXISTS seed_mappings (
    source TEXT NOT NULL,           -- e.g., 'reddit'
    entity_type TEXT NOT NULL,      -- 'thread', 'comment', 'account'
    external_id TEXT NOT NULL,      -- Reddit's ID
    local_id TEXT NOT NULL,         -- Our ULID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source, entity_type, external_id)
);
```

## Reddit API Details

### Endpoints Used

1. **Subreddit listing**: `https://www.reddit.com/r/{subreddit}.json?limit=N`
2. **Post comments**: `https://www.reddit.com/r/{subreddit}/comments/{id}.json`
3. **Subreddit about**: `https://www.reddit.com/r/{subreddit}/about.json`

### Rate Limiting

- Reddit requires rate limiting (~1 request per 2 seconds for unauthenticated)
- Include proper User-Agent header
- Handle 429 responses with exponential backoff

### Data Mapping

| Reddit Field | Forum Field |
|--------------|-------------|
| `data.title` | `Thread.Title` |
| `data.selftext` | `Thread.Content` (for self posts) |
| `data.url` | `Thread.URL` (for link posts) |
| `data.author` | Create/lookup account |
| `data.score` | `Thread.Score` |
| `data.ups` | `Thread.UpvoteCount` |
| `data.downs` | `Thread.DownvoteCount` |
| `data.created_utc` | `Thread.CreatedAt` |
| `data.num_comments` | `Thread.CommentCount` |
| `data.over_18` | `Thread.IsNSFW` |
| `data.spoiler` | `Thread.IsSpoiler` |
| `data.is_self` | Determines `Thread.Type` |

## Implementation Steps

### Phase 1: Store Interface

1. Create `store/store.go` with `Store` interface
2. Update `store/duckdb/store.go` to implement the interface
3. Add `seed_mappings` table to schema

### Phase 2: Seed Package

1. Create `pkg/seed/source.go` with data types and `Source` interface
2. Create `pkg/seed/seed.go` with `Seeder` struct and idempotent seeding logic
3. Create `pkg/seed/mapping.go` for seed mappings store

### Phase 3: Reddit Implementation

1. Create `pkg/seed/reddit/types.go` with Reddit API response types
2. Create `pkg/seed/reddit/client.go` with HTTP client and rate limiting
3. Create `pkg/seed/reddit/seeder.go` implementing `Source` interface

### Phase 4: Testing

1. Create `pkg/seed/reddit/seeder_test.go`
2. Test with real Reddit data from `/r/programming` and `/r/golang`
3. Test idempotency (run twice, verify no duplicates)

### Phase 5: CLI Integration

1. Update `cli/seed.go` to add `--reddit` flag
2. Add flags for subreddits, limits, etc.

## Test Plan

```go
func TestRedditSeeder_Programming(t *testing.T) {
    // Fetch 10 posts from /r/programming
    // Verify threads created with correct data
    // Verify comments nested correctly
}

func TestRedditSeeder_Golang(t *testing.T) {
    // Fetch 10 posts from /r/golang
    // Verify threads created with correct data
}

func TestRedditSeeder_Idempotent(t *testing.T) {
    // Seed once
    // Count records
    // Seed again with same data
    // Verify same record count (no duplicates)
}
```

## CLI Usage

```bash
# Seed from specific subreddits
forum seed --reddit golang,programming --limit 25

# Seed with comments
forum seed --reddit golang --limit 10 --with-comments

# Dry run (show what would be seeded)
forum seed --reddit golang --dry-run
```

## Dependencies

- `net/http` for Reddit API calls
- `encoding/json` for parsing responses
- No external dependencies required

## Notes

- Reddit's API returns `[deleted]` for deleted user content - skip these
- Some posts are removed by moderators - skip posts where `data.removed_by_category` is set
- Comments have nested replies in the `data.replies` field (recursive structure)
- The `depth` field in comments indicates nesting level
