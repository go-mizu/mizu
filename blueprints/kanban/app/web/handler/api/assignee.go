package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/assignees"
)

// Assignee handles assignee endpoints.
type Assignee struct {
	assignees assignees.API
}

// NewAssignee creates a new assignee handler.
func NewAssignee(assignees assignees.API) *Assignee {
	return &Assignee{assignees: assignees}
}

// List returns all assignees for an issue.
func (h *Assignee) List(c *mizu.Ctx) error {
	issueID := c.Param("issueID")

	list, err := h.assignees.List(c.Context(), issueID)
	if err != nil {
		return InternalError(c, "failed to list assignees")
	}

	return OK(c, list)
}

// Add adds an assignee to an issue.
func (h *Assignee) Add(c *mizu.Ctx) error {
	issueID := c.Param("issueID")

	var in struct {
		UserID string `json:"user_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.assignees.Add(c.Context(), issueID, in.UserID); err != nil {
		if err == assignees.ErrAlreadyAssigned {
			return BadRequest(c, "user already assigned")
		}
		return InternalError(c, "failed to add assignee")
	}

	return Created(c, map[string]string{"message": "assignee added"})
}

// Remove removes an assignee from an issue.
func (h *Assignee) Remove(c *mizu.Ctx) error {
	issueID := c.Param("issueID")
	userID := c.Param("userID")

	if err := h.assignees.Remove(c.Context(), issueID, userID); err != nil {
		return InternalError(c, "failed to remove assignee")
	}

	return OK(c, map[string]string{"message": "assignee removed"})
}
