package releases

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound       = errors.New("release not found")
	ErrReleaseExists  = errors.New("release already exists")
	ErrAssetNotFound  = errors.New("asset not found")
)

// Release represents a GitHub release
type Release struct {
	ID              int64          `json:"id"`
	NodeID          string         `json:"node_id"`
	URL             string         `json:"url"`
	HTMLURL         string         `json:"html_url"`
	AssetsURL       string         `json:"assets_url"`
	UploadURL       string         `json:"upload_url"`
	TarballURL      string         `json:"tarball_url,omitempty"`
	ZipballURL      string         `json:"zipball_url,omitempty"`
	TagName         string         `json:"tag_name"`
	TargetCommitish string         `json:"target_commitish"`
	Name            string         `json:"name,omitempty"`
	Body            string         `json:"body,omitempty"`
	Draft           bool           `json:"draft"`
	Prerelease      bool           `json:"prerelease"`
	CreatedAt       time.Time      `json:"created_at"`
	PublishedAt     *time.Time     `json:"published_at"`
	Author          *users.SimpleUser `json:"author"`
	Assets          []*Asset       `json:"assets"`
	// Internal
	RepoID   int64 `json:"-"`
	AuthorID int64 `json:"-"`
}

// Asset represents a release asset
type Asset struct {
	ID                 int64             `json:"id"`
	NodeID             string            `json:"node_id"`
	URL                string            `json:"url"`
	Name               string            `json:"name"`
	Label              string            `json:"label,omitempty"`
	State              string            `json:"state"` // uploaded, open
	ContentType        string            `json:"content_type"`
	Size               int               `json:"size"`
	DownloadCount      int               `json:"download_count"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Uploader           *users.SimpleUser `json:"uploader"`
	BrowserDownloadURL string            `json:"browser_download_url"`
	// Internal
	ReleaseID  int64  `json:"-"`
	UploaderID int64  `json:"-"`
	StoragePath string `json:"-"` // Path to stored file
}

// ReleaseNotes represents generated release notes
type ReleaseNotes struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

// CreateIn represents input for creating a release
type CreateIn struct {
	TagName              string `json:"tag_name"`
	TargetCommitish      string `json:"target_commitish,omitempty"`
	Name                 string `json:"name,omitempty"`
	Body                 string `json:"body,omitempty"`
	Draft                bool   `json:"draft,omitempty"`
	Prerelease           bool   `json:"prerelease,omitempty"`
	GenerateReleaseNotes bool   `json:"generate_release_notes,omitempty"`
}

// UpdateIn represents input for updating a release
type UpdateIn struct {
	TagName         *string `json:"tag_name,omitempty"`
	TargetCommitish *string `json:"target_commitish,omitempty"`
	Name            *string `json:"name,omitempty"`
	Body            *string `json:"body,omitempty"`
	Draft           *bool   `json:"draft,omitempty"`
	Prerelease      *bool   `json:"prerelease,omitempty"`
}

// UpdateAssetIn represents input for updating an asset
type UpdateAssetIn struct {
	Name  *string `json:"name,omitempty"`
	Label *string `json:"label,omitempty"`
}

// GenerateNotesIn represents input for generating release notes
type GenerateNotesIn struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	PreviousTagName string `json:"previous_tag_name,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// API defines the releases service interface
type API interface {
	// List returns releases for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Release, error)

	// Get retrieves a release by ID
	Get(ctx context.Context, owner, repo string, releaseID int64) (*Release, error)

	// GetLatest retrieves the latest release
	GetLatest(ctx context.Context, owner, repo string) (*Release, error)

	// GetByTag retrieves a release by tag
	GetByTag(ctx context.Context, owner, repo, tag string) (*Release, error)

	// Create creates a new release
	Create(ctx context.Context, owner, repo string, authorID int64, in *CreateIn) (*Release, error)

	// Update updates a release
	Update(ctx context.Context, owner, repo string, releaseID int64, in *UpdateIn) (*Release, error)

	// Delete removes a release
	Delete(ctx context.Context, owner, repo string, releaseID int64) error

	// GenerateNotes generates release notes
	GenerateNotes(ctx context.Context, owner, repo string, in *GenerateNotesIn) (*ReleaseNotes, error)

	// ListAssets returns assets for a release
	ListAssets(ctx context.Context, owner, repo string, releaseID int64, opts *ListOpts) ([]*Asset, error)

	// GetAsset retrieves an asset by ID
	GetAsset(ctx context.Context, owner, repo string, assetID int64) (*Asset, error)

	// UploadAsset uploads a new asset
	UploadAsset(ctx context.Context, owner, repo string, releaseID int64, uploaderID int64, name, contentType string, file io.Reader) (*Asset, error)

	// UpdateAsset updates an asset
	UpdateAsset(ctx context.Context, owner, repo string, assetID int64, in *UpdateAssetIn) (*Asset, error)

	// DeleteAsset removes an asset
	DeleteAsset(ctx context.Context, owner, repo string, assetID int64) error

	// DownloadAsset returns the asset file content
	DownloadAsset(ctx context.Context, owner, repo string, assetID int64) (io.ReadCloser, string, error)
}

// Store defines the data access interface for releases
type Store interface {
	Create(ctx context.Context, r *Release) error
	GetByID(ctx context.Context, id int64) (*Release, error)
	GetByTag(ctx context.Context, repoID int64, tag string) (*Release, error)
	GetLatest(ctx context.Context, repoID int64) (*Release, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, repoID int64, opts *ListOpts) ([]*Release, error)

	// Assets
	CreateAsset(ctx context.Context, a *Asset) error
	GetAssetByID(ctx context.Context, id int64) (*Asset, error)
	UpdateAsset(ctx context.Context, id int64, in *UpdateAssetIn) error
	DeleteAsset(ctx context.Context, id int64) error
	ListAssets(ctx context.Context, releaseID int64, opts *ListOpts) ([]*Asset, error)
	IncrementDownloadCount(ctx context.Context, assetID int64) error
}
