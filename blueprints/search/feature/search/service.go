package search

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/instant"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// Service implements the search API using an engine and cache.
type Service struct {
	engine  engine.Engine
	cache   *Cache
	store   Store
	instant *instant.Service
}

// ServiceConfig contains configuration for the search service.
type ServiceConfig struct {
	Engine    engine.Engine
	Cache     *Cache
	Store     Store
	CacheTTL  time.Duration
}

// NewService creates a new search service with the given engine.
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		engine:  cfg.Engine,
		cache:   cfg.Cache,
		store:   cfg.Store,
		instant: instant.NewService(),
	}
}

// NewServiceWithDefaults creates a service with sensible defaults for backwards compatibility.
func NewServiceWithDefaults(s Store) *Service {
	return &Service{
		engine:  nil, // No engine, falls back to store
		cache:   nil, // No caching
		store:   s,
		instant: instant.NewService(),
	}
}

// Search performs a full-text search with options.
func (s *Service) Search(ctx context.Context, query string, opts store.SearchOptions) (*store.SearchResponse, error) {
	start := time.Now()

	slog.Debug("search service starting", "query", query, "page", opts.Page)

	// Convert store options to engine options
	engineOpts := toEngineOptions(opts, engine.CategoryGeneral)
	cacheOpts := CacheOptions{Refetch: opts.Refetch, Version: opts.Version}

	// Try cache first if available
	if s.cache != nil {
		if cached, ok := s.cache.Get(ctx, query, engine.CategoryGeneral, engineOpts, cacheOpts); ok {
			slog.Debug("search cache hit", "query", query, "results", len(cached.Results))
			response := toStoreResponse(cached)
			// Still enrich with instant answer and knowledge panel
			s.enrichResponse(ctx, query, response)
			// Update search time to reflect cache hit
			response.SearchTimeMs = float64(time.Since(start).Milliseconds())
			return response, nil
		}
	}

	var response *store.SearchResponse

	// Use engine if available, otherwise fall back to store
	if s.engine != nil {
		engineResp, err := s.engine.Search(ctx, query, engineOpts)
		if err != nil {
			slog.Error("search engine error",
				"query", query,
				"engine", s.engine.Name(),
				"error", err,
			)
			return nil, err
		}

		// Warn if no results from engine
		if engineResp == nil || len(engineResp.Results) == 0 {
			slog.Warn("search engine returned empty results",
				"query", query,
				"engine", s.engine.Name(),
			)
		}

		// Cache the response
		if s.cache != nil {
			if err := s.cache.Set(ctx, query, engine.CategoryGeneral, engineOpts, engineResp); err != nil {
				slog.Warn("failed to cache search results",
					"query", query,
					"error", err,
				)
			}
		}

		response = toStoreResponse(engineResp)
	} else {
		slog.Debug("no search engine configured, falling back to store", "query", query)
		// Fall back to store-based search
		resp, err := s.store.Search().Search(ctx, query, opts)
		if err != nil {
			slog.Error("store search error", "query", query, "error", err)
			return nil, err
		}
		response = resp
	}

	// Enrich response with instant answers, knowledge panels, etc.
	s.enrichResponse(ctx, query, response)

	// Record search time
	response.SearchTimeMs = float64(time.Since(start).Milliseconds())

	// Record for suggestions - log errors instead of ignoring
	if err := s.store.Suggest().RecordQuery(ctx, query); err != nil {
		slog.Warn("failed to record query for suggestions", "query", query, "error", err)
	}

	// Record in history - log errors instead of ignoring
	if err := s.store.History().RecordSearch(ctx, &store.SearchHistory{
		Query:   query,
		Results: int(response.TotalResults),
	}); err != nil {
		slog.Warn("failed to record search history", "query", query, "error", err)
	}

	slog.Debug("search service completed",
		"query", query,
		"results", len(response.Results),
		"duration_ms", response.SearchTimeMs,
	)

	return response, nil
}

// enrichResponse adds instant answers, knowledge panels, and related searches.
func (s *Service) enrichResponse(ctx context.Context, query string, response *store.SearchResponse) {
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
}

// SearchImages searches for images.
func (s *Service) SearchImages(ctx context.Context, query string, opts store.SearchOptions) ([]store.ImageResult, error) {
	// Convert store options to engine options
	engineOpts := toEngineOptions(opts, engine.CategoryImages)
	cacheOpts := CacheOptions{Refetch: opts.Refetch, Version: opts.Version}

	// Try cache first if available
	if s.cache != nil {
		if cached, ok := s.cache.Get(ctx, query, engine.CategoryImages, engineOpts, cacheOpts); ok {
			return toStoreImageResults(cached.Results), nil
		}
	}

	// Use engine if available
	if s.engine != nil {
		engineResp, err := s.engine.Search(ctx, query, engineOpts)
		if err != nil {
			return nil, err
		}

		// Cache the response
		if s.cache != nil {
			_ = s.cache.Set(ctx, query, engine.CategoryImages, engineOpts, engineResp)
		}

		return toStoreImageResults(engineResp.Results), nil
	}

	// Fall back to store
	return s.store.Search().SearchImages(ctx, query, opts)
}

