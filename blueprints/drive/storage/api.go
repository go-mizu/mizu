// Package storage provides file storage interfaces.
package storage

import (
	"context"
	"io"
)

// Storage defines the file storage contract.
type Storage interface {
	// Save saves a file and returns its storage path.
	Save(ctx context.Context, ownerID, fileID string, r io.Reader, size int64) (string, error)

	// SaveVersion saves a file version.
	SaveVersion(ctx context.Context, ownerID, fileID string, version int, r io.Reader, size int64) (string, error)

	// Open opens a file for reading.
	Open(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete deletes a file.
	Delete(ctx context.Context, path string) error

	// DeleteAll deletes a file and all its versions.
	DeleteAll(ctx context.Context, ownerID, fileID string) error

	// Exists checks if a file exists.
	Exists(ctx context.Context, path string) (bool, error)

	// Size returns the size of a file.
	Size(ctx context.Context, path string) (int64, error)

	// Chunked upload operations

	// CreateChunkDir creates a directory for chunk uploads.
	CreateChunkDir(ctx context.Context, uploadID string) (string, error)

	// SaveChunk saves a chunk.
	SaveChunk(ctx context.Context, uploadID string, index int, r io.Reader, size int64) error

	// GetChunk opens a chunk for reading.
	GetChunk(ctx context.Context, uploadID string, index int) (io.ReadCloser, error)

	// AssembleChunks assembles chunks into a single file.
	AssembleChunks(ctx context.Context, uploadID string, totalChunks int, ownerID, fileID string) (string, error)

	// CleanupChunks removes chunk directory.
	CleanupChunks(ctx context.Context, uploadID string) error

	// Thumbnail operations

	// SaveThumbnail saves a thumbnail.
	SaveThumbnail(ctx context.Context, fileID string, size int, format string, r io.Reader) error

	// OpenThumbnail opens a thumbnail.
	OpenThumbnail(ctx context.Context, fileID string, size int, format string) (io.ReadCloser, error)

	// DeleteThumbnails deletes all thumbnails for a file.
	DeleteThumbnails(ctx context.Context, fileID string) error
}
