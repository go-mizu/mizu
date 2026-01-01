// Package local provides local filesystem storage.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage implements local filesystem storage.
type Storage struct {
	basePath string
}

// New creates a new local storage.
func New(basePath string) (*Storage, error) {
	// Create base directories
	dirs := []string{
		filepath.Join(basePath, "files"),
		filepath.Join(basePath, "thumbnails"),
		filepath.Join(basePath, "temp", "uploads"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return &Storage{basePath: basePath}, nil
}

// Save saves a file.
func (s *Storage) Save(ctx context.Context, ownerID, fileID string, r io.Reader, size int64) (string, error) {
	dir := s.buildFilePath(ownerID, fileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "current")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(path)
		return "", err
	}

	return path, nil
}

// SaveVersion saves a file version.
func (s *Storage) SaveVersion(ctx context.Context, ownerID, fileID string, version int, r io.Reader, size int64) (string, error) {
	dir := filepath.Join(s.buildFilePath(ownerID, fileID), "versions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, fmt.Sprintf("%d", version))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(path)
		return "", err
	}

	return path, nil
}

// Open opens a file.
func (s *Storage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// Delete deletes a file.
func (s *Storage) Delete(ctx context.Context, path string) error {
	return os.Remove(path)
}

// DeleteAll deletes a file and all versions.
func (s *Storage) DeleteAll(ctx context.Context, ownerID, fileID string) error {
	dir := s.buildFilePath(ownerID, fileID)
	return os.RemoveAll(dir)
}

// Exists checks if a file exists.
func (s *Storage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Size returns file size.
func (s *Storage) Size(ctx context.Context, path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CreateChunkDir creates a chunk upload directory.
func (s *Storage) CreateChunkDir(ctx context.Context, uploadID string) (string, error) {
	dir := filepath.Join(s.basePath, "temp", "uploads", uploadID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// SaveChunk saves a chunk.
func (s *Storage) SaveChunk(ctx context.Context, uploadID string, index int, r io.Reader, size int64) error {
	path := filepath.Join(s.basePath, "temp", "uploads", uploadID, fmt.Sprintf("chunk_%d", index))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

// GetChunk opens a chunk.
func (s *Storage) GetChunk(ctx context.Context, uploadID string, index int) (io.ReadCloser, error) {
	path := filepath.Join(s.basePath, "temp", "uploads", uploadID, fmt.Sprintf("chunk_%d", index))
	return os.Open(path)
}

// AssembleChunks assembles chunks into final file.
func (s *Storage) AssembleChunks(ctx context.Context, uploadID string, totalChunks int, ownerID, fileID string) (string, error) {
	// Create destination
	dir := s.buildFilePath(ownerID, fileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(dir, "current")
	dest, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	// Copy chunks in order
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(s.basePath, "temp", "uploads", uploadID, fmt.Sprintf("chunk_%d", i))
		chunk, err := os.Open(chunkPath)
		if err != nil {
			os.Remove(destPath)
			return "", fmt.Errorf("open chunk %d: %w", i, err)
		}

		if _, err := io.Copy(dest, chunk); err != nil {
			chunk.Close()
			os.Remove(destPath)
			return "", fmt.Errorf("copy chunk %d: %w", i, err)
		}
		chunk.Close()
	}

	return destPath, nil
}

// CleanupChunks removes chunk directory.
func (s *Storage) CleanupChunks(ctx context.Context, uploadID string) error {
	dir := filepath.Join(s.basePath, "temp", "uploads", uploadID)
	return os.RemoveAll(dir)
}

// SaveThumbnail saves a thumbnail.
func (s *Storage) SaveThumbnail(ctx context.Context, fileID string, size int, format string, r io.Reader) error {
	dir := filepath.Join(s.basePath, "thumbnails", fileID[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, fmt.Sprintf("%s.%d.%s", fileID, size, format))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

// OpenThumbnail opens a thumbnail.
func (s *Storage) OpenThumbnail(ctx context.Context, fileID string, size int, format string) (io.ReadCloser, error) {
	path := filepath.Join(s.basePath, "thumbnails", fileID[:2], fmt.Sprintf("%s.%d.%s", fileID, size, format))
	return os.Open(path)
}

// DeleteThumbnails deletes all thumbnails for a file.
func (s *Storage) DeleteThumbnails(ctx context.Context, fileID string) error {
	dir := filepath.Join(s.basePath, "thumbnails", fileID[:2])
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	prefix := fileID + "."
	for _, entry := range entries {
		if len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}

func (s *Storage) buildFilePath(ownerID, fileID string) string {
	return filepath.Join(
		s.basePath,
		"files",
		ownerID[:2],
		ownerID,
		fileID[:2],
		fileID,
	)
}
