package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/columns"
)

// Column handles column endpoints.
type Column struct {
	columns columns.API
}

// NewColumn creates a new column handler.
func NewColumn(columns columns.API) *Column {
	return &Column{columns: columns}
}

// List returns all columns for a project.
func (h *Column) List(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	list, err := h.columns.ListByProject(c.Context(), projectID)
	if err != nil {
		return InternalError(c, "failed to list columns")
	}

	return OK(c, list)
}

// Create creates a new column.
func (h *Column) Create(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	var in columns.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	column, err := h.columns.Create(c.Context(), projectID, &in)
	if err != nil {
		return InternalError(c, "failed to create column")
	}

	return Created(c, column)
}

// Update updates a column.
func (h *Column) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in columns.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	column, err := h.columns.Update(c.Context(), id, &in)
	if err != nil {
		if err == columns.ErrNotFound {
			return NotFound(c, "column not found")
		}
		return InternalError(c, "failed to update column")
	}

	return OK(c, column)
}

// UpdatePosition updates a column's position.
func (h *Column) UpdatePosition(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Position int `json:"position"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.columns.UpdatePosition(c.Context(), id, in.Position); err != nil {
		return InternalError(c, "failed to update position")
	}

	return OK(c, map[string]string{"message": "position updated"})
}

// SetDefault sets a column as the default.
func (h *Column) SetDefault(c *mizu.Ctx) error {
	projectID := c.Param("projectID")
	id := c.Param("id")

	if err := h.columns.SetDefault(c.Context(), projectID, id); err != nil {
		return InternalError(c, "failed to set default")
	}

	return OK(c, map[string]string{"message": "default set"})
}

// Archive archives a column.
func (h *Column) Archive(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.columns.Archive(c.Context(), id); err != nil {
		return InternalError(c, "failed to archive column")
	}

	return OK(c, map[string]string{"message": "column archived"})
}

// Unarchive unarchives a column.
func (h *Column) Unarchive(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.columns.Unarchive(c.Context(), id); err != nil {
		return InternalError(c, "failed to unarchive column")
	}

	return OK(c, map[string]string{"message": "column unarchived"})
}

// Delete deletes a column.
func (h *Column) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.columns.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete column")
	}

	return OK(c, map[string]string{"message": "column deleted"})
}
