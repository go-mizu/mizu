package comments

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
)

// Errors
var (
	ErrNotFound       = errors.New("comment not found")
	ErrThreadLocked   = errors.New("thread is locked")
	ErrThreadRemoved  = errors.New("thread is removed")
	ErrCommentRemoved = errors.New("comment is removed")
	ErrNotAuthor      = errors.New("not the comment author")
	ErrMaxDepth       = errors.New("maximum nesting depth reached")
)

// Validation constants
const (
	ContentMaxLen     = 10000
	MaxDepth          = 10
	DefaultCollapseAt = 5
)

// CommentSort defines sorting options for comments.
type CommentSort string

const (
	CommentSortBest          CommentSort = "best"
	CommentSortTop           CommentSort = "top"
	CommentSortNew           CommentSort = "new"
	CommentSortOld           CommentSort = "old"
	CommentSortControversial CommentSort = "controversial"
	CommentSortQA            CommentSort = "qa"
)

// Comment represents a comment on a thread.
type Comment struct {
	ID            string     `json:"id"`
	ThreadID      string     `json:"thread_id"`
	ParentID      string     `json:"parent_id,omitempty"`
	AuthorID      string     `json:"author_id"`
	Content       string     `json:"content"`
	ContentHTML   string     `json:"content_html"`
	Score         int64      `json:"score"`
	UpvoteCount   int64      `json:"upvote_count"`
	DownvoteCount int64      `json:"downvote_count"`
	Depth         int        `json:"depth"`
	Path          string     `json:"path"`
	ChildCount    int64      `json:"child_count"`
	IsRemoved     bool       `json:"is_removed"`
	IsDeleted     bool       `json:"is_deleted"`
	RemoveReason  string     `json:"remove_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	EditedAt      *time.Time `json:"edited_at,omitempty"`

	// Relationships
	Author   *accounts.Account `json:"author,omitempty"`
	Parent   *Comment          `json:"parent,omitempty"`
	Children []*Comment        `json:"children,omitempty"`

	// Viewer state
	Vote         int  `json:"vote,omitempty"`
	IsBookmarked bool `json:"is_bookmarked,omitempty"`
	IsOwner      bool `json:"is_owner,omitempty"`
	IsCollapsed  bool `json:"is_collapsed,omitempty"`
	CanEdit      bool `json:"can_edit,omitempty"`
	CanDelete    bool `json:"can_delete,omitempty"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	ThreadID string `json:"thread_id"`
	ParentID string `json:"parent_id,omitempty"`
	Content  string `json:"content"`
}

// ListOpts contains options for listing comments.
type ListOpts struct {
	Limit  int
	Cursor string
	SortBy CommentSort
}

// TreeOpts contains options for building comment trees.
type TreeOpts struct {
	Sort       CommentSort
	Limit      int
	MaxDepth   int
	CollapseAt int
}

// API defines the comments service interface.
type API interface {
	// Comment management
	Create(ctx context.Context, authorID string, in CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, content string) (*Comment, error)
	Delete(ctx context.Context, id string) error

	// Listing
	ListByThread(ctx context.Context, threadID string, opts ListOpts) ([]*Comment, error)
	ListByParent(ctx context.Context, parentID string, opts ListOpts) ([]*Comment, error)
	ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Comment, error)

	// Tree operations
	GetTree(ctx context.Context, threadID string, opts TreeOpts) ([]*Comment, error)
	GetSubtree(ctx context.Context, parentID string, depth int) ([]*Comment, error)
	BuildTree(comments []*Comment) []*Comment

	// Moderation
	Remove(ctx context.Context, id string, reason string) error
	Approve(ctx context.Context, id string) error

	// Voting
	UpdateVotes(ctx context.Context, id string, upDelta, downDelta int64) error

	// Viewer state
	EnrichComment(ctx context.Context, comment *Comment, viewerID string) error
	EnrichComments(ctx context.Context, comments []*Comment, viewerID string) error
}

// Store defines the data storage interface for comments.
type Store interface {
	Create(ctx context.Context, comment *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id string) error

	// Lists
	ListByThread(ctx context.Context, threadID string, opts ListOpts) ([]*Comment, error)
	ListByParent(ctx context.Context, parentID string, opts ListOpts) ([]*Comment, error)
	ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Comment, error)
	ListByPath(ctx context.Context, pathPrefix string, opts ListOpts) ([]*Comment, error)

	// Update child count
	IncrementChildCount(ctx context.Context, id string, delta int64) error
}

// WilsonScore calculates the Wilson score for comment ranking.
func WilsonScore(ups, downs int64) float64 {
	n := float64(ups + downs)
	if n == 0 {
		return 0
	}

	z := 1.96 // 95% confidence
	phat := float64(ups) / n

	return (phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)) / (1 + z*z/n)
}

// ControversialScore calculates the controversial score.
func ControversialScore(ups, downs int64) float64 {
	if ups <= 0 || downs <= 0 {
		return 0
	}

	magnitude := float64(ups + downs)
	balance := float64(min(ups, downs)) / float64(max(ups, downs))

	return magnitude * balance
}
