// Package comments provides commenting functionality.
package comments

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// TargetType indicates what the comment is attached to.
type TargetType string

const (
	TargetPage        TargetType = "page"
	TargetBlock       TargetType = "block"
	TargetDatabaseRow TargetType = "database_row"
)

// Comment represents a comment on a page, block, or database row.
type Comment struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	TargetType  TargetType        `json:"target_type"`
	TargetID    string            `json:"target_id"`
	ParentID    string            `json:"parent_id,omitempty"` // For replies
	Content     []blocks.RichText `json:"content"`
	AuthorID    string            `json:"author_id"`
	IsResolved  bool              `json:"is_resolved"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	// Enriched
	Author  *users.User `json:"author,omitempty"`
	Replies []*Comment  `json:"replies,omitempty"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	WorkspaceID string            `json:"workspace_id"`
	TargetType  TargetType        `json:"target_type"`
	TargetID    string            `json:"target_id"`
	ParentID    string            `json:"parent_id,omitempty"`
	Content     []blocks.RichText `json:"content"`
	AuthorID    string            `json:"-"`
}

// API defines the comments service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, content []blocks.RichText) (*Comment, error)
	Delete(ctx context.Context, id string) error

	// List by target type and ID
	ListByTarget(ctx context.Context, workspaceID string, targetType TargetType, targetID string) ([]*Comment, error)

	// Legacy methods for backwards compatibility
	ListByPage(ctx context.Context, pageID string) ([]*Comment, error)
	ListByBlock(ctx context.Context, blockID string) ([]*Comment, error)

	Resolve(ctx context.Context, id string) error
	Unresolve(ctx context.Context, id string) error
}

// Store defines the data access contract for comments.
type Store interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, content []blocks.RichText) error
	Delete(ctx context.Context, id string) error
	ListByTarget(ctx context.Context, workspaceID string, targetType TargetType, targetID string) ([]*Comment, error)
	SetResolved(ctx context.Context, id string, resolved bool) error
}
