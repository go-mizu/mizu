package tags

import (
	"context"
)

// Service implements the tags.API interface.
type Service struct {
	store Store
}

// NewService creates a new tags service.
func NewService(store Store) *Service {
	return &Service{store: store}
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
