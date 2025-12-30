package wpapi

import (
	"encoding/json"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/pages"
)

// ListPages handles GET /wp/v2/pages
func (h *Handler) ListPages(c *mizu.Ctx) error {
	params := ParseListParams(c)

	in := &pages.ListIn{
		Search: params.Search,
		Limit:  params.PerPage,
		Offset: params.Offset,
	}

	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Status filter
	status := c.Query("status")
	if status == "" || !h.IsAuthenticated(c) {
		in.Status = "published"
	} else {
		internalStatus, _ := MapWPPostStatus(status)
		in.Status = internalStatus
	}

	// Parent filter
	if parent := c.Query("parent"); parent != "" {
		in.ParentID = parent
	}

	// Author filter
	if author := c.Query("author"); author != "" {
		in.AuthorID = author
	}

	// Slug filter
	if slug := c.Query("slug"); slug != "" {
		page, err := h.pages.GetBySlug(c.Context(), slug)
		if err != nil {
			return OKList(c, []WPPage{}, 0, params.Page, params.PerPage)
		}
		wpPage := h.pageToWP(page, params.Context)
		return OKList(c, []WPPage{wpPage}, 1, params.Page, params.PerPage)
	}

	list, total, err := h.pages.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read pages")
	}

	wpPages := make([]WPPage, len(list))
	for i, page := range list {
		wpPages[i] = h.pageToWP(page, params.Context)
	}

	return OKList(c, wpPages, total, params.Page, params.PerPage)
}

// CreatePage handles POST /wp/v2/pages
func (h *Handler) CreatePage(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	var req WPCreatePageRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	title := ExtractRawContent(req.Title)
	if title == "" {
		return ErrorInvalidParam(c, "title", "Title is required")
	}

	status, visibility := MapWPPostStatus(req.Status)
	if req.Status == "" {
		status = "draft"
		visibility = "public"
	}

	in := &pages.CreateIn{
		Title:         title,
		Slug:          req.Slug,
		Content:       ExtractRawContent(req.Content),
		Status:        status,
		Visibility:    visibility,
		ContentFormat: "html",
		Template:      req.Template,
		SortOrder:     req.MenuOrder,
	}

	// Parent
	if req.Parent > 0 {
		in.ParentID = strconv.FormatInt(req.Parent, 10)
	}

	// Featured media
	if req.FeaturedMedia > 0 {
		in.FeaturedImageID = strconv.FormatInt(req.FeaturedMedia, 10)
	}

	// Meta
	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		in.Meta = string(metaBytes)
	}

	userID := h.getUserID(c)
	page, err := h.pages.Create(c.Context(), userID, in)
	if err != nil {
		if err == pages.ErrMissingTitle {
			return ErrorInvalidParam(c, "title", "Title is required")
		}
		return ErrorInternal(c, "rest_cannot_create", "Could not create page")
	}

	return Created(c, h.pageToWP(page, ContextEdit))
}

// GetPage handles GET /wp/v2/pages/{id}
func (h *Handler) GetPage(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	page, err := h.pages.GetByID(c.Context(), id)
	if err != nil {
		if err == pages.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid page ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read page")
	}

	if page.Status != "published" && !h.IsAuthenticated(c) {
		return ErrorNotFound(c, "rest_post_invalid_id", "Invalid page ID.")
	}

	return OK(c, h.pageToWP(page, context))
}

