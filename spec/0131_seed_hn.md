# 0131: Enhanced Seed Package - Reddit Improvements & Hacker News Support

## Overview

This spec covers two major enhancements to the forum seed package:
1. **Reddit Enhancements**: Extended crawling options including subreddit listing, resumable pagination, and improved comment fetching
2. **Hacker News Support**: New source implementation with multiple crawling modes and deduplication

## Goals

- Provide comprehensive crawling capabilities for both Reddit and Hacker News
- Support resumable operations for large-scale seeding
- Implement smart deduplication to avoid re-crawling existing items
- Maintain consistency with existing source interface patterns

---

## Part 1: Reddit Enhancements

### 1.1 New Client Methods

```go
// ListSubreddits fetches popular/trending subreddits
// Endpoint: /subreddits/popular.json or /subreddits/{where}.json
func (c *Client) ListSubreddits(ctx context.Context, opts ListSubredditsOpts) (*SubredditListResult, error)

// FetchThreadsWithCursor returns threads with pagination cursor for resumable crawling
func (c *Client) FetchThreadsWithCursor(ctx context.Context, subreddit string, opts FetchOpts) (*ThreadListResult, error)

// FetchAllComments fetches all comments for a thread with depth control
func (c *Client) FetchAllComments(ctx context.Context, subreddit, threadID string, opts CommentOpts) ([]*seed.CommentData, error)
```

### 1.2 New Types

```go
type ListSubredditsOpts struct {
    Where string // "popular", "new", "default"
    Limit int
    After string // Pagination cursor
}

type SubredditListResult struct {
    Subreddits []*seed.SubredditData
    After      string // Next page cursor
    HasMore    bool
}

type ThreadListResult struct {
    Threads []*seed.ThreadData
    After   string // Next page cursor
    HasMore bool
}

type CommentOpts struct {
    Limit     int
    Depth     int
    Sort      string // "best", "top", "new", "controversial", "old"
}
```

### 1.3 Enhanced FetchOpts

```go
type FetchOpts struct {
    Limit   int
    After   string   // Pagination cursor
    SortBy  string   // "hot", "new", "top", "rising"
    TimeRange string // For "top": "hour", "day", "week", "month", "year", "all"
}
```

### 1.4 CLI Enhancements

New flags for `forum seed reddit`:
- `--sort`: Sort order (hot, new, top, rising)
- `--time`: Time range for top posts (hour, day, week, month, year, all)
- `--comment-sort`: Comment sort order
- `--resume-from`: Resume from cursor position
- `--list-subreddits`: List available subreddits instead of seeding

---

## Part 2: Hacker News Implementation

### 2.1 HN API Overview

Base URL: `https://hacker-news.firebaseio.com/v0/`

Key endpoints:
- `/item/{id}.json` - Individual item (story, comment, job, poll)
- `/user/{id}.json` - User profile
- `/topstories.json` - Top 500 story IDs
- `/newstories.json` - New 500 story IDs
- `/beststories.json` - Best story IDs
- `/askstories.json` - Ask HN stories (200)
- `/showstories.json` - Show HN stories (200)
- `/jobstories.json` - Job stories (200)
- `/maxitem` - Current max item ID

### 2.2 HN Types (`pkg/seed/hn/types.go`)

```go
package hn

import "time"

// ItemType represents the type of HN item
type ItemType string

const (
    ItemTypeStory   ItemType = "story"
    ItemTypeComment ItemType = "comment"
    ItemTypeJob     ItemType = "job"
    ItemTypePoll    ItemType = "poll"
    ItemTypePollOpt ItemType = "pollopt"
)

// Item represents a Hacker News item
type Item struct {
    ID          int      `json:"id"`
    Type        ItemType `json:"type"`
    By          string   `json:"by"`
    Time        int64    `json:"time"`
    Text        string   `json:"text"`        // HTML content
    Title       string   `json:"title"`       // Story/job title
    URL         string   `json:"url"`         // Story URL
    Score       int      `json:"score"`
    Kids        []int    `json:"kids"`        // Child comment IDs
    Parent      int      `json:"parent"`      // Parent story/comment ID
    Poll        int      `json:"poll"`        // Associated poll
    Parts       []int    `json:"parts"`       // Poll options
    Descendants int      `json:"descendants"` // Total comment count
    Deleted     bool     `json:"deleted"`
    Dead        bool     `json:"dead"`
}

// CreatedTime returns the creation time as time.Time
func (i *Item) CreatedTime() time.Time {
    return time.Unix(i.Time, 0)
}

// IsDeleted returns true if the item is deleted or dead
func (i *Item) IsDeleted() bool {
    return i.Deleted || i.Dead
}

// User represents a Hacker News user profile
type User struct {
    ID        string `json:"id"`
    Created   int64  `json:"created"`
    Karma     int    `json:"karma"`
    About     string `json:"about"`
    Submitted []int  `json:"submitted"`
}

// FeedType represents different HN feeds
type FeedType string

const (
    FeedTop  FeedType = "top"
    FeedNew  FeedType = "new"
    FeedBest FeedType = "best"
    FeedAsk  FeedType = "ask"
    FeedShow FeedType = "show"
    FeedJobs FeedType = "jobs"
)
```

