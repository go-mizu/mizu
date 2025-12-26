// Package media provides media upload and management.
package media

import (
	"context"
	"errors"
	"io"
	"time"
)

// Errors
var (
	ErrNotFound      = errors.New("media not found")
	ErrFileTooLarge  = errors.New("file too large")
	ErrInvalidType   = errors.New("invalid file type")
	ErrUploadFailed  = errors.New("upload failed")
	ErrUnauthorized  = errors.New("unauthorized")
)

// MediaType represents the type of media.
type MediaType string

const (
	TypeImage    MediaType = "image"
	TypeVideo    MediaType = "video"
	TypeAudio    MediaType = "audio"
	TypeDocument MediaType = "document"
	TypeVoice    MediaType = "voice"
	TypeSticker  MediaType = "sticker"
)

// Media represents an uploaded media file.
type Media struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	MessageID        string     `json:"message_id,omitempty"`
	Type             MediaType  `json:"type"`
	Filename         string     `json:"filename"`
	OriginalFilename string     `json:"original_filename"`
	ContentType      string     `json:"content_type"`
	Size             int64      `json:"size"`
	URL              string     `json:"url"`
	ThumbnailURL     string     `json:"thumbnail_url,omitempty"`
	Width            int        `json:"width,omitempty"`
	Height           int        `json:"height,omitempty"`
	Duration         int        `json:"duration,omitempty"` // milliseconds for audio/video
	Waveform         string     `json:"waveform,omitempty"` // JSON array for voice messages
	Blurhash         string     `json:"blurhash,omitempty"` // for progressive loading
	IsViewOnce       bool       `json:"is_view_once,omitempty"`
	ViewCount        int        `json:"view_count,omitempty"`
	ViewedAt         *time.Time `json:"viewed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// UploadIn contains input for uploading media.
type UploadIn struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	Size        int64
	Type        MediaType
	IsViewOnce  bool
}

// ListOpts contains options for listing media.
type ListOpts struct {
	Type   MediaType
	Limit  int
	Offset int
}

// API defines the media service contract.
type API interface {
	Upload(ctx context.Context, userID string, in *UploadIn) (*Media, error)
	GetByID(ctx context.Context, id string) (*Media, error)
	Delete(ctx context.Context, id, userID string) error
	GenerateThumbnail(ctx context.Context, id string) (string, error)
	ExtractMetadata(ctx context.Context, id string) error

	// List operations
	ListByUser(ctx context.Context, userID string, opts ListOpts) ([]*Media, error)
	ListByChat(ctx context.Context, chatID string, opts ListOpts) ([]*Media, error)
	ListByMessage(ctx context.Context, messageID string) ([]*Media, error)

	// Attach media to a message
	AttachToMessage(ctx context.Context, mediaID, messageID, userID string) error

	// View once support
	ViewMedia(ctx context.Context, id, userID string) (*Media, error)

	// Get file path for serving
	GetFilePath(ctx context.Context, id string) (string, error)
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, m *Media) error
	GetByID(ctx context.Context, id string) (*Media, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, updates map[string]any) error
	ListByUser(ctx context.Context, userID string, mediaType string, limit, offset int) ([]*Media, error)
	ListByChat(ctx context.Context, chatID string, mediaType string, limit, offset int) ([]*Media, error)
	ListByMessage(ctx context.Context, messageID string) ([]*Media, error)
	AttachToMessage(ctx context.Context, mediaID, messageID string) error
	IncrementViewCount(ctx context.Context, id string) error
}

// FileStore defines the file storage contract.
type FileStore interface {
	Save(ctx context.Context, filename string, contentType string, reader io.Reader) (url string, err error)
	Delete(ctx context.Context, url string) error
	GetURL(ctx context.Context, path string) string
	GetPath(ctx context.Context, url string) string
	GenerateThumbnail(ctx context.Context, sourcePath string, mediaType MediaType) (string, error)
}
