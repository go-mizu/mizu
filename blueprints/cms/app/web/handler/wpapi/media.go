package wpapi

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/media"
)

// ListMedia handles GET /wp/v2/media
func (h *Handler) ListMedia(c *mizu.Ctx) error {
	params := ParseListParams(c)

	in := &media.ListIn{
		Search:  params.Search,
		Limit:   params.PerPage,
		Offset:  params.Offset,
		OrderBy: mapMediaOrderBy(params.OrderBy),
		Order:   params.Order,
	}

	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Author filter
	if author := c.Query("author"); author != "" {
		in.UploaderID = author
	}

	// Media type filter
	if mediaType := c.Query("media_type"); mediaType != "" {
		// Convert WordPress media_type to mime type prefix
		switch mediaType {
		case "image":
			in.MimeType = "image/"
		case "video":
			in.MimeType = "video/"
		case "audio":
			in.MimeType = "audio/"
		case "application":
			in.MimeType = "application/"
		}
	}

	// Mime type filter
	if mimeType := c.Query("mime_type"); mimeType != "" {
		in.MimeType = mimeType
	}

	list, total, err := h.media.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read media")
	}

	wpMedia := make([]WPMedia, len(list))
	for i, m := range list {
		wpMedia[i] = h.mediaToWP(m, params.Context)
	}

	return OKList(c, wpMedia, total, params.Page, params.PerPage)
}

// UploadMedia handles POST /wp/v2/media
func (h *Handler) UploadMedia(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	// Parse multipart form
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil { // 32MB max
		return ErrorBadRequest(c, "rest_upload_no_data", "No data supplied")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return ErrorBadRequest(c, "rest_upload_no_data", "No data supplied")
	}
	defer file.Close()

	// Get optional fields from form
	title := c.Request().FormValue("title")
	if title == "" {
		// Use filename without extension as title
		title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	in := &media.UploadIn{
		File:        file,
		Filename:    header.Filename,
		MimeType:    header.Header.Get("Content-Type"),
		FileSize:    header.Size,
		Title:       title,
		AltText:     c.Request().FormValue("alt_text"),
		Caption:     c.Request().FormValue("caption"),
		Description: c.Request().FormValue("description"),
	}

	// Detect mime type if not provided or if it's a generic type
	if in.MimeType == "" || in.MimeType == "application/octet-stream" {
		in.MimeType = detectMimeType(header.Filename)
	}

	userID := h.getUserID(c)
	m, err := h.media.Upload(c.Context(), userID, in)
	if err != nil {
		return ErrorInternal(c, "rest_upload_failed", "Could not upload media")
	}

	return Created(c, h.mediaToWP(m, ContextEdit))
}

// GetMedia handles GET /wp/v2/media/{id}
func (h *Handler) GetMedia(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	m, err := h.media.GetByID(c.Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid media ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read media")
	}

	return OK(c, h.mediaToWP(m, context))
}

// UpdateMedia handles POST/PUT/PATCH /wp/v2/media/{id}
func (h *Handler) UpdateMedia(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPUpdateMediaRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &media.UpdateIn{}

	if req.Title != nil {
		title := ExtractRawContent(req.Title)
		in.Title = &title
	}

	if req.AltText != "" {
		in.AltText = &req.AltText
	}

	if req.Caption != nil {
		caption := ExtractRawContent(req.Caption)
		in.Caption = &caption
	}

	if req.Description != nil {
		description := ExtractRawContent(req.Description)
		in.Description = &description
	}

	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		metaStr := string(metaBytes)
		in.Meta = &metaStr
	}

	m, err := h.media.Update(c.Context(), id, in)
	if err != nil {
		if err == media.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid media ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update media")
	}

	return OK(c, h.mediaToWP(m, ContextEdit))
}

// DeleteMedia handles DELETE /wp/v2/media/{id}
func (h *Handler) DeleteMedia(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)
	force := c.Query("force") == "true"

	if !force {
		return ErrorBadRequest(c, "rest_trash_not_supported", "Media does not support trashing. Set force=true to delete.")
	}

	m, err := h.media.GetByID(c.Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return ErrorNotFound(c, "rest_post_invalid_id", "Invalid media ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read media")
	}

	if err := h.media.Delete(c.Context(), id); err != nil {
		return ErrorInternal(c, "rest_cannot_delete", "Could not delete media")
	}

	wpMedia := h.mediaToWP(m, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpMedia,
	})
}

// mediaToWP converts an internal media item to WordPress format.
func (h *Handler) mediaToWP(m *media.Media, context string) WPMedia {
	numericID := ULIDToNumericID(m.ID)

	// Determine media type
	mediaType := "file"
	if strings.HasPrefix(m.MimeType, "image/") {
		mediaType = "image"
	} else if strings.HasPrefix(m.MimeType, "video/") {
		mediaType = "video"
	} else if strings.HasPrefix(m.MimeType, "audio/") {
		mediaType = "audio"
	}

	// Build media details
	details := WPMediaDetails{
		Width:    m.Width,
		Height:   m.Height,
		File:     m.Filename,
		FileSize: m.FileSize,
	}

	// Add image sizes for images
	if mediaType == "image" {
		details.Sizes = map[string]WPImageSize{
			"full": {
				File:      m.Filename,
				Width:     m.Width,
				Height:    m.Height,
				MimeType:  m.MimeType,
				SourceURL: h.MediaURL(m.URL),
			},
		}
	}

	// Title defaults to filename
	title := m.Title
	if title == "" {
		title = m.OriginalFilename
	}

	wp := WPMedia{
		ID:          numericID,
		Date:        FormatWPDateTime(m.CreatedAt),
		DateGMT:     FormatWPDateTimeGMT(m.CreatedAt),
		GUID:        WPRendered{Rendered: h.MediaURL(m.URL)},
		Modified:    FormatWPDateTime(m.UpdatedAt),
		ModifiedGMT: FormatWPDateTimeGMT(m.UpdatedAt),
		Slug:        m.Filename,
		Status:      "inherit",
		Type:        "attachment",
		Link:        h.MediaURL(m.URL),
		Title: WPRendered{
			Rendered: title,
		},
		Author:        ULIDToNumericID(m.UploaderID),
		CommentStatus: "closed",
		PingStatus:    "closed",
		Template:      "",
		Meta:          []any{},
		Description: WPContent{
			Rendered: m.Description,
		},
		Caption: WPContent{
			Rendered: m.Caption,
		},
		AltText:      m.AltText,
		MediaType:    mediaType,
		MimeType:     m.MimeType,
		MediaDetails: details,
		Post:         0, // Not associated with a post
		SourceURL:    h.MediaURL(m.URL),
	}

	if context == ContextEdit {
		wp.Title.Raw = title
		wp.Description.Raw = m.Description
		wp.Caption.Raw = m.Caption
	}

	wp.Links = map[string][]WPLink{
		"self":       {h.SelfLink("/media/" + strconv.FormatInt(numericID, 10))},
		"collection": {h.CollectionLink("/media")},
		"about":      {h.AboutLink("/types/attachment")},
		"author":     {h.EmbeddableLink("/users/" + strconv.FormatInt(ULIDToNumericID(m.UploaderID), 10))},
	}

	return wp
}

// mapMediaOrderBy maps WordPress orderby to internal field.
func mapMediaOrderBy(orderBy string) string {
	switch orderBy {
	case "date":
		return "created_at"
	case "modified":
		return "updated_at"
	case "title":
		return "title"
	case "id", "include":
		return "id"
	default:
		return "created_at"
	}
}

// detectMimeType detects mime type from filename extension.
func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}
