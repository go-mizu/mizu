// Package trending provides trending topics and posts calculation.
package trending

import (
	"context"
)

// TrendingTag represents a trending hashtag.
type TrendingTag struct {
	Name       string `json:"name"`
	PostsCount int    `json:"posts_count"`
	Accounts   int    `json:"accounts,omitempty"` // Unique accounts using this tag
}

// API defines the trending service contract.
type API interface {
	Tags(ctx context.Context, limit int) ([]*TrendingTag, error)
	Posts(ctx context.Context, limit int) ([]string, error)
	SuggestedAccounts(ctx context.Context, accountID string, limit int) ([]string, error)
}

// Store defines the data access contract for trending.
type Store interface {
	Tags(ctx context.Context, limit int) ([]*TrendingTag, error)
	Posts(ctx context.Context, limit int) ([]string, error)
	SuggestedAccounts(ctx context.Context, accountID string, limit int) ([]string, error)
}
