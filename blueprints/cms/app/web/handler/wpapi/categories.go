package wpapi

import (
	"encoding/json"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/categories"
)

// ListCategories handles GET /wp/v2/categories
func (h *Handler) ListCategories(c *mizu.Ctx) error {
	params := ParseListParams(c)

	in := &categories.ListIn{
		Search: params.Search,
		Limit:  params.PerPage,
		Offset: params.Offset,
	}

	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Parent filter
	if parent := c.Query("parent"); parent != "" {
		in.ParentID = parent
	}

	// Hide empty filter
	hideEmpty := c.Query("hide_empty") == "true"

	// Slug filter
	if slug := c.Query("slug"); slug != "" {
		cat, err := h.categories.GetBySlug(c.Context(), slug)
		if err != nil {
			return OKList(c, []WPCategory{}, 0, params.Page, params.PerPage)
		}
		wpCat := h.categoryToWP(cat, params.Context)
		return OKList(c, []WPCategory{wpCat}, 1, params.Page, params.PerPage)
	}

	list, total, err := h.categories.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read categories")
	}

	// Filter empty if needed
	if hideEmpty {
		filtered := make([]*categories.Category, 0)
		for _, cat := range list {
			if cat.PostCount > 0 {
				filtered = append(filtered, cat)
			}
		}
		list = filtered
		total = len(filtered)
	}

	wpCategories := make([]WPCategory, len(list))
	for i, cat := range list {
		wpCategories[i] = h.categoryToWP(cat, params.Context)
	}

	return OKList(c, wpCategories, total, params.Page, params.PerPage)
}

// CreateCategory handles POST /wp/v2/categories
func (h *Handler) CreateCategory(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	var req WPCreateCategoryRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	if req.Name == "" {
		return ErrorInvalidParam(c, "name", "Name is required")
	}

	in := &categories.CreateIn{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if req.Parent > 0 {
		in.ParentID = strconv.FormatInt(req.Parent, 10)
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		in.Meta = string(metaBytes)
	}

	cat, err := h.categories.Create(c.Context(), in)
	if err != nil {
		if err == categories.ErrMissingName {
			return ErrorInvalidParam(c, "name", "Name is required")
		}
		return ErrorInternal(c, "rest_cannot_create", "Could not create category")
	}

	return Created(c, h.categoryToWP(cat, ContextEdit))
}

// GetCategory handles GET /wp/v2/categories/{id}
func (h *Handler) GetCategory(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	cat, err := h.categories.GetByID(c.Context(), id)
	if err != nil {
		if err == categories.ErrNotFound {
			return ErrorNotFound(c, "rest_term_invalid", "Invalid category ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read category")
	}

	return OK(c, h.categoryToWP(cat, context))
}

// UpdateCategory handles POST/PUT/PATCH /wp/v2/categories/{id}
func (h *Handler) UpdateCategory(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPCreateCategoryRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &categories.UpdateIn{}

	if req.Name != "" {
		in.Name = &req.Name
	}

	if req.Slug != "" {
		in.Slug = &req.Slug
	}

	if req.Description != "" {
		in.Description = &req.Description
	}

	if req.Parent > 0 {
		parentID := strconv.FormatInt(req.Parent, 10)
		in.ParentID = &parentID
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		metaStr := string(metaBytes)
		in.Meta = &metaStr
	}

	cat, err := h.categories.Update(c.Context(), id, in)
	if err != nil {
		if err == categories.ErrNotFound {
			return ErrorNotFound(c, "rest_term_invalid", "Invalid category ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update category")
	}

	return OK(c, h.categoryToWP(cat, ContextEdit))
}

// DeleteCategory handles DELETE /wp/v2/categories/{id}
func (h *Handler) DeleteCategory(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)
	force := c.Query("force") == "true"

	if !force {
		return ErrorBadRequest(c, "rest_trash_not_supported", "Terms do not support trashing. Set force=true to delete.")
	}

	cat, err := h.categories.GetByID(c.Context(), id)
	if err != nil {
		if err == categories.ErrNotFound {
			return ErrorNotFound(c, "rest_term_invalid", "Invalid category ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read category")
	}

	if err := h.categories.Delete(c.Context(), id); err != nil {
		return ErrorInternal(c, "rest_cannot_delete", "Could not delete category")
	}

	wpCat := h.categoryToWP(cat, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpCat,
	})
}

// categoryToWP converts an internal category to WordPress format.
func (h *Handler) categoryToWP(cat *categories.Category, context string) WPCategory {
	numericID := ULIDToNumericID(cat.ID)

	var parent int64
	if cat.ParentID != "" {
		parent = ULIDToNumericID(cat.ParentID)
	}

	wp := WPCategory{
		ID:          numericID,
		Count:       cat.PostCount,
		Description: cat.Description,
		Link:        h.CategoryURL(cat.Slug),
		Name:        cat.Name,
		Slug:        cat.Slug,
		Taxonomy:    "category",
		Parent:      parent,
		Meta:        []any{},
	}

	wp.Links = map[string][]WPLink{
		"self":           {h.SelfLink("/categories/" + strconv.FormatInt(numericID, 10))},
		"collection":     {h.CollectionLink("/categories")},
		"about":          {h.AboutLink("/taxonomies/category")},
		"wp:post_type":   {h.CollectionLink("/posts?categories=" + strconv.FormatInt(numericID, 10))},
	}

	if parent > 0 {
		wp.Links["up"] = []WPLink{h.EmbeddableLink("/categories/" + strconv.FormatInt(parent, 10))}
	}

	return wp
}
