package releases

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("release not found")
	ErrExists         = errors.New("release with this tag already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrMissingTag     = errors.New("tag name is required")
	ErrAssetNotFound  = errors.New("asset not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrAlreadyPublished = errors.New("release is already published")
)

// Release represents a release
type Release struct {
	ID              string     `json:"id"`
	RepoID          string     `json:"repo_id"`
	TagName         string     `json:"tag_name"`
	TargetCommitish string     `json:"target_commitish"`
	Name            string     `json:"name"`
	Body            string     `json:"body"`
	IsDraft         bool       `json:"is_draft"`
	IsPrerelease    bool       `json:"is_prerelease"`
	AuthorID        string     `json:"author_id"`
	CreatedAt       time.Time  `json:"created_at"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`

	Assets []*Asset `json:"assets,omitempty"`
}

// Asset represents a release asset
type Asset struct {
	ID            string    `json:"id"`
	ReleaseID     string    `json:"release_id"`
	Name          string    `json:"name"`
	Label         string    `json:"label"`
	ContentType   string    `json:"content_type"`
	SizeBytes     int64     `json:"size_bytes"`
	DownloadCount int       `json:"download_count"`
	UploaderID    string    `json:"uploader_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateIn is the input for creating a release
type CreateIn struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	IsDraft         bool   `json:"is_draft"`
	IsPrerelease    bool   `json:"is_prerelease"`
}

// UpdateIn is the input for updating a release
type UpdateIn struct {
	TagName         *string `json:"tag_name,omitempty"`
	TargetCommitish *string `json:"target_commitish,omitempty"`
	Name            *string `json:"name,omitempty"`
	Body            *string `json:"body,omitempty"`
	IsDraft         *bool   `json:"is_draft,omitempty"`
	IsPrerelease    *bool   `json:"is_prerelease,omitempty"`
}

// UploadAssetIn is the input for uploading an asset
type UploadAssetIn struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

// ListOpts are options for listing releases
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the releases service interface
type API interface {
	// Release CRUD
	Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*Release, error)
	GetByID(ctx context.Context, id string) (*Release, error)
	GetByTag(ctx context.Context, repoID, tagName string) (*Release, error)
	GetLatest(ctx context.Context, repoID string) (*Release, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Release, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, opts *ListOpts) ([]*Release, error)
	Publish(ctx context.Context, id string) (*Release, error)

	// Assets
	UploadAsset(ctx context.Context, releaseID, uploaderID string, in *UploadAssetIn) (*Asset, error)
	GetAsset(ctx context.Context, id string) (*Asset, error)
	UpdateAsset(ctx context.Context, id string, name, label string) (*Asset, error)
	DeleteAsset(ctx context.Context, id string) error
	ListAssets(ctx context.Context, releaseID string) ([]*Asset, error)
	IncrementDownload(ctx context.Context, assetID string) error
}

// Store is the releases data store interface
type Store interface {
	// Releases
	Create(ctx context.Context, r *Release) error
	GetByID(ctx context.Context, id string) (*Release, error)
	GetByTag(ctx context.Context, repoID, tagName string) (*Release, error)
	GetLatest(ctx context.Context, repoID string) (*Release, error)
	Update(ctx context.Context, r *Release) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, limit, offset int) ([]*Release, error)

	// Assets
	CreateAsset(ctx context.Context, a *Asset) error
	GetAsset(ctx context.Context, id string) (*Asset, error)
	UpdateAsset(ctx context.Context, a *Asset) error
	DeleteAsset(ctx context.Context, id string) error
	ListAssets(ctx context.Context, releaseID string) ([]*Asset, error)
	IncrementDownload(ctx context.Context, assetID string) error
}
