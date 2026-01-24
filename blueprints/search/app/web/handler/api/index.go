package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// IndexHandler handles index management
type IndexHandler struct {
	store store.Store
}

// NewIndexHandler creates a new index handler
func NewIndexHandler(s store.Store) *IndexHandler {
	return &IndexHandler{store: s}
}

// Stats returns index statistics
func (h *IndexHandler) Stats(c *mizu.Ctx) error {
	stats, err := h.store.Index().GetIndexStats(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, stats)
}

// Rebuild rebuilds the search index
func (h *IndexHandler) Rebuild(c *mizu.Ctx) error {
	if err := h.store.Index().RebuildIndex(c.Context()); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "index rebuilt"})
}
