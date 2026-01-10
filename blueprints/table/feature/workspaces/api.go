// Package workspaces provides workspace management functionality.
package workspaces

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound    = errors.New("workspace not found")
	ErrSlugTaken   = errors.New("workspace slug already taken")
	ErrNotMember   = errors.New("not a member of this workspace")
	ErrNotOwner    = errors.New("not the owner of this workspace")
	ErrInvalidRole = errors.New("invalid role")
)

// Roles
const (
	RoleOwner    = "owner"
	RoleAdmin    = "admin"
	RoleMember   = "member"
	RoleReadonly = "readonly"
)

// Workspace represents a workspace/organization.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Icon      string    `json:"icon,omitempty"`
	Plan      string    `json:"plan"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Member represents a workspace membership.
type Member struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

// CreateIn contains input for creating a workspace.
type CreateIn struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Icon string `json:"icon,omitempty"`
}

// UpdateIn contains input for updating a workspace.
type UpdateIn struct {
	Name *string `json:"name,omitempty"`
	Slug *string `json:"slug,omitempty"`
	Icon *string `json:"icon,omitempty"`
}

// API defines the workspaces service interface.
type API interface {
	Create(ctx context.Context, ownerID string, in CreateIn) (*Workspace, error)
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Workspace, error)
	Delete(ctx context.Context, id string) error

	// Members
	AddMember(ctx context.Context, workspaceID, userID, role string) error
	RemoveMember(ctx context.Context, workspaceID, userID string) error
	UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error
	GetMember(ctx context.Context, workspaceID, userID string) (*Member, error)
	ListMembers(ctx context.Context, workspaceID string) ([]*Member, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
}

// Store defines the workspaces data access interface.
type Store interface {
	Create(ctx context.Context, ws *Workspace) error
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	Update(ctx context.Context, ws *Workspace) error
	Delete(ctx context.Context, id string) error

	// Members
	AddMember(ctx context.Context, member *Member) error
	RemoveMember(ctx context.Context, workspaceID, userID string) error
	UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error
	GetMember(ctx context.Context, workspaceID, userID string) (*Member, error)
	ListMembers(ctx context.Context, workspaceID string) ([]*Member, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
}
