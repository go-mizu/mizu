package teams

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("team not found")
	ErrExists         = errors.New("team already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrMissingName    = errors.New("team name is required")
	ErrMemberExists   = errors.New("member already exists")
	ErrMemberNotFound = errors.New("member not found")
	ErrRepoExists     = errors.New("repository already added")
	ErrRepoNotFound   = errors.New("repository not found in team")
	ErrAccessDenied   = errors.New("access denied")
)

// Permission levels
const (
	PermissionRead     = "read"
	PermissionTriage   = "triage"
	PermissionWrite    = "write"
	PermissionMaintain = "maintain"
	PermissionAdmin    = "admin"
)

// Member roles
const (
	RoleMaintainer = "maintainer"
	RoleMember     = "member"
)

// Team represents a team within an organization
type Team struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Permission  string    `json:"permission"`
	ParentID    string    `json:"parent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeamMember represents a team member
// Uses composite PK (team_id, user_id) - no ID field
type TeamMember struct {
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// TeamRepo represents a team's access to a repository
// Uses composite PK (team_id, repo_id) - no ID field
type TeamRepo struct {
	TeamID     string    `json:"team_id"`
	RepoID     string    `json:"repo_id"`
	Permission string    `json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateIn is the input for creating a team
type CreateIn struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Permission  string `json:"permission"`
	ParentID    string `json:"parent_id,omitempty"`
}

// UpdateIn is the input for updating a team
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Permission  *string `json:"permission,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
}

// ListOpts are options for listing
type ListOpts struct {
	Limit  int
	Offset int
}

// API is the teams service interface
type API interface {
	// Team CRUD
	Create(ctx context.Context, orgID string, in *CreateIn) (*Team, error)
	GetByID(ctx context.Context, id string) (*Team, error)
	GetBySlug(ctx context.Context, orgID, slug string) (*Team, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Team, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, orgID string, opts *ListOpts) ([]*Team, error)
	ListChildren(ctx context.Context, teamID string) ([]*Team, error)

	// Members
	AddMember(ctx context.Context, teamID, userID string, role string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
	UpdateMemberRole(ctx context.Context, teamID, userID string, role string) error
	GetMember(ctx context.Context, teamID, userID string) (*TeamMember, error)
	ListMembers(ctx context.Context, teamID string, opts *ListOpts) ([]*TeamMember, error)
	ListUserTeams(ctx context.Context, orgID, userID string) ([]*Team, error)

	// Repos
	AddRepo(ctx context.Context, teamID, repoID string, permission string) error
	RemoveRepo(ctx context.Context, teamID, repoID string) error
	UpdateRepoPermission(ctx context.Context, teamID, repoID string, permission string) error
	GetRepoAccess(ctx context.Context, teamID, repoID string) (*TeamRepo, error)
	ListRepos(ctx context.Context, teamID string, opts *ListOpts) ([]*TeamRepo, error)
}

// Store is the teams data store interface
type Store interface {
	// Team
	Create(ctx context.Context, t *Team) error
	GetByID(ctx context.Context, id string) (*Team, error)
	GetBySlug(ctx context.Context, orgID, slug string) (*Team, error)
	Update(ctx context.Context, t *Team) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, orgID string, limit, offset int) ([]*Team, error)
	ListChildren(ctx context.Context, parentID string) ([]*Team, error)

	// Members
	AddMember(ctx context.Context, m *TeamMember) error
	RemoveMember(ctx context.Context, teamID, userID string) error
	UpdateMember(ctx context.Context, m *TeamMember) error
	GetMember(ctx context.Context, teamID, userID string) (*TeamMember, error)
	ListMembers(ctx context.Context, teamID string, limit, offset int) ([]*TeamMember, error)
	ListByUser(ctx context.Context, orgID, userID string) ([]*Team, error)

	// Repos
	AddRepo(ctx context.Context, tr *TeamRepo) error
	RemoveRepo(ctx context.Context, teamID, repoID string) error
	UpdateRepo(ctx context.Context, tr *TeamRepo) error
	GetRepo(ctx context.Context, teamID, repoID string) (*TeamRepo, error)
	ListRepos(ctx context.Context, teamID string, limit, offset int) ([]*TeamRepo, error)
}
