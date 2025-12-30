package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local storage.
func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

func (s *LocalStorage) Save(ctx context.Context, filename string, reader io.Reader) (string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	fullPath := filepath.Join(s.basePath, filename)
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("write file: %w", err)
	}

	return filename, nil
}

func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	return os.Open(fullPath)
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	return os.Remove(fullPath)
}

func (s *LocalStorage) URL(path string) string {
	return s.baseURL + "/" + path
}
