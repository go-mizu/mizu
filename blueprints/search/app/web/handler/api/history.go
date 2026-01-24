package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/store/postgres"
)

// HistoryHandler handles search history
type HistoryHandler struct {
	store *postgres.Store
}

// NewHistoryHandler creates a new history handler
func NewHistoryHandler(store *postgres.Store) *HistoryHandler {
	return &HistoryHandler{store: store}
}

// List returns search history
func (h *HistoryHandler) List(c *mizu.Ctx) error {
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	history, err := h.store.History().GetHistory(c.Context(), limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, history)
}

// Clear clears all search history
func (h *HistoryHandler) Clear(c *mizu.Ctx) error {
	if err := h.store.History().ClearHistory(c.Context()); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "cleared"})
}

// Delete deletes a single history entry
func (h *HistoryHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}

	if err := h.store.History().DeleteHistoryEntry(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "deleted"})
}
