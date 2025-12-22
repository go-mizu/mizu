// Package posts provides post/comment management functionality.
package posts

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/forum/feature/accounts"
)

var (
	// ErrNotFound is returned when a post is not found.
	ErrNotFound = errors.New("post not found")

	// ErrUnauthorized is returned when user lacks permission.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrThreadLocked is returned when thread is locked.
	ErrThreadLocked = errors.New("thread locked")

	// ErrMaxDepth is returned when max nesting depth is exceeded.
	ErrMaxDepth = errors.New("max depth exceeded")
)

const (
	// MaxDepth is the maximum nesting depth for posts.
	MaxDepth = 10
)

// Post represents a post/comment in a thread.
type Post struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	AccountID string    `json:"account_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Content   string    `json:"content"`
	Depth     int       `json:"depth"`
	Score     int       `json:"score"`
	Upvotes   int       `json:"upvotes"`
	Downvotes int       `json:"downvotes"`
	IsBest    bool      `json:"is_best"`
	Type      string    `json:"type"` // comment, best_answer, mod_note
	Path      string    `json:"path"` // Materialized path for tree queries
	CreatedAt time.Time `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`

	// Relationships
	Account  *accounts.Account `json:"account,omitempty"`
	Children []*Post           `json:"children,omitempty"`

	// Current user state
	UserVote int  `json:"user_vote,omitempty"` // -1, 0, 1
	IsSaved  bool `json:"is_saved"`
	IsOwner  bool `json:"is_owner"`
}

// CreateIn contains input for creating a post.
type CreateIn struct {
	ThreadID string `json:"thread_id"`
	ParentID string `json:"parent_id,omitempty"`
	Content  string `json:"content"`
}

// UpdateIn contains input for updating a post.
type UpdateIn struct {
	Content *string `json:"content,omitempty"`
}

// PostList is a list of posts.
type PostList struct {
	Posts []*Post `json:"posts"`
	Total int     `json:"total"`
}

// API defines the posts service contract.
type API interface {
	// Post operations
	Create(ctx context.Context, accountID string, in *CreateIn) (*Post, error)
	GetByID(ctx context.Context, id, viewerID string) (*Post, error)
	Update(ctx context.Context, id, accountID string, in *UpdateIn) (*Post, error)
	Delete(ctx context.Context, id, accountID string) error

	// Listing
	ListByThread(ctx context.Context, threadID, viewerID string, sort string, limit, offset int) (*PostList, error)
	ListByAccount(ctx context.Context, accountID, viewerID string, limit, offset int) (*PostList, error)
	GetTree(ctx context.Context, threadID, viewerID string) ([]*Post, error)
}

// Store defines the data access contract for posts.
type Store interface {
	// Post operations
	Insert(ctx context.Context, p *Post) error
	GetByID(ctx context.Context, id string) (*Post, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	GetOwner(ctx context.Context, id string) (accountID, threadID string, err error)
	GetDepth(ctx context.Context, parentID string) (int, error)

	// Listing
	ListByThread(ctx context.Context, threadID string, sort string, limit, offset int) ([]*Post, int, error)
	ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*Post, int, error)
	GetTree(ctx context.Context, threadID string) ([]*Post, error)
	GetChildren(ctx context.Context, postID string) ([]*Post, error)

	// Scores
	UpdateScores(ctx context.Context, postID string, score, upvotes, downvotes int) error
}
