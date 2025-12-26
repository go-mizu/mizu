// Package workspaces provides workspace management functionality.
package workspaces

import (
	"context"
	"time"
)

// Workspace represents a team workspace.
type Workspace struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Member represents a workspace member.
type Member struct {
	ID          string    `json:"id"`
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
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateIn contains input for updating a workspace.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// AddMemberIn contains input for adding a member.
type AddMemberIn struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// API defines the workspaces service contract.
type API interface {
	Create(ctx context.Context, userID string, in *CreateIn) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	ListByUser(ctx context.Context, userID string) ([]*Workspace, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Workspace, error)
	Delete(ctx context.Context, id string) error
	AddMember(ctx context.Context, workspaceID string, in *AddMemberIn) (*Member, error)
	GetMember(ctx context.Context, workspaceID, userID string) (*Member, error)
	ListMembers(ctx context.Context, workspaceID string) ([]*Member, error)
	RemoveMember(ctx context.Context, memberID string) error
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
	UpdateMemberRole(ctx context.Context, id, role string) error
	RemoveMember(ctx context.Context, id string) error
}
