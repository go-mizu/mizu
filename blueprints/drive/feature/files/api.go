// Package files provides file management functionality.
package files

import (
	"context"
	"time"
)

// File represents a file.
type File struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ParentID    string    `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	MimeType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	StorageKey  string    `json:"-"`
	Checksum    string    `json:"checksum,omitempty"`
	Description string    `json:"description,omitempty"`
	IsStarred   bool      `json:"is_starred"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TrashedAt   time.Time `json:"trashed_at,omitempty"`
}

// FileVersion represents a file version.
type FileVersion struct {
	ID         string    `json:"id"`
	FileID     string    `json:"file_id"`
	Version    int       `json:"version"`
	Size       int64     `json:"size"`
	StorageKey string    `json:"-"`
	Checksum   string    `json:"checksum,omitempty"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateIn contains input for creating a file.
type CreateIn struct {
	Name        string `json:"name"`
	ParentID    string `json:"parent_id,omitempty"`
	MimeType    string `json:"mime_type"`
	Size        int64  `json:"size"`
	StorageKey  string `json:"-"`
	Checksum    string `json:"checksum,omitempty"`
	Description string `json:"description,omitempty"`
}

// UpdateIn contains input for updating a file.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// MoveIn contains input for moving a file.
type MoveIn struct {
	ParentID string `json:"parent_id"`
}

// CopyIn contains input for copying a file.
type CopyIn struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name,omitempty"`
}

// API defines the files service contract.
type API interface {
	Create(ctx context.Context, userID string, in *CreateIn) (*File, error)
	GetByID(ctx context.Context, id string) (*File, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*File, error)
	Move(ctx context.Context, id string, in *MoveIn) (*File, error)
	Copy(ctx context.Context, id, userID string, in *CopyIn) (*File, error)
	Delete(ctx context.Context, id string) error
	Trash(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	Star(ctx context.Context, id, userID string) error
	Unstar(ctx context.Context, id, userID string) error
	ListByUser(ctx context.Context, userID, parentID string) ([]*File, error)
	ListStarred(ctx context.Context, userID string) ([]*File, error)
	ListRecent(ctx context.Context, userID string, limit int) ([]*File, error)
	ListTrashed(ctx context.Context, userID string) ([]*File, error)
	Search(ctx context.Context, userID, query string) ([]*File, error)
	ListVersions(ctx context.Context, fileID string) ([]*FileVersion, error)
	GetVersion(ctx context.Context, fileID string, version int) (*FileVersion, error)
	RestoreVersion(ctx context.Context, fileID string, version int, userID string) (*File, error)
}
