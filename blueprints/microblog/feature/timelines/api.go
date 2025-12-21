// Package timelines provides timeline/feed generation.
package timelines

import (
	"context"

	"github.com/go-mizu/blueprints/microblog/feature/posts"
)

// API defines the timelines service contract.
type API interface {
	Home(ctx context.Context, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error)
	Local(ctx context.Context, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error)
	Hashtag(ctx context.Context, tag, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error)
	Account(ctx context.Context, accountID, viewerID string, limit int, maxID string, onlyMedia, excludeReplies bool) ([]*posts.Post, error)
	List(ctx context.Context, listID, viewerID string, limit int, maxID string) ([]*posts.Post, error)
	Bookmarks(ctx context.Context, accountID string, limit int, maxID string) ([]*posts.Post, error)
}

// Store defines the data access contract for timelines.
type Store interface {
	Home(ctx context.Context, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error)
	Local(ctx context.Context, limit int, maxID, sinceID string) ([]*posts.Post, error)
	Hashtag(ctx context.Context, tag string, limit int, maxID, sinceID string) ([]*posts.Post, error)
	Account(ctx context.Context, accountID, viewerID string, limit int, maxID string, onlyMedia, excludeReplies bool, isFollowing bool) ([]*posts.Post, error)
	List(ctx context.Context, listID string, limit int, maxID string) ([]*posts.Post, error)
	Bookmarks(ctx context.Context, accountID string, limit int, maxID string) ([]*posts.Post, error)
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)

	// Viewer state
	CheckLiked(ctx context.Context, accountID, postID string) (bool, error)
	CheckReposted(ctx context.Context, accountID, postID string) (bool, error)
	CheckBookmarked(ctx context.Context, accountID, postID string) (bool, error)
}
