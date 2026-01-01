// Package shares provides file and folder sharing functionality.
package shares

import (
	"context"
	"time"
)

// Share represents a share.
type Share struct {
	ID               string    `json:"id"`
	ResourceType     string    `json:"resource_type"`
	ResourceID       string    `json:"resource_id"`
	OwnerID          string    `json:"owner_id"`
	SharedWithID     string    `json:"shared_with_id,omitempty"`
	Permission       string    `json:"permission"`
	LinkToken        string    `json:"link_token,omitempty"`
	LinkPasswordHash string    `json:"-"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	DownloadLimit    int64     `json:"download_limit,omitempty"`
	DownloadCount    int       `json:"download_count"`
	PreventDownload  bool      `json:"prevent_download"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a share.
type CreateIn struct {
	Permission      string    `json:"permission"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	DownloadLimit   int64     `json:"download_limit,omitempty"`
	PreventDownload bool      `json:"prevent_download"`
}

// CreateLinkIn contains input for creating a link share.
type CreateLinkIn struct {
	Permission      string    `json:"permission"`
	Password        string    `json:"password,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	DownloadLimit   int64     `json:"download_limit,omitempty"`
	PreventDownload bool      `json:"prevent_download"`
}

// UpdateIn contains input for updating a share.
type UpdateIn struct {
	Permission      *string    `json:"permission,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	DownloadLimit   *int64     `json:"download_limit,omitempty"`
	PreventDownload *bool      `json:"prevent_download,omitempty"`
}

// API defines the shares service contract.
type API interface {
	Create(ctx context.Context, ownerID, resourceID, resourceType, sharedWithID, permission string) (*Share, error)
	CreateLink(ctx context.Context, ownerID, resourceID, resourceType string, in *CreateLinkIn) (*Share, error)
	GetByID(ctx context.Context, id string) (*Share, error)
	GetByToken(ctx context.Context, token string) (*Share, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Share, error)
	Delete(ctx context.Context, id string) error
	ListByOwner(ctx context.Context, ownerID string) ([]*Share, error)
	ListSharedWithMe(ctx context.Context, userID string) ([]*Share, error)
	ListForResource(ctx context.Context, resourceType, resourceID string) ([]*Share, error)
	CheckAccess(ctx context.Context, userID, resourceType, resourceID string) (*Share, error)
	IncrementDownload(ctx context.Context, id string) error
}
