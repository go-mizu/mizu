package rest

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/settings"
)

// Settings handles settings endpoints.
type Settings struct {
	settings settings.API
}

// NewSettings creates a new settings handler.
func NewSettings(settings settings.API) *Settings {
	return &Settings{settings: settings}
}

// GetAll retrieves all settings.
func (h *Settings) GetAll(c *mizu.Ctx) error {
	list, err := h.settings.GetAll(c.Context())
	if err != nil {
		return InternalError(c, "failed to get settings")
	}

	return OK(c, list)
}

// GetPublic retrieves public settings.
func (h *Settings) GetPublic(c *mizu.Ctx) error {
	list, err := h.settings.GetPublic(c.Context())
	if err != nil {
		return InternalError(c, "failed to get settings")
	}

	return OK(c, list)
}

// Get retrieves a setting by key.
func (h *Settings) Get(c *mizu.Ctx) error {
	key := c.Param("key")
	setting, err := h.settings.Get(c.Context(), key)
	if err != nil {
		if err == settings.ErrNotFound {
			return NotFound(c, "setting not found")
		}
		return InternalError(c, "failed to get setting")
	}

	return OK(c, setting)
}

// Set sets a setting value.
func (h *Settings) Set(c *mizu.Ctx) error {
	var in settings.SetIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	// If key is in URL, use it
	if key := c.Param("key"); key != "" {
		in.Key = key
	}

	setting, err := h.settings.Set(c.Context(), &in)
	if err != nil {
		if err == settings.ErrMissingKey {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to set setting")
	}

	return OK(c, setting)
}

// SetBulk sets multiple settings at once.
func (h *Settings) SetBulk(c *mizu.Ctx) error {
	var settings []*settings.SetIn
	if err := c.BindJSON(&settings, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.settings.SetBulk(c.Context(), settings); err != nil {
		return InternalError(c, "failed to set settings")
	}

	return OK(c, map[string]string{"message": "settings updated"})
}

// Delete deletes a setting.
func (h *Settings) Delete(c *mizu.Ctx) error {
	key := c.Param("key")

	if err := h.settings.Delete(c.Context(), key); err != nil {
		return InternalError(c, "failed to delete setting")
	}

	return OK(c, map[string]string{"message": "setting deleted"})
}
