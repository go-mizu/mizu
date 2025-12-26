package handler

import (
	"net/http"

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
	key := c.Param("key")
	// Note: In a real implementation, we'd resolve the key to issueID
	// For now, we'll use the key directly (assumes it's the issue ID)

	list, err := h.comments.ListByIssue(c.Context(), key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list comments"))
	}

	return c.JSON(http.StatusOK, list)
}

// Create creates a new comment.
func (h *Comment) Create(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := h.getUserID(c)

	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	comment, err := h.comments.Create(c.Context(), key, userID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to create comment"))
	}

	return c.JSON(http.StatusCreated, comment)
}

// Update updates a comment.
func (h *Comment) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	comment, err := h.comments.Update(c.Context(), id, in.Content)
	if err != nil {
		if err == comments.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("comment not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to update comment"))
	}

	return c.JSON(http.StatusOK, comment)
}

// Delete deletes a comment.
func (h *Comment) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to delete comment"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "comment deleted"})
}
