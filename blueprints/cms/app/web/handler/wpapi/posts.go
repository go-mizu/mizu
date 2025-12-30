package wpapi

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/posts"
)

// ListPosts handles GET /wp/v2/posts
func (h *Handler) ListPosts(c *mizu.Ctx) error {
	params := ParseListParams(c)

	// Build list input
	in := &posts.ListIn{
		Search:  params.Search,
		Limit:   params.PerPage,
		Offset:  params.Offset,
		OrderBy: mapPostOrderBy(params.OrderBy),
		Order:   params.Order,
	}

	// Apply offset from pagination
	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Status filter - default to published for unauthenticated
	status := c.Query("status")
	if status == "" || !h.IsAuthenticated(c) {
		in.Status = "published"
	} else {
		internalStatus, _ := MapWPPostStatus(status)
		in.Status = internalStatus
	}

	// Author filter
	if author := c.Query("author"); author != "" {
		in.AuthorID = author
	}

	// Category filter
	if categories := c.Query("categories"); categories != "" {
		// Take first category for now
		parts := strings.Split(categories, ",")
		if len(parts) > 0 {
			in.CategoryID = parts[0]
		}
	}

	// Tag filter
	if tags := c.Query("tags"); tags != "" {
		parts := strings.Split(tags, ",")
		if len(parts) > 0 {
			in.TagID = parts[0]
		}
	}

	// Sticky filter
	if sticky := c.Query("sticky"); sticky != "" {
		val := sticky == "true"
		in.IsFeatured = &val
	}

	// Slug filter
	if slug := c.Query("slug"); slug != "" {
		post, err := h.posts.GetBySlug(c.Context(), slug)
		if err != nil {
			return OKList(c, []WPPost{}, 0, params.Page, params.PerPage)
		}
		wpPost := h.postToWP(post, params.Context)
		return OKList(c, []WPPost{wpPost}, 1, params.Page, params.PerPage)
	}

	list, total, err := h.posts.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read posts")
	}

	// Convert to WordPress format
	wpPosts := make([]WPPost, len(list))
	for i, post := range list {
		wpPosts[i] = h.postToWP(post, params.Context)
	}

	return OKList(c, wpPosts, total, params.Page, params.PerPage)
}

// CreatePost handles POST /wp/v2/posts
func (h *Handler) CreatePost(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	var req WPCreatePostRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	// Extract title
	title := ExtractRawContent(req.Title)
	if title == "" {
		return ErrorInvalidParam(c, "title", "Title is required")
	}

	// Map status
	status, visibility := MapWPPostStatus(req.Status)
	if req.Status == "" {
		status = "draft"
		visibility = "public"
	}

	// Build create input
	in := &posts.CreateIn{
		Title:         title,
		Slug:          req.Slug,
		Content:       ExtractRawContent(req.Content),
		Excerpt:       ExtractRawContent(req.Excerpt),
		Status:        status,
		Visibility:    visibility,
		ContentFormat: "html",
	}

	// Featured media
	if req.FeaturedMedia > 0 {
		in.FeaturedImageID = strconv.FormatInt(req.FeaturedMedia, 10)
	}

	// Comment status
	if req.CommentStatus == "closed" {
		allowComments := false
		in.AllowComments = &allowComments
	} else if req.CommentStatus == "open" {
		allowComments := true
		in.AllowComments = &allowComments
	}

	// Categories and tags (convert numeric IDs to string)
	for _, catID := range req.Categories {
		in.CategoryIDs = append(in.CategoryIDs, strconv.FormatInt(catID, 10))
	}
	for _, tagID := range req.Tags {
		in.TagIDs = append(in.TagIDs, strconv.FormatInt(tagID, 10))
	}

	// Meta
	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		in.Meta = string(metaBytes)
	}

	userID := h.getUserID(c)
	post, err := h.posts.Create(c.Context(), userID, in)
	if err != nil {
		if err == posts.ErrMissingTitle {
			return ErrorInvalidParam(c, "title", "Title is required")
		}
		return ErrorInternal(c, "rest_cannot_create", "Could not create post")
	}

	// Handle sticky
	if req.Sticky != nil && *req.Sticky {
		stickyVal := true
		h.posts.Update(c.Context(), post.ID, &posts.UpdateIn{IsSticky: &stickyVal})
		post.IsSticky = true
	}

	return Created(c, h.postToWP(post, ContextEdit))
}

// GetPost handles GET /wp/v2/posts/{id}
func (h *Handler) GetPost(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	post, err := h.posts.GetByID(c.Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid post ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read post")
	}

	// Check if can view non-published posts
	if post.Status != "published" && !h.IsAuthenticated(c) {
		return ErrorNotFound(c, "rest_post_invalid_id", "Invalid post ID.")
	}

	return OK(c, h.postToWP(post, context))
}

