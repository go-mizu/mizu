// Package trending provides trending content functionality.
package trending

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

// TrendingTag represents a trending hashtag.
type TrendingTag struct {
	Name          string       `json:"name"`
	URL           string       `json:"url,omitempty"`
	PostsCount    int          `json:"posts_count"`
	AccountsCount int          `json:"accounts_count,omitempty"`
	History       []DayHistory `json:"history,omitempty"`
}

// DayHistory represents usage stats for a day.
type DayHistory struct {
	Day      time.Time `json:"day"`
	Uses     int       `json:"uses"`
	Accounts int       `json:"accounts"`
}

// TrendingOpts specifies options for trending queries.
type TrendingOpts struct {
	Limit  int
	Offset int
}

// API defines the trending service contract.
type API interface {
	GetTrendingTags(ctx context.Context, opts TrendingOpts) ([]*TrendingTag, error)
	GetTrendingPosts(ctx context.Context, opts TrendingOpts) ([]*posts.Post, error)
	RefreshTrending(ctx context.Context) error
}

// Store defines the data access contract for trending.
type Store interface {
	GetTrendingTags(ctx context.Context, limit, offset int) ([]*TrendingTag, error)
	GetTrendingPosts(ctx context.Context, limit, offset int) ([]*posts.Post, error)
	ComputeTrendingTags(ctx context.Context, window time.Duration, limit int) ([]*TrendingTag, error)
	ComputeTrendingPosts(ctx context.Context, window time.Duration, limit int) ([]*posts.Post, error)
}