// SearchVideos searches for videos.
func (s *Service) SearchVideos(ctx context.Context, query string, opts store.SearchOptions) ([]store.VideoResult, error) {
	// Convert store options to engine options
	engineOpts := toEngineOptions(opts, engine.CategoryVideos)
	cacheOpts := CacheOptions{Refetch: opts.Refetch, Version: opts.Version}

	// Try cache first if available
	if s.cache != nil {
		if cached, ok := s.cache.Get(ctx, query, engine.CategoryVideos, engineOpts, cacheOpts); ok {
			return toStoreVideoResults(cached.Results), nil
		}
	}

	// Use engine if available
	if s.engine != nil {
		engineResp, err := s.engine.Search(ctx, query, engineOpts)
		if err != nil {
			return nil, err
		}

		// Cache the response
		if s.cache != nil {
			_ = s.cache.Set(ctx, query, engine.CategoryVideos, engineOpts, engineResp)
		}

		return toStoreVideoResults(engineResp.Results), nil
	}

	// Fall back to store
	return s.store.Search().SearchVideos(ctx, query, opts)
}

// SearchNews searches for news articles.
func (s *Service) SearchNews(ctx context.Context, query string, opts store.SearchOptions) ([]store.NewsResult, error) {
	// Convert store options to engine options
	engineOpts := toEngineOptions(opts, engine.CategoryNews)
	cacheOpts := CacheOptions{Refetch: opts.Refetch, Version: opts.Version}

	// Try cache first if available
	if s.cache != nil {
		if cached, ok := s.cache.Get(ctx, query, engine.CategoryNews, engineOpts, cacheOpts); ok {
			return toStoreNewsResults(cached.Results), nil
		}
	}

	// Use engine if available
	if s.engine != nil {
		engineResp, err := s.engine.Search(ctx, query, engineOpts)
		if err != nil {
			return nil, err
		}

		// Cache the response
		if s.cache != nil {
			_ = s.cache.Set(ctx, query, engine.CategoryNews, engineOpts, engineResp)
		}

		return toStoreNewsResults(engineResp.Results), nil
	}

	// Fall back to store
	return s.store.Search().SearchNews(ctx, query, opts)
}

// toEngineOptions converts store.SearchOptions to engine.SearchOptions.
func toEngineOptions(opts store.SearchOptions, category engine.Category) engine.SearchOptions {
	safeSearch := 0
	switch opts.SafeSearch {
	case "off":
		safeSearch = 0
	case "moderate":
		safeSearch = 1
	case "strict":
		safeSearch = 2
	}

	return engine.SearchOptions{
		Category:   category,
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		TimeRange:  opts.TimeRange,
		Language:   opts.Language,
		Region:     opts.Region,
		SafeSearch: safeSearch,
	}
}

// toStoreResponse converts an engine.SearchResponse to store.SearchResponse.
func toStoreResponse(resp *engine.SearchResponse) *store.SearchResponse {
	results := make([]types.SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		result := types.SearchResult{
			URL:     r.URL,
			Title:   r.Title,
			Snippet: r.Content,
			Domain:  extractDomain(r.URL),
			Score:   r.Score,
			Engine:  r.Engine,
			Engines: r.Engines,
		}
		// Add published date if available
		if !r.PublishedAt.IsZero() {
			result.Published = &r.PublishedAt
		}
		// Add thumbnail if available
		if r.ThumbnailURL != "" {
			result.Thumbnail = &types.Thumbnail{
				URL: r.ThumbnailURL,
			}
		}
		results = append(results, result)
	}

	return &store.SearchResponse{
		Query:          resp.Query,
		CorrectedQuery: resp.CorrectedQuery,
		TotalResults:   resp.TotalResults,
		Results:        results,
		Suggestions:    resp.Suggestions,
		SearchTimeMs:   resp.SearchTimeMs,
		Page:           resp.Page,
		PerPage:        resp.PerPage,
	}
}

// toStoreImageResults converts engine results to store image results.
func toStoreImageResults(results []engine.Result) []store.ImageResult {
	images := make([]store.ImageResult, 0, len(results))
	for i, r := range results {
		images = append(images, store.ImageResult{
			ID:           strconv.Itoa(i),
			URL:          r.ImageURL,
			ThumbnailURL: r.ThumbnailURL,
			Title:        r.Title,
			SourceURL:    r.URL,
			SourceDomain: r.Source,
			Format:       r.ImgFormat,
			Engine:       r.Engine,
		})
	}
	return images
}

// toStoreVideoResults converts engine results to store video results.
func toStoreVideoResults(results []engine.Result) []store.VideoResult {
	videos := make([]store.VideoResult, 0, len(results))
	for i, r := range results {
		videos = append(videos, store.VideoResult{
			ID:           strconv.Itoa(i),
			URL:          r.URL,
			ThumbnailURL: r.ThumbnailURL,
			Title:        r.Title,
			Description:  r.Content,
			PublishedAt:  r.PublishedAt,
			EmbedURL:     r.EmbedURL,
			Engine:       r.Engine,
		})
	}
	return videos
}

// toStoreNewsResults converts engine results to store news results.
func toStoreNewsResults(results []engine.Result) []store.NewsResult {
	news := make([]store.NewsResult, 0, len(results))
	for i, r := range results {
		news = append(news, store.NewsResult{
			ID:          strconv.Itoa(i),
			URL:         r.URL,
			Title:       r.Title,
			Snippet:     r.Content,
			Source:      r.Source,
			ImageURL:    r.ThumbnailURL,
			PublishedAt: r.PublishedAt,
			Engine:      r.Engine,
		})
	}
	return news
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) string {
	// Simple extraction - could use net/url for more robust parsing
	if len(rawURL) < 8 {
		return ""
	}
	// Skip protocol
	start := 0
	if rawURL[:8] == "https://" {
		start = 8
	} else if rawURL[:7] == "http://" {
		start = 7
	}
	// Find end of domain
	end := start
	for end < len(rawURL) && rawURL[end] != '/' && rawURL[end] != '?' {
		end++
	}
	return rawURL[start:end]
}
