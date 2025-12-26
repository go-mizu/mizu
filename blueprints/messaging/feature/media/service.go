package media

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

// Size limits by type
const (
	maxImageSize    = 20 * 1024 * 1024  // 20MB
	maxVideoSize    = 100 * 1024 * 1024 // 100MB
	maxAudioSize    = 50 * 1024 * 1024  // 50MB
	maxDocumentSize = 100 * 1024 * 1024 // 100MB
	maxVoiceSize    = 10 * 1024 * 1024  // 10MB
)

// Allowed content types mapped to media types
var allowedTypes = map[string]MediaType{
	// Images
	"image/jpeg":    TypeImage,
	"image/png":     TypeImage,
	"image/gif":     TypeImage,
	"image/webp":    TypeImage,
	"image/svg+xml": TypeImage,
	// Videos
	"video/mp4":       TypeVideo,
	"video/webm":      TypeVideo,
	"video/quicktime": TypeVideo,
	"video/x-msvideo": TypeVideo,
	// Audio
	"audio/mpeg":  TypeAudio,
	"audio/mp4":   TypeAudio,
	"audio/ogg":   TypeAudio,
	"audio/wav":   TypeAudio,
	"audio/x-wav": TypeAudio,
	"audio/webm":  TypeVoice,
	// Documents
	"application/pdf":                                                              TypeDocument,
	"application/msword":                                                           TypeDocument,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":      TypeDocument,
	"application/vnd.ms-excel":                                                     TypeDocument,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":            TypeDocument,
	"application/vnd.ms-powerpoint":                                                TypeDocument,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation":    TypeDocument,
	"text/plain":    TypeDocument,
	"application/zip": TypeDocument,
	"application/x-zip-compressed": TypeDocument,
	"application/json": TypeDocument,
	"text/csv":       TypeDocument,
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
	// Determine media type
	mediaType := in.Type
	if mediaType == "" {
		var ok bool
		mediaType, ok = allowedTypes[in.ContentType]
		if !ok {
			// Default to document for unknown types
			mediaType = TypeDocument
		}
	}

	// Check size limits based on type
	maxSize := s.getMaxSize(mediaType)
	if in.Size > maxSize {
		return nil, ErrFileTooLarge
	}

	// Generate unique ID and filename
	id := ulid.New()
	ext := filepath.Ext(in.Filename)
	if ext == "" {
		ext = s.getExtensionFromContentType(in.ContentType)
	}
	storageName := id + ext

	// Create storage path: userID/type/filename
	storagePath := filepath.Join(userID, string(mediaType), storageName)

	// Upload to file store
	url, err := s.fileStore.Save(ctx, storagePath, in.ContentType, in.Reader)
	if err != nil {
		return nil, ErrUploadFailed
	}

	m := &Media{
		ID:               id,
		UserID:           userID,
		Type:             mediaType,
		Filename:         storageName,
		OriginalFilename: in.Filename,
		ContentType:      in.ContentType,
		Size:             in.Size,
		URL:              url,
		IsViewOnce:       in.IsViewOnce,
		CreatedAt:        time.Now(),
	}

	if err := s.store.Insert(ctx, m); err != nil {
		// Try to clean up uploaded file
		s.fileStore.Delete(ctx, url)
		return nil, err
	}

	// Generate thumbnail for images and videos asynchronously
	go s.generateThumbnailAsync(m.ID)

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
		return ErrUnauthorized
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

	// Only generate thumbnails for images and videos
	if m.Type != TypeImage && m.Type != TypeVideo {
		return "", nil
	}

	// Get file path
	sourcePath := s.fileStore.GetPath(ctx, m.URL)
	if sourcePath == "" {
		return m.URL, nil
	}

	// Generate thumbnail
	thumbURL, err := s.fileStore.GenerateThumbnail(ctx, sourcePath, m.Type)
	if err != nil {
		// Fall back to original URL for images
		if m.Type == TypeImage {
			return m.URL, nil
		}
		return "", err
	}

	// Update record with thumbnail URL
	if thumbURL != "" {
		s.store.Update(ctx, id, map[string]any{"thumbnail_url": thumbURL})
	}

	return thumbURL, nil
}

// ExtractMetadata extracts metadata from media.
func (s *Service) ExtractMetadata(ctx context.Context, id string) error {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updates := make(map[string]any)

	// TODO: Implement actual metadata extraction using image/video libraries
	// For now, just mark as processed
	switch m.Type {
	case TypeImage:
		// Would extract width, height using image package
	case TypeVideo:
		// Would extract width, height, duration using ffprobe
	case TypeAudio, TypeVoice:
		// Would extract duration, waveform using ffprobe
	}

	if len(updates) > 0 {
		return s.store.Update(ctx, id, updates)
	}

	return nil
}

// ListByUser lists media uploaded by a user.
func (s *Service) ListByUser(ctx context.Context, userID string, opts ListOpts) ([]*Media, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.ListByUser(ctx, userID, string(opts.Type), limit, opts.Offset)
}

// ListByChat lists media in a chat.
func (s *Service) ListByChat(ctx context.Context, chatID string, opts ListOpts) ([]*Media, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.ListByChat(ctx, chatID, string(opts.Type), limit, opts.Offset)
}

// ListByMessage lists media attached to a message.
func (s *Service) ListByMessage(ctx context.Context, messageID string) ([]*Media, error) {
	return s.store.ListByMessage(ctx, messageID)
}

// AttachToMessage attaches media to a message.
func (s *Service) AttachToMessage(ctx context.Context, mediaID, messageID, userID string) error {
	m, err := s.store.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}

	// Only the owner can attach their media
	if m.UserID != userID {
		return ErrUnauthorized
	}

	// Can't re-attach already attached media
	if m.MessageID != "" {
		return ErrUnauthorized
	}

	return s.store.AttachToMessage(ctx, mediaID, messageID)
}

// ViewMedia views media (for view-once support).
func (s *Service) ViewMedia(ctx context.Context, id, userID string) (*Media, error) {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// For view-once media, check if already viewed
	if m.IsViewOnce && m.ViewCount > 0 && m.UserID != userID {
		return nil, ErrNotFound // Don't reveal that it was view-once
	}

	// Increment view count for view-once
	if m.IsViewOnce && m.UserID != userID {
		if err := s.store.IncrementViewCount(ctx, id); err != nil {
			return nil, err
		}
		m.ViewCount++
	}

	return m, nil
}

// GetFilePath returns the file path for serving.
func (s *Service) GetFilePath(ctx context.Context, id string) (string, error) {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return s.fileStore.GetPath(ctx, m.URL), nil
}

// Helper methods

func (s *Service) getMaxSize(t MediaType) int64 {
	switch t {
	case TypeImage:
		return maxImageSize
	case TypeVideo:
		return maxVideoSize
	case TypeAudio:
		return maxAudioSize
	case TypeVoice:
		return maxVoiceSize
	default:
		return maxDocumentSize
	}
}

func (s *Service) getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "audio/mpeg":
		return ".mp3"
	case "audio/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	default:
		// Try to extract from content type
		parts := strings.Split(contentType, "/")
		if len(parts) == 2 {
			return "." + parts[1]
		}
		return ""
	}
}

func (s *Service) generateThumbnailAsync(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s.GenerateThumbnail(ctx, id)
}
