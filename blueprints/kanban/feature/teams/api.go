// Package teams provides team management functionality.
package teams

import (
	"context"
	"time"
)

// Team represents a team within a workspace.
type Team struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
}

// Member represents a team membership.
type Member struct {
	TeamID   string    `json:"team_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"` // lead, member
	JoinedAt time.Time `json:"joined_at"`
}

// Role constants
const (
	RoleLead   = "lead"
	RoleMember = "member"
)

// CreateIn contains input for creating a team.
type CreateIn struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// UpdateIn contains input for updating a team.
type UpdateIn struct {
	Key  *string `json:"key,omitempty"`
	Name *string `json:"name,omitempty"`
}

// API defines the teams service contract.
type API interface {
	Create(ctx context.Context, workspaceID string, in *CreateIn) (*Team, error)
	GetByID(ctx context.Context, id string) (*Team, error)
	GetByKey(ctx context.Context, workspaceID, key string) (*Team, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Team, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Team, error)
	Delete(ctx context.Context, id string) error
	AddMember(ctx context.Context, teamID, userID, role string) error
	GetMember(ctx context.Context, teamID, userID string) (*Member, error)
	ListMembers(ctx context.Context, teamID string) ([]*Member, error)
	UpdateMemberRole(ctx context.Context, teamID, userID, role string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
}

// Store defines the data access contract for teams.
type Store interface {
	Create(ctx context.Context, t *Team) error
	GetByID(ctx context.Context, id string) (*Team, error)
	GetByKey(ctx context.Context, workspaceID, key string) (*Team, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Team, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	AddMember(ctx context.Context, m *Member) error
	GetMember(ctx context.Context, teamID, userID string) (*Member, error)
	ListMembers(ctx context.Context, teamID string) ([]*Member, error)
	UpdateMemberRole(ctx context.Context, teamID, userID, role string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
}
