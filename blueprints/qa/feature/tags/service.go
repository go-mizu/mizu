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
	if len(names) == 0 {
		return nil
	}

	// Normalize names
	normalizedNames := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			normalizedNames = append(normalizedNames, name)
		}
	}

	// Batch check which tags already exist
	existing, _ := s.store.GetByNames(ctx, normalizedNames)

	// Create only missing tags
	for _, name := range normalizedNames {
		if _, exists := existing[name]; !exists {
			_, _ = s.Create(ctx, name, "")
		}
	}
	return nil
}

// GetByName gets a tag by name.
func (s *Service) GetByName(ctx context.Context, name string) (*Tag, error) {
	return s.store.GetByName(ctx, name)
}

// GetByNames gets multiple tags by names.
func (s *Service) GetByNames(ctx context.Context, names []string) (map[string]*Tag, error) {
	return s.store.GetByNames(ctx, names)
}

// List lists tags.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Tag, error) {
	return s.store.List(ctx, opts)
}

// IncrementQuestionCount updates question count.
func (s *Service) IncrementQuestionCount(ctx context.Context, name string, delta int64) error {
	return s.store.IncrementQuestionCount(ctx, name, delta)
}

// IncrementQuestionCountBatch updates question counts for multiple tags.
func (s *Service) IncrementQuestionCountBatch(ctx context.Context, names []string, delta int64) error {
	return s.store.IncrementQuestionCountBatch(ctx, names, delta)
}
