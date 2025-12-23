package tags

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
)

// Service implements the tags.API interface.
type Service struct {
	store Store
}

// NewService creates a new tags service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new tag.
func (s *Service) Create(ctx context.Context, in CreateIn) (*Tag, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Check if name is taken
	if existing, _ := s.store.GetByName(ctx, in.Name); existing != nil {
		return nil, ErrNameTaken
	}

	tag := &Tag{
		ID:          ulid.New(),
		Name:        in.Name,
		Description: in.Description,
		Color:       in.Color,
	}

	if tag.Color == "" {
		tag.Color = "#666666"
	}

	if err := s.store.Create(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// GetByID retrieves a tag by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Tag, error) {
	return s.store.GetByID(ctx, id)
}

// GetByName retrieves a tag by name.
func (s *Service) GetByName(ctx context.Context, name string) (*Tag, error) {
	return s.store.GetByName(ctx, name)
}

// GetByNames retrieves tags by names.
func (s *Service) GetByNames(ctx context.Context, names []string) ([]*Tag, error) {
	return s.store.GetByNames(ctx, names)
}

// Update updates a tag.
func (s *Service) Update(ctx context.Context, id string, in CreateIn) (*Tag, error) {
	tag, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := in.Validate(); err != nil {
		return nil, err
	}

	tag.Name = in.Name
	tag.Description = in.Description
	if in.Color != "" {
		tag.Color = in.Color
	}

	if err := s.store.Update(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// Delete deletes a tag.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// List lists all tags.
func (s *Service) List(ctx context.Context, limit int) ([]*Tag, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.store.List(ctx, limit)
}

// ListPopular lists popular tags.
func (s *Service) ListPopular(ctx context.Context, limit int) ([]*Tag, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.ListPopular(ctx, limit)
}

// IncrementCount increments a tag's story count.
func (s *Service) IncrementCount(ctx context.Context, id string, delta int64) error {
	return s.store.IncrementCount(ctx, id, delta)
}
