package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/posts"
)

// Posts handles post endpoints.
type Posts struct {
	posts     posts.API
	getUserID func(*mizu.Ctx) string
}

// NewPosts creates a new posts handler.
func NewPosts(posts posts.API, getUserID func(*mizu.Ctx) string) *Posts {
	return &Posts{posts: posts, getUserID: getUserID}
}

// List lists posts.
func (h *Posts) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	in := &posts.ListIn{
		AuthorID:   c.Query("author_id"),
		Status:     c.Query("status"),
		Visibility: c.Query("visibility"),
		CategoryID: c.Query("category_id"),
		TagID:      c.Query("tag_id"),
		Search:     c.Query("search"),
		Limit:      perPage,
		Offset:     (page - 1) * perPage,
		OrderBy:    c.Query("sort"),
		Order:      c.Query("order"),
	}

	if c.Query("is_featured") == "true" {
		featured := true
		in.IsFeatured = &featured
	}

	list, total, err := h.posts.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list posts")
	}

	return List(c, list, total, page, perPage)
}

// Create creates a new post.
func (h *Posts) Create(c *mizu.Ctx) error {
	var in posts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	userID := h.getUserID(c)
	post, err := h.posts.Create(c.Context(), userID, &in)
	if err != nil {
		if err == posts.ErrMissingTitle {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create post")
	}

	return Created(c, post)
}

// Get retrieves a post by ID.
func (h *Posts) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	post, err := h.posts.GetByID(c.Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "post not found")
		}
		return InternalError(c, "failed to get post")
	}

	return OK(c, post)
}

// GetBySlug retrieves a post by slug.
func (h *Posts) GetBySlug(c *mizu.Ctx) error {
	slug := c.Param("slug")
	post, err := h.posts.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "post not found")
		}
		return InternalError(c, "failed to get post")
	}

	return OK(c, post)
}

// Update updates a post.
func (h *Posts) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in posts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	post, err := h.posts.Update(c.Context(), id, &in)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "post not found")
		}
		return InternalError(c, "failed to update post")
	}

	return OK(c, post)
}

// Delete deletes a post.
func (h *Posts) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.posts.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete post")
	}

	return OK(c, map[string]string{"message": "post deleted"})
}

// Publish publishes a post.
func (h *Posts) Publish(c *mizu.Ctx) error {
	id := c.Param("id")

	post, err := h.posts.Publish(c.Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "post not found")
		}
		return InternalError(c, "failed to publish post")
	}

	return OK(c, post)
}

// Unpublish unpublishes a post.
func (h *Posts) Unpublish(c *mizu.Ctx) error {
	id := c.Param("id")

	post, err := h.posts.Unpublish(c.Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "post not found")
		}
		return InternalError(c, "failed to unpublish post")
	}

	return OK(c, post)
}
