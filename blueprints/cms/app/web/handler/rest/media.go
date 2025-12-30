package rest

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/media"
)

// Media handles media endpoints.
type Media struct {
	media     media.API
	getUserID func(*mizu.Ctx) string
}

// NewMedia creates a new media handler.
func NewMedia(media media.API, getUserID func(*mizu.Ctx) string) *Media {
	return &Media{media: media, getUserID: getUserID}
}

// List lists media.
func (h *Media) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	in := &media.ListIn{
		UploaderID: c.Query("uploader_id"),
		MimeType:   c.Query("mime_type"),
		Search:     c.Query("search"),
		Limit:      perPage,
		Offset:     (page - 1) * perPage,
		OrderBy:    c.Query("sort"),
		Order:      c.Query("order"),
	}

	list, total, err := h.media.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list media")
	}

	return List(c, list, total, page, perPage)
}

// Upload handles file upload.
func (h *Media) Upload(c *mizu.Ctx) error {
	// Parse multipart form (max 32MB)
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return BadRequest(c, "failed to parse form")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return BadRequest(c, "file is required")
	}
	defer file.Close()

	userID := h.getUserID(c)
	in := &media.UploadIn{
		File:        file,
		Filename:    header.Filename,
		MimeType:    header.Header.Get("Content-Type"),
		FileSize:    header.Size,
		AltText:     c.Request().FormValue("alt_text"),
		Caption:     c.Request().FormValue("caption"),
		Title:       c.Request().FormValue("title"),
		Description: c.Request().FormValue("description"),
	}

	m, err := h.media.Upload(c.Context(), userID, in)
	if err != nil {
		if err == media.ErrInvalidMimeType {
			return BadRequest(c, "invalid file type")
		}
		return InternalError(c, "failed to upload media")
	}

	return Created(c, m)
}

// Get retrieves a media item by ID.
func (h *Media) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	m, err := h.media.GetByID(c.Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to get media")
	}

	return OK(c, m)
}

// Update updates media metadata.
func (h *Media) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in media.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	m, err := h.media.Update(c.Context(), id, &in)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to update media")
	}

	return OK(c, m)
}

// Delete deletes a media item.
func (h *Media) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.media.Delete(c.Context(), id); err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to delete media")
	}

	return OK(c, map[string]string{"message": "media deleted"})
}

// ServeFile serves the actual media file.
func (h *Media) ServeFile(c *mizu.Ctx) error {
	id := c.Param("id")

	reader, m, err := h.media.GetFile(c.Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to get file")
	}
	defer reader.Close()

	c.Writer().Header().Set("Content-Type", m.MimeType)
	c.Writer().Header().Set("Content-Disposition", "inline; filename=\""+m.OriginalFilename+"\"")
	c.Writer().Header().Set("Cache-Control", "public, max-age=31536000")
	c.Writer().WriteHeader(http.StatusOK)
	io.Copy(c.Writer(), reader)
	return nil
}
