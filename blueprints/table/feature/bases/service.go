package bases

import (
	"context"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the bases API.
type Service struct {
	store Store
}

// NewService creates a new bases service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new base.
func (s *Service) Create(ctx context.Context, userID string, in CreateIn) (*Base, error) {
	base := &Base{
		ID:          ulid.New(),
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Description: in.Description,
		Icon:        in.Icon,
		Color:       in.Color,
		CreatedBy:   userID,
	}

	if base.Color == "" {
		base.Color = "#2563EB" // Default blue
	}

	if err := s.store.Create(ctx, base); err != nil {
		return nil, err
	}

	return base, nil
}

// GetByID retrieves a base by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Base, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a base.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Base, error) {
	base, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		base.Name = *in.Name
	}
	if in.Description != nil {
		base.Description = *in.Description
	}
	if in.Icon != nil {
		base.Icon = *in.Icon
	}
	if in.Color != nil {
		base.Color = *in.Color
	}

	if err := s.store.Update(ctx, base); err != nil {
		return nil, err
	}

	return base, nil
}

// Delete deletes a base.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Duplicate duplicates a base.
func (s *Service) Duplicate(ctx context.Context, id string, newName string) (*Base, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	base := &Base{
		ID:          ulid.New(),
		WorkspaceID: original.WorkspaceID,
		Name:        newName,
		Description: original.Description,
		Icon:        original.Icon,
		Color:       original.Color,
		CreatedBy:   original.CreatedBy,
	}

	if err := s.store.Create(ctx, base); err != nil {
		return nil, err
	}

	return base, nil
}

// ListByWorkspace lists all bases in a workspace.
func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Base, error) {
	return s.store.ListByWorkspace(ctx, workspaceID)
}