// UpdatePage handles POST/PUT/PATCH /wp/v2/pages/{id}
func (h *Handler) UpdatePage(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPCreatePageRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &pages.UpdateIn{}

	if req.Title != nil {
		title := ExtractRawContent(req.Title)
		in.Title = &title
	}

	if req.Content != nil {
		content := ExtractRawContent(req.Content)
		in.Content = &content
	}

	if req.Slug != "" {
		in.Slug = &req.Slug
	}

	if req.Status != "" {
		status, visibility := MapWPPostStatus(req.Status)
		in.Status = &status
		in.Visibility = &visibility
	}

	if req.Parent > 0 {
		parentID := strconv.FormatInt(req.Parent, 10)
		in.ParentID = &parentID
	}

	if req.FeaturedMedia > 0 {
		mediaID := strconv.FormatInt(req.FeaturedMedia, 10)
		in.FeaturedImageID = &mediaID
	}

	if req.Template != "" {
		in.Template = &req.Template
	}

	if req.MenuOrder != 0 {
		in.SortOrder = &req.MenuOrder
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		metaStr := string(metaBytes)
		in.Meta = &metaStr
	}

	page, err := h.pages.Update(c.Context(), id, in)
	if err != nil {
		if err == pages.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid page ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update page")
	}

	return OK(c, h.pageToWP(page, ContextEdit))
}

// DeletePage handles DELETE /wp/v2/pages/{id}
func (h *Handler) DeletePage(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)
	force := c.Query("force") == "true"

	page, err := h.pages.GetByID(c.Context(), id)
	if err != nil {
		if err == pages.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid page ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read page")
	}

	if force {
		if err := h.pages.Delete(c.Context(), id); err != nil {
			return ErrorInternal(c, "rest_cannot_delete", "Could not delete page")
		}
	} else {
		trashStatus := "trash"
		if _, err := h.pages.Update(c.Context(), id, &pages.UpdateIn{Status: &trashStatus}); err != nil {
			return ErrorInternal(c, "rest_cannot_delete", "Could not trash page")
		}
	}

	wpPage := h.pageToWP(page, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpPage,
	})
}

// pageToWP converts an internal page to WordPress format.
func (h *Handler) pageToWP(p *pages.Page, context string) WPPage {
	numericID := ULIDToNumericID(p.ID)

	var parent int64
	if p.ParentID != "" {
		parent = ULIDToNumericID(p.ParentID)
	}

	var featuredMedia int64
	if p.FeaturedImageID != "" {
		featuredMedia = ULIDToNumericID(p.FeaturedImageID)
	}

	wpStatus := MapPostStatus(p.Status, p.Visibility)

	wp := WPPage{
		ID:          numericID,
		Date:        FormatWPDateTime(p.CreatedAt),
		DateGMT:     FormatWPDateTimeGMT(p.CreatedAt),
		GUID:        WPRendered{Rendered: h.PageURL(p.Slug)},
		Modified:    FormatWPDateTime(p.UpdatedAt),
		ModifiedGMT: FormatWPDateTimeGMT(p.UpdatedAt),
		Slug:        p.Slug,
		Status:      wpStatus,
		Type:        "page",
		Link:        h.PageURL(p.Slug),
		Title: WPRendered{
			Rendered: p.Title,
		},
		Content: WPContent{
			Rendered:  p.Content,
			Protected: p.Visibility == "password",
		},
		Excerpt: WPContent{
			Rendered:  "",
			Protected: p.Visibility == "password",
		},
		Author:        ULIDToNumericID(p.AuthorID),
		FeaturedMedia: featuredMedia,
		Parent:        parent,
		MenuOrder:     p.SortOrder,
		CommentStatus: "closed",
		PingStatus:    "closed",
		Template:      p.Template,
		Meta:          []any{},
	}

	if context == ContextEdit {
		wp.Title.Raw = p.Title
		wp.Content.Raw = p.Content
	}

	wp.Links = map[string][]WPLink{
		"self":       {h.SelfLink("/pages/" + strconv.FormatInt(numericID, 10))},
		"collection": {h.CollectionLink("/pages")},
		"about":      {h.AboutLink("/types/page")},
		"author":     {h.EmbeddableLink("/users/" + strconv.FormatInt(ULIDToNumericID(p.AuthorID), 10))},
	}

	if featuredMedia > 0 {
		wp.Links["wp:featuredmedia"] = []WPLink{h.EmbeddableLink("/media/" + strconv.FormatInt(featuredMedia, 10))}
	}

	return wp
}
