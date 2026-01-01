// Package files provides file management.
package files

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound       = errors.New("file not found")
	ErrNameTaken      = errors.New("file name already exists")
	ErrQuotaExceeded  = errors.New("storage quota exceeded")
	ErrNotOwner       = errors.New("not file owner")
	ErrFileLocked     = errors.New("file is locked")
	ErrInvalidUpload  = errors.New("invalid upload")
	ErrChunkMissing   = errors.New("chunk missing")
	ErrUploadExpired  = errors.New("upload session expired")
	ErrUploadNotFound = errors.New("upload not found")
)

// File represents a file.
type File struct {
	ID             string     `json:"id"`
	OwnerID        string     `json:"owner_id"`
	FolderID       string     `json:"folder_id,omitempty"`
	Name           string     `json:"name"`
	Path           string     `json:"path"`
	Size           int64      `json:"size"`
	MimeType       string     `json:"mime_type"`
	Extension      string     `json:"extension"`
	StoragePath    string     `json:"-"`
	ChecksumSHA256 string     `json:"checksum_sha256,omitempty"`
	HasThumbnail   bool       `json:"has_thumbnail"`
	ThumbnailPath  string     `json:"-"`
	Starred        bool       `json:"starred"`
	Trashed        bool       `json:"trashed"`
	TrashedAt      *time.Time `json:"trashed_at,omitempty"`
	Locked         bool       `json:"locked"`
	LockedBy       string     `json:"locked_by,omitempty"`
	LockedAt       *time.Time `json:"locked_at,omitempty"`
	LockExpiresAt  *time.Time `json:"lock_expires_at,omitempty"`
	VersionCount   int        `json:"version_count"`
	CurrentVersion int        `json:"current_version"`
	Description    string     `json:"description,omitempty"`
	Metadata       string     `json:"metadata,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	AccessedAt     time.Time  `json:"accessed_at"`
}

// FileVersion represents a file version.
type FileVersion struct {
	ID             string    `json:"id"`
	FileID         string    `json:"file_id"`
	VersionNumber  int       `json:"version_number"`
	Size           int64     `json:"size"`
	StoragePath    string    `json:"-"`
	ChecksumSHA256 string    `json:"checksum_sha256,omitempty"`
	UploadedBy     string    `json:"uploaded_by"`
	Comment        string    `json:"comment,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChunkedUpload represents a chunked upload session.
type ChunkedUpload struct {
	ID           string    `json:"upload_id"`
	AccountID    string    `json:"-"`
	FolderID     string    `json:"folder_id,omitempty"`
	Filename     string    `json:"filename"`
	TotalSize    int64     `json:"total_size"`
	ChunkSize    int       `json:"chunk_size"`
	TotalChunks  int       `json:"total_chunks"`
	MimeType     string    `json:"mime_type,omitempty"`
	Status       string    `json:"status"`
	TempPath     string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UploadChunk represents a single chunk.
type UploadChunk struct {
	UploadID    string    `json:"upload_id"`
	ChunkIndex  int       `json:"chunk_index"`
	Size        int       `json:"size"`
	Checksum    string    `json:"checksum,omitempty"`
	StoragePath string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

// UploadIn contains file upload input.
type UploadIn struct {
	Filename    string
	FolderID    string
	Description string
	MimeType    string
	Size        int64
	Reader      io.Reader
}

// CreateChunkedUploadIn contains chunked upload creation input.
type CreateChunkedUploadIn struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type,omitempty"`
	FolderID  string `json:"folder_id,omitempty"`
	ChunkSize int    `json:"chunk_size,omitempty"`
}

// UpdateIn contains file update input.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ListIn contains file listing input.
type ListIn struct {
	FolderID string
	Starred  *bool
	Trashed  bool
	MimeType string
	Limit    int
	Offset   int
	OrderBy  string
	Order    string
}

// UploadProgress represents upload progress.
type UploadProgress struct {
	UploadID       string `json:"upload_id"`
	Status         string `json:"status"`
	TotalChunks    int    `json:"total_chunks"`
	UploadedChunks []int  `json:"uploaded_chunks"`
	MissingChunks  []int  `json:"missing_chunks"`
	ProgressPct    float64 `json:"progress_percent"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// API defines the files service contract.
type API interface {
	Upload(ctx context.Context, ownerID string, in *UploadIn) (*File, error)
	GetByID(ctx context.Context, id string) (*File, error)
	List(ctx context.Context, ownerID string, in *ListIn) ([]*File, error)
	ListRecent(ctx context.Context, ownerID string, limit int) ([]*File, error)
	Update(ctx context.Context, id, ownerID string, in *UpdateIn) (*File, error)
	Move(ctx context.Context, id, ownerID, newFolderID string) (*File, error)
	Copy(ctx context.Context, id, ownerID, destFolderID string, newName string) (*File, error)
	Delete(ctx context.Context, id, ownerID string) error
	Star(ctx context.Context, id, ownerID string, starred bool) error
	Lock(ctx context.Context, id, ownerID string, duration time.Duration) error
	Unlock(ctx context.Context, id, ownerID string) error
	Open(ctx context.Context, id string) (io.ReadCloser, *File, error)
	UpdateAccessed(ctx context.Context, id string) error

	// Chunked uploads
	CreateChunkedUpload(ctx context.Context, ownerID string, in *CreateChunkedUploadIn) (*ChunkedUpload, error)
	UploadChunk(ctx context.Context, uploadID string, index int, r io.Reader, size int64, checksum string) error
	GetUploadProgress(ctx context.Context, uploadID string) (*UploadProgress, error)
	CompleteUpload(ctx context.Context, uploadID, ownerID string, checksum string) (*File, error)
	CancelUpload(ctx context.Context, uploadID, ownerID string) error

	// Versions
	ListVersions(ctx context.Context, fileID string) ([]*FileVersion, error)
	UploadVersion(ctx context.Context, fileID, ownerID string, r io.Reader, size int64, comment string) (*FileVersion, error)
	OpenVersion(ctx context.Context, fileID string, version int) (io.ReadCloser, *FileVersion, error)
	RestoreVersion(ctx context.Context, fileID, ownerID string, version int) (*File, error)
}

// Store defines the data access contract.
type Store interface {
	Create(ctx context.Context, f *File) error
	GetByID(ctx context.Context, id string) (*File, error)
	GetByOwnerAndFolderAndName(ctx context.Context, ownerID, folderID, name string) (*File, error)
	List(ctx context.Context, ownerID string, in *ListIn) ([]*File, error)
	ListRecent(ctx context.Context, ownerID string, limit int) ([]*File, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdateFolder(ctx context.Context, id, folderID, path string) error
	UpdateTrashed(ctx context.Context, id string, trashed bool) error
	UpdateStarred(ctx context.Context, id string, starred bool) error
	UpdateLock(ctx context.Context, id string, locked bool, lockedBy string, expiresAt *time.Time) error
	UpdateVersion(ctx context.Context, id string, versionCount, currentVersion int) error
	UpdateAccessed(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// VersionStore defines version data access.
type VersionStore interface {
	Create(ctx context.Context, v *FileVersion) error
	GetByFileAndVersion(ctx context.Context, fileID string, version int) (*FileVersion, error)
	ListByFile(ctx context.Context, fileID string) ([]*FileVersion, error)
	Delete(ctx context.Context, id string) error
}

// ChunkedUploadStore defines chunked upload data access.
type ChunkedUploadStore interface {
	Create(ctx context.Context, u *ChunkedUpload) error
	GetByID(ctx context.Context, id string) (*ChunkedUpload, error)
	UpdateStatus(ctx context.Context, id, status string) error
	Delete(ctx context.Context, id string) error
	CreateChunk(ctx context.Context, c *UploadChunk) error
	GetChunks(ctx context.Context, uploadID string) ([]*UploadChunk, error)
	DeleteChunks(ctx context.Context, uploadID string) error
}
