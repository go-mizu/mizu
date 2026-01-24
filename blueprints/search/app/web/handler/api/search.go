package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/search"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SearchHandler handles search API requests
type SearchHandler struct {
	service *search.Service
}

// NewSearchHandler creates a new search handler with default configuration.
// For SearXNG integration, use NewSearchHandlerWithConfig.
func NewSearchHandler(s store.Store) *SearchHandler {
	return &SearchHandler{service: search.NewServiceWithDefaults(s)}
}

// NewSearchHandlerWithConfig creates a new search handler with custom configuration.
func NewSearchHandlerWithConfig(cfg search.ServiceConfig) *SearchHandler {
	return &SearchHandler{service: search.NewService(cfg)}
}

// Search handles the main search endpoint
func (h *SearchHandler) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query parameter 'q' is required"})
	}

	opts := parseSearchOptions(c)

	// Perform search via service
	response, err := h.service.Search(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, response)
}

// SearchImages handles image search
func (h *SearchHandler) SearchImages(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query parameter 'q' is required"})
	}

	opts := parseSearchOptions(c)

	results, err := h.service.SearchImages(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Estimate total results for pagination
	totalResults := estimateTotal(len(results), opts.Page, opts.PerPage)

	return c.JSON(200, map[string]any{
		"query":         query,
		"results":       results,
		"total_results": totalResults,
		"page":          opts.Page,
		"per_page":      opts.PerPage,
	})
}

// SearchVideos handles video search
func (h *SearchHandler) SearchVideos(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query parameter 'q' is required"})
	}

	opts := parseSearchOptions(c)

	results, err := h.service.SearchVideos(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Estimate total results for pagination
	totalResults := estimateTotal(len(results), opts.Page, opts.PerPage)

	return c.JSON(200, map[string]any{
		"query":         query,
		"results":       results,
		"total_results": totalResults,
		"page":          opts.Page,
		"per_page":      opts.PerPage,
	})
}

// SearchNews handles news search
func (h *SearchHandler) SearchNews(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query parameter 'q' is required"})
	}

	opts := parseSearchOptions(c)

	results, err := h.service.SearchNews(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Estimate total results for pagination
	totalResults := estimateTotal(len(results), opts.Page, opts.PerPage)

	return c.JSON(200, map[string]any{
		"query":         query,
		"results":       results,
		"total_results": totalResults,
		"page":          opts.Page,
		"per_page":      opts.PerPage,
	})
}

// estimateTotal estimates total results for pagination when the exact count is unknown.
// If we got a full page of results, assume there are more pages.
// Otherwise, this is likely the last page.
func estimateTotal(resultCount, page, perPage int) int {
	if perPage <= 0 {
		perPage = 10
	}
	if resultCount >= perPage {
		// Got a full page, estimate at least 10 pages
		return perPage * 10
	}
	// Partial page - calculate actual total
	return (page-1)*perPage + resultCount
}

// parseSearchOptions extracts search options from query parameters
func parseSearchOptions(c *mizu.Ctx) store.SearchOptions {
	opts := store.SearchOptions{
		Page:    1,
		PerPage: 10,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			opts.Page = p
		}
	}

	if perPage := c.Query("per_page"); perPage != "" {
		if pp, err := strconv.Atoi(perPage); err == nil && pp > 0 && pp <= 50 {
			opts.PerPage = pp
		}
	}

	opts.TimeRange = c.Query("time")
	opts.Region = c.Query("region")
	opts.Language = c.Query("lang")
	opts.SafeSearch = c.Query("safe")
	opts.Site = c.Query("site")
	opts.ExcludeSite = c.Query("exclude_site")
	opts.FileType = c.Query("filetype")
	opts.Lens = c.Query("lens")

	if c.Query("verbatim") == "true" {
		opts.Verbatim = true
	}

	// Cache control options
	if c.Query("refetch") == "true" {
		opts.Refetch = true
	}

	if version := c.Query("version"); version != "" {
		if v, err := strconv.Atoi(version); err == nil && v > 0 {
			opts.Version = v
		}
	}

	return opts
}
