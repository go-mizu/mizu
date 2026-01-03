package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/rowcomments"
)

// RowComment handles row comment endpoints.
type RowComment struct {
	comments  rowcomments.API
	getUserID func(c *mizu.Ctx) string
}

// NewRowComment creates a new RowComment handler.
func NewRowComment(comments rowcomments.API, getUserID func(c *mizu.Ctx) string) *RowComment {
	return &RowComment{comments: comments, getUserID: getUserID}
}

// Create creates a new comment on a row.
func (h *RowComment) Create(c *mizu.Ctx) error {
	rowID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if in.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}

	comment, err := h.comments.Create(c.Request().Context(), &rowcomments.CreateIn{
		RowID:   rowID,
		Content: in.Content,
		UserID:  userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, comment)
}

// List lists all comments for a row.
func (h *RowComment) List(c *mizu.Ctx) error {
	rowID := c.Param("id")

	comments, err := h.comments.ListByRow(c.Request().Context(), rowID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if comments == nil {
		comments = []*rowcomments.Comment{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"comments": comments})
}

// Get retrieves a single comment.
func (h *RowComment) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	comment, err := h.comments.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "comment not found"})
	}

	return c.JSON(http.StatusOK, comment)
}

// Update updates a comment.
func (h *RowComment) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	comment, err := h.comments.Update(c.Request().Context(), id, &rowcomments.UpdateIn{
		Content: in.Content,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, comment)
}

// Delete deletes a comment.
func (h *RowComment) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Resolve marks a comment as resolved.
func (h *RowComment) Resolve(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Resolve(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "resolved"})
}

// Unresolve marks a comment as unresolved.
func (h *RowComment) Unresolve(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Unresolve(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "unresolved"})
}
