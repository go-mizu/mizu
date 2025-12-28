package orgs

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
)

// Service implements the orgs API
type Service struct {
	store     Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new orgs service
func NewService(store Store, userStore users.Store, baseURL string) *Service {
	return &Service{store: store, userStore: userStore, baseURL: baseURL}
}

// Create creates a new organization
func (s *Service) Create(ctx context.Context, creatorID int64, in *CreateIn) (*Organization, error) {
	// Check if org exists
	existing, err := s.store.GetByLogin(ctx, in.Login)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrOrgExists
	}

	// Also check if a user with this login exists
	existingUser, err := s.userStore.GetByLogin(ctx, in.Login)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrOrgExists
	}

	now := time.Now()
	org := &Organization{
		Login:                        in.Login,
		Name:                         in.Name,
		Description:                  in.Description,
		Email:                        in.Email,
		Type:                         "Organization",
		HasOrganizationProjects:      true,
		HasRepositoryProjects:        true,
		MembersCanCreateRepositories: true,
		MembersCanCreatePublicRepositories: true,
		MembersCanCreatePrivateRepositories: true,
		DefaultRepositoryPermission:  "read",
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}

	if err := s.store.Create(ctx, org); err != nil {
		return nil, err
	}

	// Add creator as owner
	if err := s.store.AddMember(ctx, org.ID, creatorID, "admin", true); err != nil {
		return nil, err
	}

	s.populateURLs(org)
	return org, nil
}

// Get retrieves an organization by login
func (s *Service) Get(ctx context.Context, login string) (*Organization, error) {
	org, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}
	s.populateURLs(org)
	return org, nil
}

// GetByID retrieves an organization by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Organization, error) {
	org, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}
	s.populateURLs(org)
	return org, nil
}

// Update updates an organization
func (s *Service) Update(ctx context.Context, login string, in *UpdateIn) (*Organization, error) {
	org, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, org.ID, in); err != nil {
		return nil, err
	}

	return s.Get(ctx, login)
}

// Delete removes an organization
func (s *Service) Delete(ctx context.Context, login string) error {
	org, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return err
	}
	if org == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, org.ID)
}

// List returns all organizations with pagination
func (s *Service) List(ctx context.Context, opts *ListOpts) ([]*OrgSimple, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	return s.store.List(ctx, opts)
}

// ListForUser returns organizations for a specific user
func (s *Service) ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*OrgSimple, error) {
	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	return s.store.ListForUser(ctx, user.ID, opts)
}

// ListMembers returns members of an organization
func (s *Service) ListMembers(ctx context.Context, org string, opts *ListMembersOpts) ([]*users.SimpleUser, error) {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListMembersOpts{ListOpts: ListOpts{PerPage: 30}}
	}
	return s.store.ListMembers(ctx, o.ID, opts)
}

// IsMember checks if a user is a member of an organization
func (s *Service) IsMember(ctx context.Context, org, username string) (bool, error) {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return false, err
	}
	if o == nil {
		return false, ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	return s.store.IsMember(ctx, o.ID, user.ID)
}

// GetMembership retrieves a user's membership in an organization
func (s *Service) GetMembership(ctx context.Context, org, username string) (*Membership, error) {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	member, err := s.store.GetMember(ctx, o.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	return &Membership{
		URL:             fmt.Sprintf("%s/api/v3/orgs/%s/memberships/%s", s.baseURL, org, username),
		State:           "active",
		Role:            member.Role,
		OrganizationURL: fmt.Sprintf("%s/api/v3/orgs/%s", s.baseURL, org),
		Organization:    o.ToSimple(),
		User:            member.SimpleUser,
	}, nil
}

// SetMembership sets a user's membership in an organization
func (s *Service) SetMembership(ctx context.Context, org, username, role string) (*Membership, error) {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	// Check if already a member
	isMember, err := s.store.IsMember(ctx, o.ID, user.ID)
	if err != nil {
		return nil, err
	}

	if isMember {
		// Update role
		if err := s.store.UpdateMemberRole(ctx, o.ID, user.ID, role); err != nil {
			return nil, err
		}
	} else {
		// Add as member
		if err := s.store.AddMember(ctx, o.ID, user.ID, role, false); err != nil {
			return nil, err
		}
	}

	return s.GetMembership(ctx, org, username)
}

// RemoveMember removes a user from an organization
func (s *Service) RemoveMember(ctx context.Context, org, username string) error {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return err
	}
	if user == nil {
		return users.ErrNotFound
	}

	// Check if this is the last owner
	member, err := s.store.GetMember(ctx, o.ID, user.ID)
	if err != nil {
		return err
	}
	if member == nil {
		return nil // Not a member, no-op
	}

	if member.Role == "admin" {
		count, err := s.store.CountOwners(ctx, o.ID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	return s.store.RemoveMember(ctx, o.ID, user.ID)
}

// ListPublicMembers returns public members of an organization
func (s *Service) ListPublicMembers(ctx context.Context, org string, opts *ListOpts) ([]*users.SimpleUser, error) {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	return s.store.ListPublicMembers(ctx, o.ID, opts)
}

// IsPublicMember checks if a user is a public member
func (s *Service) IsPublicMember(ctx context.Context, org, username string) (bool, error) {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return false, err
	}
	if o == nil {
		return false, ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	return s.store.IsPublicMember(ctx, o.ID, user.ID)
}

// PublicizeMembership makes membership public
func (s *Service) PublicizeMembership(ctx context.Context, org, username string) error {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return err
	}
	if user == nil {
		return users.ErrNotFound
	}

	return s.store.SetMemberPublicity(ctx, o.ID, user.ID, true)
}

// ConcealMembership hides membership
func (s *Service) ConcealMembership(ctx context.Context, org, username string) error {
	o, err := s.store.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return err
	}
	if user == nil {
		return users.ErrNotFound
	}

	return s.store.SetMemberPublicity(ctx, o.ID, user.ID, false)
}

// populateURLs fills in the URL fields for an organization
func (s *Service) populateURLs(o *Organization) {
	o.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Organization:%d", o.ID)))
	o.URL = fmt.Sprintf("%s/api/v3/orgs/%s", s.baseURL, o.Login)
	o.HTMLURL = fmt.Sprintf("%s/%s", s.baseURL, o.Login)
	o.ReposURL = fmt.Sprintf("%s/api/v3/orgs/%s/repos", s.baseURL, o.Login)
	o.EventsURL = fmt.Sprintf("%s/api/v3/orgs/%s/events", s.baseURL, o.Login)
	o.HooksURL = fmt.Sprintf("%s/api/v3/orgs/%s/hooks", s.baseURL, o.Login)
	o.IssuesURL = fmt.Sprintf("%s/api/v3/orgs/%s/issues", s.baseURL, o.Login)
	o.MembersURL = fmt.Sprintf("%s/api/v3/orgs/%s/members{/member}", s.baseURL, o.Login)
	o.PublicMembersURL = fmt.Sprintf("%s/api/v3/orgs/%s/public_members{/member}", s.baseURL, o.Login)
	if o.AvatarURL == "" {
		o.AvatarURL = fmt.Sprintf("%s/avatars/%s", s.baseURL, o.Login)
	}
}
