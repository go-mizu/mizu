package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SettingsHandler handles user settings
type SettingsHandler struct {
	store store.Store
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(s store.Store) *SettingsHandler {
	return &SettingsHandler{store: s}
}

// Get returns user settings
func (h *SettingsHandler) Get(c *mizu.Ctx) error {
	settings, err := h.store.Preference().GetSettings(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, settings)
}

// Update updates user settings
func (h *SettingsHandler) Update(c *mizu.Ctx) error {
	var req store.SearchSettings

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.Preference().UpdateSettings(c.Context(), &req); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, &req)
}
