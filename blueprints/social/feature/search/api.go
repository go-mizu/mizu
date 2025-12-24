// Package search provides search functionality.
package search

import (
	"context"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
)

// SearchType represents the type of search result.
const (
	TypeAccounts = "accounts"
	TypePosts    = "posts"
	TypeHashtags = "hashtags"
)

// Hashtag represents a hashtag search result.
type Hashtag struct {
	Name       string `json:"name"`
	URL        string `json:"url,omitempty"`
	PostsCount int    `json:"posts_count"`
}

// SearchResult contains search results.
type SearchResult struct {
	Accounts []*accounts.Account `json:"accounts"`
	Posts    []*posts.Post       `json:"posts"`
	Hashtags []*Hashtag          `json:"hashtags"`
}

// SearchOpts specifies search options.
type SearchOpts struct {
	Query      string
	Type       string // accounts, posts, hashtags, or empty for all
	Limit      int
	Offset     int
	AccountID  string // For filtering "from:user" searches
	Following  bool   // Only search accounts user follows
	Resolve    bool   // Attempt to resolve remote accounts
	MinLikes   int    // Minimum likes filter
	MinReposts int    // Minimum reposts filter
	HasMedia   bool   // Only posts with media
}

// API defines the search service contract.
type API interface {
	Search(ctx context.Context, opts SearchOpts) (*SearchResult, error)
	SearchAccounts(ctx context.Context, query string, limit, offset int, viewerID string) ([]*accounts.Account, error)
	SearchPosts(ctx context.Context, query string, limit, offset int, viewerID string) ([]*posts.Post, error)
	SearchHashtags(ctx context.Context, query string, limit int) ([]*Hashtag, error)
	Suggest(ctx context.Context, query string, limit int) ([]string, error)
}

// Store defines the data access contract for search.
type Store interface {
	SearchAccounts(ctx context.Context, query string, limit, offset int) ([]*accounts.Account, error)
	SearchPosts(ctx context.Context, query string, limit, offset int, minLikes, minReposts int, hasMedia bool) ([]*posts.Post, error)
	SearchHashtags(ctx context.Context, query string, limit int) ([]*Hashtag, error)
	SuggestHashtags(ctx context.Context, prefix string, limit int) ([]string, error)
}
