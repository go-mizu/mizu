package media

import (
	"context"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

const (
	maxFileSize = 100 * 1024 * 1024 // 100MB
)

var allowedTypes = map[string]MediaType{
	"image/jpeg":      TypeImage,
	"image/png":       TypeImage,
	"image/gif":       TypeImage,
	"image/webp":      TypeImage,
	"video/mp4":       TypeVideo,
	"video/webm":      TypeVideo,
	"video/quicktime": TypeVideo,
	"audio/mpeg":      TypeAudio,
	"audio/mp4":       TypeAudio,
	"audio/ogg":       TypeAudio,
	"audio/webm":      TypeVoice,
	"application/pdf": TypeDocument,
}

// Service implements the media API.
type Service struct {
	store     Store
	fileStore FileStore
}

// NewService creates a new media service.
func NewService(store Store, fileStore FileStore) *Service {
	return &Service{
		store:     store,
		fileStore: fileStore,
	}
}

// Upload uploads a media file.
func (s *Service) Upload(ctx context.Context, userID string, in *UploadIn) (*Media, error) {
	if in.Size > maxFileSize {
		return nil, ErrFileTooLarge
	}

	mediaType := in.Type
	if mediaType == "" {
		var ok bool
		mediaType, ok = allowedTypes[in.ContentType]
		if !ok {
			mediaType = TypeDocument
		}
	}

	// Generate unique filename
	id := ulid.New()
	ext := filepath.Ext(in.Filename)
	storagePath := userID + "/" + id + ext

	// Upload to file store
	url, err := s.fileStore.Save(ctx, storagePath, in.ContentType, in.Reader)
	if err != nil {
		return nil, ErrUploadFailed
	}

	m := &Media{
		ID:          id,
		UserID:      userID,
		Type:        mediaType,
		Filename:    in.Filename,
		ContentType: in.ContentType,
		Size:        in.Size,
		URL:         url,
		CreatedAt:   time.Now(),
	}

	if err := s.store.Insert(ctx, m); err != nil {
		// Try to clean up uploaded file
		s.fileStore.Delete(ctx, url)
		return nil, err
	}

	return m, nil
}

// GetByID retrieves media by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Media, error) {
	return s.store.GetByID(ctx, id)
}

// Delete deletes media.
func (s *Service) Delete(ctx context.Context, id, userID string) error {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if m.UserID != userID {
		return ErrNotFound // Don't reveal existence
	}

	// Delete from file store
	if err := s.fileStore.Delete(ctx, m.URL); err != nil {
		// Log but continue
	}

	// Delete thumbnail if exists
	if m.ThumbnailURL != "" {
		s.fileStore.Delete(ctx, m.ThumbnailURL)
	}

	return s.store.Delete(ctx, id)
}

// GenerateThumbnail generates a thumbnail for media.
func (s *Service) GenerateThumbnail(ctx context.Context, id string) (string, error) {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// TODO: Implement actual thumbnail generation
	// For now, return the original URL for images
	if m.Type == TypeImage {
		return m.URL, nil
	}

	return "", nil
}

// ExtractMetadata extracts metadata from media.
func (s *Service) ExtractMetadata(ctx context.Context, id string) error {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updates := make(map[string]any)

	// TODO: Implement actual metadata extraction
	// For now, just set some defaults
	switch m.Type {
	case TypeImage:
		// Would extract width, height
	case TypeVideo:
		// Would extract width, height, duration
	case TypeAudio, TypeVoice:
		// Would extract duration, waveform
	}

	if len(updates) > 0 {
		return s.store.Update(ctx, id, updates)
	}

	return nil
}
