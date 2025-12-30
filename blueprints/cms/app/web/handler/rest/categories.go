package rest

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/feature/terms"
)

// Categories handles REST API requests for categories.
type Categories struct {
	terms     terms.API
	getUserID func(*mizu.Ctx) string
}

// NewCategories creates a new categories handler.
func NewCategories(t terms.API, getUserID func(*mizu.Ctx) string) *Categories {
	return &Categories{terms: t, getUserID: getUserID}
}

// List lists categories.
func (h *Categories) List(c *mizu.Ctx) error {
	page, perPage := ParsePagination(c)
	orderBy, order := ParseOrder(c, "name", "asc")

	opts := terms.ListOpts{
		Page:      page,
		PerPage:   perPage,
		OrderBy:   orderBy,
		Order:     order,
		Search:    c.Query("search"),
		Taxonomy:  "category",
		HideEmpty: c.Query("hide_empty") == "true",
	}

	if parent := c.Query("parent"); parent != "" {
		opts.Parent = &parent
	}

	results, total, err := h.terms.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Error retrieving categories")
	}

	totalPages := (total + perPage - 1) / perPage
	return SuccessWithPagination(c, h.formatTerms(results), total, totalPages)
}

// Get retrieves a single category.
func (h *Categories) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	term, err := h.terms.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == terms.ErrNotFound {
			return NotFound(c, "Invalid category ID.")
		}
		return InternalError(c, "Error retrieving category")
	}

	if term.Taxonomy != "category" {
		return NotFound(c, "Invalid category ID.")
	}

	return Success(c, h.formatTerm(term))
}

// Create creates a new category.
func (h *Categories) Create(c *mizu.Ctx) error {
	var in terms.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.Taxonomy = "category"

	term, err := h.terms.Create(c.Request().Context(), in)
	if err != nil {
		if err == terms.ErrSlugTaken {
			return Conflict(c, "A category with that slug already exists.")
		}
		return InternalError(c, "Error creating category")
	}

	return Created(c, h.formatTerm(term))
}

// Update updates a category.
func (h *Categories) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in terms.UpdateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	term, err := h.terms.Update(c.Request().Context(), id, in)
	if err != nil {
		if err == terms.ErrNotFound {
			return NotFound(c, "Invalid category ID.")
		}
		return InternalError(c, "Error updating category")
	}

	return Success(c, h.formatTerm(term))
}

// Delete deletes a category.
func (h *Categories) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	force := c.Query("force") == "true"

	term, err := h.terms.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == terms.ErrNotFound {
			return NotFound(c, "Invalid category ID.")
		}
		return InternalError(c, "Error retrieving category")
	}

	if err := h.terms.Delete(c.Request().Context(), id, force); err != nil {
		return InternalError(c, "Error deleting category")
	}

	return Deleted(c, h.formatTerm(term))
}

func (h *Categories) formatTerms(tt []*terms.Term) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tt))
	for i, t := range tt {
		result[i] = h.formatTerm(t)
	}
	return result
}

func (h *Categories) formatTerm(t *terms.Term) map[string]interface{} {
	return map[string]interface{}{
		"id":          t.ID,
		"count":       t.Count,
		"description": t.Description,
		"link":        "",
		"name":        t.Name,
		"slug":        t.Slug,
		"taxonomy":    "category",
		"parent":      t.Parent,
	}
}
