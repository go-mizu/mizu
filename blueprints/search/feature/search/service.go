package search

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/feature/instant"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// Service implements the search API.
type Service struct {
	store   Store
	instant *instant.Service
}

// NewService creates a new search service.
func NewService(s Store) *Service {
	return &Service{
		store:   s,
		instant: instant.NewService(),
	}
}

// Search performs a full-text search with options.
func (s *Service) Search(ctx context.Context, query string, opts store.SearchOptions) (*store.SearchResponse, error) {
	// Perform the search
	response, err := s.store.Search().Search(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	// Record for suggestions
	_ = s.store.Suggest().RecordQuery(ctx, query)

	// Try to detect instant answer
	if answer := s.instant.Detect(query); answer != nil {
		response.InstantAnswer = answer
	}

	// Try to get knowledge panel
	if panel, err := s.store.Knowledge().GetEntity(ctx, query); err == nil && panel != nil {
		response.KnowledgePanel = panel
	}

	// Get related searches
	if suggestions, err := s.store.Suggest().GetSuggestions(ctx, query, 5); err == nil {
		for _, sug := range suggestions {
			if sug.Text != query {
				response.RelatedSearches = append(response.RelatedSearches, sug.Text)
			}
		}
	}

	// Record in history
	_ = s.store.History().RecordSearch(ctx, &store.SearchHistory{
		Query:   query,
		Results: int(response.TotalResults),
	})

	return response, nil
}

// SearchImages searches for images.
func (s *Service) SearchImages(ctx context.Context, query string, opts store.SearchOptions) ([]store.ImageResult, error) {
	return s.store.Search().SearchImages(ctx, query, opts)
}

// SearchVideos searches for videos.
func (s *Service) SearchVideos(ctx context.Context, query string, opts store.SearchOptions) ([]store.VideoResult, error) {
	return s.store.Search().SearchVideos(ctx, query, opts)
}

// SearchNews searches for news articles.
func (s *Service) SearchNews(ctx context.Context, query string, opts store.SearchOptions) ([]store.NewsResult, error) {
	return s.store.Search().SearchNews(ctx, query, opts)
}
