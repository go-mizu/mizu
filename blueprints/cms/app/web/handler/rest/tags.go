package rest

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/feature/terms"
)

// Tags handles REST API requests for tags.
type Tags struct {
	terms     terms.API
	getUserID func(*mizu.Ctx) string
}

// NewTags creates a new tags handler.
func NewTags(t terms.API, getUserID func(*mizu.Ctx) string) *Tags {
	return &Tags{terms: t, getUserID: getUserID}
}

// List lists tags.
func (h *Tags) List(c *mizu.Ctx) error {
	page, perPage := ParsePagination(c)
	orderBy, order := ParseOrder(c, "name", "asc")

	opts := terms.ListOpts{
		Page:      page,
		PerPage:   perPage,
		OrderBy:   orderBy,
		Order:     order,
		Search:    c.Query("search"),
		Taxonomy:  "post_tag",
		HideEmpty: c.Query("hide_empty") == "true",
	}

	results, total, err := h.terms.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Error retrieving tags")
	}

	totalPages := (total + perPage - 1) / perPage
	return SuccessWithPagination(c, h.formatTerms(results), total, totalPages)
}

// Get retrieves a single tag.
func (h *Tags) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	term, err := h.terms.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == terms.ErrNotFound {
			return NotFound(c, "Invalid tag ID.")
		}
		return InternalError(c, "Error retrieving tag")
	}

	if term.Taxonomy != "post_tag" {
		return NotFound(c, "Invalid tag ID.")
	}

	return Success(c, h.formatTerm(term))
}

// Create creates a new tag.
func (h *Tags) Create(c *mizu.Ctx) error {
	var in terms.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.Taxonomy = "post_tag"

	term, err := h.terms.Create(c.Request().Context(), in)
	if err != nil {
		if err == terms.ErrSlugTaken {
			return Conflict(c, "A tag with that slug already exists.")
		}
		return InternalError(c, "Error creating tag")
	}

	return Created(c, h.formatTerm(term))
}

// Update updates a tag.
func (h *Tags) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in terms.UpdateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	term, err := h.terms.Update(c.Request().Context(), id, in)
	if err != nil {
		if err == terms.ErrNotFound {
			return NotFound(c, "Invalid tag ID.")
		}
		return InternalError(c, "Error updating tag")
	}

	return Success(c, h.formatTerm(term))
}

// Delete deletes a tag.
func (h *Tags) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	force := c.Query("force") == "true"

	term, err := h.terms.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == terms.ErrNotFound {
			return NotFound(c, "Invalid tag ID.")
		}
		return InternalError(c, "Error retrieving tag")
	}

	if err := h.terms.Delete(c.Request().Context(), id, force); err != nil {
		return InternalError(c, "Error deleting tag")
	}

	return Deleted(c, h.formatTerm(term))
}

func (h *Tags) formatTerms(tt []*terms.Term) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tt))
	for i, t := range tt {
		result[i] = h.formatTerm(t)
	}
	return result
}

func (h *Tags) formatTerm(t *terms.Term) map[string]interface{} {
	return map[string]interface{}{
		"id":          t.ID,
		"count":       t.Count,
		"description": t.Description,
		"link":        "",
		"name":        t.Name,
		"slug":        t.Slug,
		"taxonomy":    "post_tag",
	}
}
