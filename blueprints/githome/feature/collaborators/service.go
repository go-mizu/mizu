package collaborators

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the collaborators API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new collaborators service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// List returns collaborators for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Collaborator, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.List(ctx, r.ID, opts)
}

// IsCollaborator checks if a user is a collaborator
func (s *Service) IsCollaborator(ctx context.Context, owner, repo, username string) (bool, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	if r == nil {
		return false, repos.ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	collab, err := s.store.Get(ctx, r.ID, user.ID)
	return collab != nil, err
}

// GetPermission returns a user's permission level
func (s *Service) GetPermission(ctx context.Context, owner, repo, username string) (*PermissionLevel, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	// Check if owner
	if r.OwnerID == user.ID {
		return &PermissionLevel{
			Permission: "admin",
			RoleName:   "admin",
			User:       user.ToSimple(),
		}, nil
	}

	// Check if collaborator
	collab, err := s.store.Get(ctx, r.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if collab != nil {
		return &PermissionLevel{
			Permission: collab.RoleName,
			RoleName:   collab.RoleName,
			User:       user.ToSimple(),
		}, nil
	}

	// Default: no permission for private repos, read for public
	permission := "none"
	if !r.Private {
		permission = "read"
	}

	return &PermissionLevel{
		Permission: permission,
		RoleName:   permission,
		User:       user.ToSimple(),
	}, nil
}

// Add adds a collaborator (or invites if they don't have access)
func (s *Service) Add(ctx context.Context, owner, repo, username string, permission string) (*Invitation, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if permission == "" {
		permission = "push"
	}

	// Check if already a collaborator
	existing, err := s.store.Get(ctx, r.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update permission
		if err := s.store.UpdatePermission(ctx, r.ID, user.ID, permission); err != nil {
			return nil, err
		}
		return nil, nil // No invitation needed
	}

	// Create invitation
	now := time.Now()
	inv := &Invitation{
		Invitee:     user.ToSimple(),
		Permissions: permission,
		CreatedAt:   now,
		RepoID:      r.ID,
		InviteeID:   user.ID,
	}

	if err := s.store.CreateInvitation(ctx, inv); err != nil {
		return nil, err
	}

	s.populateInvitationURLs(inv, owner, repo)
	return inv, nil
}

// Remove removes a collaborator
func (s *Service) Remove(ctx context.Context, owner, repo, username string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return err
	}
	if user == nil {
		return users.ErrNotFound
	}

	return s.store.Remove(ctx, r.ID, user.ID)
}

// ListInvitations returns pending invitations for a repo
func (s *Service) ListInvitations(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Invitation, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	invitations, err := s.store.ListInvitationsForRepo(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, inv := range invitations {
		s.populateInvitationURLs(inv, owner, repo)
	}
	return invitations, nil
}

// UpdateInvitation updates an invitation's permission
func (s *Service) UpdateInvitation(ctx context.Context, owner, repo string, invitationID int64, permission string) (*Invitation, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	inv, err := s.store.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if inv == nil || inv.RepoID != r.ID {
		return nil, ErrInvitationNotFound
	}

	if err := s.store.UpdateInvitation(ctx, invitationID, permission); err != nil {
		return nil, err
	}

	inv.Permissions = permission
	s.populateInvitationURLs(inv, owner, repo)
	return inv, nil
}

// DeleteInvitation deletes an invitation
func (s *Service) DeleteInvitation(ctx context.Context, owner, repo string, invitationID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	inv, err := s.store.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv == nil || inv.RepoID != r.ID {
		return ErrInvitationNotFound
	}

	return s.store.DeleteInvitation(ctx, invitationID)
}

// ListUserInvitations returns invitations for the authenticated user
func (s *Service) ListUserInvitations(ctx context.Context, userID int64, opts *ListOpts) ([]*Invitation, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListInvitationsForUser(ctx, userID, opts)
}

// AcceptInvitation accepts an invitation
func (s *Service) AcceptInvitation(ctx context.Context, userID int64, invitationID int64) error {
	inv, err := s.store.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv == nil {
		return ErrInvitationNotFound
	}
	if inv.InviteeID != userID {
		return ErrInvitationNotFound
	}
	if inv.Expired {
		return ErrInvitationExpired
	}

	// Add as collaborator
	if err := s.store.Add(ctx, inv.RepoID, userID, inv.Permissions); err != nil {
		return err
	}

	// Accept (delete) invitation
	return s.store.AcceptInvitation(ctx, invitationID)
}

// DeclineInvitation declines an invitation
func (s *Service) DeclineInvitation(ctx context.Context, userID int64, invitationID int64) error {
	inv, err := s.store.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv == nil {
		return ErrInvitationNotFound
	}
	if inv.InviteeID != userID {
		return ErrInvitationNotFound
	}

	return s.store.DeleteInvitation(ctx, invitationID)
}

// populateInvitationURLs fills in the URL fields for an invitation
func (s *Service) populateInvitationURLs(inv *Invitation, owner, repo string) {
	inv.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("RepositoryInvitation:%d", inv.ID)))
	inv.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/invitations/%d", s.baseURL, owner, repo, inv.ID)
	inv.HTMLURL = fmt.Sprintf("%s/%s/%s/invitations", s.baseURL, owner, repo)
	if inv.Repository == nil {
		inv.Repository = &Repository{
			Name:     repo,
			FullName: fmt.Sprintf("%s/%s", owner, repo),
			URL:      fmt.Sprintf("%s/api/v3/repos/%s/%s", s.baseURL, owner, repo),
			HTMLURL:  fmt.Sprintf("%s/%s/%s", s.baseURL, owner, repo),
		}
	}
}
