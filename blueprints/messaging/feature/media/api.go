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
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Type         MediaType `json:"type"`
	Filename     string    `json:"filename"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Duration     int       `json:"duration,omitempty"` // seconds for audio/video
	Waveform     string    `json:"waveform,omitempty"` // for voice messages
	CreatedAt    time.Time `json:"created_at"`
}

// UploadIn contains input for uploading media.
type UploadIn struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	Size        int64
	Type        MediaType
}

// API defines the media service contract.
type API interface {
	Upload(ctx context.Context, userID string, in *UploadIn) (*Media, error)
	GetByID(ctx context.Context, id string) (*Media, error)
	Delete(ctx context.Context, id, userID string) error
	GenerateThumbnail(ctx context.Context, id string) (string, error)
	ExtractMetadata(ctx context.Context, id string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, m *Media) error
	GetByID(ctx context.Context, id string) (*Media, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, updates map[string]any) error
}

// FileStore defines the file storage contract.
type FileStore interface {
	Save(ctx context.Context, filename string, contentType string, reader io.Reader) (url string, err error)
	Delete(ctx context.Context, url string) error
	GetURL(ctx context.Context, path string) string
}
