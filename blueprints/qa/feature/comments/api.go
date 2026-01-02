package comments

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
)

var (
	ErrNotFound = errors.New("comment not found")
	ErrInvalid  = errors.New("invalid comment")
)

const (
	BodyMaxLen = 1000
)

// TargetType defines comment target.
type TargetType string

const (
	TargetQuestion TargetType = "question"
	TargetAnswer   TargetType = "answer"
)

// Comment represents a comment.
type Comment struct {
	ID         string     `json:"id"`
	TargetType TargetType `json:"target_type"`
	TargetID   string     `json:"target_id"`
	AuthorID   string     `json:"author_id"`
	Body       string     `json:"body"`
	Score      int64      `json:"score"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	Author *accounts.Account `json:"author,omitempty"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	TargetType TargetType `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Body       string     `json:"body"`
}

// ListOpts contains options for listing comments.
type ListOpts struct {
	Limit int
}

// API defines the comments service interface.
type API interface {
	Create(ctx context.Context, authorID string, in CreateIn) (*Comment, error)
	ListByTarget(ctx context.Context, targetType TargetType, targetID string, opts ListOpts) ([]*Comment, error)
	ListByTargets(ctx context.Context, targetType TargetType, targetIDs []string, opts ListOpts) (map[string][]*Comment, error)
	Delete(ctx context.Context, id string) error
	UpdateScore(ctx context.Context, id string, delta int64) error
}

// Store defines the data storage interface for comments.
type Store interface {
	Create(ctx context.Context, comment *Comment) error
	ListByTarget(ctx context.Context, targetType TargetType, targetID string, opts ListOpts) ([]*Comment, error)
	ListByTargets(ctx context.Context, targetType TargetType, targetIDs []string, opts ListOpts) (map[string][]*Comment, error)
	Delete(ctx context.Context, id string) error
	UpdateScore(ctx context.Context, id string, delta int64) error
}
