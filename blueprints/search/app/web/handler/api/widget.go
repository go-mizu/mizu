package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/widget"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// WidgetHandler handles widget-related API requests.
type WidgetHandler struct {
	service *widget.Service
}

// NewWidgetHandler creates a new widget handler.
func NewWidgetHandler(st store.Store) *WidgetHandler {
	return &WidgetHandler{
		service: widget.NewService(st.Widget()),
	}
}

// GetSettings returns widget settings for the current user.
func (h *WidgetHandler) GetSettings(c *mizu.Ctx) error {
	// For now, use a default user ID
	userID := "default"

	settings, err := h.service.GetSettings(c.Context(), userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, settings)
}

// UpdateSettings updates a widget setting.
func (h *WidgetHandler) UpdateSettings(c *mizu.Ctx) error {
	var req struct {
		WidgetType string `json:"widget_type"`
		Enabled    bool   `json:"enabled"`
		Position   int    `json:"position"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.WidgetType == "" {
		return c.JSON(400, map[string]string{"error": "widget_type required"})
	}

	setting := &types.WidgetSetting{
		UserID:     "default",
		WidgetType: types.WidgetType(req.WidgetType),
		Enabled:    req.Enabled,
		Position:   req.Position,
	}

	if err := h.service.UpdateSetting(c.Context(), setting); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, setting)
}

// GetCheatSheet returns a programming cheat sheet.
func (h *WidgetHandler) GetCheatSheet(c *mizu.Ctx) error {
	language := c.Param("language")
	if language == "" {
		return c.JSON(400, map[string]string{"error": "language required"})
	}

	sheet, err := h.service.GetCheatSheet(c.Context(), language)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	if sheet == nil {
		return c.JSON(404, map[string]string{"error": "cheat sheet not found"})
	}

	return c.JSON(200, sheet)
}

// ListCheatSheets returns all available cheat sheets.
func (h *WidgetHandler) ListCheatSheets(c *mizu.Ctx) error {
	sheets, err := h.service.ListCheatSheets(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, sheets)
}

// GetRelated returns related searches for a query.
func (h *WidgetHandler) GetRelated(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	related, err := h.service.GetRelatedSearches(c.Context(), query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"query":   query,
		"related": related,
	})
}
