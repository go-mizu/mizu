package seed

import (
	"context"
	"time"
)

// FetchOpts contains options for fetching data.
type FetchOpts struct {
	Limit     int
	After     string // Pagination cursor
	SortBy    string // e.g., "hot", "new", "top"
	TimeRange string // For "top": "hour", "day", "week", "month", "year", "all"
}

// ThreadListResult contains threads with pagination info for resumable crawling.
type ThreadListResult struct {
	Threads []*ThreadData
	After   string // Next page cursor
	HasMore bool
}

// CommentOpts contains options for fetching comments.
type CommentOpts struct {
	Limit int
	Depth int
	Sort  string // "best", "top", "new", "controversial", "old"
}

// ListSubredditsOpts contains options for listing subreddits.
type ListSubredditsOpts struct {
	Where string // "popular", "new", "default"
	Limit int
	After string // Pagination cursor
}

// SubredditListResult contains subreddits with pagination info.
type SubredditListResult struct {
	Subreddits []*SubredditData
	After      string
	HasMore    bool
}

// SubredditData represents subreddit metadata from any source.
type SubredditData struct {
	Name        string
	Title       string
	Description string
	Subscribers int64
}

// ThreadData represents thread data from any source.
type ThreadData struct {
	ExternalID   string
	Title        string
	Content      string // Self text for text posts
	URL          string // External URL for link posts
	Author       string
	Score        int64
	UpvoteCount  int64
	DownvoteCount int64
	CommentCount int64
	CreatedAt    time.Time
	IsNSFW       bool
	IsSpoiler    bool
	IsSelf       bool // True for text posts, false for link posts
	Domain       string
	Permalink    string
}

// CommentData represents comment data from any source.
type CommentData struct {
	ExternalID       string
	ExternalParentID string // Empty for top-level comments
	ExternalThreadID string // The thread this comment belongs to
	Author           string
	Content          string
	Score            int64
	UpvoteCount      int64
	DownvoteCount    int64
	CreatedAt        time.Time
	Depth            int
	Replies          []*CommentData // Nested replies
}

// Source represents a data source for seeding.
type Source interface {
	// Name returns the source identifier (e.g., "reddit").
	Name() string

	// FetchSubreddit fetches metadata for a subreddit.
	FetchSubreddit(ctx context.Context, name string) (*SubredditData, error)

	// FetchThreads fetches threads from a subreddit.
	FetchThreads(ctx context.Context, subreddit string, opts FetchOpts) ([]*ThreadData, error)

	// FetchComments fetches comments for a thread.
	FetchComments(ctx context.Context, subreddit, threadID string) ([]*CommentData, error)
}
