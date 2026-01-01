// Package mime provides MIME type detection.
package mime

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Common MIME type categories.
const (
	CategoryDocument = "document"
	CategoryImage    = "image"
	CategoryVideo    = "video"
	CategoryAudio    = "audio"
	CategoryArchive  = "archive"
	CategoryCode     = "code"
	CategoryOther    = "other"
)

// DetectFromBytes detects MIME type from file content.
func DetectFromBytes(data []byte) string {
	mimeType := http.DetectContentType(data)
	// Clean up charset suffix if present
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	return mimeType
}

// DetectFromFilename guesses MIME type from filename extension.
func DetectFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	// Clean up charset suffix
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	return mimeType
}

// Category returns the category for a MIME type.
func Category(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return CategoryImage
	case strings.HasPrefix(mimeType, "video/"):
		return CategoryVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return CategoryAudio
	case isDocument(mimeType):
		return CategoryDocument
	case isArchive(mimeType):
		return CategoryArchive
	case isCode(mimeType):
		return CategoryCode
	default:
		return CategoryOther
	}
}

func isDocument(mimeType string) bool {
	docs := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument",
		"application/vnd.ms-excel",
		"application/vnd.ms-powerpoint",
		"application/vnd.oasis.opendocument",
		"text/plain",
		"text/rtf",
		"text/csv",
		"text/markdown",
	}
	for _, d := range docs {
		if strings.HasPrefix(mimeType, d) {
			return true
		}
	}
	return false
}

func isArchive(mimeType string) bool {
	archives := []string{
		"application/zip",
		"application/x-rar",
		"application/x-7z",
		"application/x-tar",
		"application/gzip",
		"application/x-bzip2",
	}
	for _, a := range archives {
		if mimeType == a {
			return true
		}
	}
	return false
}

func isCode(mimeType string) bool {
	if strings.HasPrefix(mimeType, "text/x-") {
		return true
	}
	codes := []string{
		"application/javascript",
		"application/json",
		"application/xml",
		"text/html",
		"text/css",
	}
	for _, c := range codes {
		if mimeType == c {
			return true
		}
	}
	return false
}

// Extension returns the file extension for a MIME type.
func Extension(mimeType string) string {
	exts, _ := mime.ExtensionsByType(mimeType)
	if len(exts) > 0 {
		return strings.TrimPrefix(exts[0], ".")
	}
	return ""
}

// IsThumbnailable returns true if thumbnails can be generated.
func IsThumbnailable(mimeType string) bool {
	if strings.HasPrefix(mimeType, "image/") {
		// Exclude SVG and some others
		if mimeType == "image/svg+xml" {
			return false
		}
		return true
	}
	if mimeType == "application/pdf" {
		return true
	}
	return false
}
