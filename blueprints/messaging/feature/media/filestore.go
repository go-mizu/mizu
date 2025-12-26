package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalFileStore implements FileStore for local filesystem storage.
type LocalFileStore struct {
	basePath string
	baseURL  string
}

// NewLocalFileStore creates a new LocalFileStore.
// basePath is the filesystem directory for storing files.
// baseURL is the URL prefix for accessing files (e.g., "/media").
func NewLocalFileStore(basePath, baseURL string) *LocalFileStore {
	return &LocalFileStore{
		basePath: basePath,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
	}
}

// EnsureDir creates the base directory if it doesn't exist.
func (s *LocalFileStore) EnsureDir() error {
	return os.MkdirAll(s.basePath, 0755)
}

// Save saves a file to the local filesystem.
func (s *LocalFileStore) Save(ctx context.Context, filename string, contentType string, reader io.Reader) (url string, err error) {
	// Create full path
	fullPath := filepath.Join(s.basePath, filename)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(fullPath) // Clean up on error
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return URL
	return s.GetURL(ctx, filename), nil
}

// Delete deletes a file from the local filesystem.
func (s *LocalFileStore) Delete(ctx context.Context, url string) error {
	path := s.GetPath(ctx, url)
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

// GetURL returns the URL for a file path.
func (s *LocalFileStore) GetURL(ctx context.Context, path string) string {
	// Normalize path separators for URL
	urlPath := strings.ReplaceAll(path, string(os.PathSeparator), "/")
	return s.baseURL + "/" + urlPath
}

// GetPath returns the filesystem path for a URL.
func (s *LocalFileStore) GetPath(ctx context.Context, url string) string {
	// Remove base URL prefix
	if !strings.HasPrefix(url, s.baseURL) {
		return ""
	}
	relativePath := strings.TrimPrefix(url, s.baseURL+"/")
	return filepath.Join(s.basePath, relativePath)
}

// GenerateThumbnail generates a thumbnail for an image or video.
// For now, this returns the original URL for images and empty for videos.
// TODO: Implement actual thumbnail generation using image library or ffmpeg.
func (s *LocalFileStore) GenerateThumbnail(ctx context.Context, sourcePath string, mediaType MediaType) (string, error) {
	switch mediaType {
	case TypeImage:
		// For images, we could resize but for now just return original
		// In production, use imaging library to create 300px wide thumbnail
		return "", nil

	case TypeVideo:
		// For videos, we would use ffmpeg to extract a frame
		// ffmpeg -i input.mp4 -ss 00:00:01 -frames:v 1 thumbnail.jpg
		return "", nil

	default:
		return "", nil
	}
}

// GetBasePath returns the base storage path.
func (s *LocalFileStore) GetBasePath() string {
	return s.basePath
}

// FileExists checks if a file exists.
func (s *LocalFileStore) FileExists(path string) bool {
	fullPath := filepath.Join(s.basePath, path)
	_, err := os.Stat(fullPath)
	return err == nil
}
