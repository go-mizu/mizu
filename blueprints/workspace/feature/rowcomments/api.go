// Package rowcomments provides comment functionality for database rows.
package rowcomments

import (
	"context"
	"time"
)

// Comment represents a comment on a database row.
type Comment struct {
	ID         string    `json:"id"`
	RowID      string    `json:"row_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name,omitempty"`
	UserAvatar string    `json:"user_avatar,omitempty"`
	Content    string    `json:"content"`
	Resolved   bool      `json:"resolved"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	RowID   string `json:"row_id"`
	Content string `json:"content"`
	UserID  string `json:"-"`
}

// UpdateIn contains input for updating a comment.
type UpdateIn struct {
	Content string `json:"content"`
}

// API defines the row comments service contract.
type API interface {
	// CRUD
	Create(ctx context.Context, in *CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Comment, error)
	Delete(ctx context.Context, id string) error

	// List
	ListByRow(ctx context.Context, rowID string) ([]*Comment, error)

	// Actions
	Resolve(ctx context.Context, id string) error
	Unresolve(ctx context.Context, id string) error
}

// Store defines the data access contract for row comments.
type Store interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, content string) error
	Delete(ctx context.Context, id string) error
	ListByRow(ctx context.Context, rowID string) ([]*Comment, error)
	SetResolved(ctx context.Context, id string, resolved bool) error
}
