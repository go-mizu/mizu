// Package enrich provides enrichment APIs (Teclis/TinyGem style).
package enrich

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// Service handles small web and news enrichment.
type Service struct {
	store store.SmallWebStore
}

// NewService creates a new enrichment service.
func NewService(st store.SmallWebStore) *Service {
	return &Service{store: st}
}

// SearchWeb searches the small web index (Teclis-style).
func (s *Service) SearchWeb(ctx context.Context, query string, limit int) (*types.EnrichmentResponse, error) {
	start := time.Now()

	if limit <= 0 {
		limit = 10
	}

	results, err := s.store.SearchWeb(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return &types.EnrichmentResponse{
		Meta: types.EnrichmentMeta{
			ID:   generateID(),
			Node: "local",
			Ms:   time.Since(start).Milliseconds(),
		},
		Data: toEnrichmentResults(results),
	}, nil
}

// SearchNews searches for non-mainstream news (TinyGem-style).
func (s *Service) SearchNews(ctx context.Context, query string, limit int) (*types.EnrichmentResponse, error) {
	start := time.Now()

	if limit <= 0 {
		limit = 10
	}

	results, err := s.store.SearchNews(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return &types.EnrichmentResponse{
		Meta: types.EnrichmentMeta{
			ID:   generateID(),
			Node: "local",
			Ms:   time.Since(start).Milliseconds(),
		},
		Data: toEnrichmentResults(results),
	}, nil
}

// Index indexes a new small web entry.
func (s *Service) Index(ctx context.Context, entry *types.SmallWebEntry) error {
	return s.store.IndexEntry(ctx, entry)
}

// SeedSmallWeb seeds sample small web entries.
func (s *Service) SeedSmallWeb(ctx context.Context) error {
	return s.store.SeedSmallWeb(ctx)
}

// toEnrichmentResults converts store results to response format.
func toEnrichmentResults(results []*store.EnrichmentResult) []types.EnrichmentResult {
	out := make([]types.EnrichmentResult, 0, len(results))
	for _, r := range results {
		out = append(out, types.EnrichmentResult{
			Type:      r.Type,
			Rank:      r.Rank,
			URL:       r.URL,
			Title:     r.Title,
			Snippet:   r.Snippet,
			Published: r.Published,
		})
	}
	return out
}

// generateID generates a unique ID.
func generateID() string {
	return time.Now().Format("20060102150405.000")
}
