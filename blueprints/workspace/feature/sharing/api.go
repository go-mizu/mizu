// Package sharing provides page sharing and permissions.
package sharing

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// ShareType represents the type of share.
type ShareType string

const (
	ShareUser   ShareType = "user"
	ShareLink   ShareType = "link"
	ShareDomain ShareType = "domain"
	SharePublic ShareType = "public"
)

// Permission represents the access level.
type Permission string

const (
	PermRead       Permission = "read"
	PermComment    Permission = "comment"
	PermEdit       Permission = "edit"
	PermFullAccess Permission = "full_access"
)

// Share represents a share configuration.
type Share struct {
	ID         string      `json:"id"`
	PageID     string      `json:"page_id"`
	Type       ShareType   `json:"type"`
	Permission Permission  `json:"permission"`
	UserID     string      `json:"user_id,omitempty"`
	Token      string      `json:"token,omitempty"`
	Password   string      `json:"-"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
	Domain     string      `json:"domain,omitempty"`
	CreatedBy  string      `json:"created_by"`
	CreatedAt  time.Time   `json:"created_at"`

	// Enriched
	User *users.User `json:"user,omitempty"`
}

// LinkOpts contains options for creating a share link.
type LinkOpts struct {
	Permission Permission
	Password   string
	ExpiresAt  *time.Time
}

// API defines the sharing service contract.
type API interface {
	// User shares
	ShareWithUser(ctx context.Context, pageID, userID string, perm Permission, createdBy string) (*Share, error)
	UpdateUserPermission(ctx context.Context, id string, perm Permission) error
	RemoveUserShare(ctx context.Context, id string) error

	// Link shares
	CreateShareLink(ctx context.Context, pageID string, opts LinkOpts, createdBy string) (*Share, error)
	UpdateShareLink(ctx context.Context, id string, opts LinkOpts) error
	DeleteShareLink(ctx context.Context, id string) error
	GetByToken(ctx context.Context, token string) (*Share, error)

	// Public
	EnablePublic(ctx context.Context, pageID, createdBy string) (*Share, error)
	DisablePublic(ctx context.Context, pageID string) error

	// List
	ListByPage(ctx context.Context, pageID string) ([]*Share, error)

	// Check permissions
	CanAccess(ctx context.Context, userID, pageID string) (Permission, error)
}

// Store defines the data access contract for sharing.
type Store interface {
	Create(ctx context.Context, s *Share) error
	GetByID(ctx context.Context, id string) (*Share, error)
	GetByToken(ctx context.Context, token string) (*Share, error)
	GetByPageAndUser(ctx context.Context, pageID, userID string) (*Share, error)
	GetPublicByPage(ctx context.Context, pageID string) (*Share, error)
	Update(ctx context.Context, id string, perm Permission) error
	UpdateLink(ctx context.Context, id string, opts LinkOpts) error
	Delete(ctx context.Context, id string) error
	ListByPage(ctx context.Context, pageID string) ([]*Share, error)
}
