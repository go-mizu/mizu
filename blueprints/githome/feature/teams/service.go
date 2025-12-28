package teams

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the teams API
type Service struct {
	store     Store
	orgStore  orgs.Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new teams service
func NewService(store Store, orgStore orgs.Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		orgStore:  orgStore,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// List returns teams for an organization
func (s *Service) List(ctx context.Context, org string, opts *ListOpts) ([]*Team, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
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

	teams, err := s.store.List(ctx, o.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, t := range teams {
		s.populateURLs(t, org)
	}
	return teams, nil
}

// GetBySlug retrieves a team by slug
func (s *Service) GetBySlug(ctx context.Context, org, slug string) (*Team, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(t, org)
	return t, nil
}

// GetByID retrieves a team by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Team, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

// Create creates a new team
func (s *Service) Create(ctx context.Context, org string, in *CreateIn) (*Team, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	slug := slugify(in.Name)

	// Check if team exists
	existing, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrTeamExists
	}

	privacy := in.Privacy
	if privacy == "" {
		privacy = "secret"
	}

	permission := in.Permission
	if permission == "" {
		permission = "pull"
	}

	now := time.Now()
	t := &Team{
		Name:        in.Name,
		Slug:        slug,
		Description: in.Description,
		Privacy:     privacy,
		Permission:  permission,
		ParentID:    in.ParentTeamID,
		OrgID:       o.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, t); err != nil {
		return nil, err
	}

	// Add maintainers
	for _, login := range in.Maintainers {
		user, err := s.userStore.GetByLogin(ctx, login)
		if err != nil || user == nil {
			continue
		}
		_ = s.store.AddMember(ctx, t.ID, user.ID, "maintainer")
		_ = s.store.IncrementMembers(ctx, t.ID, 1)
	}

	s.populateURLs(t, org)
	return t, nil
}

// Update updates a team
func (s *Service) Update(ctx context.Context, org, slug string, in *UpdateIn) (*Team, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, t.ID, in); err != nil {
		return nil, err
	}

	// Get updated slug if name changed
	newSlug := slug
	if in.Name != nil {
		newSlug = slugify(*in.Name)
	}
	return s.GetBySlug(ctx, org, newSlug)
}

// Delete removes a team
func (s *Service) Delete(ctx context.Context, org, slug string) error {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrNotFound
	}

	return s.store.Delete(ctx, t.ID)
}

// ListMembers returns members of a team
func (s *Service) ListMembers(ctx context.Context, org, slug string, opts *ListOpts) ([]*users.SimpleUser, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListMembers(ctx, t.ID, opts)
}

// GetMembership retrieves a user's membership in a team
func (s *Service) GetMembership(ctx context.Context, org, slug, username string) (*Membership, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	m, err := s.store.GetMember(ctx, t.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotMember
	}

	m.URL = fmt.Sprintf("%s/api/v3/orgs/%s/teams/%s/memberships/%s", s.baseURL, org, slug, username)
	return m, nil
}

// AddMembership adds or updates a user's membership in a team
func (s *Service) AddMembership(ctx context.Context, org, slug, username, role string) (*Membership, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if role == "" {
		role = "member"
	}

	// Check if already member
	isMember, err := s.store.IsMember(ctx, t.ID, user.ID)
	if err != nil {
		return nil, err
	}

	if isMember {
		if err := s.store.UpdateMemberRole(ctx, t.ID, user.ID, role); err != nil {
			return nil, err
		}
	} else {
		if err := s.store.AddMember(ctx, t.ID, user.ID, role); err != nil {
			return nil, err
		}
		_ = s.store.IncrementMembers(ctx, t.ID, 1)
	}

	return s.GetMembership(ctx, org, slug, username)
}

// RemoveMembership removes a user from a team
func (s *Service) RemoveMembership(ctx context.Context, org, slug, username string) error {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return err
	}
	if user == nil {
		return users.ErrNotFound
	}

	if err := s.store.RemoveMember(ctx, t.ID, user.ID); err != nil {
		return err
	}

	return s.store.IncrementMembers(ctx, t.ID, -1)
}

// ListRepos returns repositories for a team
func (s *Service) ListRepos(ctx context.Context, org, slug string, opts *ListOpts) ([]*Repository, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListRepos(ctx, t.ID, opts)
}

// CheckRepoPermission checks a team's permission on a repo
func (s *Service) CheckRepoPermission(ctx context.Context, org, slug, owner, repo string) (*RepoPermission, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	return s.store.GetRepoPermission(ctx, t.ID, r.ID)
}

// AddRepo adds a repo to a team
func (s *Service) AddRepo(ctx context.Context, org, slug, owner, repo, permission string) error {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrNotFound
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	if permission == "" {
		permission = "pull"
	}

	// Check if already has permission
	existing, _ := s.store.GetRepoPermission(ctx, t.ID, r.ID)
	if existing != nil {
		return s.store.UpdateRepoPermission(ctx, t.ID, r.ID, permission)
	}

	if err := s.store.AddRepo(ctx, t.ID, r.ID, permission); err != nil {
		return err
	}
	return s.store.IncrementRepos(ctx, t.ID, 1)
}

// RemoveRepo removes a repo from a team
func (s *Service) RemoveRepo(ctx context.Context, org, slug, owner, repo string) error {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return err
	}
	if o == nil {
		return orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrNotFound
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	if err := s.store.RemoveRepo(ctx, t.ID, r.ID); err != nil {
		return err
	}
	return s.store.IncrementRepos(ctx, t.ID, -1)
}

// ListChildren returns child teams
func (s *Service) ListChildren(ctx context.Context, org, slug string, opts *ListOpts) ([]*Team, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	t, err := s.store.GetBySlug(ctx, o.ID, slug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	children, err := s.store.ListChildren(ctx, t.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		s.populateURLs(child, org)
	}
	return children, nil
}

// populateURLs fills in the URL fields for a team
func (s *Service) populateURLs(t *Team, org string) {
	t.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Team:%d", t.ID)))
	t.URL = fmt.Sprintf("%s/api/v3/orgs/%s/teams/%s", s.baseURL, org, t.Slug)
	t.HTMLURL = fmt.Sprintf("%s/orgs/%s/teams/%s", s.baseURL, org, t.Slug)
	t.MembersURL = fmt.Sprintf("%s/api/v3/orgs/%s/teams/%s/members{/member}", s.baseURL, org, t.Slug)
	t.RepositoriesURL = fmt.Sprintf("%s/api/v3/orgs/%s/teams/%s/repos", s.baseURL, org, t.Slug)
}

// slugify converts a name to a URL-friendly slug
func slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
