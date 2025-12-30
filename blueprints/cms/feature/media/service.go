package media

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound        = errors.New("media not found")
	ErrMissingFile     = errors.New("file is required")
	ErrMissingFilename = errors.New("filename is required")
	ErrInvalidMimeType = errors.New("invalid mime type")
)

// AllowedMimeTypes contains the allowed MIME types for upload.
var AllowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"application/pdf": true,
	"video/mp4":       true,
	"video/webm":      true,
	"audio/mpeg":      true,
	"audio/wav":       true,
}

// Service implements the media API.
type Service struct {
	store   Store
	storage Storage
}

// NewService creates a new media service.
func NewService(store Store, storage Storage) *Service {
	return &Service{
		store:   store,
		storage: storage,
	}
}

func (s *Service) Upload(ctx context.Context, uploaderID string, in *UploadIn) (*Media, error) {
	if in.File == nil {
		return nil, ErrMissingFile
	}
	if in.Filename == "" {
		return nil, ErrMissingFilename
	}

	// Validate mime type
	if !AllowedMimeTypes[in.MimeType] {
		return nil, ErrInvalidMimeType
	}

	// Generate unique filename
	ext := filepath.Ext(in.Filename)
	id := ulid.New()
	filename := id + ext

	// Save file to storage
	storagePath, err := s.storage.Save(ctx, filename, in.File)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	media := &Media{
		ID:               id,
		UploaderID:       uploaderID,
		Filename:         filename,
		OriginalFilename: in.Filename,
		MimeType:         in.MimeType,
		FileSize:         in.FileSize,
		StoragePath:      storagePath,
		StorageProvider:  "local",
		URL:              s.storage.URL(storagePath),
		AltText:          in.AltText,
		Caption:          in.Caption,
		Title:            in.Title,
		Description:      in.Description,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Try to extract image dimensions
	if strings.HasPrefix(in.MimeType, "image/") {
		// TODO: Extract width/height from image
	}

	if err := s.store.Create(ctx, media); err != nil {
		// Clean up file on error
		_ = s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	return media, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Media, error) {
	media, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if media == nil {
		return nil, ErrNotFound
	}
	return media, nil
}

func (s *Service) List(ctx context.Context, in *ListIn) ([]*Media, int, error) {
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.OrderBy == "" {
		in.OrderBy = "created_at"
	}
	if in.Order == "" {
		in.Order = "desc"
	}
	return s.store.List(ctx, in)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Media, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	media, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if media == nil {
		return ErrNotFound
	}

	// Delete from storage
	if err := s.storage.Delete(ctx, media.StoragePath); err != nil {
		// Log but continue with DB deletion
	}

	return s.store.Delete(ctx, id)
}

func (s *Service) GetFile(ctx context.Context, id string) (io.ReadCloser, *Media, error) {
	media, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if media == nil {
		return nil, nil, ErrNotFound
	}

	reader, err := s.storage.Get(ctx, media.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	return reader, media, nil
}
