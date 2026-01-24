package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/suggest"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SuggestHandler handles autocomplete suggestions
type SuggestHandler struct {
	service *suggest.Service
}

// NewSuggestHandler creates a new suggest handler
func NewSuggestHandler(s store.Store) *SuggestHandler {
	return &SuggestHandler{service: suggest.NewService(s)}
}

// Suggest returns autocomplete suggestions
func (h *SuggestHandler) Suggest(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.JSON(200, []any{})
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	suggestions, err := h.service.GetSuggestions(c.Context(), q, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, suggestions)
}

// Trending returns trending search queries
func (h *SuggestHandler) Trending(c *mizu.Ctx) error {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	queries, err := h.service.GetTrending(c.Context(), limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, queries)
}
