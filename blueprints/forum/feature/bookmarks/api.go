package bookmarks

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("bookmark not found")
	ErrAlreadyExists  = errors.New("bookmark already exists")
)

// Target types
const (
	TargetThread  = "thread"
	TargetComment = "comment"
)

// Bookmark represents a saved item.
type Bookmark struct {
	ID         string    `json:"id"`
	AccountID  string    `json:"account_id"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListOpts contains options for listing bookmarks.
type ListOpts struct {
	Limit  int
	Cursor string
}

// API defines the bookmarks service interface.
type API interface {
	Create(ctx context.Context, accountID, targetType, targetID string) error
	Delete(ctx context.Context, accountID, targetType, targetID string) error
	IsBookmarked(ctx context.Context, accountID, targetType, targetID string) (bool, error)
	List(ctx context.Context, accountID, targetType string, opts ListOpts) ([]*Bookmark, error)

	// Batch check
	GetBookmarked(ctx context.Context, accountID, targetType string, targetIDs []string) (map[string]bool, error)
}

// Store defines the data storage interface for bookmarks.
type Store interface {
	Create(ctx context.Context, bookmark *Bookmark) error
	GetByTarget(ctx context.Context, accountID, targetType, targetID string) (*Bookmark, error)
	Delete(ctx context.Context, accountID, targetType, targetID string) error
	List(ctx context.Context, accountID, targetType string, opts ListOpts) ([]*Bookmark, error)
	GetByTargets(ctx context.Context, accountID, targetType string, targetIDs []string) ([]*Bookmark, error)
}
