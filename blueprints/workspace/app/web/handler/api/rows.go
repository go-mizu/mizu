package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/rows"
)

// Row handles database row endpoints.
type Row struct {
	rows      rows.API
	getUserID func(c *mizu.Ctx) string
}

// NewRow creates a new Row handler.
func NewRow(rows rows.API, getUserID func(c *mizu.Ctx) string) *Row {
	return &Row{rows: rows, getUserID: getUserID}
}

// Create creates a new row in a database.
func (h *Row) Create(c *mizu.Ctx) error {
	dbID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	row, err := h.rows.Create(c.Request().Context(), &rows.CreateIn{
		DatabaseID: dbID,
		Properties: in.Properties,
		CreatedBy:  userID,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, row)
}

// Get retrieves a single row.
func (h *Row) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	row, err := h.rows.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "row not found"})
	}

	return c.JSON(http.StatusOK, row)
}

// Update updates a row's properties.
func (h *Row) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	row, err := h.rows.Update(c.Request().Context(), id, &rows.UpdateIn{
		Properties: in.Properties,
		UpdatedBy:  userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, row)
}

// Delete deletes a row.
func (h *Row) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.rows.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// List lists rows in a database with optional filters and sorts.
func (h *Row) List(c *mizu.Ctx) error {
	dbID := c.Param("id")

	var filters []rows.Filter
	var sorts []rows.Sort

	// Parse filters from query param
	if filtersJSON := c.Query("filters"); filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid filters"})
		}
	}

	// Parse sorts from query param
	if sortsJSON := c.Query("sorts"); sortsJSON != "" {
		if err := json.Unmarshal([]byte(sortsJSON), &sorts); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid sorts"})
		}
	}

	cursor := c.Query("cursor")

	result, err := h.rows.List(c.Request().Context(), &rows.ListIn{
		DatabaseID: dbID,
		Filters:    filters,
		Sorts:      sorts,
		Cursor:     cursor,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// Duplicate creates a copy of a row.
func (h *Row) Duplicate(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	row, err := h.rows.DuplicateRow(c.Request().Context(), id, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, row)
}
