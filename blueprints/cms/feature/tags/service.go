package tags

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/slug"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound    = errors.New("tag not found")
	ErrMissingName = errors.New("name is required")
)

// Service implements the tags API.
type Service struct {
	store Store
}

// NewService creates a new tags service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in *CreateIn) (*Tag, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	now := time.Now()
	tagSlug := in.Slug
	if tagSlug == "" {
		tagSlug = slug.Generate(in.Name)
	}

	tag := &Tag{
		ID:              ulid.New(),
		Name:            in.Name,
		Slug:            tagSlug,
		Description:     in.Description,
		FeaturedImageID: in.FeaturedImageID,
		Meta:            in.Meta,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.store.Create(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Tag, error) {
	tag, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, ErrNotFound
	}
	return tag, nil
}

func (s *Service) GetBySlug(ctx context.Context, tagSlug string) (*Tag, error) {
	tag, err := s.store.GetBySlug(ctx, tagSlug)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, ErrNotFound
	}
	return tag, nil
}

func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]*Tag, error) {
	return s.store.GetByIDs(ctx, ids)
}

func (s *Service) List(ctx context.Context, in *ListIn) ([]*Tag, int, error) {
	if in.Limit <= 0 {
		in.Limit = 100
	}
	if in.OrderBy == "" {
		in.OrderBy = "name"
	}
	if in.Order == "" {
		in.Order = "asc"
	}
	return s.store.List(ctx, in)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Tag, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) IncrementPostCount(ctx context.Context, id string) error {
	return s.store.IncrementPostCount(ctx, id)
}

func (s *Service) DecrementPostCount(ctx context.Context, id string) error {
	return s.store.DecrementPostCount(ctx, id)
}
