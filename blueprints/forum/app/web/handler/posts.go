package handler

import (
	"github.com/go-mizu/blueprints/forum/feature/posts"
	"github.com/go-mizu/mizu"
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

// ListByThread lists posts in a thread.
func (h *Post) ListByThread(c *mizu.Ctx) error {
	threadID := c.Param("id")
	viewerID := h.optionalAuth(c)

	sort := StringQuery(c, "sort", "best")
	limit := IntQuery(c, "limit", 50)
	if limit > 200 {
		limit = 200
	}
	offset := IntQuery(c, "offset", 0)

	list, err := h.posts.ListByThread(c.Request().Context(), threadID, viewerID, sort, limit, offset)
	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"posts": list.Posts,
		"total": list.Total,
	}))
}

// GetTree returns posts in a tree structure.
func (h *Post) GetTree(c *mizu.Ctx) error {
	threadID := c.Param("id")
	viewerID := h.optionalAuth(c)

	tree, err := h.posts.GetTree(c.Request().Context(), threadID, viewerID)
	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"posts": tree,
	}))
}

// Get returns a specific post.
func (h *Post) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	viewerID := h.optionalAuth(c)

	post, err := h.posts.GetByID(c.Request().Context(), id, viewerID)
	if err != nil {
		return c.JSON(404, ErrorResponse("Post not found"))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"post": post,
	}))
}

// Create creates a new post.
func (h *Post) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)

	var in posts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	post, err := h.posts.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(201, DataResponse(map[string]any{
		"post": post,
	}))
}

// Update updates a post.
func (h *Post) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	var in posts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	post, err := h.posts.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"post": post,
	}))
}

// Delete deletes a post.
func (h *Post) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.posts.Delete(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Post deleted successfully",
	}))
}

// ListByAccount lists posts created by an account.
func (h *Post) ListByAccount(c *mizu.Ctx) error {
	accountID := c.Param("id")
	viewerID := h.optionalAuth(c)

	limit := IntQuery(c, "limit", 50)
	if limit > 200 {
		limit = 200
	}
	offset := IntQuery(c, "offset", 0)

	list, err := h.posts.ListByAccount(c.Request().Context(), accountID, viewerID, limit, offset)
	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"posts": list.Posts,
		"total": list.Total,
	}))
}
