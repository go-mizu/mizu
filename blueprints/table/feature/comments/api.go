// Package comments provides record comment functionality.
package comments

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("comment not found")
)

// Comment represents a comment on a record.
type Comment struct {
	ID         string    `json:"id"`
	RecordID   string    `json:"record_id"`
	ParentID   string    `json:"parent_id,omitempty"`
	UserID     string    `json:"user_id"`
	Content    string    `json:"content"`
	IsResolved bool      `json:"is_resolved"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	RecordID string `json:"record_id"`
	ParentID string `json:"parent_id,omitempty"`
	Content  string `json:"content"`
}

// API defines the comments service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, content string) (*Comment, error)
	Delete(ctx context.Context, id string) error
	Resolve(ctx context.Context, id string) error
	Unresolve(ctx context.Context, id string) error
	ListByRecord(ctx context.Context, recordID string) ([]*Comment, error)
}

// Store defines the comments data access interface.
type Store interface {
	Create(ctx context.Context, comment *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id string) error
	ListByRecord(ctx context.Context, recordID string) ([]*Comment, error)
	DeleteByRecord(ctx context.Context, recordID string) error
}
