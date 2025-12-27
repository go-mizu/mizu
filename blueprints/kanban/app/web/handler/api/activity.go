package api

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/activities"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// Activity handles activity endpoints.
type Activity struct {
	activities activities.API
	workspaces workspaces.API
	getUserID  func(*mizu.Ctx) string
}

// NewActivity creates a new activity handler.
func NewActivity(activities activities.API, workspaces workspaces.API, getUserID func(*mizu.Ctx) string) *Activity {
	return &Activity{activities: activities, workspaces: workspaces, getUserID: getUserID}
}

// ListByIssue returns all activities for an issue.
func (h *Activity) ListByIssue(c *mizu.Ctx) error {
	issueID := c.Param("issueID")

	list, err := h.activities.ListByIssue(c.Context(), issueID)
	if err != nil {
		return InternalError(c, "failed to list activities")
	}

	return OK(c, list)
}

// ListByWorkspace returns activities for a workspace.
func (h *Activity) ListByWorkspace(c *mizu.Ctx) error {
	slug := c.Param("slug")

	// Get workspace by slug
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return InternalError(c, "failed to get workspace")
	}
	if ws == nil {
		return NotFound(c, "workspace not found")
	}

	// Parse pagination params
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	list, err := h.activities.ListByWorkspace(c.Context(), ws.ID, limit, offset)
	if err != nil {
		return InternalError(c, "failed to list activities")
	}

	return OK(c, list)
}
