package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/categories"
)

// Categories handles category endpoints.
type Categories struct {
	categories categories.API
}

// NewCategories creates a new categories handler.
func NewCategories(categories categories.API) *Categories {
	return &Categories{categories: categories}
}

// List lists categories.
func (h *Categories) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 100
	}

	in := &categories.ListIn{
		ParentID: c.Query("parent_id"),
		Search:   c.Query("search"),
		Limit:    perPage,
		Offset:   (page - 1) * perPage,
	}

	list, total, err := h.categories.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list categories")
	}

	return List(c, list, total, page, perPage)
}

// Create creates a new category.
func (h *Categories) Create(c *mizu.Ctx) error {
	var in categories.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	cat, err := h.categories.Create(c.Context(), &in)
	if err != nil {
		if err == categories.ErrMissingName {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create category")
	}

	return Created(c, cat)
}

// Get retrieves a category by ID.
func (h *Categories) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	cat, err := h.categories.GetByID(c.Context(), id)
	if err != nil {
		if err == categories.ErrNotFound {
			return NotFound(c, "category not found")
		}
		return InternalError(c, "failed to get category")
	}

	return OK(c, cat)
}

// GetTree retrieves the category hierarchy.
func (h *Categories) GetTree(c *mizu.Ctx) error {
	tree, err := h.categories.GetTree(c.Context())
	if err != nil {
		return InternalError(c, "failed to get category tree")
	}

	return OK(c, tree)
}

// Update updates a category.
func (h *Categories) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in categories.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	cat, err := h.categories.Update(c.Context(), id, &in)
	if err != nil {
		if err == categories.ErrNotFound {
			return NotFound(c, "category not found")
		}
		return InternalError(c, "failed to update category")
	}

	return OK(c, cat)
}

// Delete deletes a category.
func (h *Categories) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.categories.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete category")
	}

	return OK(c, map[string]string{"message": "category deleted"})
}