### 2.3 HN Client (`pkg/seed/hn/client.go`)

```go
package hn

import (
    "context"
    "encoding/json"
    "fmt"
    "html"
    "net/http"
    "regexp"
    "strings"
    "sync"
    "time"

    "github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
)

const (
    baseURL    = "https://hacker-news.firebaseio.com/v0"
    defaultUA  = "ForumSeeder/1.0"
    maxWorkers = 10 // Concurrent item fetches
)

type Client struct {
    httpClient  *http.Client
    userAgent   string
    rateLimit   time.Duration
    lastReq     time.Time
    mu          sync.Mutex
    concurrency int
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        userAgent:   defaultUA,
        rateLimit:   100 * time.Millisecond, // HN has no rate limit but be polite
        concurrency: maxWorkers,
    }
}

// Name returns the source identifier
func (c *Client) Name() string {
    return "hn"
}

// FetchFeed fetches story IDs from a specific feed
func (c *Client) FetchFeed(ctx context.Context, feed FeedType) ([]int, error)

// FetchItem fetches a single item by ID
func (c *Client) FetchItem(ctx context.Context, id int) (*Item, error)

// FetchItems fetches multiple items concurrently
func (c *Client) FetchItems(ctx context.Context, ids []int) ([]*Item, error)

// FetchUser fetches a user profile
func (c *Client) FetchUser(ctx context.Context, id string) (*User, error)

// FetchMaxItem returns the current maximum item ID
func (c *Client) FetchMaxItem(ctx context.Context) (int, error)

// FetchItemsSince fetches items from startID to maxItem
func (c *Client) FetchItemsSince(ctx context.Context, startID int, limit int) ([]*Item, int, error)
```

### 2.4 Source Interface Implementation

The HN client implements `seed.Source` with HN-specific adaptations:

```go
// FetchSubreddit returns HN as a single "board"
func (c *Client) FetchSubreddit(ctx context.Context, name string) (*seed.SubredditData, error) {
    // Returns static HN metadata
    return &seed.SubredditData{
        Name:        "hackernews",
        Title:       "Hacker News",
        Description: "Social news website focusing on computer science and entrepreneurship",
        Subscribers: 0, // Not applicable
    }, nil
}

// FetchThreads fetches stories based on options
func (c *Client) FetchThreads(ctx context.Context, _ string, opts seed.FetchOpts) ([]*seed.ThreadData, error) {
    // Use opts.SortBy to determine feed type
    // "hot"/"top" -> topstories, "new" -> newstories, etc.
}

// FetchComments fetches comments for a story
func (c *Client) FetchComments(ctx context.Context, _, threadID string) ([]*seed.CommentData, error) {
    // Fetches comment tree recursively
}
```

### 2.5 HN-Specific Fetch Options

```go
type HNFetchOpts struct {
    Feed        FeedType // top, new, best, ask, show, jobs
    Limit       int      // Max stories to fetch
    FromItemID  int      // Start from specific item ID (for resumable)
    SkipExisting bool    // Skip items that already exist in seed_mappings
    Force       bool     // Force re-fetch even if exists
}
```

### 2.6 Deduplication Strategy

```go
// CheckExists checks if items already exist in seed_mappings
func (s *Seeder) CheckExists(ctx context.Context, source string, externalIDs []string) (map[string]bool, error)

// FilterNewItems returns only items not already seeded
func (s *Seeder) FilterNewItems(ctx context.Context, source string, items []*seed.ThreadData) ([]*seed.ThreadData, error)
```

### 2.7 CLI Command (`cli/seed_hn.go`)

```go
// NewSeedHN creates the seed hn command
func NewSeedHN() *cobra.Command

// Flags:
// --feed: Feed type (top, new, best, ask, show, jobs) [default: top]
// --limit: Number of stories to fetch [default: 25]
// --from-id: Start from specific item ID
// --with-comments: Also fetch comments
// --comment-depth: Max comment depth [default: 5]
// --skip-existing: Skip already seeded items [default: true]
// --force: Force re-fetch existing items
// --dry-run: Preview without making changes
```

---

## Part 3: Implementation Plan

### Phase 1: Reddit Enhancements

1. **Extend FetchOpts** in `source.go`:
   - Add TimeRange field
   - Ensure backward compatibility

2. **Update reddit/client.go**:
   - Add `ListSubreddits()` method
   - Add `FetchThreadsWithCursor()` returning pagination info
   - Enhance `FetchComments()` with sort options
   - Update `FetchThreads()` to support sort and time range

