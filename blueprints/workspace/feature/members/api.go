// Package members provides workspace membership management.
package members

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// Role represents a member's role in a workspace.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleGuest  Role = "guest"
)

// Member represents a workspace member.
type Member struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	UserID      string     `json:"user_id"`
	Role        Role       `json:"role"`
	JoinedAt    time.Time  `json:"joined_at"`
	InvitedBy   string     `json:"invited_by,omitempty"`
	User        *users.User `json:"user,omitempty"`
}

// Invite represents a pending invitation.
type Invite struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Email       string    `json:"email"`
	Role        Role      `json:"role"`
	Token       string    `json:"-"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// API defines the members service contract.
type API interface {
	// Members
	Add(ctx context.Context, workspaceID, userID string, role Role, inviterID string) (*Member, error)
	GetByID(ctx context.Context, id string) (*Member, error)
	GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID string) (*Member, error)
	List(ctx context.Context, workspaceID string) ([]*Member, error)
	UpdateRole(ctx context.Context, id string, role Role) error
	Remove(ctx context.Context, id string) error

	// Invites
	Invite(ctx context.Context, workspaceID, email string, role Role, inviterID string) (*Invite, error)
	GetInvite(ctx context.Context, token string) (*Invite, error)
	AcceptInvite(ctx context.Context, token string, userID string) (*Member, error)
	RevokeInvite(ctx context.Context, id string) error
	ListPendingInvites(ctx context.Context, workspaceID string) ([]*Invite, error)
}

// Store defines the data access contract for members.
type Store interface {
	Create(ctx context.Context, m *Member) error
	GetByID(ctx context.Context, id string) (*Member, error)
	GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID string) (*Member, error)
	List(ctx context.Context, workspaceID string) ([]*Member, error)
	UpdateRole(ctx context.Context, id string, role Role) error
	Delete(ctx context.Context, id string) error

	CreateInvite(ctx context.Context, inv *Invite) error
	GetInviteByToken(ctx context.Context, token string) (*Invite, error)
	DeleteInvite(ctx context.Context, id string) error
	ListPendingInvites(ctx context.Context, workspaceID string) ([]*Invite, error)
}
