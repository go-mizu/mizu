package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
)

// Sheet handles sheet endpoints.
type Sheet struct {
	sheets    sheets.API
	getUserID func(*mizu.Ctx) string
}

// NewSheet creates a new Sheet handler.
func NewSheet(sheets sheets.API, getUserID func(*mizu.Ctx) string) *Sheet {
	return &Sheet{
		sheets:    sheets,
		getUserID: getUserID,
	}
}

// Create creates a new sheet.
func (h *Sheet) Create(c *mizu.Ctx) error {
	var in sheets.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	sheet, err := h.sheets.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, sheet)
}

// Get retrieves a sheet by ID.
func (h *Sheet) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	sheet, err := h.sheets.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == sheets.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "sheet not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sheet)
}

// Update updates a sheet.
func (h *Sheet) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in sheets.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	sheet, err := h.sheets.Update(c.Request().Context(), id, &in)
	if err != nil {
		if err == sheets.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "sheet not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sheet)
}

// Delete deletes a sheet.
func (h *Sheet) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	// Get the sheet to find its workbook
	sheet, err := h.sheets.GetByID(ctx, id)
	if err != nil {
		if err == sheets.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "sheet not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Check if this is the last sheet in the workbook
	allSheets, err := h.sheets.List(ctx, sheet.WorkbookID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(allSheets) <= 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot delete the last sheet in a workbook"})
	}

	if err := h.sheets.Delete(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
