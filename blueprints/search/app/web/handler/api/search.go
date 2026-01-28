package api

import (
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/bang"
	"github.com/go-mizu/mizu/blueprints/search/feature/search"
	"github.com/go-mizu/mizu/blueprints/search/feature/widget"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SearchHandler handles search API requests
type SearchHandler struct {
	service       *search.Service
	bangService   *bang.Service
	widgetService *widget.Service
}

// NewSearchHandler creates a new search handler with default configuration.
// For SearXNG integration, use NewSearchHandlerWithConfig.
func NewSearchHandler(s store.Store) *SearchHandler {
	return &SearchHandler{
		service:       search.NewServiceWithDefaults(s),
		bangService:   bang.NewService(s.Bang()),
		widgetService: widget.NewService(s.Widget()),
	}
}

// NewSearchHandlerWithConfig creates a new search handler with custom configuration.
// Pass nil for fullStore if you don't need bang/widget integration.
func NewSearchHandlerWithConfig(cfg search.ServiceConfig, fullStore store.Store) *SearchHandler {
	h := &SearchHandler{service: search.NewService(cfg)}
	if fullStore != nil {
		h.bangService = bang.NewService(fullStore.Bang())
		h.widgetService = widget.NewService(fullStore.Widget())
	}
	return h
}

// Service returns the underlying search service.
func (h *SearchHandler) Service() *search.Service {
	return h.service
}

// Search handles the main search endpoint
func (h *SearchHandler) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query parameter 'q' is required"})
	}

	// Parse bangs from query
	if h.bangService != nil {
		bangResult, err := h.bangService.Parse(c.Context(), query)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}

		// Handle external bang redirect
		if bangResult.RedirectURL != "" {
			return c.JSON(200, map[string]any{
				"redirect": bangResult.RedirectURL,
				"bang":     bangResult.Bang,
			})
		}

		// Handle internal bangs
		if bangResult.Internal {
			switch bangResult.Category {
			case "images":
				return h.searchImagesWithQuery(c, bangResult.Query)
			case "news":
				return h.searchNewsWithQuery(c, bangResult.Query)
			case "videos":
				return h.searchVideosWithQuery(c, bangResult.Query)
			case "maps":
				// For maps, redirect to OpenStreetMap
				return c.JSON(200, map[string]any{
					"redirect": "https://www.openstreetmap.org/search?query=" + bangResult.Query,
					"category": "maps",
				})
			case "ai":
				// AI mode handled separately via /api/ai endpoints
				return c.JSON(200, map[string]any{
					"ai_mode":  true,
					"query":    bangResult.Query,
					"redirect": "/ai?q=" + bangResult.Query,
				})
			case "summarize":
				return c.JSON(200, map[string]any{
					"summarize": true,
					"query":     bangResult.Query,
					"redirect":  "/api/summarize?url=" + bangResult.Query,
				})
			case "lucky":
				// "Feeling lucky" - will return first result
				opts := parseSearchOptions(c)
				opts.PerPage = 1
				response, err := h.service.Search(c.Context(), bangResult.Query, opts)
				if err != nil {
					return c.JSON(500, map[string]string{"error": err.Error()})
				}
				if len(response.Results) > 0 {
					return c.JSON(200, map[string]any{
						"redirect": response.Results[0].URL,
						"lucky":    true,
					})
				}
				// No results, fall through to normal search
				query = bangResult.Query
			default:
				// Check for time filter
				if strings.HasPrefix(bangResult.Category, "time:") {
					timeRange := strings.TrimPrefix(bangResult.Category, "time:")
					query = bangResult.Query
					// Set time range in query params for parseSearchOptions to pick up
					c.Request().URL.RawQuery += "&time=" + timeRange
				} else {
					query = bangResult.Query
				}
			}
		} else {
			query = bangResult.Query
		}
	}

	opts := parseSearchOptions(c)

	// Perform search via service
	response, err := h.service.Search(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Generate widgets for the response
	if h.widgetService != nil {
		widgets := h.widgetService.GenerateWidgets(c.Context(), query, response.Results)
		if len(widgets) > 0 {
			response.Widgets = widgets
		}
	}

	// Set HasMore based on results count
	response.HasMore = len(response.Results) >= opts.PerPage

	return c.JSON(200, response)
}

// searchImagesWithQuery handles image search from bang redirect
func (h *SearchHandler) searchImagesWithQuery(c *mizu.Ctx, query string) error {
	opts := parseSearchOptions(c)
	results, err := h.service.SearchImages(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	totalResults := estimateTotal(len(results), opts.Page, opts.PerPage)
	return c.JSON(200, map[string]any{
		"query":         query,
		"results":       results,
		"total_results": totalResults,
		"page":          opts.Page,
		"per_page":      opts.PerPage,
		"category":      "images",
	})
}

// searchNewsWithQuery handles news search from bang redirect
func (h *SearchHandler) searchNewsWithQuery(c *mizu.Ctx, query string) error {
	opts := parseSearchOptions(c)
	results, err := h.service.SearchNews(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	totalResults := estimateTotal(len(results), opts.Page, opts.PerPage)
	return c.JSON(200, map[string]any{
		"query":         query,
		"results":       results,
		"total_results": totalResults,
		"page":          opts.Page,
		"per_page":      opts.PerPage,
		"category":      "news",
	})
}

// searchVideosWithQuery handles video search from bang redirect
func (h *SearchHandler) searchVideosWithQuery(c *mizu.Ctx, query string) error {
	opts := parseSearchOptions(c)
	results, err := h.service.SearchVideos(c.Context(), query, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	totalResults := estimateTotal(len(results), opts.Page, opts.PerPage)
	return c.JSON(200, map[string]any{
		"query":         query,
		"results":       results,
		"total_results": totalResults,
		"page":          opts.Page,
		"per_page":      opts.PerPage,
		"category":      "videos",
	})
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

	// Date filtering (Kagi-style)
	opts.DateBefore = c.Query("before")
	opts.DateAfter = c.Query("after")

	// Safe search level (0=off, 1=moderate, 2=strict)
	if safeLevel := c.Query("safe_level"); safeLevel != "" {
		if sl, err := strconv.Atoi(safeLevel); err == nil && sl >= 0 && sl <= 2 {
			opts.SafeLevel = sl
		}
	}

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
