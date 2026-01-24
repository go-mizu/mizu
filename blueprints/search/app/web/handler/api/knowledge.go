package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/store/postgres"
)

// KnowledgeHandler handles knowledge panel requests
type KnowledgeHandler struct {
	store *postgres.Store
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(store *postgres.Store) *KnowledgeHandler {
	return &KnowledgeHandler{store: store}
}

// GetEntity returns a knowledge panel for an entity
func (h *KnowledgeHandler) GetEntity(c *mizu.Ctx) error {
	query := c.Param("query")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query parameter required"})
	}

	panel, err := h.store.Knowledge().GetEntity(c.Context(), query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	if panel == nil {
		return c.JSON(404, map[string]string{"error": "entity not found"})
	}

	return c.JSON(200, panel)
}
