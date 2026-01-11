package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/dashboard"
	"github.com/go-mizu/blueprints/table/feature/views"
)

// Dashboard handles dashboard endpoints.
type Dashboard struct {
	dashboard *dashboard.Service
	views     views.API
	getUserID func(*mizu.Ctx) string
}

// NewDashboard creates a new dashboard handler.
func NewDashboard(dashboard *dashboard.Service, views views.API, getUserID func(*mizu.Ctx) string) *Dashboard {
	return &Dashboard{dashboard: dashboard, views: views, getUserID: getUserID}
}

// GetData retrieves aggregated data for dashboard widgets.
func (h *Dashboard) GetData(c *mizu.Ctx) error {
	viewID := c.Param("id")

	var req struct {
		WidgetIDs []string `json:"widget_ids"`
	}
	// Optional body
	_ = c.BindJSON(&req, 1<<20)

	data, err := h.dashboard.GetDashboardData(c.Context(), &dashboard.GetDataIn{
		ViewID:    viewID,
		WidgetIDs: req.WidgetIDs,
	})
	if err != nil {
		return InternalError(c, "failed to get dashboard data")
	}

	return OK(c, data)
}

// AddWidget adds a new widget to the dashboard.
func (h *Dashboard) AddWidget(c *mizu.Ctx) error {
	viewID := c.Param("id")

	var req struct {
		Widget dashboard.Widget `json:"widget"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	widget, err := h.dashboard.AddWidget(c.Context(), &dashboard.AddWidgetIn{
		ViewID: viewID,
		Widget: req.Widget,
	})
	if err != nil {
		return InternalError(c, "failed to add widget")
	}

	return Created(c, widget)
}

// UpdateWidget updates an existing widget.
func (h *Dashboard) UpdateWidget(c *mizu.Ctx) error {
	viewID := c.Param("id")
	widgetID := c.Param("widgetId")

	var req struct {
		Widget dashboard.Widget `json:"widget"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	widget, err := h.dashboard.UpdateWidget(c.Context(), &dashboard.UpdateWidgetIn{
		ViewID:   viewID,
		WidgetID: widgetID,
		Widget:   req.Widget,
	})
	if err != nil {
		return InternalError(c, "failed to update widget")
	}

	return OK(c, widget)
}

// DeleteWidget removes a widget from the dashboard.
func (h *Dashboard) DeleteWidget(c *mizu.Ctx) error {
	viewID := c.Param("id")
	widgetID := c.Param("widgetId")

	err := h.dashboard.DeleteWidget(c.Context(), &dashboard.DeleteWidgetIn{
		ViewID:   viewID,
		WidgetID: widgetID,
	})
	if err != nil {
		return InternalError(c, "failed to delete widget")
	}

	return NoContent(c)
}
