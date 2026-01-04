package synced_blocks

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
)

// SyncedBlock represents a block that can be synced across multiple pages.
// The original block content is stored once, and references link to it.
type SyncedBlock struct {
	ID          string           `json:"id"`
	WorkspaceID string           `json:"workspace_id"`
	OriginalID  string           `json:"original_id"`    // Original block ID
	PageID      string           `json:"page_id"`        // Page containing the original
	PageName    string           `json:"page_name"`      // Name of the original page
	Content     []blocks.Block   `json:"content"`        // The synced block content
	LastUpdated time.Time        `json:"last_updated"`
	CreatedAt   time.Time        `json:"created_at"`
	CreatedBy   string           `json:"created_by"`
}

// SyncedBlockReference tracks where a synced block is used.
type SyncedBlockReference struct {
	ID            string    `json:"id"`
	SyncedBlockID string    `json:"synced_block_id"`
	PageID        string    `json:"page_id"`       // Page where this reference exists
	BlockID       string    `json:"block_id"`      // The block containing this reference
	CreatedAt     time.Time `json:"created_at"`
}

// CreateIn is the input for creating a synced block.
type CreateIn struct {
	WorkspaceID string `json:"workspace_id"`
	PageID      string `json:"page_id"`
	BlockIDs    []string `json:"block_ids"` // Blocks to sync
	CreatedBy   string `json:"created_by"`
}

// UpdateIn is the input for updating a synced block.
type UpdateIn struct {
	Content []blocks.Block `json:"content,omitempty"`
}

// API defines the synced blocks service interface.
type API interface {
	// Create creates a new synced block from existing blocks.
	Create(ctx context.Context, in *CreateIn) (*SyncedBlock, error)

	// GetByID retrieves a synced block by its ID.
	GetByID(ctx context.Context, id string) (*SyncedBlock, error)

	// Update updates a synced block's content.
	Update(ctx context.Context, id string, in *UpdateIn) (*SyncedBlock, error)

	// Delete removes a synced block and all its references.
	Delete(ctx context.Context, id string) error

	// ListByPage returns all synced blocks originating from a page.
	ListByPage(ctx context.Context, pageID string) ([]*SyncedBlock, error)

	// ListByWorkspace returns all synced blocks in a workspace.
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*SyncedBlock, error)

	// AddReference adds a reference to a synced block from another page.
	AddReference(ctx context.Context, syncedBlockID, pageID, blockID string) (*SyncedBlockReference, error)

	// RemoveReference removes a reference to a synced block.
	RemoveReference(ctx context.Context, refID string) error

	// GetReferences returns all references to a synced block.
	GetReferences(ctx context.Context, syncedBlockID string) ([]*SyncedBlockReference, error)
}

// Store defines the storage interface for synced blocks.
type Store interface {
	Create(ctx context.Context, sb *SyncedBlock) error
	GetByID(ctx context.Context, id string) (*SyncedBlock, error)
	Update(ctx context.Context, sb *SyncedBlock) error
	Delete(ctx context.Context, id string) error
	ListByPage(ctx context.Context, pageID string) ([]*SyncedBlock, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*SyncedBlock, error)

	CreateReference(ctx context.Context, ref *SyncedBlockReference) error
	DeleteReference(ctx context.Context, id string) error
	GetReferencesByBlock(ctx context.Context, syncedBlockID string) ([]*SyncedBlockReference, error)
}
