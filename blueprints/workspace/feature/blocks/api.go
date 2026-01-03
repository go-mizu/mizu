// Package blocks provides block content management.
package blocks

import (
	"context"
	"time"
)

// BlockType represents the type of block.
type BlockType string

const (
	// Text blocks
	BlockParagraph  BlockType = "paragraph"
	BlockHeading1   BlockType = "heading_1"
	BlockHeading2   BlockType = "heading_2"
	BlockHeading3   BlockType = "heading_3"
	BlockQuote      BlockType = "quote"
	BlockCallout    BlockType = "callout"

	// List blocks
	BlockBulletList BlockType = "bulleted_list"
	BlockNumberList BlockType = "numbered_list"
	BlockToggle     BlockType = "toggle"
	BlockTodo       BlockType = "to_do"

	// Media blocks
	BlockImage      BlockType = "image"
	BlockVideo      BlockType = "video"
	BlockFile       BlockType = "file"
	BlockBookmark   BlockType = "bookmark"
	BlockEmbed      BlockType = "embed"

	// Advanced blocks
	BlockCode       BlockType = "code"
	BlockEquation   BlockType = "equation"
	BlockTable      BlockType = "table"
	BlockTableRow   BlockType = "table_row"
	BlockDivider    BlockType = "divider"
	BlockColumnList BlockType = "column_list"
	BlockColumn     BlockType = "column"

	// Database blocks
	BlockChildPage  BlockType = "child_page"
	BlockChildDB    BlockType = "child_database"
	BlockLinkedDB   BlockType = "linked_database"
	BlockSyncedBlock BlockType = "synced_block"
)

// Block represents a content block.
type Block struct {
	ID        string    `json:"id"`
	PageID    string    `json:"page_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Type      BlockType `json:"type"`
	Content   Content   `json:"content"`
	Position  int       `json:"position"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`

	// Enriched
	Children []*Block `json:"children,omitempty"`
}

// Content holds block-specific content.
type Content struct {
	// Text content (with rich text)
	RichText []RichText `json:"rich_text,omitempty"`

	// Todo specific
	Checked *bool `json:"checked,omitempty"`

	// Callout specific
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`

	// Code specific
	Language string `json:"language,omitempty"`

	// Media specific
	URL     string     `json:"url,omitempty"`
	Caption []RichText `json:"caption,omitempty"`

	// Bookmark specific
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// Table specific
	TableWidth int  `json:"table_width,omitempty"`
	HasHeader  bool `json:"has_header,omitempty"`

	// Database reference
	DatabaseID string `json:"database_id,omitempty"`

	// Synced block
	SyncedFrom string `json:"synced_from,omitempty"`
}

// RichText represents formatted text.
type RichText struct {
	Type        string      `json:"type"` // text, mention, equation
	Text        string      `json:"text"`
	Annotations Annotations `json:"annotations,omitempty"`
	Link        string      `json:"link,omitempty"`
	Mention     *Mention    `json:"mention,omitempty"`
}

// Annotations holds text formatting.
type Annotations struct {
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Color         string `json:"color,omitempty"`
}

// Mention represents a mention in rich text.
type Mention struct {
	Type   string `json:"type"` // user, page, date
	UserID string `json:"user_id,omitempty"`
	PageID string `json:"page_id,omitempty"`
	Date   string `json:"date,omitempty"`
}

// CreateIn contains input for creating a block.
type CreateIn struct {
	PageID    string    `json:"page_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Type      BlockType `json:"type"`
	Content   Content   `json:"content"`
	Position  int       `json:"position"`
	CreatedBy string    `json:"-"`
}

// UpdateIn contains input for updating a block.
type UpdateIn struct {
	ID        string    `json:"id"`
	Type      BlockType `json:"type,omitempty"`
	Content   Content   `json:"content"`
	UpdatedBy string    `json:"-"`
}

// API defines the blocks service contract.
type API interface {
	// CRUD
	Create(ctx context.Context, in *CreateIn) (*Block, error)
	GetByID(ctx context.Context, id string) (*Block, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Block, error)
	Delete(ctx context.Context, id string) error

	// Bulk operations
	GetByPage(ctx context.Context, pageID string) ([]*Block, error)
	GetChildren(ctx context.Context, blockID string) ([]*Block, error)
	BatchCreate(ctx context.Context, blocks []*CreateIn) ([]*Block, error)
	BatchUpdate(ctx context.Context, updates []*UpdateIn) error
	BatchDelete(ctx context.Context, ids []string) error

	// Reorder
	Move(ctx context.Context, id string, newParentID string, position int) error
	Reorder(ctx context.Context, parentID string, blockIDs []string) error

	// Special operations
	ConvertType(ctx context.Context, id string, newType BlockType) (*Block, error)
	Duplicate(ctx context.Context, id string, userID string) (*Block, error)
}

// Store defines the data access contract for blocks.
type Store interface {
	Create(ctx context.Context, b *Block) error
	GetByID(ctx context.Context, id string) (*Block, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	GetByPage(ctx context.Context, pageID string) ([]*Block, error)
	GetChildren(ctx context.Context, blockID string) ([]*Block, error)
	Move(ctx context.Context, id string, newParentID string, position int) error
	Reorder(ctx context.Context, parentID string, blockIDs []string) error
	DeleteByPage(ctx context.Context, pageID string) error
}
