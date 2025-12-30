package rest

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/feature/posts"
)

// Pages handles REST API requests for pages.
type Pages struct {
	posts     posts.API
	getUserID func(*mizu.Ctx) string
}

// NewPages creates a new pages handler.
func NewPages(p posts.API, getUserID func(*mizu.Ctx) string) *Pages {
	return &Pages{posts: p, getUserID: getUserID}
}

// List lists pages.
func (h *Pages) List(c *mizu.Ctx) error {
	page, perPage := ParsePagination(c)
	orderBy, order := ParseOrder(c, "menu_order", "asc")

	opts := posts.ListOpts{
		Page:    page,
		PerPage: perPage,
		OrderBy: orderBy,
		Order:   order,
		Search:  c.Query("search"),
		Type:    "page",
	}

	if status := c.Query("status"); status != "" {
		opts.Status = []string{status}
	} else {
		opts.Status = []string{"publish"}
	}

	if parent := c.Query("parent"); parent != "" {
		opts.Parent = []string{parent}
	}

	results, total, err := h.posts.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Error retrieving pages")
	}

	totalPages := (total + perPage - 1) / perPage
	return SuccessWithPagination(c, h.formatPages(results), total, totalPages)
}

// Get retrieves a single page.
func (h *Pages) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	page, err := h.posts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "Invalid page ID.")
		}
		return InternalError(c, "Error retrieving page")
	}

	if page.Type != "page" {
		return NotFound(c, "Invalid page ID.")
	}

	return Success(c, h.formatPage(page))
}

// Create creates a new page.
func (h *Pages) Create(c *mizu.Ctx) error {
	var in posts.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.Type = "page"
	in.Author = h.getUserID(c)

	page, err := h.posts.Create(c.Request().Context(), in)
	if err != nil {
		return InternalError(c, "Error creating page")
	}

	return Created(c, h.formatPage(page))
}

// Update updates a page.
func (h *Pages) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in posts.UpdateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	page, err := h.posts.Update(c.Request().Context(), id, in)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "Invalid page ID.")
		}
		return InternalError(c, "Error updating page")
	}

	return Success(c, h.formatPage(page))
}

// Delete deletes a page.
func (h *Pages) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	force := c.Query("force") == "true"

	page, err := h.posts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "Invalid page ID.")
		}
		return InternalError(c, "Error retrieving page")
	}

	if err := h.posts.Delete(c.Request().Context(), id, force); err != nil {
		return InternalError(c, "Error deleting page")
	}

	return Deleted(c, h.formatPage(page))
}

func (h *Pages) formatPages(pp []*posts.Post) []map[string]interface{} {
	result := make([]map[string]interface{}, len(pp))
	for i, p := range pp {
		result[i] = h.formatPage(p)
	}
	return result
}

func (h *Pages) formatPage(p *posts.Post) map[string]interface{} {
	return map[string]interface{}{
		"id":             p.ID,
		"date":           p.Date.Format("2006-01-02T15:04:05"),
		"date_gmt":       p.DateGmt.Format("2006-01-02T15:04:05"),
		"guid":           map[string]interface{}{"rendered": p.GUID.Rendered},
		"modified":       p.Modified.Format("2006-01-02T15:04:05"),
		"modified_gmt":   p.ModifiedGmt.Format("2006-01-02T15:04:05"),
		"slug":           p.Slug,
		"status":         p.Status,
		"type":           "page",
		"link":           "",
		"title":          map[string]interface{}{"rendered": p.Title.Rendered},
		"content":        map[string]interface{}{"rendered": p.Content.Rendered, "protected": p.Content.Protected},
		"excerpt":        map[string]interface{}{"rendered": p.Excerpt.Rendered, "protected": p.Excerpt.Protected},
		"author":         p.Author,
		"featured_media": p.FeaturedMedia,
		"comment_status": p.CommentStatus,
		"ping_status":    p.PingStatus,
		"parent":         p.Parent,
		"menu_order":     p.MenuOrder,
		"template":       p.Template,
	}
}
