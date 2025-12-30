package wpapi

import (
	"encoding/json"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/comments"
)

// ListComments handles GET /wp/v2/comments
func (h *Handler) ListComments(c *mizu.Ctx) error {
	params := ParseListParams(c)

	in := &comments.ListIn{
		Limit:  params.PerPage,
		Offset: params.Offset,
	}

	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Status filter - default to approved for unauthenticated
	status := c.Query("status")
	if status == "" {
		if h.IsAuthenticated(c) {
			// Show all for authenticated users
		} else {
			in.Status = "approved"
		}
	} else {
		in.Status = MapWPCommentStatus(status)
	}

	// Post filter
	if post := c.Query("post"); post != "" {
		in.PostID = post
	}

	// Parent filter
	if parent := c.Query("parent"); parent != "" {
		in.ParentID = parent
	}

	// Author filter
	if author := c.Query("author"); author != "" {
		in.AuthorID = author
	}

	list, total, err := h.comments.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read comments")
	}

	wpComments := make([]WPComment, len(list))
	for i, comment := range list {
		wpComments[i] = h.commentToWP(comment, params.Context)
	}

	return OKList(c, wpComments, total, params.Page, params.PerPage)
}

// CreateComment handles POST /wp/v2/comments
func (h *Handler) CreateComment(c *mizu.Ctx) error {
	var req WPCreateCommentRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	// Post is required
	if req.Post == 0 {
		return ErrorInvalidParam(c, "post", "Post ID is required")
	}

	// Content is required
	content := ExtractRawContent(req.Content)
	if content == "" {
		return ErrorInvalidParam(c, "content", "Content is required")
	}

	in := &comments.CreateIn{
		PostID:  strconv.FormatInt(req.Post, 10),
		Content: content,
	}

	// Parent comment
	if req.Parent > 0 {
		in.ParentID = strconv.FormatInt(req.Parent, 10)
	}

	// Author info
	if h.IsAuthenticated(c) {
		in.AuthorID = h.getUserID(c)
	} else {
		// Anonymous comment
		if req.AuthorName == "" {
			return ErrorInvalidParam(c, "author_name", "Author name is required for anonymous comments")
		}
		in.AuthorName = req.AuthorName
		in.AuthorEmail = req.AuthorEmail
		in.AuthorURL = req.AuthorURL
	}

	// Capture request info
	in.IPAddress = c.Request().RemoteAddr
	in.UserAgent = c.Request().UserAgent()

	comment, err := h.comments.Create(c.Context(), in)
	if err != nil {
		if err == comments.ErrMissingContent {
			return ErrorInvalidParam(c, "content", "Content is required")
		}
		if err == comments.ErrMissingPostID {
			return ErrorInvalidParam(c, "post", "Post ID is required")
		}
		return ErrorInternal(c, "rest_cannot_create", "Could not create comment")
	}

	return Created(c, h.commentToWP(comment, ContextEdit))
}

// GetComment handles GET /wp/v2/comments/{id}
func (h *Handler) GetComment(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	comment, err := h.comments.GetByID(c.Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return ErrorNotFound(c, "rest_comment_invalid_id", "Invalid comment ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read comment")
	}

	// Non-approved comments require auth
	if comment.Status != "approved" && !h.IsAuthenticated(c) {
		return ErrorNotFound(c, "rest_comment_invalid_id", "Invalid comment ID.")
	}

	return OK(c, h.commentToWP(comment, context))
}

// UpdateComment handles POST/PUT/PATCH /wp/v2/comments/{id}
func (h *Handler) UpdateComment(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPCreateCommentRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &comments.UpdateIn{}

	if req.Content != nil {
		content := ExtractRawContent(req.Content)
		in.Content = &content
	}

	if req.Status != "" {
		status := MapWPCommentStatus(req.Status)
		in.Status = &status
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		metaStr := string(metaBytes)
		in.Meta = &metaStr
	}

	comment, err := h.comments.Update(c.Context(), id, in)
	if err != nil {
		if err == comments.ErrNotFound {
			return ErrorNotFound(c, "rest_comment_invalid_id", "Invalid comment ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update comment")
	}

	return OK(c, h.commentToWP(comment, ContextEdit))
}

// DeleteComment handles DELETE /wp/v2/comments/{id}
func (h *Handler) DeleteComment(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)
	force := c.Query("force") == "true"

	comment, err := h.comments.GetByID(c.Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return ErrorNotFound(c, "rest_comment_invalid_id", "Invalid comment ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read comment")
	}

	if force {
		if err := h.comments.Delete(c.Context(), id); err != nil {
			return ErrorInternal(c, "rest_cannot_delete", "Could not delete comment")
		}
	} else {
		// Move to trash
		trashStatus := "trash"
		if _, err := h.comments.Update(c.Context(), id, &comments.UpdateIn{Status: &trashStatus}); err != nil {
			return ErrorInternal(c, "rest_cannot_delete", "Could not trash comment")
		}
	}

	wpComment := h.commentToWP(comment, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpComment,
	})
}

// commentToWP converts an internal comment to WordPress format.
func (h *Handler) commentToWP(cm *comments.Comment, context string) WPComment {
	numericID := ULIDToNumericID(cm.ID)

	var author int64
	if cm.AuthorID != "" {
		author = ULIDToNumericID(cm.AuthorID)
	}

	var parent int64
	if cm.ParentID != "" {
		parent = ULIDToNumericID(cm.ParentID)
	}

	wp := WPComment{
		ID:         numericID,
		Post:       ULIDToNumericID(cm.PostID),
		Parent:     parent,
		Author:     author,
		AuthorName: cm.AuthorName,
		AuthorURL:  cm.AuthorURL,
		Date:       FormatWPDateTime(cm.CreatedAt),
		DateGMT:    FormatWPDateTimeGMT(cm.CreatedAt),
		Content: WPContent{
			Rendered: cm.Content,
		},
		Link:             h.CommentURL("", numericID), // Simplified URL
		Status:           MapCommentStatus(cm.Status),
		Type:             "comment",
		AuthorAvatarURLs: AvatarURLs(cm.AuthorEmail),
		Meta:             []any{},
	}

	// Edit context includes more fields
	if context == ContextEdit {
		wp.AuthorEmail = cm.AuthorEmail
		wp.AuthorIP = cm.IPAddress
		wp.AuthorUserAgent = cm.UserAgent
		wp.Content.Raw = cm.Content
	}

	wp.Links = map[string][]WPLink{
		"self":       {h.SelfLink("/comments/" + strconv.FormatInt(numericID, 10))},
		"collection": {h.CollectionLink("/comments")},
		"up":         {h.EmbeddableLink("/posts/" + strconv.FormatInt(ULIDToNumericID(cm.PostID), 10))},
	}

	if author > 0 {
		wp.Links["author"] = []WPLink{h.EmbeddableLink("/users/" + strconv.FormatInt(author, 10))}
	}

	if parent > 0 {
		wp.Links["in-reply-to"] = []WPLink{h.EmbeddableLink("/comments/" + strconv.FormatInt(parent, 10))}
	}

	return wp
}
