package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-mizu/blueprints/messaging/feature/media"
	"github.com/go-mizu/mizu"
)

// MediaHandler handles media-related requests.
type MediaHandler struct {
	media     media.API
	fileStore *media.LocalFileStore
	getUserID func(*mizu.Ctx) string
}

// NewMediaHandler creates a new MediaHandler.
func NewMediaHandler(m media.API, fs *media.LocalFileStore, getUserID func(*mizu.Ctx) string) *MediaHandler {
	return &MediaHandler{
		media:     m,
		fileStore: fs,
		getUserID: getUserID,
	}
}

// Upload handles file uploads.
func (h *MediaHandler) Upload(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "authentication required")
	}

	// Parse multipart form (max 100MB)
	if err := c.Request().ParseMultipartForm(100 << 20); err != nil {
		return BadRequest(c, "invalid form data")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return BadRequest(c, "no file provided")
	}
	defer file.Close()

	// Get optional parameters
	mediaType := media.MediaType(c.Request().FormValue("type"))
	isViewOnce := c.Request().FormValue("view_once") == "true"

	// Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		// Try to detect from extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		contentType = mimeTypeFromExt(ext)
	}

	in := &media.UploadIn{
		Reader:      file,
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
		Type:        mediaType,
		IsViewOnce:  isViewOnce,
	}

	m, err := h.media.Upload(c.Request().Context(), userID, in)
	if err != nil {
		switch err {
		case media.ErrFileTooLarge:
			return BadRequest(c, "file too large")
		case media.ErrInvalidType:
			return BadRequest(c, "invalid file type")
		default:
			return InternalError(c, "upload failed")
		}
	}

	return Created(c, m)
}

// Download serves a media file.
func (h *MediaHandler) Download(c *mizu.Ctx) error {
	id := c.Param("id")

	m, err := h.media.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to get media")
	}

	// Get file path
	filePath := h.fileStore.GetPath(c.Request().Context(), m.URL)
	if filePath == "" {
		return NotFound(c, "file not found")
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return NotFound(c, "file not found")
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return InternalError(c, "failed to read file")
	}

	// Set headers
	c.Writer().Header().Set("Content-Type", m.ContentType)
	c.Writer().Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	c.Writer().Header().Set("Content-Disposition", "inline; filename=\""+m.OriginalFilename+"\"")
	c.Writer().Header().Set("Cache-Control", "private, max-age=31536000")

	// Serve file with range support
	http.ServeContent(c.Writer(), c.Request(), m.OriginalFilename, stat.ModTime(), file)
	return nil
}

// Thumbnail serves a media thumbnail.
func (h *MediaHandler) Thumbnail(c *mizu.Ctx) error {
	id := c.Param("id")

	m, err := h.media.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to get media")
	}

	// Use thumbnail URL if available, otherwise fall back to original
	url := m.ThumbnailURL
	if url == "" {
		url = m.URL
	}

	filePath := h.fileStore.GetPath(c.Request().Context(), url)
	if filePath == "" {
		return NotFound(c, "file not found")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return NotFound(c, "file not found")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return InternalError(c, "failed to read file")
	}

	c.Writer().Header().Set("Content-Type", m.ContentType)
	c.Writer().Header().Set("Cache-Control", "public, max-age=31536000")

	http.ServeContent(c.Writer(), c.Request(), "", stat.ModTime(), file)
	return nil
}

// Stream streams a video or audio file.
func (h *MediaHandler) Stream(c *mizu.Ctx) error {
	// Stream is handled the same as Download with range support
	return h.Download(c)
}

// Delete deletes a media file.
func (h *MediaHandler) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "authentication required")
	}

	id := c.Param("id")

	if err := h.media.Delete(c.Request().Context(), id, userID); err != nil {
		switch err {
		case media.ErrNotFound:
			return NotFound(c, "media not found")
		case media.ErrUnauthorized:
			return Forbidden(c, "not authorized")
		default:
			return InternalError(c, "failed to delete media")
		}
	}

	return Success(c, nil)
}

// GetByID returns media metadata.
func (h *MediaHandler) GetByID(c *mizu.Ctx) error {
	id := c.Param("id")

	m, err := h.media.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to get media")
	}

	return Success(c, m)
}

// View handles view-once media viewing.
func (h *MediaHandler) View(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "authentication required")
	}

	id := c.Param("id")

	m, err := h.media.ViewMedia(c.Request().Context(), id, userID)
	if err != nil {
		if err == media.ErrNotFound {
			return NotFound(c, "media not found")
		}
		return InternalError(c, "failed to view media")
	}

	// For view-once, return the file directly
	filePath := h.fileStore.GetPath(c.Request().Context(), m.URL)
	if filePath == "" {
		return NotFound(c, "file not found")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return NotFound(c, "file not found")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return InternalError(c, "failed to read file")
	}

	c.Writer().Header().Set("Content-Type", m.ContentType)
	c.Writer().Header().Set("Cache-Control", "no-store")

	http.ServeContent(c.Writer(), c.Request(), m.OriginalFilename, stat.ModTime(), file)
	return nil
}

// ListChatMedia lists media in a chat.
func (h *MediaHandler) ListChatMedia(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "authentication required")
	}

	chatID := c.Param("id")
	mediaType := media.MediaType(c.Request().URL.Query().Get("type"))
	limit, _ := strconv.Atoi(c.Request().URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(c.Request().URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	opts := media.ListOpts{
		Type:   mediaType,
		Limit:  limit,
		Offset: offset,
	}

	list, err := h.media.ListByChat(c.Request().Context(), chatID, opts)
	if err != nil {
		return InternalError(c, "failed to list media")
	}

	return Success(c, list)
}

// ServeMedia serves media files directly from URL path.
// This handles /media/* routes.
func (h *MediaHandler) ServeMedia(c *mizu.Ctx) error {
	// Get the path after /media/
	path := strings.TrimPrefix(c.Request().URL.Path, "/media/")
	if path == "" {
		return NotFound(c, "file not found")
	}

	filePath := filepath.Join(h.fileStore.GetBasePath(), path)

	// Security: ensure path is within base directory
	absBase, _ := filepath.Abs(h.fileStore.GetBasePath())
	absPath, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absPath, absBase) {
		return Forbidden(c, "invalid path")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return NotFound(c, "file not found")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		return NotFound(c, "file not found")
	}

	// Detect content type
	ext := filepath.Ext(filePath)
	contentType := mimeTypeFromExt(ext)

	c.Writer().Header().Set("Content-Type", contentType)
	c.Writer().Header().Set("Cache-Control", "public, max-age=31536000")

	http.ServeContent(c.Writer(), c.Request(), stat.Name(), stat.ModTime(), file)
	return nil
}

// Helper to get MIME type from file extension
func mimeTypeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	// Images
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
	// Videos
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	// Audio
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	// Documents
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// Compile-time interface check
var _ io.Closer = (*os.File)(nil)
