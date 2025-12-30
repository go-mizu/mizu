// Package versions provides document versioning with draft/publish workflow.
package versions

import (
	"context"
	"time"
)

// Version represents a document version.
type Version struct {
	ID        string         `json:"id"`
	Parent    string         `json:"parent"`
	Version   int            `json:"version"`
	Snapshot  map[string]any `json:"snapshot"`
	Published bool           `json:"published"`
	Autosave  bool           `json:"autosave"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedBy string         `json:"updatedBy,omitempty"`
}

// GlobalVersion represents a global document version.
type GlobalVersion struct {
	ID         string         `json:"id"`
	GlobalSlug string         `json:"globalSlug"`
	Version    int            `json:"version"`
	Snapshot   map[string]any `json:"snapshot"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedBy  string         `json:"updatedBy,omitempty"`
}

// VersionOptions holds options for creating versions.
type VersionOptions struct {
	Published bool
	Autosave  bool
	UpdatedBy string
}

// ListOptions holds options for listing versions.
type ListOptions struct {
	Limit int
	Page  int
}

// VersionsResult holds the result of listing versions.
type VersionsResult struct {
	Docs          []*Version `json:"docs"`
	TotalDocs     int        `json:"totalDocs"`
	Limit         int        `json:"limit"`
	TotalPages    int        `json:"totalPages"`
	Page          int        `json:"page"`
	HasPrevPage   bool       `json:"hasPrevPage"`
	HasNextPage   bool       `json:"hasNextPage"`
	PrevPage      *int       `json:"prevPage"`
	NextPage      *int       `json:"nextPage"`
}

// GlobalVersionsResult holds the result of listing global versions.
type GlobalVersionsResult struct {
	Docs          []*GlobalVersion `json:"docs"`
	TotalDocs     int              `json:"totalDocs"`
	Limit         int              `json:"limit"`
	TotalPages    int              `json:"totalPages"`
	Page          int              `json:"page"`
	HasPrevPage   bool             `json:"hasPrevPage"`
	HasNextPage   bool             `json:"hasNextPage"`
	PrevPage      *int             `json:"prevPage"`
	NextPage      *int             `json:"nextPage"`
}

// VersionDiff represents differences between two versions.
type VersionDiff struct {
	Added   map[string]any     `json:"added"`
	Removed map[string]any     `json:"removed"`
	Changed map[string]DiffPair `json:"changed"`
}

// DiffPair holds before/after values for a changed field.
type DiffPair struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

// Service defines the versions service interface.
type Service interface {
	// Collection version operations
	CreateVersion(ctx context.Context, collection, parentID string, data map[string]any, opts *VersionOptions) (*Version, error)
	GetVersion(ctx context.Context, collection, versionID string) (*Version, error)
	ListVersions(ctx context.Context, collection, parentID string, opts *ListOptions) (*VersionsResult, error)
	RestoreVersion(ctx context.Context, collection, versionID string) (map[string]any, error)
	CompareVersions(ctx context.Context, collection, versionID1, versionID2 string) (*VersionDiff, error)

	// Draft operations
	SaveDraft(ctx context.Context, collection, parentID string, data map[string]any, userID string) (map[string]any, error)
	PublishDraft(ctx context.Context, collection, parentID string, userID string) (map[string]any, error)
	GetLatestDraft(ctx context.Context, collection, parentID string) (*Version, error)

	// Autosave operations
	Autosave(ctx context.Context, collection, parentID string, data map[string]any, userID string) error
	GetLatestAutosave(ctx context.Context, collection, parentID string) (*Version, error)

	// Global version operations
	CreateGlobalVersion(ctx context.Context, slug string, data map[string]any, userID string) (*GlobalVersion, error)
	GetGlobalVersion(ctx context.Context, versionID string) (*GlobalVersion, error)
	ListGlobalVersions(ctx context.Context, slug string, opts *ListOptions) (*GlobalVersionsResult, error)
	RestoreGlobalVersion(ctx context.Context, versionID string) (map[string]any, error)
}
