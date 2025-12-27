package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/comments"
)

// Comment handles comment endpoints.
type Comment struct {
	comments  comments.API
	getUserID func(*mizu.Ctx) string
}

// NewComment creates a new comment handler.
func NewComment(comments comments.API, getUserID func(*mizu.Ctx) string) *Comment {
	return &Comment{comments: comments, getUserID: getUserID}
}

// List returns all comments for an issue.
func (h *Comment) List(c *mizu.Ctx) error {
	issueID := c.Param("issueID")

	list, err := h.comments.ListByIssue(c.Context(), issueID)
	if err != nil {
		return InternalError(c, "failed to list comments")
	}

	return OK(c, list)
}

// Create creates a new comment.
func (h *Comment) Create(c *mizu.Ctx) error {
	issueID := c.Param("issueID")
	userID := h.getUserID(c)

	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.comments.Create(c.Context(), issueID, userID, &in)
	if err != nil {
		return InternalError(c, "failed to create comment")
	}

	return Created(c, comment)
}

// Update updates a comment.
func (h *Comment) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.comments.Update(c.Context(), id, in.Content)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "comment not found")
		}
		return InternalError(c, "failed to update comment")
	}

	return OK(c, comment)
}

// Delete deletes a comment.
func (h *Comment) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete comment")
	}

	return OK(c, map[string]string{"message": "comment deleted"})
}
