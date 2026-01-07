package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
)

// Charts handles chart endpoints.
type Charts struct {
	charts    charts.API
	getUserID func(*mizu.Ctx) string
}

// NewCharts creates a new Charts handler.
func NewCharts(charts charts.API, getUserID func(*mizu.Ctx) string) *Charts {
	return &Charts{
		charts:    charts,
		getUserID: getUserID,
	}
}

// Create creates a new chart.
func (h *Charts) Create(c *mizu.Ctx) error {
	var in charts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	chart, err := h.charts.Create(c.Request().Context(), &in)
	if err != nil {
		if err == charts.ErrInvalidType {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid chart type"})
		}
		if err == charts.ErrEmptyDataRange {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "data range is required"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, chart)
}

// Get retrieves a chart by ID.
func (h *Charts) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	chart, err := h.charts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, chart)
}

// Update updates a chart.
func (h *Charts) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in charts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	chart, err := h.charts.Update(c.Request().Context(), id, &in)
	if err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		if err == charts.ErrInvalidType {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid chart type"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, chart)
}

// Delete deletes a chart.
func (h *Charts) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.charts.Delete(c.Request().Context(), id); err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListBySheet lists charts in a sheet.
func (h *Charts) ListBySheet(c *mizu.Ctx) error {
	sheetID := c.Param("sheetId")

	chartList, err := h.charts.ListBySheet(c.Request().Context(), sheetID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, chartList)
}

// Duplicate duplicates a chart.
func (h *Charts) Duplicate(c *mizu.Ctx) error {
	id := c.Param("id")

	chart, err := h.charts.Duplicate(c.Request().Context(), id)
	if err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, chart)
}

// GetData retrieves resolved chart data from cell values.
func (h *Charts) GetData(c *mizu.Ctx) error {
	id := c.Param("id")

	data, err := h.charts.GetData(c.Request().Context(), id)
	if err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		if err == charts.ErrEmptyDataRange {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "chart has no data range"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, data)
}
