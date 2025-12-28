package orgs

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("organization not found")
	ErrExists         = errors.New("organization already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrMissingName    = errors.New("organization name is required")
	ErrMemberExists   = errors.New("member already exists")
	ErrMemberNotFound = errors.New("member not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrLastOwner      = errors.New("cannot remove the last owner")
)

// Member roles
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Organization represents an organization
type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	AvatarURL   string    `json:"avatar_url"`
	Location    string    `json:"location"`
	Website     string    `json:"website"`
	Email       string    `json:"email"`
	IsVerified  bool      `json:"is_verified"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Member represents an organization member
type Member struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateIn is the input for creating an organization
type CreateIn struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Email       string `json:"email"`
}

// UpdateIn is the input for updating an organization
type UpdateIn struct {
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Location    *string `json:"location,omitempty"`
	Website     *string `json:"website,omitempty"`
	Email       *string `json:"email,omitempty"`
}

// ListOpts are options for listing
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the orgs service interface
type API interface {
	// Organization CRUD
	Create(ctx context.Context, creatorID string, in *CreateIn) (*Organization, error)
	GetByID(ctx context.Context, id string) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Organization, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, opts *ListOpts) ([]*Organization, error)

	// Members
	AddMember(ctx context.Context, orgID, userID string, role string) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	UpdateMemberRole(ctx context.Context, orgID, userID string, role string) error
	GetMember(ctx context.Context, orgID, userID string) (*Member, error)
	ListMembers(ctx context.Context, orgID string, opts *ListOpts) ([]*Member, error)
	ListUserOrgs(ctx context.Context, userID string) ([]*Organization, error)
	IsMember(ctx context.Context, orgID, userID string) (bool, error)
	IsOwner(ctx context.Context, orgID, userID string) (bool, error)
}

// Store is the orgs data store interface
type Store interface {
	// Organization
	Create(ctx context.Context, o *Organization) error
	GetByID(ctx context.Context, id string) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
	Update(ctx context.Context, o *Organization) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*Organization, error)

	// Members
	AddMember(ctx context.Context, m *Member) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	UpdateMember(ctx context.Context, m *Member) error
	GetMember(ctx context.Context, orgID, userID string) (*Member, error)
	ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*Member, error)
	ListByUser(ctx context.Context, userID string) ([]*Organization, error)
	CountOwners(ctx context.Context, orgID string) (int, error)
}
