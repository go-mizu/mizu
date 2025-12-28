package orgs

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound   = errors.New("organization not found")
	ErrOrgExists  = errors.New("organization already exists")
	ErrNotMember  = errors.New("user is not a member")
	ErrLastOwner  = errors.New("cannot remove last owner")
)

// Organization represents a GitHub organization
type Organization struct {
	ID                          int64     `json:"id"`
	NodeID                      string    `json:"node_id"`
	Login                       string    `json:"login"`
	URL                         string    `json:"url"`
	ReposURL                    string    `json:"repos_url"`
	EventsURL                   string    `json:"events_url"`
	HooksURL                    string    `json:"hooks_url"`
	IssuesURL                   string    `json:"issues_url"`
	MembersURL                  string    `json:"members_url"`
	PublicMembersURL            string    `json:"public_members_url"`
	AvatarURL                   string    `json:"avatar_url"`
	Description                 string    `json:"description,omitempty"`
	Name                        string    `json:"name,omitempty"`
	Company                     string    `json:"company,omitempty"`
	Blog                        string    `json:"blog,omitempty"`
	Location                    string    `json:"location,omitempty"`
	Email                       string    `json:"email,omitempty"`
	TwitterUsername             string    `json:"twitter_username,omitempty"`
	IsVerified                  bool      `json:"is_verified"`
	HasOrganizationProjects     bool      `json:"has_organization_projects"`
	HasRepositoryProjects       bool      `json:"has_repository_projects"`
	PublicRepos                 int       `json:"public_repos"`
	PublicGists                 int       `json:"public_gists"`
	Followers                   int       `json:"followers"`
	Following                   int       `json:"following"`
	HTMLURL                     string    `json:"html_url"`
	Type                        string    `json:"type"`
	TotalPrivateRepos           int       `json:"total_private_repos"`
	OwnedPrivateRepos           int       `json:"owned_private_repos"`
	DefaultRepositoryPermission string    `json:"default_repository_permission,omitempty"`
	MembersCanCreateRepositories bool     `json:"members_can_create_repositories"`
	MembersCanCreatePublicRepositories bool `json:"members_can_create_public_repositories"`
	MembersCanCreatePrivateRepositories bool `json:"members_can_create_private_repositories"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

// OrgSimple is a compact organization representation
type OrgSimple struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Login       string `json:"login"`
	URL         string `json:"url"`
	AvatarURL   string `json:"avatar_url"`
	Description string `json:"description,omitempty"`
}

// Membership represents a user's membership in an org
type Membership struct {
	URL             string           `json:"url"`
	State           string           `json:"state"` // active, pending
	Role            string           `json:"role"`  // admin, member
	OrganizationURL string           `json:"organization_url"`
	Organization    *OrgSimple       `json:"organization"`
	User            *users.SimpleUser `json:"user"`
}

// Member represents an org member with role info
type Member struct {
	*users.SimpleUser
	Role string `json:"role,omitempty"`
}

// CreateIn represents the input for creating an organization
type CreateIn struct {
	Login       string `json:"login"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Email       string `json:"email,omitempty"`
}

// UpdateIn represents the input for updating an organization
type UpdateIn struct {
	Name                        *string `json:"name,omitempty"`
	Description                 *string `json:"description,omitempty"`
	Email                       *string `json:"email,omitempty"`
	Company                     *string `json:"company,omitempty"`
	Location                    *string `json:"location,omitempty"`
	Blog                        *string `json:"blog,omitempty"`
	TwitterUsername             *string `json:"twitter_username,omitempty"`
	HasOrganizationProjects     *bool   `json:"has_organization_projects,omitempty"`
	HasRepositoryProjects       *bool   `json:"has_repository_projects,omitempty"`
	DefaultRepositoryPermission *string `json:"default_repository_permission,omitempty"`
	MembersCanCreateRepositories *bool  `json:"members_can_create_repositories,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
	Since   int64 `json:"since,omitempty"`
}

// ListMembersOpts contains options for listing members
type ListMembersOpts struct {
	ListOpts
	Filter string `json:"filter,omitempty"` // 2fa_disabled, all
	Role   string `json:"role,omitempty"`   // all, admin, member
}

// API defines the organizations service interface
type API interface {
	// Create creates a new organization
	Create(ctx context.Context, creatorID int64, in *CreateIn) (*Organization, error)

	// Get retrieves an organization by login
	Get(ctx context.Context, login string) (*Organization, error)

	// GetByID retrieves an organization by ID
	GetByID(ctx context.Context, id int64) (*Organization, error)

	// Update updates an organization
	Update(ctx context.Context, login string, in *UpdateIn) (*Organization, error)

	// Delete removes an organization
	Delete(ctx context.Context, login string) error

	// List returns all organizations with pagination
	List(ctx context.Context, opts *ListOpts) ([]*OrgSimple, error)

	// ListForUser returns organizations for a specific user
	ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*OrgSimple, error)

	// ListMembers returns members of an organization
	ListMembers(ctx context.Context, org string, opts *ListMembersOpts) ([]*users.SimpleUser, error)

	// IsMember checks if a user is a member of an organization
	IsMember(ctx context.Context, org, username string) (bool, error)

	// GetMembership retrieves a user's membership in an organization
	GetMembership(ctx context.Context, org, username string) (*Membership, error)

	// SetMembership sets a user's membership in an organization
	SetMembership(ctx context.Context, org, username, role string) (*Membership, error)

	// RemoveMember removes a user from an organization
	RemoveMember(ctx context.Context, org, username string) error

	// ListPublicMembers returns public members of an organization
	ListPublicMembers(ctx context.Context, org string, opts *ListOpts) ([]*users.SimpleUser, error)

	// IsPublicMember checks if a user is a public member
	IsPublicMember(ctx context.Context, org, username string) (bool, error)

	// PublicizeMembership makes membership public
	PublicizeMembership(ctx context.Context, org, username string) error

	// ConcealMembership hides membership
	ConcealMembership(ctx context.Context, org, username string) error
}

// Store defines the data access interface for organizations
type Store interface {
	Create(ctx context.Context, org *Organization) error
	GetByLogin(ctx context.Context, login string) (*Organization, error)
	GetByID(ctx context.Context, id int64) (*Organization, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, opts *ListOpts) ([]*OrgSimple, error)
	ListForUser(ctx context.Context, userID int64, opts *ListOpts) ([]*OrgSimple, error)

	// Membership operations
	AddMember(ctx context.Context, orgID, userID int64, role string, isPublic bool) error
	UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error
	RemoveMember(ctx context.Context, orgID, userID int64) error
	GetMember(ctx context.Context, orgID, userID int64) (*Member, error)
	ListMembers(ctx context.Context, orgID int64, opts *ListMembersOpts) ([]*users.SimpleUser, error)
	ListPublicMembers(ctx context.Context, orgID int64, opts *ListOpts) ([]*users.SimpleUser, error)
	IsMember(ctx context.Context, orgID, userID int64) (bool, error)
	IsPublicMember(ctx context.Context, orgID, userID int64) (bool, error)
	SetMemberPublicity(ctx context.Context, orgID, userID int64, isPublic bool) error
	CountOwners(ctx context.Context, orgID int64) (int, error)
}

// ToSimple converts an Organization to OrgSimple
func (o *Organization) ToSimple() *OrgSimple {
	return &OrgSimple{
		ID:          o.ID,
		NodeID:      o.NodeID,
		Login:       o.Login,
		URL:         o.URL,
		AvatarURL:   o.AvatarURL,
		Description: o.Description,
	}
}
