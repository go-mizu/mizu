package rowblocks

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the row blocks API.
type Service struct {
	store Store
}

// NewService creates a new row blocks service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new content block.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Block, error) {
	// Get the max order to place this block at the end (or after specific block)
	maxOrder, err := s.store.GetMaxOrder(ctx, in.RowID)
	if err != nil {
		maxOrder = -1
	}

	block := &Block{
		ID:         ulid.Make().String(),
		RowID:      in.RowID,
		ParentID:   in.ParentID,
		Type:       in.Type,
		Content:    in.Content,
		Properties: in.Properties,
		Order:      maxOrder + 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Initialize properties for specific block types
	if block.Properties == nil {
		block.Properties = make(map[string]interface{})
	}

	// Set defaults for to_do blocks
	if in.Type == BlockTypeToDo {
		if _, ok := block.Properties["checked"]; !ok {
			block.Properties["checked"] = false
		}
	}

	if err := s.store.Create(ctx, block); err != nil {
		return nil, err
	}

	return block, nil
}

// GetByID retrieves a block by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Block, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a block's content or properties.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Block, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete deletes a block.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByRow retrieves all blocks for a row, ordered by sort_order.
func (s *Service) ListByRow(ctx context.Context, rowID string) ([]*Block, error) {
	blocks, err := s.store.ListByRow(ctx, rowID)
	if err != nil {
		return nil, err
	}

	// Build tree structure for nested blocks
	blockMap := make(map[string]*Block)
	var rootBlocks []*Block

	for _, block := range blocks {
		blockMap[block.ID] = block
	}

	for _, block := range blocks {
		if block.ParentID == "" {
			rootBlocks = append(rootBlocks, block)
		} else if parent, ok := blockMap[block.ParentID]; ok {
			parent.Children = append(parent.Children, block)
		}
	}

	return rootBlocks, nil
}

// Reorder updates the order of blocks.
func (s *Service) Reorder(ctx context.Context, rowID string, in *ReorderIn) error {
	return s.store.UpdateOrders(ctx, rowID, in.BlockIDs)
}
