package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
)

// Table handles table endpoints.
type Table struct {
	tables    *tables.Service
	fields    *fields.Service
	views     *views.Service
	getUserID func(*mizu.Ctx) string
}

// NewTable creates a new table handler.
func NewTable(tables *tables.Service, fields *fields.Service, views *views.Service, getUserID func(*mizu.Ctx) string) *Table {
	return &Table{tables: tables, fields: fields, views: views, getUserID: getUserID}
}

// Create creates a new table.
func (h *Table) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in tables.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	tbl, err := h.tables.Create(c.Context(), userID, in)
	if err != nil {
		return InternalError(c, "failed to create table")
	}

	return Created(c, map[string]any{"table": tbl})
}

// Get returns a table by ID with fields and views.
func (h *Table) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	tbl, err := h.tables.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "table not found")
	}

	fieldList, _ := h.fields.ListByTable(c.Context(), id)
	viewList, _ := h.views.ListByTable(c.Context(), id)

	// Note: Select options are fetched separately via /fields/{id}/options

	return OK(c, map[string]any{
		"table":  tbl,
		"fields": fieldList,
		"views":  viewList,
	})
}

// Update updates a table.
func (h *Table) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in tables.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	tbl, err := h.tables.Update(c.Context(), id, in)
	if err != nil {
		if err == tables.ErrNotFound {
			return NotFound(c, "table not found")
		}
		return InternalError(c, "failed to update table")
	}

	return OK(c, map[string]any{"table": tbl})
}

// Delete deletes a table.
func (h *Table) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.tables.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete table")
	}

	return NoContent(c)
}

// ListFields returns all fields in a table.
func (h *Table) ListFields(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.fields.ListByTable(c.Context(), id)
	if err != nil {
		return InternalError(c, "failed to list fields")
	}

	return OK(c, map[string]any{"fields": list})
}

// ListViews returns all views for a table.
func (h *Table) ListViews(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.views.ListByTable(c.Context(), id)
	if err != nil {
		return InternalError(c, "failed to list views")
	}

	return OK(c, map[string]any{"views": list})
}
