package columns

import (
	"context"
	"errors"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("column not found")
)

// Service implements the columns API.
type Service struct {
	store Store
}

// NewService creates a new columns service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, projectID string, in *CreateIn) (*Column, error) {
	// Get existing columns to determine if this should be default
	existing, err := s.store.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	isDefault := in.IsDefault
	if len(existing) == 0 {
		// First column is always default
		isDefault = true
	}

	position := in.Position
	if position == 0 && len(existing) > 0 {
		// Default position is at the end
		position = len(existing)
	}

	column := &Column{
		ID:         ulid.New(),
		ProjectID:  projectID,
		Name:       in.Name,
		Position:   position,
		IsDefault:  isDefault,
		IsArchived: false,
	}

	if err := s.store.Create(ctx, column); err != nil {
		return nil, err
	}

	return column, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Column, error) {
	c, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Column, error) {
	return s.store.ListByProject(ctx, projectID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Column, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) UpdatePosition(ctx context.Context, id string, position int) error {
	return s.store.UpdatePosition(ctx, id, position)
}

func (s *Service) SetDefault(ctx context.Context, projectID, columnID string) error {
	return s.store.SetDefault(ctx, projectID, columnID)
}

func (s *Service) Archive(ctx context.Context, id string) error {
	return s.store.Archive(ctx, id)
}

func (s *Service) Unarchive(ctx context.Context, id string) error {
	return s.store.Unarchive(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) GetDefault(ctx context.Context, projectID string) (*Column, error) {
	c, err := s.store.GetDefault(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}
