// Package shares provides sharing and permissions.
package shares

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("share not found")
	ErrNotOwner        = errors.New("not owner")
	ErrAlreadyShared   = errors.New("already shared with this user")
	ErrLinkNotFound    = errors.New("share link not found")
	ErrLinkExpired     = errors.New("share link expired")
	ErrLinkDisabled    = errors.New("share link disabled")
	ErrDownloadLimit   = errors.New("download limit reached")
	ErrInvalidPassword = errors.New("invalid password")
	ErrNoPermission    = errors.New("no permission")
)

// Permission levels.
const (
	PermissionViewer    = "viewer"
	PermissionCommenter = "commenter"
	PermissionEditor    = "editor"
	PermissionOwner     = "owner"
)

// Share represents a share with a user.
type Share struct {
	ID         string    `json:"id"`
	ItemID     string    `json:"item_id"`
	ItemType   string    `json:"item_type"` // "file" or "folder"
	OwnerID    string    `json:"owner_id"`
	SharedWith string    `json:"shared_with"`
	Permission string    `json:"permission"`
	Notify     bool      `json:"notify"`
	Message    string    `json:"message,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ShareLink represents a public share link.
type ShareLink struct {
	ID            string     `json:"id"`
	ItemID        string     `json:"item_id"`
	ItemType      string     `json:"item_type"`
	OwnerID       string     `json:"owner_id"`
	Token         string     `json:"token"`
	URL           string     `json:"url,omitempty"`
	Permission    string     `json:"permission"`
	PasswordHash  string     `json:"-"`
	HasPassword   bool       `json:"has_password"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	DownloadLimit *int       `json:"download_limit,omitempty"`
	DownloadCount int        `json:"download_count"`
	AllowDownload bool       `json:"allow_download"`
	Disabled      bool       `json:"disabled"`
	CreatedAt     time.Time  `json:"created_at"`
	AccessedAt    *time.Time `json:"accessed_at,omitempty"`
}

// CreateShareIn contains share creation input.
type CreateShareIn struct {
	ItemID     string `json:"item_id"`
	ItemType   string `json:"item_type"`
	SharedWith string `json:"shared_with"`
	Permission string `json:"permission"`
	Notify     bool   `json:"notify"`
	Message    string `json:"message,omitempty"`
}

// CreateLinkIn contains share link creation input.
type CreateLinkIn struct {
	ItemID        string     `json:"item_id"`
	ItemType      string     `json:"item_type"`
	Permission    string     `json:"permission"`
	Password      string     `json:"password,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	DownloadLimit *int       `json:"download_limit,omitempty"`
	AllowDownload bool       `json:"allow_download"`
}

// UpdateLinkIn contains share link update input.
type UpdateLinkIn struct {
	Permission    *string    `json:"permission,omitempty"`
	Password      *string    `json:"password,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	DownloadLimit *int       `json:"download_limit,omitempty"`
	AllowDownload *bool      `json:"allow_download,omitempty"`
	Disabled      *bool      `json:"disabled,omitempty"`
}

// EffectivePermission represents resolved permission.
type EffectivePermission struct {
	Permission string `json:"permission"`
	Source     string `json:"source"` // "owner", "share", "link", "inherited"
	ItemID     string `json:"item_id"`
	ItemType   string `json:"item_type"`
}

// SharedItem represents an item shared with user.
type SharedItem struct {
	Share     *Share `json:"share"`
	ItemID    string `json:"item_id"`
	ItemType  string `json:"item_type"`
	ItemName  string `json:"item_name"`
	OwnerName string `json:"owner_name"`
}

// API defines the shares service contract.
type API interface {
	// User shares
	Create(ctx context.Context, ownerID string, in *CreateShareIn) (*Share, error)
	GetByID(ctx context.Context, id string) (*Share, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*Share, error)
	ListSharedWithMe(ctx context.Context, accountID string) ([]*SharedItem, error)
	ListForItem(ctx context.Context, itemID, itemType string) ([]*Share, error)
	Update(ctx context.Context, id, ownerID string, permission string) (*Share, error)
	Delete(ctx context.Context, id, ownerID string) error

	// Share links
	CreateLink(ctx context.Context, ownerID string, in *CreateLinkIn) (*ShareLink, error)
	GetLinkByID(ctx context.Context, id string) (*ShareLink, error)
	GetLinkByToken(ctx context.Context, token string) (*ShareLink, error)
	ListLinksForItem(ctx context.Context, itemID, itemType string) ([]*ShareLink, error)
	UpdateLink(ctx context.Context, id, ownerID string, in *UpdateLinkIn) (*ShareLink, error)
	DeleteLink(ctx context.Context, id, ownerID string) error
	VerifyLinkPassword(ctx context.Context, token, password string) (bool, error)
	RecordLinkAccess(ctx context.Context, token string) error

	// Permissions
	GetEffectivePermission(ctx context.Context, accountID, itemID, itemType string) (*EffectivePermission, error)
	CanPerform(ctx context.Context, accountID, itemID, itemType, action string) (bool, error)
}

// Store defines the data access contract.
type Store interface {
	Create(ctx context.Context, s *Share) error
	GetByID(ctx context.Context, id string) (*Share, error)
	GetByItemAndUser(ctx context.Context, itemID, itemType, sharedWith string) (*Share, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*Share, error)
	ListBySharedWith(ctx context.Context, accountID string) ([]*Share, error)
	ListByItem(ctx context.Context, itemID, itemType string) ([]*Share, error)
	Update(ctx context.Context, id string, permission string) error
	Delete(ctx context.Context, id string) error
}

// LinkStore defines the link data access contract.
type LinkStore interface {
	Create(ctx context.Context, l *ShareLink) error
	GetByID(ctx context.Context, id string) (*ShareLink, error)
	GetByToken(ctx context.Context, token string) (*ShareLink, error)
	ListByItem(ctx context.Context, itemID, itemType string) ([]*ShareLink, error)
	Update(ctx context.Context, id string, in *UpdateLinkIn, passwordHash string) error
	UpdateAccess(ctx context.Context, id string) error
	IncrementDownloads(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}
