package blocks

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("block not found")
)

// Service implements the blocks API.
type Service struct {
	store Store
}

// NewService creates a new blocks service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new block.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Block, error) {
	now := time.Now()
	block := &Block{
		ID:        ulid.New(),
		PageID:    in.PageID,
		ParentID:  in.ParentID,
		Type:      in.Type,
		Content:   in.Content,
		Position:  in.Position,
		CreatedBy: in.CreatedBy,
		CreatedAt: now,
		UpdatedBy: in.CreatedBy,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, block); err != nil {
		return nil, err
	}

	return block, nil
}

// GetByID retrieves a block by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Block, error) {
	block, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Load children
	children, _ := s.store.GetChildren(ctx, id)
	block.Children = children

	return block, nil
}

// Update updates a block.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Block, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a block.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// GetByPage retrieves all blocks for a page.
func (s *Service) GetByPage(ctx context.Context, pageID string) ([]*Block, error) {
	blocks, err := s.store.GetByPage(ctx, pageID)
	if err != nil {
		return nil, err
	}

	// Build tree structure
	return s.buildTree(blocks), nil
}

// GetChildren retrieves children of a block.
func (s *Service) GetChildren(ctx context.Context, blockID string) ([]*Block, error) {
	return s.store.GetChildren(ctx, blockID)
}

// BatchCreate creates multiple blocks in a single batch operation.
func (s *Service) BatchCreate(ctx context.Context, inputs []*CreateIn) ([]*Block, error) {
	if len(inputs) == 0 {
		return []*Block{}, nil
	}

	now := time.Now()
	blocks := make([]*Block, len(inputs))
	for i, in := range inputs {
		blocks[i] = &Block{
			ID:        ulid.New(),
			PageID:    in.PageID,
			ParentID:  in.ParentID,
			Type:      in.Type,
			Content:   in.Content,
			Position:  in.Position,
			CreatedBy: in.CreatedBy,
			CreatedAt: now,
			UpdatedBy: in.CreatedBy,
			UpdatedAt: now,
		}
	}

	if err := s.store.BatchCreate(ctx, blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

// BatchUpdate updates multiple blocks.
func (s *Service) BatchUpdate(ctx context.Context, updates []*UpdateIn) error {
	// Note: True batch update would require a store method that accepts multiple updates
	// For now, individual updates are still necessary as each block may have different content
	for _, u := range updates {
		if err := s.store.Update(ctx, u.ID, u); err != nil {
			return err
		}
	}
	return nil
}

// BatchDelete deletes multiple blocks in a single batch operation.
func (s *Service) BatchDelete(ctx context.Context, ids []string) error {
	return s.store.BatchDelete(ctx, ids)
}

// Move moves a block to a new position.
func (s *Service) Move(ctx context.Context, id string, newParentID string, position int) error {
	return s.store.Move(ctx, id, newParentID, position)
}

// Reorder reorders blocks within a parent.
func (s *Service) Reorder(ctx context.Context, parentID string, blockIDs []string) error {
	return s.store.Reorder(ctx, parentID, blockIDs)
}

// ConvertType converts a block to a different type.
func (s *Service) ConvertType(ctx context.Context, id string, newType BlockType) (*Block, error) {
	block, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, id, &UpdateIn{
		Type:    newType,
		Content: block.Content,
	}); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, id)
}

// Duplicate creates a copy of a block.
func (s *Service) Duplicate(ctx context.Context, id string, userID string) (*Block, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	return s.Create(ctx, &CreateIn{
		PageID:    original.PageID,
		ParentID:  original.ParentID,
		Type:      original.Type,
		Content:   original.Content,
		Position:  original.Position + 1,
		CreatedBy: userID,
	})
}

// buildTree organizes flat blocks into a tree structure.
func (s *Service) buildTree(blocks []*Block) []*Block {
	// Create a map for quick lookup
	blockMap := make(map[string]*Block)
	for _, b := range blocks {
		b.Children = []*Block{}
		blockMap[b.ID] = b
	}

	// Build tree
	var roots []*Block
	for _, b := range blocks {
		if b.ParentID == "" {
			roots = append(roots, b)
		} else if parent, ok := blockMap[b.ParentID]; ok {
			parent.Children = append(parent.Children, b)
		} else {
			roots = append(roots, b)
		}
	}

	return roots
}
