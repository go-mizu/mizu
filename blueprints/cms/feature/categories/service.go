package categories

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/slug"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound    = errors.New("category not found")
	ErrMissingName = errors.New("name is required")
)

// Service implements the categories API.
type Service struct {
	store Store
}

// NewService creates a new categories service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in *CreateIn) (*Category, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	now := time.Now()
	catSlug := in.Slug
	if catSlug == "" {
		catSlug = slug.Generate(in.Name)
	}

	cat := &Category{
		ID:              ulid.New(),
		ParentID:        in.ParentID,
		Name:            in.Name,
		Slug:            catSlug,
		Description:     in.Description,
		FeaturedImageID: in.FeaturedImageID,
		Meta:            in.Meta,
		SortOrder:       in.SortOrder,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.store.Create(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Category, error) {
	cat, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrNotFound
	}
	return cat, nil
}

func (s *Service) GetBySlug(ctx context.Context, catSlug string) (*Category, error) {
	cat, err := s.store.GetBySlug(ctx, catSlug)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrNotFound
	}
	return cat, nil
}

func (s *Service) List(ctx context.Context, in *ListIn) ([]*Category, int, error) {
	if in.Limit <= 0 {
		in.Limit = 100
	}
	return s.store.List(ctx, in)
}

func (s *Service) GetTree(ctx context.Context) ([]*Category, error) {
	return s.store.GetTree(ctx)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Category, error) {
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
