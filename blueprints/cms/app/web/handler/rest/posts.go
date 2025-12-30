package rest

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/feature/posts"
)

// Posts handles REST API requests for posts.
type Posts struct {
	posts     posts.API
	getUserID func(*mizu.Ctx) string
}

// NewPosts creates a new posts handler.
func NewPosts(p posts.API, getUserID func(*mizu.Ctx) string) *Posts {
	return &Posts{posts: p, getUserID: getUserID}
}

// List lists posts.
func (h *Posts) List(c *mizu.Ctx) error {
	page, perPage := ParsePagination(c)
	orderBy, order := ParseOrder(c, "date", "desc")

	opts := posts.ListOpts{
		Page:    page,
		PerPage: perPage,
		OrderBy: orderBy,
		Order:   order,
		Search:  c.Query("search"),
		Type:    "post",
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		opts.Status = []string{status}
	} else {
		opts.Status = []string{"publish"}
	}

	results, total, err := h.posts.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Error retrieving posts")
	}

	totalPages := (total + perPage - 1) / perPage
	return SuccessWithPagination(c, h.formatPosts(results), total, totalPages)
}

// Get retrieves a single post.
func (h *Posts) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	post, err := h.posts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "Invalid post ID.")
		}
		return InternalError(c, "Error retrieving post")
	}

	return Success(c, h.formatPost(post))
}

// Create creates a new post.
func (h *Posts) Create(c *mizu.Ctx) error {
	var in posts.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.Type = "post"
	in.Author = h.getUserID(c)

	post, err := h.posts.Create(c.Request().Context(), in)
	if err != nil {
		return InternalError(c, "Error creating post")
	}

	return Created(c, h.formatPost(post))
}

// Update updates a post.
func (h *Posts) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in posts.UpdateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	post, err := h.posts.Update(c.Request().Context(), id, in)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "Invalid post ID.")
		}
		return InternalError(c, "Error updating post")
	}

	return Success(c, h.formatPost(post))
}

// Delete deletes a post.
func (h *Posts) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	force := c.Query("force") == "true"

	post, err := h.posts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return NotFound(c, "Invalid post ID.")
		}
		return InternalError(c, "Error retrieving post")
	}

	if err := h.posts.Delete(c.Request().Context(), id, force); err != nil {
		return InternalError(c, "Error deleting post")
	}

	return Deleted(c, h.formatPost(post))
}

func (h *Posts) formatPosts(pp []*posts.Post) []map[string]interface{} {
	result := make([]map[string]interface{}, len(pp))
	for i, p := range pp {
		result[i] = h.formatPost(p)
	}
	return result
}

func (h *Posts) formatPost(p *posts.Post) map[string]interface{} {
	return map[string]interface{}{
		"id":             p.ID,
		"date":           p.Date.Format("2006-01-02T15:04:05"),
		"date_gmt":       p.DateGmt.Format("2006-01-02T15:04:05"),
		"guid":           map[string]interface{}{"rendered": p.GUID.Rendered},
		"modified":       p.Modified.Format("2006-01-02T15:04:05"),
		"modified_gmt":   p.ModifiedGmt.Format("2006-01-02T15:04:05"),
		"slug":           p.Slug,
		"status":         p.Status,
		"type":           p.Type,
		"link":           "",
		"title":          map[string]interface{}{"rendered": p.Title.Rendered},
		"content":        map[string]interface{}{"rendered": p.Content.Rendered, "protected": p.Content.Protected},
		"excerpt":        map[string]interface{}{"rendered": p.Excerpt.Rendered, "protected": p.Excerpt.Protected},
		"author":         p.Author,
		"featured_media": p.FeaturedMedia,
		"comment_status": p.CommentStatus,
		"ping_status":    p.PingStatus,
		"sticky":         p.Sticky,
		"template":       p.Template,
		"format":         p.Format,
		"categories":     p.Categories,
		"tags":           p.Tags,
	}
}
