package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/tags"
)

// Tags handles tag endpoints.
type Tags struct {
	tags tags.API
}

// NewTags creates a new tags handler.
func NewTags(tags tags.API) *Tags {
	return &Tags{tags: tags}
}

// List lists tags.
func (h *Tags) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 100
	}

	in := &tags.ListIn{
		Search:  c.Query("search"),
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
		OrderBy: c.Query("sort"),
		Order:   c.Query("order"),
	}

	list, total, err := h.tags.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list tags")
	}

	return List(c, list, total, page, perPage)
}

// Create creates a new tag.
func (h *Tags) Create(c *mizu.Ctx) error {
	var in tags.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	tag, err := h.tags.Create(c.Context(), &in)
	if err != nil {
		if err == tags.ErrMissingName {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create tag")
	}

	return Created(c, tag)
}

// Get retrieves a tag by ID.
func (h *Tags) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	tag, err := h.tags.GetByID(c.Context(), id)
	if err != nil {
		if err == tags.ErrNotFound {
			return NotFound(c, "tag not found")
		}
		return InternalError(c, "failed to get tag")
	}

	return OK(c, tag)
}

// Update updates a tag.
func (h *Tags) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in tags.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	tag, err := h.tags.Update(c.Context(), id, &in)
	if err != nil {
		if err == tags.ErrNotFound {
			return NotFound(c, "tag not found")
		}
		return InternalError(c, "failed to update tag")
	}

	return OK(c, tag)
}

// Delete deletes a tag.
func (h *Tags) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.tags.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete tag")
	}

	return OK(c, map[string]string{"message": "tag deleted"})
}
