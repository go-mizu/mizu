package meta

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Service provides metadata extraction functionality.
type Service struct {
	cache sync.Map // map[string]*FileMetadata
}

// New creates a new metadata service.
func New() *Service {
	return &Service{}
}

// Extract extracts metadata from a file.
func (s *Service) Extract(ctx context.Context, filePath string) (*FileMetadata, error) {
	// Check cache first
	if cached, ok := s.cache.Load(filePath); ok {
		return cached.(*FileMetadata), nil
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Determine MIME type from extension
	mimeType := getMimeType(filePath)

	// Create base metadata
	meta := &FileMetadata{
		FileID:      filePath,
		ExtractedAt: time.Now(),
		MimeType:    mimeType,
		Size:        info.Size(),
	}

	// Extract type-specific metadata
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		if imgMeta, err := extractImageMetadata(ctx, filePath); err == nil {
			meta.Image = imgMeta
		}
	case strings.HasPrefix(mimeType, "audio/"):
		if audioMeta, err := extractAudioMetadata(ctx, filePath); err == nil {
			meta.Audio = audioMeta
		}
	case strings.HasPrefix(mimeType, "video/"):
		if videoMeta, err := extractVideoMetadata(ctx, filePath); err == nil {
			meta.Video = videoMeta
		}
	case mimeType == "application/pdf":
		if docMeta, err := extractPDFMetadata(ctx, filePath); err == nil {
			meta.Document = docMeta
		}
	}

	// Cache the result
	s.cache.Store(filePath, meta)

	return meta, nil
}

// Invalidate removes cached metadata for a file.
func (s *Service) Invalidate(filePath string) {
	s.cache.Delete(filePath)
}

func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "text/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".rar":  "application/vnd.rar",
		".7z":   "application/x-7z-compressed",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".heic": "image/heic",
		".heif": "image/heif",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".bmp":  "image/bmp",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",
		".wma":  "audio/x-ms-wma",
		".aiff": "audio/aiff",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".m4v":  "video/x-m4v",
		".go":   "text/x-go",
		".py":   "text/x-python",
		".rs":   "text/x-rust",
		".java": "text/x-java",
		".c":    "text/x-c",
		".cpp":  "text/x-c++",
		".h":    "text/x-c",
		".md":   "text/markdown",
		".yaml": "text/yaml",
		".yml":  "text/yaml",
		".toml": "text/toml",
		".sql":  "text/x-sql",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
