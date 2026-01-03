// Package comments provides commenting functionality.
package comments

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// Comment represents a comment on a page or block.
type Comment struct {
	ID         string            `json:"id"`
	PageID     string            `json:"page_id"`
	BlockID    string            `json:"block_id,omitempty"`
	ParentID   string            `json:"parent_id,omitempty"` // For replies
	Content    []blocks.RichText `json:"content"`
	AuthorID   string            `json:"author_id"`
	IsResolved bool              `json:"is_resolved"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`

	// Enriched
	Author  *users.User `json:"author,omitempty"`
	Replies []*Comment  `json:"replies,omitempty"`
}

// CreateIn contains input for creating a comment.
type CreateIn struct {
	PageID   string            `json:"page_id"`
	BlockID  string            `json:"block_id,omitempty"`
	ParentID string            `json:"parent_id,omitempty"`
	Content  []blocks.RichText `json:"content"`
	AuthorID string            `json:"-"`
}

// API defines the comments service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Comment, error)
	GetByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, id string, content []blocks.RichText) (*Comment, error)
	Delete(ctx context.Context, id string) error

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
	ListByPage(ctx context.Context, pageID string) ([]*Comment, error)
	ListByBlock(ctx context.Context, blockID string) ([]*Comment, error)
	SetResolved(ctx context.Context, id string, resolved bool) error
}
