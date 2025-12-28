package comments

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("comment not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrMissingBody  = errors.New("comment body is required")
	ErrAccessDenied = errors.New("access denied")
)

// Target types
const (
	TargetIssue       = "issue"
	TargetPullRequest = "pull_request"
)

// Comment represents a comment on an issue or pull request
type Comment struct {
	ID         string    `json:"id"`
	TargetType string    `json:"target_type"` // issue, pull_request
	TargetID   string    `json:"target_id"`
	UserID     string    `json:"user_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateIn is the input for creating a comment
type CreateIn struct {
	Body string `json:"body"`
}

// UpdateIn is the input for updating a comment
type UpdateIn struct {
	Body string `json:"body"`
}

// ListOpts are options for listing comments
type ListOpts struct {
	Sort      string // created, updated
	Direction string // asc, desc
	Since     *time.Time
	Limit     int
	Offset    int
}

// API is the comments service interface
type API interface {
	Create(ctx context.Context, targetType, targetID, userID string, in *CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Comment, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, targetType, targetID string, opts *ListOpts) ([]*Comment, int, error)
	CountByTarget(ctx context.Context, targetType, targetID string) (int, error)
}

// Store is the comments data store interface
type Store interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, c *Comment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, targetType, targetID string, limit, offset int) ([]*Comment, int, error)
	CountByTarget(ctx context.Context, targetType, targetID string) (int, error)
}