// UpdatePost handles POST/PUT/PATCH /wp/v2/posts/{id}
func (h *Handler) UpdatePost(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPCreatePostRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &posts.UpdateIn{}

	// Title
	if req.Title != nil {
		title := ExtractRawContent(req.Title)
		in.Title = &title
	}

	// Content
	if req.Content != nil {
		content := ExtractRawContent(req.Content)
		in.Content = &content
	}

	// Excerpt
	if req.Excerpt != nil {
		excerpt := ExtractRawContent(req.Excerpt)
		in.Excerpt = &excerpt
	}

	// Slug
	if req.Slug != "" {
		in.Slug = &req.Slug
	}

	// Status
	if req.Status != "" {
		status, visibility := MapWPPostStatus(req.Status)
		in.Status = &status
		in.Visibility = &visibility
	}

	// Featured media
	if req.FeaturedMedia > 0 {
		mediaID := strconv.FormatInt(req.FeaturedMedia, 10)
		in.FeaturedImageID = &mediaID
	}

	// Comment status
	if req.CommentStatus != "" {
		allowComments := req.CommentStatus == "open"
		in.AllowComments = &allowComments
	}

	// Sticky
	if req.Sticky != nil {
		in.IsSticky = req.Sticky
	}

	// Categories and tags
	if len(req.Categories) > 0 {
		for _, catID := range req.Categories {
			in.CategoryIDs = append(in.CategoryIDs, strconv.FormatInt(catID, 10))
		}
	}
	if len(req.Tags) > 0 {
		for _, tagID := range req.Tags {
			in.TagIDs = append(in.TagIDs, strconv.FormatInt(tagID, 10))
		}
	}

	// Meta
	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		metaStr := string(metaBytes)
		in.Meta = &metaStr
	}

	post, err := h.posts.Update(c.Context(), id, in)
	if err != nil {
		if err == posts.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid post ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update post")
	}

	return OK(c, h.postToWP(post, ContextEdit))
}

// DeletePost handles DELETE /wp/v2/posts/{id}
func (h *Handler) DeletePost(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)
	force := c.Query("force") == "true"

	// Get post first for response
	post, err := h.posts.GetByID(c.Context(), id)
	if err != nil {
		if err == posts.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid post ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read post")
	}

	if force {
		if err := h.posts.Delete(c.Context(), id); err != nil {
			return ErrorInternal(c, "rest_cannot_delete", "Could not delete post")
		}
	} else {
		// Move to trash (set status to trash)
		trashStatus := "trash"
		if _, err := h.posts.Update(c.Context(), id, &posts.UpdateIn{Status: &trashStatus}); err != nil {
			return ErrorInternal(c, "rest_cannot_delete", "Could not trash post")
		}
	}

	wpPost := h.postToWP(post, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpPost,
	})
}

// postToWP converts an internal post to WordPress format.
func (h *Handler) postToWP(p *posts.Post, context string) WPPost {
	numericID := ULIDToNumericID(p.ID)

	// Determine publish date
	publishDate := p.CreatedAt
	if p.PublishedAt != nil {
		publishDate = *p.PublishedAt
	}

	// Get category and tag IDs (we skip this for now to avoid nil context issues)
	// Categories and tags can be fetched separately if needed
	categoryIDs := make([]int64, 0)
	tagIDs := make([]int64, 0)

	// Featured media
	var featuredMedia int64
	if p.FeaturedImageID != "" {
		featuredMedia = ULIDToNumericID(p.FeaturedImageID)
	}

	// Comment status
	commentStatus := "closed"
	if p.AllowComments {
		commentStatus = "open"
	}

	// Map status
	wpStatus := MapPostStatus(p.Status, p.Visibility)

	wp := WPPost{
		ID:          numericID,
		Date:        FormatWPDateTime(publishDate),
		DateGMT:     FormatWPDateTimeGMT(publishDate),
		GUID:        WPRendered{Rendered: h.PostURL(p.Slug)},
		Modified:    FormatWPDateTime(p.UpdatedAt),
		ModifiedGMT: FormatWPDateTimeGMT(p.UpdatedAt),
		Slug:        p.Slug,
		Status:      wpStatus,
		Type:        "post",
		Link:        h.PostURL(p.Slug),
		Title: WPRendered{
			Rendered: p.Title,
		},
		Content: WPContent{
			Rendered:  p.Content,
			Protected: p.Visibility == "password",
		},
		Excerpt: WPContent{
			Rendered:  p.Excerpt,
			Protected: p.Visibility == "password",
		},
		Author:        ULIDToNumericID(p.AuthorID),
		FeaturedMedia: featuredMedia,
		CommentStatus: commentStatus,
		PingStatus:    "closed",
		Sticky:        p.IsSticky,
		Template:      "",
		Format:        "standard",
		Meta:          []any{},
		Categories:    categoryIDs,
		Tags:          tagIDs,
	}

	// Add raw content in edit context
	if context == ContextEdit {
		wp.Title.Raw = p.Title
		wp.Content.Raw = p.Content
		wp.Excerpt.Raw = p.Excerpt
	}

	// Add links
	wp.Links = map[string][]WPLink{
		"self":       {h.SelfLink("/posts/" + strconv.FormatInt(numericID, 10))},
		"collection": {h.CollectionLink("/posts")},
		"about":      {h.AboutLink("/types/post")},
		"author":     {h.EmbeddableLink("/users/" + strconv.FormatInt(ULIDToNumericID(p.AuthorID), 10))},
		"replies":    {h.EmbeddableLink("/comments?post=" + strconv.FormatInt(numericID, 10))},
	}

	if len(categoryIDs) > 0 {
		wp.Links["wp:term"] = append(wp.Links["wp:term"], h.TaxonomyLink("/categories?post="+strconv.FormatInt(numericID, 10), "category"))
	}
	if len(tagIDs) > 0 {
		wp.Links["wp:term"] = append(wp.Links["wp:term"], h.TaxonomyLink("/tags?post="+strconv.FormatInt(numericID, 10), "post_tag"))
	}
	if featuredMedia > 0 {
		wp.Links["wp:featuredmedia"] = []WPLink{h.EmbeddableLink("/media/" + strconv.FormatInt(featuredMedia, 10))}
	}

	return wp
}

// mapPostOrderBy maps WordPress orderby to internal field.
func mapPostOrderBy(orderBy string) string {
	switch orderBy {
	case "date":
		return "created_at"
	case "modified":
		return "updated_at"
	case "title":
		return "title"
	case "slug":
		return "slug"
	case "author":
		return "author_id"
	case "id", "include":
		return "id"
	default:
		return "created_at"
	}
}
