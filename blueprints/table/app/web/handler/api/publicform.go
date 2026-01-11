package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
)

// PublicForm handles public form endpoints (no auth required).
type PublicForm struct {
	views   *views.Service
	tables  *tables.Service
	fields  *fields.Service
	records *records.Service
}

// NewPublicForm creates a new public form handler.
func NewPublicForm(views *views.Service, tables *tables.Service, fields *fields.Service, records *records.Service) *PublicForm {
	return &PublicForm{views: views, tables: tables, fields: fields, records: records}
}

// GetForm returns form view configuration and fields for public access.
func (h *PublicForm) GetForm(c *mizu.Ctx) error {
	viewID := c.Param("viewId")

	// Get the view
	view, err := h.views.GetByID(c.Context(), viewID)
	if err != nil {
		return NotFound(c, "form not found")
	}

	// Must be a form view
	if view.Type != views.TypeForm {
		return NotFound(c, "form not found")
	}

	// Get the table
	table, err := h.tables.GetByID(c.Context(), view.TableID)
	if err != nil {
		return InternalError(c, "failed to load table")
	}

	// Get fields
	fieldList, err := h.fields.ListByTable(c.Context(), view.TableID)
	if err != nil {
		return InternalError(c, "failed to load fields")
	}

	return OK(c, map[string]any{
		"view":   view,
		"table":  table,
		"fields": fieldList,
	})
}

// SubmitFormRequest is the request body for form submission.
type SubmitFormRequest struct {
	Values map[string]any `json:"values"`
}

// SubmitForm handles public form submission.
func (h *PublicForm) SubmitForm(c *mizu.Ctx) error {
	viewID := c.Param("viewId")

	// Get the view to validate it's a form
	view, err := h.views.GetByID(c.Context(), viewID)
	if err != nil {
		return NotFound(c, "form not found")
	}

	if view.Type != views.TypeForm {
		return NotFound(c, "form not found")
	}

	// Parse request body
	var req SubmitFormRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	// Create the record (use "anonymous" as creator for public submissions)
	record, err := h.records.Create(c.Context(), view.TableID, req.Values, "anonymous")
	if err != nil {
		return InternalError(c, "failed to submit form")
	}

	return Created(c, map[string]any{
		"record": record,
		"message": "Form submitted successfully",
	})
}