3. **Update reddit/types.go**:
   - Add `SubredditListResult` and `ThreadListResult` types
   - Add `ListSubredditsOpts` and `CommentOpts` types

4. **Update cli/seed_reddit.go**:
   - Add new flags for sort, time range, resume

5. **Write tests**:
   - Test list subreddits
   - Test pagination with cursor
   - Test sort options

### Phase 2: Hacker News Implementation

1. **Create pkg/seed/hn/types.go**:
   - Define Item, User, ItemType, FeedType

2. **Create pkg/seed/hn/client.go**:
   - Implement HN API client
   - Implement seed.Source interface
   - Add concurrent item fetching

3. **Update pkg/seed/seed.go**:
   - Add HN-specific seeding logic
   - Update username normalization for HN
   - Add batch existence check for dedup

4. **Create cli/seed_hn.go**:
   - Implement CLI command with all flags
   - Add progress reporting

5. **Update cli/seed.go**:
   - Add HN subcommand

6. **Write tests**:
   - Client tests for all endpoints
   - Integration tests for seeding
   - Idempotency tests

### Phase 3: Testing & Verification

1. Run all existing tests to ensure no regression
2. Test Reddit enhancements with live API
3. Test HN client with live API
4. Test full seeding workflow for both sources
5. Verify deduplication works correctly

---

## Part 4: File Structure

```
pkg/seed/
├── seed.go          # Core seeder (updated for HN)
├── source.go        # Source interface (updated with new types)
├── reddit/
│   ├── client.go    # Reddit client (enhanced)
│   ├── types.go     # Reddit types (extended)
│   ├── client_test.go
│   └── seeder_test.go
└── hn/
    ├── client.go    # HN client (new)
    ├── types.go     # HN types (new)
    ├── client_test.go (new)
    └── seeder_test.go (new)

cli/
├── seed.go          # Seed command (add hn subcommand)
├── seed_reddit.go   # Reddit command (enhanced flags)
├── seed_hn.go       # HN command (new)
└── seed_test.go     # Tests
```

---

## Part 5: API Reference

### Reddit API Endpoints Used

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/r/{sub}/about.json` | GET | Subreddit metadata |
| `/r/{sub}.json` | GET | Thread listing |
| `/r/{sub}/comments/{id}.json` | GET | Thread comments |
| `/subreddits/{where}.json` | GET | Subreddit listing |

### Hacker News API Endpoints Used

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/topstories.json` | GET | Top 500 story IDs |
| `/newstories.json` | GET | Newest 500 story IDs |
| `/beststories.json` | GET | Best story IDs |
| `/askstories.json` | GET | Ask HN story IDs |
| `/showstories.json` | GET | Show HN story IDs |
| `/jobstories.json` | GET | Job story IDs |
| `/item/{id}.json` | GET | Single item |
| `/user/{id}.json` | GET | User profile |
| `/maxitem` | GET | Max item ID |

---

## Part 6: Usage Examples

### Reddit

```bash
# Basic seeding
forum seed reddit --subreddits golang,programming --limit 25

# With comments and sorting
forum seed reddit --subreddits golang --limit 50 --with-comments --sort top --time week

# Resume from cursor
forum seed reddit --subreddits golang --limit 100 --resume-from "t3_abc123"

# List available subreddits
forum seed reddit --list-subreddits --limit 20
```

### Hacker News

```bash
# Seed top stories
forum seed hn --feed top --limit 25

# Seed with comments
forum seed hn --feed best --limit 50 --with-comments

# Seed new stories, skip existing
forum seed hn --feed new --limit 100 --skip-existing

# Resume from item ID
forum seed hn --from-id 12345678 --limit 100

# Force re-seed existing items
forum seed hn --feed top --limit 25 --force

# Dry run to preview
forum seed hn --feed top --limit 10 --dry-run
```

---

## Part 7: Error Handling

### Reddit
- Rate limiting (429): Exponential backoff with max 3 retries
- Not found (404): Skip and continue
- Server errors (5xx): Retry with backoff

### Hacker News
- Deleted/dead items: Skip silently
- Missing items: Log warning and continue
- Network errors: Retry with exponential backoff

### Common
- Database errors: Fail fast with clear error message
- Mapping conflicts: Use existing mapping, don't overwrite
- Invalid data: Log and skip, continue with remaining items

---

## Part 8: Performance Considerations

### Reddit
- Rate limit: 2 seconds between requests (Reddit requirement)
- Batch size: Max 100 items per request

### Hacker News
- No official rate limit, but use 100ms delay between requests
- Concurrent fetching: Up to 10 parallel item requests
- Batch comment fetching to reduce API calls

### Database
- Use transactions for batch inserts when possible
- Cache user and board lookups during a seed run
- Batch existence checks for deduplication
