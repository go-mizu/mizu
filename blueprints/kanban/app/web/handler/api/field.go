package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/fields"
)

// Field handles field endpoints.
type Field struct {
	fields fields.API
}

// NewField creates a new field handler.
func NewField(fields fields.API) *Field {
	return &Field{fields: fields}
}

// List returns all fields for a project.
func (h *Field) List(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	list, err := h.fields.ListByProject(c.Context(), projectID)
	if err != nil {
		return InternalError(c, "failed to list fields")
	}

	return OK(c, list)
}

// Create creates a new field.
func (h *Field) Create(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	var in fields.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	field, err := h.fields.Create(c.Context(), projectID, &in)
	if err != nil {
		if err == fields.ErrKeyExists {
			return BadRequest(c, "field key already exists")
		}
		return InternalError(c, "failed to create field")
	}

	return Created(c, field)
}

// Get returns a field by ID.
func (h *Field) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	field, err := h.fields.GetByID(c.Context(), id)
	if err != nil {
		if err == fields.ErrNotFound {
			return NotFound(c, "field not found")
		}
		return InternalError(c, "failed to get field")
	}

	return OK(c, field)
}

// Update updates a field.
func (h *Field) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in fields.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	field, err := h.fields.Update(c.Context(), id, &in)
	if err != nil {
		if err == fields.ErrNotFound {
			return NotFound(c, "field not found")
		}
		return InternalError(c, "failed to update field")
	}

	return OK(c, field)
}

// UpdatePosition updates a field's position.
func (h *Field) UpdatePosition(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Position int `json:"position"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.fields.UpdatePosition(c.Context(), id, in.Position); err != nil {
		return InternalError(c, "failed to update position")
	}

	return OK(c, map[string]string{"message": "position updated"})
}

// Archive archives a field.
func (h *Field) Archive(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.fields.Archive(c.Context(), id); err != nil {
		return InternalError(c, "failed to archive field")
	}

	return OK(c, map[string]string{"message": "field archived"})
}

// Unarchive unarchives a field.
func (h *Field) Unarchive(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.fields.Unarchive(c.Context(), id); err != nil {
		return InternalError(c, "failed to unarchive field")
	}

	return OK(c, map[string]string{"message": "field unarchived"})
}

// Delete deletes a field.
func (h *Field) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.fields.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete field")
	}

	return OK(c, map[string]string{"message": "field deleted"})
}
