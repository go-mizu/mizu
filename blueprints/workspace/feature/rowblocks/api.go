// Package rowblocks provides content block functionality for database rows.
package rowblocks

import (
	"context"
	"time"
)

// BlockType represents the type of content block.
type BlockType string

const (
	BlockTypeHeading1     BlockType = "heading_1"
	BlockTypeHeading2     BlockType = "heading_2"
	BlockTypeHeading3     BlockType = "heading_3"
	BlockTypeParagraph    BlockType = "paragraph"
	BlockTypeBulletedList BlockType = "bulleted_list"
	BlockTypeNumberedList BlockType = "numbered_list"
	BlockTypeToDo         BlockType = "to_do"
	BlockTypeToggle       BlockType = "toggle"
	BlockTypeCode         BlockType = "code"
	BlockTypeQuote        BlockType = "quote"
	BlockTypeDivider      BlockType = "divider"
	BlockTypeCallout      BlockType = "callout"
	BlockTypeImage        BlockType = "image"
	BlockTypeFile         BlockType = "file"
)

// Block represents a content block within a database row.
type Block struct {
	ID         string                 `json:"id"`
	RowID      string                 `json:"row_id"`
	ParentID   string                 `json:"parent_id,omitempty"`
	Type       BlockType              `json:"type"`
	Content    string                 `json:"content"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Order      int                    `json:"order"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`

	// Nested blocks (for toggle blocks)
	Children []*Block `json:"children,omitempty"`
}

// CreateIn contains input for creating a block.
type CreateIn struct {
	RowID      string                 `json:"row_id"`
	ParentID   string                 `json:"parent_id,omitempty"`
	Type       BlockType              `json:"type"`
	Content    string                 `json:"content"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	AfterID    string                 `json:"after_id,omitempty"` // Insert after this block
}

// UpdateIn contains input for updating a block.
type UpdateIn struct {
	Content    string                 `json:"content,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Checked    *bool                  `json:"checked,omitempty"` // For to_do blocks
}

// ReorderIn contains input for reordering blocks.
type ReorderIn struct {
	BlockIDs []string `json:"block_ids"`
}

// API defines the row blocks service contract.
type API interface {
	// CRUD
	Create(ctx context.Context, in *CreateIn) (*Block, error)
	GetByID(ctx context.Context, id string) (*Block, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Block, error)
	Delete(ctx context.Context, id string) error

	// List
	ListByRow(ctx context.Context, rowID string) ([]*Block, error)

	// Reorder
	Reorder(ctx context.Context, rowID string, in *ReorderIn) error
}

// Store defines the data access contract for row blocks.
type Store interface {
	Create(ctx context.Context, b *Block) error
	GetByID(ctx context.Context, id string) (*Block, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByRow(ctx context.Context, rowID string) ([]*Block, error)
	GetMaxOrder(ctx context.Context, rowID string) (int, error)
	UpdateOrders(ctx context.Context, rowID string, blockIDs []string) error
}
