package synced_blocks

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/oklog/ulid/v2"
)

// Service implements the synced blocks API.
type Service struct {
	store  Store
	blocks blocks.API
	pages  pages.API
}

// NewService creates a new synced blocks service.
func NewService(store Store, blocks blocks.API, pages pages.API) *Service {
	return &Service{
		store:  store,
		blocks: blocks,
		pages:  pages,
	}
}

// Create creates a new synced block from existing blocks.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*SyncedBlock, error) {
	// Get page info
	page, err := s.pages.GetByID(ctx, in.PageID)
	if err != nil {
		return nil, err
	}

	// Get the blocks to sync
	var content []blocks.Block
	for _, blockID := range in.BlockIDs {
		block, err := s.blocks.GetByID(ctx, blockID)
		if err != nil {
			continue // Skip blocks that don't exist
		}
		content = append(content, *block)
	}

	now := time.Now()
	sb := &SyncedBlock{
		ID:          ulid.Make().String(),
		WorkspaceID: in.WorkspaceID,
		OriginalID:  in.BlockIDs[0], // First block is the original
		PageID:      in.PageID,
		PageName:    page.Title,
		Content:     content,
		LastUpdated: now,
		CreatedAt:   now,
		CreatedBy:   in.CreatedBy,
	}

	if err := s.store.Create(ctx, sb); err != nil {
		return nil, err
	}

	return sb, nil
}

// GetByID retrieves a synced block by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*SyncedBlock, error) {
	sb, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Refresh page name in case it changed
	if page, err := s.pages.GetByID(ctx, sb.PageID); err == nil {
		sb.PageName = page.Title
	}

	return sb, nil
}

// Update updates a synced block's content.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*SyncedBlock, error) {
	sb, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Content != nil {
		sb.Content = in.Content
	}
	sb.LastUpdated = time.Now()

	if err := s.store.Update(ctx, sb); err != nil {
		return nil, err
	}

	return sb, nil
}

// Delete removes a synced block and all its references.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByPage returns all synced blocks originating from a page.
func (s *Service) ListByPage(ctx context.Context, pageID string) ([]*SyncedBlock, error) {
	return s.store.ListByPage(ctx, pageID)
}

// ListByWorkspace returns all synced blocks in a workspace.
func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string) ([]*SyncedBlock, error) {
	return s.store.ListByWorkspace(ctx, workspaceID)
}

// AddReference adds a reference to a synced block from another page.
func (s *Service) AddReference(ctx context.Context, syncedBlockID, pageID, blockID string) (*SyncedBlockReference, error) {
	ref := &SyncedBlockReference{
		ID:            ulid.Make().String(),
		SyncedBlockID: syncedBlockID,
		PageID:        pageID,
		BlockID:       blockID,
		CreatedAt:     time.Now(),
	}

	if err := s.store.CreateReference(ctx, ref); err != nil {
		return nil, err
	}

	return ref, nil
}

// RemoveReference removes a reference to a synced block.
func (s *Service) RemoveReference(ctx context.Context, refID string) error {
	return s.store.DeleteReference(ctx, refID)
}

// GetReferences returns all references to a synced block.
func (s *Service) GetReferences(ctx context.Context, syncedBlockID string) ([]*SyncedBlockReference, error) {
	return s.store.GetReferencesByBlock(ctx, syncedBlockID)
}
