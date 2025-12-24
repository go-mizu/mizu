package seed

import (
	"context"
	"time"
)

// FetchOpts contains options for fetching data.
type FetchOpts struct {
	Limit  int
	SortBy string // "top", "new", "best", "ask", "show"
}

// StoryData represents story data from any source.
type StoryData struct {
	ExternalID   string
	Title        string
	Content      string // Text for self posts
	URL          string // External URL for link posts
	Author       string
	Score        int64
	CommentCount int64
	CreatedAt    time.Time
	IsSelf       bool // True for text posts, false for link posts
	Domain       string
}

// CommentData represents comment data from any source.
type CommentData struct {
	ExternalID       string
	ExternalParentID string // Empty for top-level comments
	ExternalStoryID  string // The story this comment belongs to
	Author           string
	Content          string
	Score            int64
	CreatedAt        time.Time
	Depth            int
	Replies          []*CommentData // Nested replies
}

// Source represents a data source for seeding.
type Source interface {
	// Name returns the source identifier (e.g., "hn").
	Name() string

	// FetchStories fetches stories from the source.
	FetchStories(ctx context.Context, opts FetchOpts) ([]*StoryData, error)

	// FetchComments fetches comments for a story.
	FetchComments(ctx context.Context, storyID string) ([]*CommentData, error)
}
