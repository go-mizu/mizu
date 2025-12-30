package wpapi

import (
	"encoding/json"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/tags"
)

// ListTags handles GET /wp/v2/tags
func (h *Handler) ListTags(c *mizu.Ctx) error {
	params := ParseListParams(c)

	in := &tags.ListIn{
		Search:  params.Search,
		Limit:   params.PerPage,
		Offset:  params.Offset,
		OrderBy: mapTagOrderBy(params.OrderBy),
		Order:   params.Order,
	}

	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Hide empty filter
	hideEmpty := c.Query("hide_empty") == "true"

	// Slug filter
	if slug := c.Query("slug"); slug != "" {
		tag, err := h.tags.GetBySlug(c.Context(), slug)
		if err != nil {
			return OKList(c, []WPTag{}, 0, params.Page, params.PerPage)
		}
		wpTag := h.tagToWP(tag, params.Context)
		return OKList(c, []WPTag{wpTag}, 1, params.Page, params.PerPage)
	}

	list, total, err := h.tags.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read tags")
	}

	// Filter empty if needed
	if hideEmpty {
		filtered := make([]*tags.Tag, 0)
		for _, tag := range list {
			if tag.PostCount > 0 {
				filtered = append(filtered, tag)
			}
		}
		list = filtered
		total = len(filtered)
	}

	wpTags := make([]WPTag, len(list))
	for i, tag := range list {
		wpTags[i] = h.tagToWP(tag, params.Context)
	}

	return OKList(c, wpTags, total, params.Page, params.PerPage)
}

// CreateTag handles POST /wp/v2/tags
func (h *Handler) CreateTag(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	var req WPCreateTagRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	if req.Name == "" {
		return ErrorInvalidParam(c, "name", "Name is required")
	}

	in := &tags.CreateIn{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		in.Meta = string(metaBytes)
	}

	tag, err := h.tags.Create(c.Context(), in)
	if err != nil {
		if err == tags.ErrMissingName {
			return ErrorInvalidParam(c, "name", "Name is required")
		}
		return ErrorInternal(c, "rest_cannot_create", "Could not create tag")
	}

	return Created(c, h.tagToWP(tag, ContextEdit))
}

// GetTag handles GET /wp/v2/tags/{id}
func (h *Handler) GetTag(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	tag, err := h.tags.GetByID(c.Context(), id)
	if err != nil {
		if err == tags.ErrNotFound {
			return ErrorNotFound(c, "rest_term_invalid", "Invalid tag ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read tag")
	}

	return OK(c, h.tagToWP(tag, context))
}

// UpdateTag handles POST/PUT/PATCH /wp/v2/tags/{id}
func (h *Handler) UpdateTag(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPCreateTagRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &tags.UpdateIn{}

	if req.Name != "" {
		in.Name = &req.Name
	}

	if req.Slug != "" {
		in.Slug = &req.Slug
	}

	if req.Description != "" {
		in.Description = &req.Description
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		metaStr := string(metaBytes)
		in.Meta = &metaStr
	}

	tag, err := h.tags.Update(c.Context(), id, in)
	if err != nil {
		if err == tags.ErrNotFound {
			return ErrorNotFound(c, "rest_term_invalid", "Invalid tag ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update tag")
	}

	return OK(c, h.tagToWP(tag, ContextEdit))
}

// DeleteTag handles DELETE /wp/v2/tags/{id}
func (h *Handler) DeleteTag(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)
	force := c.Query("force") == "true"

	if !force {
		return ErrorBadRequest(c, "rest_trash_not_supported", "Terms do not support trashing. Set force=true to delete.")
	}

	tag, err := h.tags.GetByID(c.Context(), id)
	if err != nil {
		if err == tags.ErrNotFound {
			return ErrorNotFound(c, "rest_term_invalid", "Invalid tag ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read tag")
	}

	if err := h.tags.Delete(c.Context(), id); err != nil {
		return ErrorInternal(c, "rest_cannot_delete", "Could not delete tag")
	}

	wpTag := h.tagToWP(tag, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpTag,
	})
}

// tagToWP converts an internal tag to WordPress format.
func (h *Handler) tagToWP(tag *tags.Tag, context string) WPTag {
	numericID := ULIDToNumericID(tag.ID)

	wp := WPTag{
		ID:          numericID,
		Count:       tag.PostCount,
		Description: tag.Description,
		Link:        h.TagURL(tag.Slug),
		Name:        tag.Name,
		Slug:        tag.Slug,
		Taxonomy:    "post_tag",
		Meta:        []any{},
	}

	wp.Links = map[string][]WPLink{
		"self":         {h.SelfLink("/tags/" + strconv.FormatInt(numericID, 10))},
		"collection":   {h.CollectionLink("/tags")},
		"about":        {h.AboutLink("/taxonomies/post_tag")},
		"wp:post_type": {h.CollectionLink("/posts?tags=" + strconv.FormatInt(numericID, 10))},
	}

	return wp
}

// mapTagOrderBy maps WordPress orderby to internal field.
func mapTagOrderBy(orderBy string) string {
	switch orderBy {
	case "id", "include":
		return "id"
	case "name":
		return "name"
	case "slug":
		return "slug"
	case "count":
		return "post_count"
	case "description":
		return "description"
	default:
		return "name"
	}
}
