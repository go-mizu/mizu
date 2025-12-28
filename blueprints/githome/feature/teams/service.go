package teams

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/slug"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the teams API
type Service struct {
	store Store
}

// NewService creates a new teams service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new team
func (s *Service) Create(ctx context.Context, orgID string, in *CreateIn) (*Team, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	teamSlug := slug.Make(in.Name)
	if teamSlug == "" {
		return nil, ErrInvalidInput
	}

	// Check if exists
	existing, _ := s.store.GetBySlug(ctx, orgID, teamSlug)
	if existing != nil {
		return nil, ErrExists
	}

	permission := in.Permission
	if permission == "" {
		permission = PermissionRead
	}

	now := time.Now()
	team := &Team{
		ID:          ulid.New(),
		OrgID:       orgID,
		Name:        strings.TrimSpace(in.Name),
		Slug:        teamSlug,
		Description: in.Description,
		Permission:  permission,
		ParentID:    in.ParentID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

// GetByID retrieves a team by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Team, error) {
	team, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrNotFound
	}
	return team, nil
}

// GetBySlug retrieves a team by organization ID and slug
func (s *Service) GetBySlug(ctx context.Context, orgID, teamSlug string) (*Team, error) {
	team, err := s.store.GetBySlug(ctx, orgID, teamSlug)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrNotFound
	}
	return team, nil
}

// Update updates a team
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Team, error) {
	team, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		newSlug := slug.Make(name)
		// Check if new slug conflicts
		if newSlug != team.Slug {
			existing, _ := s.store.GetBySlug(ctx, team.OrgID, newSlug)
			if existing != nil {
				return nil, ErrExists
			}
		}
		team.Name = name
		team.Slug = newSlug
	}
	if in.Description != nil {
		team.Description = *in.Description
	}
	if in.Permission != nil {
		team.Permission = *in.Permission
	}
	if in.ParentID != nil {
		team.ParentID = *in.ParentID
	}

	team.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

// Delete deletes a team
func (s *Service) Delete(ctx context.Context, id string) error {
	team, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists teams in an organization
func (s *Service) List(ctx context.Context, orgID string, opts *ListOpts) ([]*Team, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.List(ctx, orgID, limit, offset)
}

// ListChildren lists child teams of a parent team
func (s *Service) ListChildren(ctx context.Context, teamID string) ([]*Team, error) {
	return s.store.ListChildren(ctx, teamID)
}

// AddMember adds a member to a team
func (s *Service) AddMember(ctx context.Context, teamID, userID string, role string) error {
	// Validate role
	if role != RoleMaintainer && role != RoleMember {
		return ErrInvalidInput
	}

	// Check if already a member
	existing, err := s.store.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrMemberExists
	}

	member := &TeamMember{
		TeamID:    teamID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
	}

	return s.store.AddMember(ctx, member)
}

// RemoveMember removes a member from a team
func (s *Service) RemoveMember(ctx context.Context, teamID, userID string) error {
	member, err := s.store.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrMemberNotFound
	}
	return s.store.RemoveMember(ctx, teamID, userID)
}

// UpdateMemberRole updates a member's role in a team
func (s *Service) UpdateMemberRole(ctx context.Context, teamID, userID string, role string) error {
	// Validate role
	if role != RoleMaintainer && role != RoleMember {
		return ErrInvalidInput
	}

	member, err := s.store.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrMemberNotFound
	}

	member.Role = role
	return s.store.UpdateMember(ctx, member)
}

// GetMember gets a team member
func (s *Service) GetMember(ctx context.Context, teamID, userID string) (*TeamMember, error) {
	member, err := s.store.GetMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrMemberNotFound
	}
	return member, nil
}

// ListMembers lists members of a team
func (s *Service) ListMembers(ctx context.Context, teamID string, opts *ListOpts) ([]*TeamMember, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListMembers(ctx, teamID, limit, offset)
}

// ListUserTeams lists teams a user belongs to in an organization
func (s *Service) ListUserTeams(ctx context.Context, orgID, userID string) ([]*Team, error) {
	return s.store.ListByUser(ctx, orgID, userID)
}

// AddRepo adds a repository to a team
func (s *Service) AddRepo(ctx context.Context, teamID, repoID string, permission string) error {
	// Validate permission
	if !s.validPermission(permission) {
		return ErrInvalidInput
	}

	// Check if already added
	existing, err := s.store.GetRepo(ctx, teamID, repoID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrRepoExists
	}

	teamRepo := &TeamRepo{
		TeamID:     teamID,
		RepoID:     repoID,
		Permission: permission,
		CreatedAt:  time.Now(),
	}

	return s.store.AddRepo(ctx, teamRepo)
}

// RemoveRepo removes a repository from a team
func (s *Service) RemoveRepo(ctx context.Context, teamID, repoID string) error {
	existing, err := s.store.GetRepo(ctx, teamID, repoID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRepoNotFound
	}
	return s.store.RemoveRepo(ctx, teamID, repoID)
}

// UpdateRepoPermission updates a repository's permission in a team
func (s *Service) UpdateRepoPermission(ctx context.Context, teamID, repoID string, permission string) error {
	// Validate permission
	if !s.validPermission(permission) {
		return ErrInvalidInput
	}

	existing, err := s.store.GetRepo(ctx, teamID, repoID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRepoNotFound
	}

	existing.Permission = permission
	return s.store.UpdateRepo(ctx, existing)
}

// GetRepoAccess gets a team's access to a repository
func (s *Service) GetRepoAccess(ctx context.Context, teamID, repoID string) (*TeamRepo, error) {
	teamRepo, err := s.store.GetRepo(ctx, teamID, repoID)
	if err != nil {
		return nil, err
	}
	if teamRepo == nil {
		return nil, ErrRepoNotFound
	}
	return teamRepo, nil
}

// ListRepos lists repositories a team has access to
func (s *Service) ListRepos(ctx context.Context, teamID string, opts *ListOpts) ([]*TeamRepo, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListRepos(ctx, teamID, limit, offset)
}

func (s *Service) getPageParams(opts *ListOpts) (int, int) {
	limit := 30
	offset := 0
	if opts != nil {
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
	}
	return limit, offset
}

func (s *Service) validPermission(perm string) bool {
	return perm == PermissionRead || perm == PermissionTriage ||
		perm == PermissionWrite || perm == PermissionMaintain || perm == PermissionAdmin
}
