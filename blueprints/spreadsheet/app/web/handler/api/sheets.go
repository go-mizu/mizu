package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Sheet handles sheet endpoints.
type Sheet struct {
	sheets    sheets.API
	workbooks workbooks.API
	getUserID func(*mizu.Ctx) string
}

// NewSheet creates a new Sheet handler.
func NewSheet(sheets sheets.API, workbooks workbooks.API, getUserID func(*mizu.Ctx) string) *Sheet {
	return &Sheet{
		sheets:    sheets,
		workbooks: workbooks,
		getUserID: getUserID,
	}
}

// checkSheetAccess verifies the user has access to the sheet via workbook ownership.
// Returns the sheet if access is granted, or nil if access is denied.
// When nil is returned, the response has already been written.
func (h *Sheet) checkSheetAccess(c *mizu.Ctx, sheetID string) *sheets.Sheet {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return nil
	}

	sheet, err := h.sheets.GetByID(c.Request().Context(), sheetID)
	if err != nil {
		if err == sheets.ErrNotFound {
			c.JSON(http.StatusNotFound, map[string]string{"error": "sheet not found"})
		} else {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve sheet"})
		}
		return nil
	}

	// SECURITY: Verify workbook ownership to prevent IDOR attacks
	wb, err := h.workbooks.GetByID(c.Request().Context(), sheet.WorkbookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify access"})
		return nil
	}

	if wb.OwnerID != userID {
		c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
		return nil
	}

	return sheet
}

// checkWorkbookAccess verifies the user has access to the workbook.
// Returns true if access is granted, false if denied (response already written).
func (h *Sheet) checkWorkbookAccess(c *mizu.Ctx, workbookID string) bool {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}

	wb, err := h.workbooks.GetByID(c.Request().Context(), workbookID)
	if err != nil {
		if err == workbooks.ErrNotFound {
			c.JSON(http.StatusNotFound, map[string]string{"error": "workbook not found"})
		} else {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify access"})
		}
		return false
	}

	if wb.OwnerID != userID {
		c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
		return false
	}

	return true
}

// Create creates a new sheet.
func (h *Sheet) Create(c *mizu.Ctx) error {
	var in sheets.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// SECURITY: Verify user owns the workbook before creating a sheet
	if !h.checkWorkbookAccess(c, in.WorkbookID) {
		return nil // Response already written
	}

	sheet, err := h.sheets.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create sheet"})
	}

	return c.JSON(http.StatusCreated, sheet)
}

// Get retrieves a sheet by ID.
func (h *Sheet) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user owns the workbook
	sheet := h.checkSheetAccess(c, id)
	if sheet == nil {
		return nil // Response already written
	}

	return c.JSON(http.StatusOK, sheet)
}

// Update updates a sheet.
func (h *Sheet) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	// SECURITY: Verify user owns the workbook
	if h.checkSheetAccess(c, id) == nil {
		return nil // Response already written
	}

	var in sheets.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	sheet, err := h.sheets.Update(c.Request().Context(), id, &in)
	if err != nil {
		if err == sheets.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "sheet not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update sheet"})
	}

	return c.JSON(http.StatusOK, sheet)
}

// Delete deletes a sheet.
func (h *Sheet) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	// SECURITY: Verify user owns the workbook
	sheet := h.checkSheetAccess(c, id)
	if sheet == nil {
		return nil // Response already written
	}

	// Check if this is the last sheet in the workbook
	allSheets, err := h.sheets.List(ctx, sheet.WorkbookID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list sheets"})
	}

	if len(allSheets) <= 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot delete the last sheet in a workbook"})
	}

	if err := h.sheets.Delete(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete sheet"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
