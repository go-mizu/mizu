package comments

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// Errors
var (
	ErrNotFound    = errors.New("comment not found")
	ErrInvalidText = errors.New("comment text is required")
	ErrTooDeep     = errors.New("comment too deeply nested")
)

// Validation constants
const (
	TextMinLen = 1
	TextMaxLen = 20000
	MaxDepth   = 10
)

// Comment represents a comment on a story.
type Comment struct {
	ID         string    `json:"id"`
	StoryID    string    `json:"story_id"`
	ParentID   string    `json:"parent_id,omitempty"`
	AuthorID   string    `json:"author_id"`
	Text       string    `json:"text"`
	TextHTML   string    `json:"text_html"`
	Score      int64     `json:"score"`
	Depth      int       `json:"depth"`
	Path       string    `json:"-"`
	ChildCount int64     `json:"child_count"`
	IsRemoved  bool      `json:"-"`
	CreatedAt  time.Time `json:"created_at"`

	// Joined fields
	Author   *users.User `json:"author,omitempty"`
	UserVote int         `json:"user_vote,omitempty"`
	Children []*Comment  `json:"children,omitempty"`
}

// ListIn contains options for listing comments.
type ListIn struct {
	StoryID  string
	ParentID string
	Sort     string // "best", "new", "old", "top"
	Limit    int
	Offset   int
}

// API defines the comments service interface.
type API interface {
	GetByID(ctx context.Context, id string, viewerID string) (*Comment, error)

	// Lists
	ListByStory(ctx context.Context, storyID string, viewerID string) ([]*Comment, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int, viewerID string) ([]*Comment, error)
}

// Store defines the data storage interface for comments.
type Store interface {
	GetByID(ctx context.Context, id string) (*Comment, error)

	// Create
	Create(ctx context.Context, comment *Comment) error
	IncrementChildCount(ctx context.Context, commentID string) error
	GetDepth(ctx context.Context, commentID string) (int, string, error)
	UpdateScore(ctx context.Context, commentID string, delta int) error

	// Lists
	ListByStory(ctx context.Context, storyID string) ([]*Comment, error)
	ListByParent(ctx context.Context, parentID string) ([]*Comment, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*Comment, error)
}
