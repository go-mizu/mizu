package reactions

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound       = errors.New("reaction not found")
	ErrInvalidContent = errors.New("invalid reaction content")
	ErrAlreadyExists  = errors.New("reaction already exists")
)

// Valid reaction content types
const (
	ContentPlusOne  = "+1"
	ContentMinusOne = "-1"
	ContentLaugh    = "laugh"
	ContentConfused = "confused"
	ContentHeart    = "heart"
	ContentHooray   = "hooray"
	ContentRocket   = "rocket"
	ContentEyes     = "eyes"
)

// Reaction represents a reaction
type Reaction struct {
	ID        int64             `json:"id"`
	NodeID    string            `json:"node_id"`
	User      *users.SimpleUser `json:"user"`
	Content   string            `json:"content"` // +1, -1, laugh, confused, heart, hooray, rocket, eyes
	CreatedAt time.Time         `json:"created_at"`
}

// Reactions represents reaction counts (rollup)
type Reactions struct {
	URL        string `json:"url"`
	TotalCount int    `json:"total_count"`
	PlusOne    int    `json:"+1"`
	MinusOne   int    `json:"-1"`
	Laugh      int    `json:"laugh"`
	Confused   int    `json:"confused"`
	Heart      int    `json:"heart"`
	Hooray     int    `json:"hooray"`
	Rocket     int    `json:"rocket"`
	Eyes       int    `json:"eyes"`
}

// ListOpts contains options for listing reactions
type ListOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Content string `json:"content,omitempty"` // Filter by content type
}

// API defines the reactions service interface
type API interface {
	// Issue reactions
	ListForIssue(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Reaction, error)
	CreateForIssue(ctx context.Context, owner, repo string, number int, userID int64, content string) (*Reaction, error)
	DeleteForIssue(ctx context.Context, owner, repo string, number int, reactionID int64) error

	// Issue comment reactions
	ListForIssueComment(ctx context.Context, owner, repo string, commentID int64, opts *ListOpts) ([]*Reaction, error)
	CreateForIssueComment(ctx context.Context, owner, repo string, commentID int64, userID int64, content string) (*Reaction, error)
	DeleteForIssueComment(ctx context.Context, owner, repo string, commentID int64, reactionID int64) error

	// PR review comment reactions
	ListForPullReviewComment(ctx context.Context, owner, repo string, commentID int64, opts *ListOpts) ([]*Reaction, error)
	CreateForPullReviewComment(ctx context.Context, owner, repo string, commentID int64, userID int64, content string) (*Reaction, error)
	DeleteForPullReviewComment(ctx context.Context, owner, repo string, commentID int64, reactionID int64) error

	// Commit comment reactions
	ListForCommitComment(ctx context.Context, owner, repo string, commentID int64, opts *ListOpts) ([]*Reaction, error)
	CreateForCommitComment(ctx context.Context, owner, repo string, commentID int64, userID int64, content string) (*Reaction, error)
	DeleteForCommitComment(ctx context.Context, owner, repo string, commentID int64, reactionID int64) error

	// Release reactions
	ListForRelease(ctx context.Context, owner, repo string, releaseID int64, opts *ListOpts) ([]*Reaction, error)
	CreateForRelease(ctx context.Context, owner, repo string, releaseID int64, userID int64, content string) (*Reaction, error)
	DeleteForRelease(ctx context.Context, owner, repo string, releaseID int64, reactionID int64) error

	// Get reaction rollup for a subject
	GetRollup(ctx context.Context, subjectType string, subjectID int64) (*Reactions, error)
}

// Store defines the data access interface for reactions
type Store interface {
	Create(ctx context.Context, subjectType string, subjectID, userID int64, content string) (*Reaction, error)
	GetByID(ctx context.Context, id int64) (*Reaction, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, subjectType string, subjectID int64, opts *ListOpts) ([]*Reaction, error)
	GetByUserAndContent(ctx context.Context, subjectType string, subjectID, userID int64, content string) (*Reaction, error)
	GetRollup(ctx context.Context, subjectType string, subjectID int64) (*Reactions, error)
}

// ValidContent returns true if the content is a valid reaction type
func ValidContent(content string) bool {
	switch content {
	case ContentPlusOne, ContentMinusOne, ContentLaugh, ContentConfused,
		ContentHeart, ContentHooray, ContentRocket, ContentEyes:
		return true
	}
	return false
}
