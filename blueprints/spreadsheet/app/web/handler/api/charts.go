package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Charts handles chart endpoints.
type Charts struct {
	charts    charts.API
	sheets    sheets.API
	workbooks workbooks.API
	getUserID func(*mizu.Ctx) string
}

// NewCharts creates a new Charts handler.
func NewCharts(charts charts.API, sheets sheets.API, workbooks workbooks.API, getUserID func(*mizu.Ctx) string) *Charts {
	return &Charts{
		charts:    charts,
		sheets:    sheets,
		workbooks: workbooks,
		getUserID: getUserID,
	}
}

// checkSheetAccess verifies the user has access to the sheet via workbook ownership.
// Returns true if access is granted, false if denied (response already written).
func (h *Charts) checkSheetAccess(c *mizu.Ctx, sheetID string) bool {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}

	sheet, err := h.sheets.GetByID(c.Request().Context(), sheetID)
	if err != nil {
		if err == sheets.ErrNotFound {
			c.JSON(http.StatusNotFound, map[string]string{"error": "sheet not found"})
		} else {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve sheet"})
		}
		return false
	}

	// SECURITY: Verify workbook ownership to prevent IDOR attacks
	wb, err := h.workbooks.GetByID(c.Request().Context(), sheet.WorkbookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify access"})
		return false
	}

	if wb.OwnerID != userID {
		c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
		return false
	}

	return true
}

// checkChartAccess verifies the user has access to the chart via workbook ownership.
// Returns the chart if access is granted, or nil if denied (response already written).
func (h *Charts) checkChartAccess(c *mizu.Ctx, chartID string) *charts.Chart {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return nil
	}

	chart, err := h.charts.GetByID(c.Request().Context(), chartID)
	if err != nil {
		if err == charts.ErrNotFound {
			c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		} else {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve chart"})
		}
		return nil
	}

	sheet, err := h.sheets.GetByID(c.Request().Context(), chart.SheetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify access"})
		return nil
	}

	wb, err := h.workbooks.GetByID(c.Request().Context(), sheet.WorkbookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify access"})
		return nil
	}

	if wb.OwnerID != userID {
		c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
		return nil
	}

	return chart
}

// Create creates a new chart.
func (h *Charts) Create(c *mizu.Ctx) error {
	var in charts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, in.SheetID) {
		return nil // Response already written
	}

	chart, err := h.charts.Create(c.Request().Context(), &in)
	if err != nil {
		if err == charts.ErrInvalidType {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid chart type"})
		}
		if err == charts.ErrEmptyDataRange {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "data range is required"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create chart"})
	}

	return c.JSON(http.StatusCreated, chart)
}

// Get retrieves a chart by ID.
func (h *Charts) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user has access to the chart
	chart := h.checkChartAccess(c, id)
	if chart == nil {
		return nil // Response already written
	}

	return c.JSON(http.StatusOK, chart)
}

// Update updates a chart.
func (h *Charts) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user has access to the chart
	if h.checkChartAccess(c, id) == nil {
		return nil // Response already written
	}

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
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update chart"})
	}

	return c.JSON(http.StatusOK, chart)
}

// Delete deletes a chart.
func (h *Charts) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user has access to the chart
	if h.checkChartAccess(c, id) == nil {
		return nil // Response already written
	}

	if err := h.charts.Delete(c.Request().Context(), id); err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete chart"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListBySheet lists charts in a sheet.
func (h *Charts) ListBySheet(c *mizu.Ctx) error {
	sheetID := c.Param("sheetId")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	chartList, err := h.charts.ListBySheet(c.Request().Context(), sheetID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list charts"})
	}

	return c.JSON(http.StatusOK, chartList)
}

// Duplicate duplicates a chart.
func (h *Charts) Duplicate(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user has access to the chart
	if h.checkChartAccess(c, id) == nil {
		return nil // Response already written
	}

	chart, err := h.charts.Duplicate(c.Request().Context(), id)
	if err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to duplicate chart"})
	}

	return c.JSON(http.StatusCreated, chart)
}

// GetData retrieves resolved chart data from cell values.
func (h *Charts) GetData(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user has access to the chart
	if h.checkChartAccess(c, id) == nil {
		return nil // Response already written
	}

	data, err := h.charts.GetData(c.Request().Context(), id)
	if err != nil {
		if err == charts.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "chart not found"})
		}
		if err == charts.ErrEmptyDataRange {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "chart has no data range"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get chart data"})
	}

	return c.JSON(http.StatusOK, data)
}
