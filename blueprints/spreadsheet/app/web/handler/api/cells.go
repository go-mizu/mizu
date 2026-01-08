package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// Cell handles cell endpoints.
type Cell struct {
	cells     cells.API
	sheets    sheets.API
	workbooks workbooks.API
	getUserID func(*mizu.Ctx) string
}

// NewCell creates a new Cell handler.
func NewCell(cells cells.API, sheets sheets.API, workbooks workbooks.API, getUserID func(*mizu.Ctx) string) *Cell {
	return &Cell{
		cells:     cells,
		sheets:    sheets,
		workbooks: workbooks,
		getUserID: getUserID,
	}
}

// checkSheetAccess verifies the user has access to the sheet via workbook ownership.
// Returns true if access is granted, false if denied (response already written).
func (h *Cell) checkSheetAccess(c *mizu.Ctx, sheetID string) bool {
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

// Get retrieves a cell by position.
func (h *Cell) Get(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	row, _ := strconv.Atoi(c.Param("row"))
	col, _ := strconv.Atoi(c.Param("col"))

	cell, err := h.cells.Get(c.Request().Context(), sheetID, row, col)
	if err != nil {
		if err == cells.ErrNotFound {
			// Return empty cell for non-existent cells
			return c.JSON(http.StatusOK, &cells.Cell{
				SheetID: sheetID,
				Row:     row,
				Col:     col,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve cell"})
	}

	return c.JSON(http.StatusOK, cell)
}

// GetRange retrieves cells in a range.
func (h *Cell) GetRange(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	startRow := 0
	startCol := 0
	endRow := 100
	endCol := 26

	if v := c.Query("startRow"); v != "" {
		startRow, _ = strconv.Atoi(v)
	}
	if v := c.Query("startCol"); v != "" {
		startCol, _ = strconv.Atoi(v)
	}
	if v := c.Query("endRow"); v != "" {
		endRow, _ = strconv.Atoi(v)
	}
	if v := c.Query("endCol"); v != "" {
		endCol, _ = strconv.Atoi(v)
	}

	cellList, err := h.cells.GetRange(c.Request().Context(), sheetID, startRow, startCol, endRow, endCol)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve cells"})
	}

	return c.JSON(http.StatusOK, cellList)
}

// Set sets a cell value.
func (h *Cell) Set(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	row, _ := strconv.Atoi(c.Param("row"))
	col, _ := strconv.Atoi(c.Param("col"))

	var in cells.SetCellIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	cell, err := h.cells.Set(c.Request().Context(), sheetID, row, col, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to set cell"})
	}

	return c.JSON(http.StatusOK, cell)
}

// BatchUpdate updates multiple cells at once.
func (h *Cell) BatchUpdate(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in cells.BatchUpdateIn
	if err := c.BindJSON(&in, 10<<20); err != nil { // 10MB limit for batch updates
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	cellList, err := h.cells.BatchUpdate(c.Request().Context(), sheetID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update cells"})
	}

	return c.JSON(http.StatusOK, cellList)
}

// Delete deletes a cell.
func (h *Cell) Delete(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	row, _ := strconv.Atoi(c.Param("row"))
	col, _ := strconv.Atoi(c.Param("col"))

	if err := h.cells.Delete(c.Request().Context(), sheetID, row, col); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete cell"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// InsertRows inserts rows at the specified position.
func (h *Cell) InsertRows(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		RowIndex int `json:"rowIndex"`
		Count    int `json:"count"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if in.Count <= 0 {
		in.Count = 1
	}

	if err := h.cells.InsertRows(c.Request().Context(), sheetID, in.RowIndex, in.Count); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to insert rows"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteRows deletes rows at the specified position.
func (h *Cell) DeleteRows(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		StartRow int `json:"startRow"`
		Count    int `json:"count"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if in.Count <= 0 {
		in.Count = 1
	}

	if err := h.cells.DeleteRows(c.Request().Context(), sheetID, in.StartRow, in.Count); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete rows"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// InsertCols inserts columns at the specified position.
func (h *Cell) InsertCols(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		ColIndex int `json:"colIndex"`
		Count    int `json:"count"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if in.Count <= 0 {
		in.Count = 1
	}

	if err := h.cells.InsertCols(c.Request().Context(), sheetID, in.ColIndex, in.Count); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to insert columns"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteCols deletes columns at the specified position.
func (h *Cell) DeleteCols(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		StartCol int `json:"startCol"`
		Count    int `json:"count"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if in.Count <= 0 {
		in.Count = 1
	}

	if err := h.cells.DeleteCols(c.Request().Context(), sheetID, in.StartCol, in.Count); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete columns"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GetMerges retrieves merged regions for a sheet.
func (h *Cell) GetMerges(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	merges, err := h.cells.GetMergedRegions(c.Request().Context(), sheetID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve merges"})
	}

	return c.JSON(http.StatusOK, merges)
}

// Merge merges cells in a range.
func (h *Cell) Merge(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		StartRow int `json:"startRow"`
		StartCol int `json:"startCol"`
		EndRow   int `json:"endRow"`
		EndCol   int `json:"endCol"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	region, err := h.cells.Merge(c.Request().Context(), sheetID, in.StartRow, in.StartCol, in.EndRow, in.EndCol)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to merge cells"})
	}

	return c.JSON(http.StatusOK, region)
}

// Unmerge unmerges cells in a range.
func (h *Cell) Unmerge(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		StartRow int `json:"startRow"`
		StartCol int `json:"startCol"`
		EndRow   int `json:"endRow"`
		EndCol   int `json:"endCol"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.cells.Unmerge(c.Request().Context(), sheetID, in.StartRow, in.StartCol, in.EndRow, in.EndCol); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unmerge cells"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Evaluate evaluates a formula.
func (h *Cell) Evaluate(c *mizu.Ctx) error {
	var in struct {
		SheetID string `json:"sheetID"`
		Formula string `json:"formula"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// SECURITY: Verify user has access to the sheet
	if !h.checkSheetAccess(c, in.SheetID) {
		return nil // Response already written
	}

	result, err := h.cells.EvaluateFormula(c.Request().Context(), in.SheetID, in.Formula)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to evaluate formula"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"result": result,
	})
}

// CopyRange copies a range of cells to a new location.
func (h *Cell) CopyRange(c *mizu.Ctx) error {
	sheetID := c.Param("sheetID")

	// SECURITY: Verify user has access to the source sheet
	if !h.checkSheetAccess(c, sheetID) {
		return nil // Response already written
	}

	var in struct {
		SourceRange struct {
			StartRow int `json:"startRow"`
			StartCol int `json:"startCol"`
			EndRow   int `json:"endRow"`
			EndCol   int `json:"endCol"`
		} `json:"sourceRange"`
		DestRow     int    `json:"destRow"`
		DestCol     int    `json:"destCol"`
		DestSheetID string `json:"destSheetId"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	destSheetID := in.DestSheetID
	if destSheetID == "" {
		destSheetID = sheetID
	}

	// SECURITY: Also verify access to destination sheet if different
	if destSheetID != sheetID {
		if !h.checkSheetAccess(c, destSheetID) {
			return nil // Response already written
		}
	}

	sourceRange := cells.Range{
		StartRow: in.SourceRange.StartRow,
		StartCol: in.SourceRange.StartCol,
		EndRow:   in.SourceRange.EndRow,
		EndCol:   in.SourceRange.EndCol,
	}

	if err := h.cells.CopyRange(c.Request().Context(), sheetID, sourceRange, destSheetID, in.DestRow, in.DestCol); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to copy range"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
