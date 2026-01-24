package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/store/postgres"
)

// PreferencesHandler handles user preferences
type PreferencesHandler struct {
	store *postgres.Store
}

// NewPreferencesHandler creates a new preferences handler
func NewPreferencesHandler(store *postgres.Store) *PreferencesHandler {
	return &PreferencesHandler{store: store}
}

// List returns all user preferences
func (h *PreferencesHandler) List(c *mizu.Ctx) error {
	prefs, err := h.store.Preference().GetPreferences(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, prefs)
}

// Set creates or updates a preference
func (h *PreferencesHandler) Set(c *mizu.Ctx) error {
	var req struct {
		Domain string `json:"domain"`
		Action string `json:"action"` // upvote, downvote, block
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Domain == "" || req.Action == "" {
		return c.JSON(400, map[string]string{"error": "domain and action required"})
	}

	// Validate action
	if req.Action != "upvote" && req.Action != "downvote" && req.Action != "block" {
		return c.JSON(400, map[string]string{"error": "action must be upvote, downvote, or block"})
	}

	pref := &store.UserPreference{
		Domain: req.Domain,
		Action: req.Action,
	}

	if err := h.store.Preference().SetPreference(c.Context(), pref); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, pref)
}

// Delete removes a preference
func (h *PreferencesHandler) Delete(c *mizu.Ctx) error {
	domain := c.Param("domain")
	if domain == "" {
		return c.JSON(400, map[string]string{"error": "domain required"})
	}

	if err := h.store.Preference().DeletePreference(c.Context(), domain); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "deleted"})
}
