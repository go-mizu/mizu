package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

// Post handles post endpoints.
type Post struct {
	posts        posts.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewPost creates a new post handler.
func NewPost(postsSvc posts.API, getAccountID func(*mizu.Ctx) string, optionalAuth func(*mizu.Ctx) string) *Post {
	return &Post{
		posts:        postsSvc,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// Create handles POST /api/v1/posts
func (h *Post) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	var in posts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	post, err := h.posts.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		switch err {
		case posts.ErrEmpty:
			return UnprocessableEntity(c, "post content is required")
		case posts.ErrTooLong:
			return UnprocessableEntity(c, "post content is too long")
		default:
			return InternalError(c, err)
		}
	}

	_ = h.posts.PopulateAccount(c.Request().Context(), post)

	return Created(c, post)
}

// Get handles GET /api/v1/posts/:id
func (h *Post) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	post, err := h.posts.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "post")
	}

	_ = h.posts.PopulateAccount(c.Request().Context(), post)

	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.posts.PopulateViewerState(c.Request().Context(), post, viewerID)
	}

	return Success(c, post)
}

// Update handles PUT /api/v1/posts/:id
func (h *Post) Update(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	id := c.Param("id")

	var in posts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	post, err := h.posts.Update(c.Request().Context(), accountID, id, &in)
	if err != nil {
		switch err {
		case posts.ErrNotFound:
			return NotFound(c, "post")
		case posts.ErrUnauthorized:
			return Forbidden(c)
		default:
			return InternalError(c, err)
		}
	}

	_ = h.posts.PopulateAccount(c.Request().Context(), post)

	return Success(c, post)
}

// Delete handles DELETE /api/v1/posts/:id
func (h *Post) Delete(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	id := c.Param("id")

	err := h.posts.Delete(c.Request().Context(), accountID, id)
	if err != nil {
		switch err {
		case posts.ErrNotFound:
			return NotFound(c, "post")
		case posts.ErrUnauthorized:
			return Forbidden(c)
		default:
			return InternalError(c, err)
		}
	}

	return NoContent(c)
}

// GetContext handles GET /api/v1/posts/:id/context
func (h *Post) GetContext(c *mizu.Ctx) error {
	id := c.Param("id")

	ctx, err := h.posts.GetContext(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "post")
	}

	// Populate accounts
	_ = h.posts.PopulateAccounts(c.Request().Context(), ctx.Ancestors)
	_ = h.posts.PopulateAccounts(c.Request().Context(), ctx.Descendants)

	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.posts.PopulateViewerStates(c.Request().Context(), ctx.Ancestors, viewerID)
		_ = h.posts.PopulateViewerStates(c.Request().Context(), ctx.Descendants, viewerID)
	}

	return Success(c, ctx)
}
