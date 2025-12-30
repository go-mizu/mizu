// Package media provides media library management.
package media

import (
	"context"
	"io"
	"time"
)

// Media represents a media item.
type Media struct {
	ID               string    `json:"id"`
	UploaderID       string    `json:"uploader_id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	StoragePath      string    `json:"storage_path"`
	StorageProvider  string    `json:"storage_provider"`
	URL              string    `json:"url"`
	AltText          string    `json:"alt_text,omitempty"`
	Caption          string    `json:"caption,omitempty"`
	Title            string    `json:"title,omitempty"`
	Description      string    `json:"description,omitempty"`
	Width            int       `json:"width,omitempty"`
	Height           int       `json:"height,omitempty"`
	Duration         int       `json:"duration,omitempty"`
	Meta             string    `json:"meta,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UploadIn contains input for uploading media.
type UploadIn struct {
	File             io.Reader
	Filename         string
	MimeType         string
	FileSize         int64
	AltText          string
	Caption          string
	Title            string
	Description      string
}

// UpdateIn contains input for updating media metadata.
type UpdateIn struct {
	AltText     *string `json:"alt_text,omitempty"`
	Caption     *string `json:"caption,omitempty"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Meta        *string `json:"meta,omitempty"`
}

// ListIn contains input for listing media.
type ListIn struct {
	UploaderID string
	MimeType   string
	Search     string
	Limit      int
	Offset     int
	OrderBy    string
	Order      string
}

// API defines the media service contract.
type API interface {
	Upload(ctx context.Context, uploaderID string, in *UploadIn) (*Media, error)
	GetByID(ctx context.Context, id string) (*Media, error)
	List(ctx context.Context, in *ListIn) ([]*Media, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Media, error)
	Delete(ctx context.Context, id string) error
	GetFile(ctx context.Context, id string) (io.ReadCloser, *Media, error)
}

// Store defines the data access contract for media.
type Store interface {
	Create(ctx context.Context, m *Media) error
	GetByID(ctx context.Context, id string) (*Media, error)
	GetByFilename(ctx context.Context, filename string) (*Media, error)
	List(ctx context.Context, in *ListIn) ([]*Media, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
}

// Storage defines the file storage contract.
type Storage interface {
	Save(ctx context.Context, filename string, reader io.Reader) (string, error)
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	URL(path string) string
}
