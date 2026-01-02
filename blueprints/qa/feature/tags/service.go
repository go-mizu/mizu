package tags

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the tags API.
type Service struct {
	store Store
}

// NewService creates a new tags service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new tag.
func (s *Service) Create(ctx context.Context, name string, excerpt string) (*Tag, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, ErrNotFound
	}
	if existing, _ := s.store.GetByName(ctx, name); existing != nil {
		return existing, nil
	}
	tag := &Tag{
		ID:        ulid.New(),
		Name:      name,
		Excerpt:   excerpt,
		CreatedAt: time.Now(),
	}
	if err := s.store.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

// UpsertBatch ensures tags exist.
func (s *Service) UpsertBatch(ctx context.Context, names []string) error {
	for _, name := range names {
		_, _ = s.Create(ctx, name, "")
	}
	return nil
}

// GetByName gets a tag by name.
func (s *Service) GetByName(ctx context.Context, name string) (*Tag, error) {
	return s.store.GetByName(ctx, name)
}

// List lists tags.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Tag, error) {
	return s.store.List(ctx, opts)
}

// IncrementQuestionCount updates question count.
func (s *Service) IncrementQuestionCount(ctx context.Context, name string, delta int64) error {
	return s.store.IncrementQuestionCount(ctx, name, delta)
}
