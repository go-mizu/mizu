package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/views"
)

// View handles view endpoints.
type View struct {
	views     *views.Service
	getUserID func(*mizu.Ctx) string
}

// NewView creates a new view handler.
func NewView(views *views.Service, getUserID func(*mizu.Ctx) string) *View {
	return &View{views: views, getUserID: getUserID}
}

// Create creates a new view.
func (h *View) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in views.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	view, err := h.views.Create(c.Context(), userID, in)
	if err != nil {
		return InternalError(c, "failed to create view")
	}

	return Created(c, map[string]any{"view": view})
}

// Get returns a view by ID.
func (h *View) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	view, err := h.views.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "view not found")
	}

	return OK(c, map[string]any{"view": view})
}

// Update updates a view.
func (h *View) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in views.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	view, err := h.views.Update(c.Context(), id, in)
	if err != nil {
		if err == views.ErrNotFound {
			return NotFound(c, "view not found")
		}
		return InternalError(c, "failed to update view")
	}

	return OK(c, map[string]any{"view": view})
}

// Delete deletes a view.
func (h *View) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.views.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete view")
	}

	return NoContent(c)
}

// Duplicate duplicates a view.
func (h *View) Duplicate(c *mizu.Ctx) error {
	id := c.Param("id")

	// Get original view name
	original, err := h.views.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "view not found")
	}

	view, err := h.views.Duplicate(c.Context(), id, original.Name+" (copy)")
	if err != nil {
		return InternalError(c, "failed to duplicate view")
	}

	return Created(c, map[string]any{"view": view})
}

// ViewReorderRequest is the request body for reordering views.
type ViewReorderRequest struct {
	ViewIDs []string `json:"view_ids"`
}

// Reorder reorders views for a table.
func (h *View) Reorder(c *mizu.Ctx) error {
	tableID := c.Param("tableId")

	var req ViewReorderRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	// Get all views and update positions
	viewList, err := h.views.ListByTable(c.Context(), tableID)
	if err != nil {
		return InternalError(c, "failed to list views")
	}

	for i, viewID := range req.ViewIDs {
		for _, v := range viewList {
			if v.ID == viewID {
				v.Position = i
				// Update position via service
				h.views.Update(c.Context(), v.ID, views.UpdateIn{})
				break
			}
		}
	}

	return OK(c, map[string]any{"success": true})
}
