// Package timelines provides timeline generation.
package timelines

import (
	"context"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

// TimelineOpts specifies options for fetching timelines.
type TimelineOpts struct {
	Limit     int
	MaxID     string // Posts older than this ID
	MinID     string // Posts newer than this ID
	SinceID   string // Posts newer than this ID (exclusive)
	OnlyMedia bool
}

// API defines the timelines service contract.
type API interface {
	Home(ctx context.Context, accountID string, opts TimelineOpts) ([]*posts.Post, error)
	Public(ctx context.Context, opts TimelineOpts) ([]*posts.Post, error)
	User(ctx context.Context, userID string, opts TimelineOpts, includeReplies bool) ([]*posts.Post, error)
	Hashtag(ctx context.Context, tag string, opts TimelineOpts) ([]*posts.Post, error)
	List(ctx context.Context, accountID, listID string, opts TimelineOpts) ([]*posts.Post, error)
	Bookmarks(ctx context.Context, accountID string, opts TimelineOpts) ([]*posts.Post, error)
	Likes(ctx context.Context, accountID string, opts TimelineOpts) ([]*posts.Post, error)
}

// Store defines the data access contract for timelines.
type Store interface {
	GetHomeFeed(ctx context.Context, accountID string, limit int, maxID, minID string) ([]*posts.Post, error)
	GetPublicFeed(ctx context.Context, limit int, maxID, minID string, onlyMedia bool) ([]*posts.Post, error)
	GetUserFeed(ctx context.Context, userID string, limit int, maxID, minID string, includeReplies, onlyMedia bool) ([]*posts.Post, error)
	GetHashtagFeed(ctx context.Context, tag string, limit int, maxID, minID string) ([]*posts.Post, error)
	GetListFeed(ctx context.Context, listID string, limit int, maxID, minID string) ([]*posts.Post, error)
	GetBookmarksFeed(ctx context.Context, accountID string, limit int, maxID, minID string) ([]*posts.Post, error)
	GetLikesFeed(ctx context.Context, accountID string, limit int, maxID, minID string) ([]*posts.Post, error)
}
