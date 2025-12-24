// Package interactions provides like, repost, and bookmark functionality.
package interactions

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrAlreadyLiked      = errors.New("already liked")
	ErrNotLiked          = errors.New("not liked")
	ErrAlreadyReposted   = errors.New("already reposted")
	ErrNotReposted       = errors.New("not reposted")
	ErrAlreadyBookmarked = errors.New("already bookmarked")
	ErrNotBookmarked     = errors.New("not bookmarked")
	ErrCannotInteractOwn = errors.New("cannot interact with own post")
	ErrPostNotFound      = errors.New("post not found")
)

// Like represents a like on a post.
type Like struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Repost represents a repost of a post.
type Repost struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Bookmark represents a bookmark of a post.
type Bookmark struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PostState represents the viewer's interaction state with a post.
type PostState struct {
	Liked      bool `json:"liked"`
	Reposted   bool `json:"reposted"`
	Bookmarked bool `json:"bookmarked"`
}

// API defines the interactions service contract.
type API interface {
	Like(ctx context.Context, accountID, postID string) error
	Unlike(ctx context.Context, accountID, postID string) error
	GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)
	GetLikedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error)

	Repost(ctx context.Context, accountID, postID string) error
	Unrepost(ctx context.Context, accountID, postID string) error
	GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)

	Bookmark(ctx context.Context, accountID, postID string) error
	Unbookmark(ctx context.Context, accountID, postID string) error
	GetBookmarkedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error)

	GetPostState(ctx context.Context, accountID, postID string) (*PostState, error)
	GetPostStates(ctx context.Context, accountID string, postIDs []string) (map[string]*PostState, error)
}

// Store defines the data access contract for interactions.
type Store interface {
	// Likes
	InsertLike(ctx context.Context, l *Like) error
	DeleteLike(ctx context.Context, accountID, postID string) error
	ExistsLike(ctx context.Context, accountID, postID string) (bool, error)
	GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)
	GetLikedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error)
	IncrementLikesCount(ctx context.Context, postID string) error
	DecrementLikesCount(ctx context.Context, postID string) error

	// Reposts
	InsertRepost(ctx context.Context, r *Repost) error
	DeleteRepost(ctx context.Context, accountID, postID string) error
	ExistsRepost(ctx context.Context, accountID, postID string) (bool, error)
	GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error)
	IncrementRepostsCount(ctx context.Context, postID string) error
	DecrementRepostsCount(ctx context.Context, postID string) error

	// Bookmarks
	InsertBookmark(ctx context.Context, b *Bookmark) error
	DeleteBookmark(ctx context.Context, accountID, postID string) error
	ExistsBookmark(ctx context.Context, accountID, postID string) (bool, error)
	GetBookmarkedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error)

	// State queries
	GetPostState(ctx context.Context, accountID, postID string) (*PostState, error)
	GetPostStates(ctx context.Context, accountID string, postIDs []string) (map[string]*PostState, error)
}
