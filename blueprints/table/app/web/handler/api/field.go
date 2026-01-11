package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/fields"
)

// Field handles field endpoints.
type Field struct {
	fields    *fields.Service
	getUserID func(*mizu.Ctx) string
}

// NewField creates a new field handler.
func NewField(fields *fields.Service, getUserID func(*mizu.Ctx) string) *Field {
	return &Field{fields: fields, getUserID: getUserID}
}

// Create creates a new field.
func (h *Field) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in fields.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	field, err := h.fields.Create(c.Context(), userID, in)
	if err != nil {
		if err == fields.ErrInvalidType {
			return BadRequest(c, "invalid field type")
		}
		return InternalError(c, "failed to create field")
	}

	return Created(c, map[string]any{"field": field})
}

// Get returns a field by ID.
func (h *Field) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	field, err := h.fields.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "field not found")
	}

	return OK(c, map[string]any{"field": field})
}

// Update updates a field.
func (h *Field) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in fields.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	field, err := h.fields.Update(c.Context(), id, in)
	if err != nil {
		if err == fields.ErrNotFound {
			return NotFound(c, "field not found")
		}
		return InternalError(c, "failed to update field")
	}

	return OK(c, map[string]any{"field": field})
}

// Delete deletes a field.
func (h *Field) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.fields.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete field")
	}

	return NoContent(c)
}

// ReorderRequest is the request body for reordering fields.
type ReorderRequest struct {
	FieldIDs []string `json:"field_ids"`
}

// Reorder reorders fields in a table.
func (h *Field) Reorder(c *mizu.Ctx) error {
	tableID := c.Param("tableId")

	var req ReorderRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.fields.Reorder(c.Context(), tableID, req.FieldIDs); err != nil {
		return InternalError(c, "failed to reorder fields")
	}

	return OK(c, map[string]any{"success": true})
}

// ListOptions returns select options for a field.
func (h *Field) ListOptions(c *mizu.Ctx) error {
	fieldID := c.Param("id")

	choices, err := h.fields.ListSelectChoices(c.Context(), fieldID)
	if err != nil {
		return InternalError(c, "failed to list options")
	}

	return OK(c, map[string]any{"options": choices})
}

// CreateOptionRequest is the request body for creating an option.
type CreateOptionRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CreateOption creates a new select option.
func (h *Field) CreateOption(c *mizu.Ctx) error {
	fieldID := c.Param("id")

	var req CreateOptionRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	choice := &fields.SelectChoice{
		Name:  req.Name,
		Color: req.Color,
	}

	if err := h.fields.AddSelectChoice(c.Context(), fieldID, choice); err != nil {
		return InternalError(c, "failed to create option")
	}

	return Created(c, map[string]any{"option": choice})
}

// UpdateOption updates a select option.
func (h *Field) UpdateOption(c *mizu.Ctx) error {
	fieldID := c.Param("id")
	optionID := c.Param("optionId")

	var in fields.UpdateChoiceIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.fields.UpdateSelectChoice(c.Context(), fieldID, optionID, in); err != nil {
		return InternalError(c, "failed to update option")
	}

	return OK(c, map[string]any{"success": true})
}

// DeleteOption deletes a select option.
func (h *Field) DeleteOption(c *mizu.Ctx) error {
	fieldID := c.Param("id")
	optionID := c.Param("optionId")

	if err := h.fields.DeleteSelectChoice(c.Context(), fieldID, optionID); err != nil {
		return InternalError(c, "failed to delete option")
	}

	return NoContent(c)
}
