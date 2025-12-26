// Package workspaces provides workspace management functionality.
package workspaces

import (
	"context"
	"time"
)

// Workspace represents a workspace (tenant boundary).
type Workspace struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Member represents a workspace member.
type Member struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"` // owner, admin, member, guest
	JoinedAt    time.Time `json:"joined_at"`
}

// Role constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleGuest  = "guest"
)

// CreateIn contains input for creating a workspace.
type CreateIn struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// UpdateIn contains input for updating a workspace.
type UpdateIn struct {
	Name *string `json:"name,omitempty"`
}

// API defines the workspaces service contract.
type API interface {
	Create(ctx context.Context, userID string, in *CreateIn) (*Workspace, error)
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Workspace, error)
	Delete(ctx context.Context, id string) error
	AddMember(ctx context.Context, workspaceID, userID, role string) error
	GetMember(ctx context.Context, workspaceID, userID string) (*Member, error)
	ListMembers(ctx context.Context, workspaceID string) ([]*Member, error)
	UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error
	RemoveMember(ctx context.Context, workspaceID, userID string) error
}

// Store defines the data access contract for workspaces.
type Store interface {
	Create(ctx context.Context, w *Workspace) error
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	AddMember(ctx context.Context, m *Member) error
	GetMember(ctx context.Context, workspaceID, userID string) (*Member, error)
	ListMembers(ctx context.Context, workspaceID string) ([]*Member, error)
	UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error
	RemoveMember(ctx context.Context, workspaceID, userID string) error
}
