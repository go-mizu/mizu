package teams

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound   = errors.New("team not found")
	ErrTeamExists = errors.New("team already exists")
	ErrNotMember  = errors.New("user is not a member")
)

// Team represents a GitHub team
type Team struct {
	ID              int64       `json:"id"`
	NodeID          string      `json:"node_id"`
	URL             string      `json:"url"`
	HTMLURL         string      `json:"html_url"`
	Name            string      `json:"name"`
	Slug            string      `json:"slug"`
	Description     string      `json:"description,omitempty"`
	Privacy         string      `json:"privacy"` // secret, closed
	Permission      string      `json:"permission"` // pull, push, admin
	MembersURL      string      `json:"members_url"`
	RepositoriesURL string      `json:"repositories_url"`
	Parent          *TeamSimple `json:"parent,omitempty"`
	MembersCount    int         `json:"members_count"`
	ReposCount      int         `json:"repos_count"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	Organization    *OrgSimple  `json:"organization,omitempty"`
	// Internal
	OrgID    int64  `json:"-"`
	ParentID *int64 `json:"-"`
}

// TeamSimple is a compact team representation
type TeamSimple struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Privacy     string `json:"privacy"`
	Permission  string `json:"permission"`
}

// OrgSimple is a compact org representation
type OrgSimple struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Login       string `json:"login"`
	URL         string `json:"url"`
	AvatarURL   string `json:"avatar_url"`
	Description string `json:"description,omitempty"`
}

// Membership represents a user's membership in a team
type Membership struct {
	URL   string `json:"url"`
	Role  string `json:"role"` // member, maintainer
	State string `json:"state"` // active, pending
}

// RepoPermission represents a team's permission on a repo
type RepoPermission struct {
	Permission string `json:"permission"` // pull, triage, push, maintain, admin
	RoleName   string `json:"role_name"`
}

// CreateIn represents input for creating a team
type CreateIn struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	Privacy        string   `json:"privacy,omitempty"` // secret, closed
	Permission     string   `json:"permission,omitempty"` // pull, push
	ParentTeamID   *int64   `json:"parent_team_id,omitempty"`
	Maintainers    []string `json:"maintainers,omitempty"`
	RepoNames      []string `json:"repo_names,omitempty"`
}

// UpdateIn represents input for updating a team
type UpdateIn struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Privacy      *string `json:"privacy,omitempty"`
	Permission   *string `json:"permission,omitempty"`
	ParentTeamID *int64  `json:"parent_team_id,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per_page,omitempty"`
	Role    string `json:"role,omitempty"` // member, maintainer, all
}

// API defines the teams service interface
type API interface {
	// List returns teams for an organization
	List(ctx context.Context, org string, opts *ListOpts) ([]*Team, error)

	// GetBySlug retrieves a team by slug
	GetBySlug(ctx context.Context, org, slug string) (*Team, error)

	// GetByID retrieves a team by ID
	GetByID(ctx context.Context, id int64) (*Team, error)

	// Create creates a new team
	Create(ctx context.Context, org string, in *CreateIn) (*Team, error)

	// Update updates a team
	Update(ctx context.Context, org, slug string, in *UpdateIn) (*Team, error)

	// Delete removes a team
	Delete(ctx context.Context, org, slug string) error

	// ListMembers returns members of a team
	ListMembers(ctx context.Context, org, slug string, opts *ListOpts) ([]*users.SimpleUser, error)

	// GetMembership retrieves a user's membership in a team
	GetMembership(ctx context.Context, org, slug, username string) (*Membership, error)

	// AddMembership adds or updates a user's membership in a team
	AddMembership(ctx context.Context, org, slug, username, role string) (*Membership, error)

	// RemoveMembership removes a user from a team
	RemoveMembership(ctx context.Context, org, slug, username string) error

	// ListRepos returns repositories for a team
	ListRepos(ctx context.Context, org, slug string, opts *ListOpts) ([]*Repository, error)

	// CheckRepoPermission checks a team's permission on a repo
	CheckRepoPermission(ctx context.Context, org, slug, owner, repo string) (*RepoPermission, error)

	// AddRepo adds a repo to a team
	AddRepo(ctx context.Context, org, slug, owner, repo, permission string) error

	// RemoveRepo removes a repo from a team
	RemoveRepo(ctx context.Context, org, slug, owner, repo string) error

	// ListChildren returns child teams
	ListChildren(ctx context.Context, org, slug string, opts *ListOpts) ([]*Team, error)
}

// Repository is a minimal repo reference
type Repository struct {
	ID          int64             `json:"id"`
	NodeID      string            `json:"node_id"`
	Name        string            `json:"name"`
	FullName    string            `json:"full_name"`
	Owner       *users.SimpleUser `json:"owner"`
	Private     bool              `json:"private"`
	HTMLURL     string            `json:"html_url"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url"`
	Permissions *RepoPermissions  `json:"permissions,omitempty"`
}

// RepoPermissions represents permissions on a repo
type RepoPermissions struct {
	Admin    bool `json:"admin"`
	Maintain bool `json:"maintain"`
	Push     bool `json:"push"`
	Triage   bool `json:"triage"`
	Pull     bool `json:"pull"`
}

// Store defines the data access interface for teams
type Store interface {
	Create(ctx context.Context, t *Team) error
	GetByID(ctx context.Context, id int64) (*Team, error)
	GetBySlug(ctx context.Context, orgID int64, slug string) (*Team, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, orgID int64, opts *ListOpts) ([]*Team, error)
	ListChildren(ctx context.Context, parentID int64, opts *ListOpts) ([]*Team, error)

	// Membership
	AddMember(ctx context.Context, teamID, userID int64, role string) error
	UpdateMemberRole(ctx context.Context, teamID, userID int64, role string) error
	RemoveMember(ctx context.Context, teamID, userID int64) error
	GetMember(ctx context.Context, teamID, userID int64) (*Membership, error)
	ListMembers(ctx context.Context, teamID int64, opts *ListOpts) ([]*users.SimpleUser, error)
	IsMember(ctx context.Context, teamID, userID int64) (bool, error)

	// Repositories
	AddRepo(ctx context.Context, teamID, repoID int64, permission string) error
	UpdateRepoPermission(ctx context.Context, teamID, repoID int64, permission string) error
	RemoveRepo(ctx context.Context, teamID, repoID int64) error
	GetRepoPermission(ctx context.Context, teamID, repoID int64) (*RepoPermission, error)
	ListRepos(ctx context.Context, teamID int64, opts *ListOpts) ([]*Repository, error)

	// Counter operations
	IncrementMembers(ctx context.Context, teamID int64, delta int) error
	IncrementRepos(ctx context.Context, teamID int64, delta int) error
}

// ToSimple converts a Team to TeamSimple
func (t *Team) ToSimple() *TeamSimple {
	return &TeamSimple{
		ID:          t.ID,
		NodeID:      t.NodeID,
		URL:         t.URL,
		HTMLURL:     t.HTMLURL,
		Name:        t.Name,
		Slug:        t.Slug,
		Description: t.Description,
		Privacy:     t.Privacy,
		Permission:  t.Permission,
	}
}
