package api

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
)

// SettingsHandler handles settings API endpoints.
type SettingsHandler struct {
	store store.Store
}

// NewSettingsHandler creates a new settings handler.
func NewSettingsHandler(st store.Store) *SettingsHandler {
	return &SettingsHandler{store: st}
}

// Get returns the current user settings.
func (h *SettingsHandler) Get(c *mizu.Ctx) error {
	settings, err := h.store.GetSettings(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get settings"})
	}

	return c.JSON(http.StatusOK, settings)
}

// Update saves user settings.
func (h *SettingsHandler) Update(c *mizu.Ctx) error {
	var settings types.Settings
	if err := c.BindJSON(&settings, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.UpdateSettings(c.Context(), &settings); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update settings"})
	}

	// Return updated settings
	updated, err := h.store.GetSettings(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch updated settings"})
	}

	return c.JSON(http.StatusOK, updated)
}
