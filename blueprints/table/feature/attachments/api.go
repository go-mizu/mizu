// Package attachments provides file attachment functionality.
package attachments

import (
	"context"
	"errors"
	"io"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("attachment not found")
)

// Attachment represents a file attachment.
type Attachment struct {
	ID           string    `json:"id"`
	RecordID     string    `json:"record_id"`
	FieldID      string    `json:"field_id"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// API defines the attachments service interface.
type API interface {
	Upload(ctx context.Context, recordID, fieldID string, file io.Reader, filename string, size int64, mimeType string) (*Attachment, error)
	GetByID(ctx context.Context, id string) (*Attachment, error)
	Delete(ctx context.Context, id string) error
	ListByRecord(ctx context.Context, recordID, fieldID string) ([]*Attachment, error)
	GetSignedURL(ctx context.Context, id string, ttl time.Duration) (string, error)
}

// Store defines the attachments data access interface.
type Store interface {
	Create(ctx context.Context, att *Attachment) error
	GetByID(ctx context.Context, id string) (*Attachment, error)
	Delete(ctx context.Context, id string) error
	ListByRecord(ctx context.Context, recordID, fieldID string) ([]*Attachment, error)
	DeleteByRecord(ctx context.Context, recordID string) error
}
