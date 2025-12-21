package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/posts"
)

// Post contains post-related handlers.
type Post struct {
	posts        posts.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewPost creates new post handlers.
func NewPost(
	posts posts.API,
	getAccountID func(*mizu.Ctx) string,
	optionalAuth func(*mizu.Ctx) string,
) *Post {
	return &Post{
		posts:        posts,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// Create creates a new post.
func (h *Post) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	var in posts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	post, err := h.posts.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse("CREATE_FAILED", err.Error()))
	}

	return c.JSON(201, map[string]any{"data": post})
}

// Get returns a specific post.
func (h *Post) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	viewerID := h.optionalAuth(c)

	post, err := h.posts.GetByID(c.Request().Context(), id, viewerID)
	if err != nil {
		return c.JSON(404, ErrorResponse("NOT_FOUND", "Post not found"))
	}

	return c.JSON(200, map[string]any{"data": post})
}

// Update updates an existing post.
func (h *Post) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	var in posts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	post, err := h.posts.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": post})
}

// Delete deletes a post.
func (h *Post) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.posts.Delete(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse("DELETE_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}

// GetContext returns a post's thread context.
func (h *Post) GetContext(c *mizu.Ctx) error {
	id := c.Param("id")
	viewerID := h.optionalAuth(c)

	ctx, err := h.posts.GetThread(c.Request().Context(), id, viewerID)
	if err != nil {
		return c.JSON(404, ErrorResponse("NOT_FOUND", "Post not found"))
	}

	return c.JSON(200, map[string]any{"data": ctx})
}
