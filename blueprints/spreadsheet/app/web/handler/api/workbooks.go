package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Workbook handles workbook endpoints.
type Workbook struct {
	workbooks workbooks.API
	sheets    sheets.API
	getUserID func(*mizu.Ctx) string
}

// NewWorkbook creates a new Workbook handler.
func NewWorkbook(workbooks workbooks.API, sheets sheets.API, getUserID func(*mizu.Ctx) string) *Workbook {
	return &Workbook{
		workbooks: workbooks,
		sheets:    sheets,
		getUserID: getUserID,
	}
}

// List lists workbooks for the current user.
func (h *Workbook) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	list, err := h.workbooks.List(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, list)
}

// Create creates a new workbook.
func (h *Workbook) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var in workbooks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	in.OwnerID = userID
	in.CreatedBy = userID

	wb, err := h.workbooks.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create a default sheet
	sheet, err := h.sheets.Create(c.Request().Context(), &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Sheet1",
	})
	if err != nil {
		// Rollback workbook creation
		h.workbooks.Delete(c.Request().Context(), wb.ID)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"workbook": wb,
		"sheet":    sheet,
	})
}

// Get retrieves a workbook by ID.
func (h *Workbook) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	wb, err := h.workbooks.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == workbooks.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "workbook not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Get sheets for this workbook
	sheetList, err := h.sheets.List(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workbook": wb,
		"sheets":   sheetList,
	})
}

// Update updates a workbook.
func (h *Workbook) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in workbooks.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	wb, err := h.workbooks.Update(c.Request().Context(), id, &in)
	if err != nil {
		if err == workbooks.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "workbook not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, wb)
}

// Delete deletes a workbook.
func (h *Workbook) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.workbooks.Delete(c.Request().Context(), id); err != nil {
		if err == workbooks.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "workbook not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListSheets lists sheets in a workbook.
func (h *Workbook) ListSheets(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.sheets.List(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, list)
}
