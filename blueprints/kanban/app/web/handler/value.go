package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/values"
)

// Value handles field value endpoints.
type Value struct {
	values values.API
}

// NewValue creates a new value handler.
func NewValue(values values.API) *Value {
	return &Value{values: values}
}

// Set sets a field value for an issue.
func (h *Value) Set(c *mizu.Ctx) error {
	issueID := c.Param("issueID")
	fieldID := c.Param("fieldID")

	var in values.SetIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	value, err := h.values.Set(c.Context(), issueID, fieldID, &in)
	if err != nil {
		return InternalError(c, "failed to set value")
	}

	return OK(c, value)
}

// Get returns a field value for an issue.
func (h *Value) Get(c *mizu.Ctx) error {
	issueID := c.Param("issueID")
	fieldID := c.Param("fieldID")

	value, err := h.values.Get(c.Context(), issueID, fieldID)
	if err != nil {
		if err == values.ErrNotFound {
			return NotFound(c, "value not found")
		}
		return InternalError(c, "failed to get value")
	}

	return OK(c, value)
}

// ListByIssue returns all field values for an issue.
func (h *Value) ListByIssue(c *mizu.Ctx) error {
	issueID := c.Param("issueID")

	list, err := h.values.ListByIssue(c.Context(), issueID)
	if err != nil {
		return InternalError(c, "failed to list values")
	}

	return OK(c, list)
}

// Delete deletes a field value.
func (h *Value) Delete(c *mizu.Ctx) error {
	issueID := c.Param("issueID")
	fieldID := c.Param("fieldID")

	if err := h.values.Delete(c.Context(), issueID, fieldID); err != nil {
		return InternalError(c, "failed to delete value")
	}

	return OK(c, map[string]string{"message": "value deleted"})
}

// BulkSet sets multiple field values at once.
func (h *Value) BulkSet(c *mizu.Ctx) error {
	issueID := c.Param("issueID")

	var in struct {
		Values []struct {
			FieldID string       `json:"field_id"`
			Value   values.SetIn `json:"value"`
		} `json:"values"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	vs := make([]*values.Value, len(in.Values))
	for i, v := range in.Values {
		vs[i] = &values.Value{
			IssueID:   issueID,
			FieldID:   v.FieldID,
			ValueText: v.Value.ValueText,
			ValueNum:  v.Value.ValueNum,
			ValueBool: v.Value.ValueBool,
			ValueDate: v.Value.ValueDate,
			ValueTS:   v.Value.ValueTS,
			ValueRef:  v.Value.ValueRef,
			ValueJSON: v.Value.ValueJSON,
		}
	}

	if err := h.values.BulkSet(c.Context(), vs); err != nil {
		return InternalError(c, "failed to bulk set values")
	}

	return OK(c, map[string]string{"message": "values set"})
}
