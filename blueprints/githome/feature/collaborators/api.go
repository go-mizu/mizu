package collaborators

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

var (
	ErrNotFound          = errors.New("collaborator not found")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrAlreadyCollaborator = errors.New("user is already a collaborator")
	ErrInvitationExpired = errors.New("invitation has expired")
)

// Collaborator represents a repository collaborator
type Collaborator struct {
	*users.SimpleUser
	Permissions *Permissions `json:"permissions,omitempty"`
	RoleName    string       `json:"role_name"`
}

// Permissions represents collaborator permissions
type Permissions struct {
	Pull     bool `json:"pull"`
	Triage   bool `json:"triage"`
	Push     bool `json:"push"`
	Maintain bool `json:"maintain"`
	Admin    bool `json:"admin"`
}

// PermissionLevel represents a permission check response
type PermissionLevel struct {
	Permission string `json:"permission"` // admin, write, read, none
	RoleName   string `json:"role_name"`
	User       *users.SimpleUser `json:"user"`
}

// Invitation represents a repository invitation
type Invitation struct {
	ID          int64             `json:"id"`
	NodeID      string            `json:"node_id"`
	Repository  *Repository       `json:"repository"`
	Invitee     *users.SimpleUser `json:"invitee"`
	Inviter     *users.SimpleUser `json:"inviter"`
	Permissions string            `json:"permissions"` // read, triage, write, maintain, admin
	CreatedAt   time.Time         `json:"created_at"`
	Expired     bool              `json:"expired"`
	URL         string            `json:"url"`
	HTMLURL     string            `json:"html_url"`
	// Internal
	RepoID    int64 `json:"-"`
	InviteeID int64 `json:"-"`
	InviterID int64 `json:"-"`
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
}

// ListOpts contains options for listing collaborators
type ListOpts struct {
	Page        int    `json:"page,omitempty"`
	PerPage     int    `json:"per_page,omitempty"`
	Affiliation string `json:"affiliation,omitempty"` // outside, direct, all
	Permission  string `json:"permission,omitempty"`  // pull, triage, push, maintain, admin
}

// API defines the collaborators service interface
type API interface {
	// List returns collaborators for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Collaborator, error)

	// IsCollaborator checks if a user is a collaborator
	IsCollaborator(ctx context.Context, owner, repo, username string) (bool, error)

	// GetPermission returns a user's permission level
	GetPermission(ctx context.Context, owner, repo, username string) (*PermissionLevel, error)

	// Add adds a collaborator (or invites if they don't have access)
	Add(ctx context.Context, owner, repo, username string, permission string) (*Invitation, error)

	// Remove removes a collaborator
	Remove(ctx context.Context, owner, repo, username string) error

	// ListInvitations returns pending invitations for a repo
	ListInvitations(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Invitation, error)

	// UpdateInvitation updates an invitation's permission
	UpdateInvitation(ctx context.Context, owner, repo string, invitationID int64, permission string) (*Invitation, error)

	// DeleteInvitation deletes an invitation
	DeleteInvitation(ctx context.Context, owner, repo string, invitationID int64) error

	// ListUserInvitations returns invitations for the authenticated user
	ListUserInvitations(ctx context.Context, userID int64, opts *ListOpts) ([]*Invitation, error)

	// AcceptInvitation accepts an invitation
	AcceptInvitation(ctx context.Context, userID int64, invitationID int64) error

	// DeclineInvitation declines an invitation
	DeclineInvitation(ctx context.Context, userID int64, invitationID int64) error
}

// Store defines the data access interface for collaborators
type Store interface {
	// Direct collaborators
	Add(ctx context.Context, repoID, userID int64, permission string) error
	Remove(ctx context.Context, repoID, userID int64) error
	Get(ctx context.Context, repoID, userID int64) (*Collaborator, error)
	List(ctx context.Context, repoID int64, opts *ListOpts) ([]*Collaborator, error)
	UpdatePermission(ctx context.Context, repoID, userID int64, permission string) error

	// Invitations
	CreateInvitation(ctx context.Context, inv *Invitation) error
	GetInvitationByID(ctx context.Context, id int64) (*Invitation, error)
	UpdateInvitation(ctx context.Context, id int64, permission string) error
	DeleteInvitation(ctx context.Context, id int64) error
	ListInvitationsForRepo(ctx context.Context, repoID int64, opts *ListOpts) ([]*Invitation, error)
	ListInvitationsForUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Invitation, error)
	AcceptInvitation(ctx context.Context, id int64) error
}

// PermissionToLevel converts a permission string to numeric level
func PermissionToLevel(permission string) int {
	switch permission {
	case "admin":
		return 4
	case "maintain":
		return 3
	case "write", "push":
		return 2
	case "triage":
		return 1
	case "read", "pull":
		return 0
	default:
		return -1
	}
}

// PermissionToPermissions converts a permission string to Permissions struct
func PermissionToPermissions(permission string) *Permissions {
	p := &Permissions{}
	level := PermissionToLevel(permission)
	if level >= 0 {
		p.Pull = true
	}
	if level >= 1 {
		p.Triage = true
	}
	if level >= 2 {
		p.Push = true
	}
	if level >= 3 {
		p.Maintain = true
	}
	if level >= 4 {
		p.Admin = true
	}
	return p
}
