package suggest

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// Service implements the suggestion API.
type Service struct {
	store Store
}

// NewService creates a new suggestion service.
func NewService(s Store) *Service {
	return &Service{store: s}
}

// GetSuggestions returns autocomplete suggestions for a prefix.
func (s *Service) GetSuggestions(ctx context.Context, prefix string, limit int) ([]store.Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.Suggest().GetSuggestions(ctx, prefix, limit)
}

// RecordQuery records a query for future suggestions.
func (s *Service) RecordQuery(ctx context.Context, query string) error {
	return s.store.Suggest().RecordQuery(ctx, query)
}

// GetTrending returns trending/popular queries.
func (s *Service) GetTrending(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.Suggest().GetTrendingQueries(ctx, limit)
}
