package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/pages"
)

// Pages handles page endpoints.
type Pages struct {
	pages     pages.API
	getUserID func(*mizu.Ctx) string
}

// NewPages creates a new pages handler.
func NewPages(pages pages.API, getUserID func(*mizu.Ctx) string) *Pages {
	return &Pages{pages: pages, getUserID: getUserID}
}

// List lists pages.
func (h *Pages) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	in := &pages.ListIn{
		ParentID:   c.Query("parent_id"),
		AuthorID:   c.Query("author_id"),
		Status:     c.Query("status"),
		Visibility: c.Query("visibility"),
		Search:     c.Query("search"),
		Limit:      perPage,
		Offset:     (page - 1) * perPage,
	}

	list, total, err := h.pages.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list pages")
	}

	return List(c, list, total, page, perPage)
}

// Create creates a new page.
func (h *Pages) Create(c *mizu.Ctx) error {
	var in pages.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	userID := h.getUserID(c)
	pg, err := h.pages.Create(c.Context(), userID, &in)
	if err != nil {
		if err == pages.ErrMissingTitle {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create page")
	}

	return Created(c, pg)
}

// Get retrieves a page by ID.
func (h *Pages) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	pg, err := h.pages.GetByID(c.Context(), id)
	if err != nil {
		if err == pages.ErrNotFound {
			return NotFound(c, "page not found")
		}
		return InternalError(c, "failed to get page")
	}

	return OK(c, pg)
}

// GetBySlug retrieves a page by slug.
func (h *Pages) GetBySlug(c *mizu.Ctx) error {
	slug := c.Param("slug")
	pg, err := h.pages.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == pages.ErrNotFound {
			return NotFound(c, "page not found")
		}
		return InternalError(c, "failed to get page")
	}

	return OK(c, pg)
}

// GetTree retrieves the page hierarchy.
func (h *Pages) GetTree(c *mizu.Ctx) error {
	tree, err := h.pages.GetTree(c.Context())
	if err != nil {
		return InternalError(c, "failed to get page tree")
	}

	return OK(c, tree)
}

// Update updates a page.
func (h *Pages) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in pages.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	pg, err := h.pages.Update(c.Context(), id, &in)
	if err != nil {
		if err == pages.ErrNotFound {
			return NotFound(c, "page not found")
		}
		return InternalError(c, "failed to update page")
	}

	return OK(c, pg)
}

// Delete deletes a page.
func (h *Pages) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.pages.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete page")
	}

	return OK(c, map[string]string{"message": "page deleted"})
}
