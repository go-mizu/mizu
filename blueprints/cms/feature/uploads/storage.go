package uploads

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements local filesystem storage.
type LocalStorage struct {
	dir     string
	baseURL string
}

// NewLocalStorage creates a new local storage instance.
func NewLocalStorage(dir, baseURL string) *LocalStorage {
	return &LocalStorage{
		dir:     dir,
		baseURL: baseURL,
	}
}

// Store saves a file to local storage.
func (s *LocalStorage) Store(ctx context.Context, filename string, reader io.Reader) (string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(s.dir, filename)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return "", err
	}

	return filename, nil
}

// Delete removes a file from local storage.
func (s *LocalStorage) Delete(ctx context.Context, filename string) error {
	path := filepath.Join(s.dir, filename)
	return os.Remove(path)
}

// GetURL returns the URL for a file.
func (s *LocalStorage) GetURL(filename string) string {
	if s.baseURL == "" {
		return "/uploads/" + filename
	}
	return s.baseURL + "/" + filename
}

// Open opens a file for reading.
func (s *LocalStorage) Open(ctx context.Context, filename string) (io.ReadCloser, error) {
	path := filepath.Join(s.dir, filename)
	return os.Open(path)
}
