package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu"
)

// Media handles media upload endpoints.
type Media struct {
	uploadDir string
	baseURL   string
	getUserID func(c *mizu.Ctx) string
}

// NewMedia creates a new media handler.
func NewMedia(uploadDir string, getUserID func(c *mizu.Ctx) string) *Media {
	return &Media{
		uploadDir: uploadDir,
		baseURL:   "/uploads",
		getUserID: getUserID,
	}
}

// Upload handles file uploads.
func (h *Media) Upload(c *mizu.Ctx) error {
	// Parse multipart form with max 50MB
	if err := c.Request().ParseMultipartForm(50 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse form"})
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no file provided"})
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !isAllowedContentType(contentType) {
		// Try to detect from extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		contentType = getContentTypeFromExt(ext)
		if !isAllowedContentType(contentType) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported file type"})
		}
	}

	// Generate unique filename
	id := generateID()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = getExtFromContentType(contentType)
	}
	filename := id + ext

	// Ensure upload directory exists
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create upload directory"})
	}

	// Create destination file
	destPath := filepath.Join(h.uploadDir, filename)
	dest, err := os.Create(destPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file"})
	}
	defer dest.Close()

	// Copy file content
	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(destPath) // Cleanup on error
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file"})
	}

	// Return file info
	url := h.baseURL + "/" + filename
	return c.JSON(http.StatusOK, map[string]string{
		"id":       id,
		"url":      url,
		"filename": header.Filename,
		"type":     contentType,
	})
}

// generateID generates a random ID for file names.
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(bytes)
}

// isAllowedContentType checks if the content type is allowed.
func isAllowedContentType(contentType string) bool {
	allowed := map[string]bool{
		// Images
		"image/jpeg":    true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		"image/bmp":     true,
		"image/tiff":    true,
		// Videos
		"video/mp4":       true,
		"video/webm":      true,
		"video/ogg":       true,
		"video/quicktime": true,
		"video/x-msvideo": true,
		// Audio
		"audio/mpeg":  true,
		"audio/wav":   true,
		"audio/ogg":   true,
		"audio/webm":  true,
		"audio/flac":  true,
		"audio/aac":   true,
		"audio/x-m4a": true,
		// Documents
		"application/pdf":                                                               true,
		"application/msword":                                                            true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":       true,
		"application/vnd.ms-excel":                                                      true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":             true,
		"application/vnd.ms-powerpoint":                                                 true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":     true,
		"text/plain":       true,
		"text/csv":         true,
		"application/json": true,
		"application/zip":  true,
		"application/x-zip-compressed": true,
		"application/x-rar-compressed": true,
		"application/x-7z-compressed":  true,
		"application/gzip":             true,
	}
	return allowed[contentType]
}

// getContentTypeFromExt gets content type from file extension.
func getContentTypeFromExt(ext string) string {
	types := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/x-m4a",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".json": "application/json",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".gz":   "application/gzip",
	}
	if ct, ok := types[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// getExtFromContentType gets file extension from content type.
func getExtFromContentType(contentType string) string {
	exts := map[string]string{
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"image/svg+xml":   ".svg",
		"video/mp4":       ".mp4",
		"video/webm":      ".webm",
		"audio/mpeg":      ".mp3",
		"audio/wav":       ".wav",
		"application/pdf": ".pdf",
	}
	if ext, ok := exts[contentType]; ok {
		return ext
	}
	return ""
}
