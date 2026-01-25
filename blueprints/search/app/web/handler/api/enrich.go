package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/enrich"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// EnrichHandler handles enrichment API requests.
type EnrichHandler struct {
	service *enrich.Service
}

// NewEnrichHandler creates a new enrichment handler.
func NewEnrichHandler(st store.Store) *EnrichHandler {
	return &EnrichHandler{
		service: enrich.NewService(st.SmallWeb()),
	}
}

// SearchWeb handles GET /api/enrich/web requests.
func (h *EnrichHandler) SearchWeb(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	resp, err := h.service.SearchWeb(c.Context(), query, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, resp)
}

// SearchNews handles GET /api/enrich/news requests.
func (h *EnrichHandler) SearchNews(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	resp, err := h.service.SearchNews(c.Context(), query, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, resp)
}
