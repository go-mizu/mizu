package orgs

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/slug"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the orgs API
type Service struct {
	store Store
}

// NewService creates a new orgs service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new organization
func (s *Service) Create(ctx context.Context, creatorID string, in *CreateIn) (*Organization, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	orgSlug := slug.Make(in.Name)
	if orgSlug == "" {
		return nil, ErrInvalidInput
	}

	// Check if exists
	existing, _ := s.store.GetBySlug(ctx, orgSlug)
	if existing != nil {
		return nil, ErrExists
	}

	now := time.Now()
	org := &Organization{
		ID:          ulid.New(),
		Name:        strings.TrimSpace(in.Name),
		Slug:        orgSlug,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Email:       in.Email,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, org); err != nil {
		return nil, err
	}

	// Add creator as owner
	member := &Member{
		ID:        ulid.New(),
		OrgID:     org.ID,
		UserID:    creatorID,
		Role:      RoleOwner,
		CreatedAt: now,
	}
	if err := s.store.AddMember(ctx, member); err != nil {
		// Cleanup org on failure
		s.store.Delete(ctx, org.ID)
		return nil, err
	}

	return org, nil
}

// GetByID retrieves an organization by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Organization, error) {
	org, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}
	return org, nil
}

// GetBySlug retrieves an organization by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	org, err := s.store.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}
	return org, nil
}

// Update updates an organization
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Organization, error) {
	org, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}

	if in.DisplayName != nil {
		org.DisplayName = *in.DisplayName
	}
	if in.Description != nil {
		org.Description = *in.Description
	}
	if in.AvatarURL != nil {
		org.AvatarURL = *in.AvatarURL
	}
	if in.Location != nil {
		org.Location = *in.Location
	}
	if in.Website != nil {
		org.Website = *in.Website
	}
	if in.Email != nil {
		org.Email = *in.Email
	}

	org.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

// Delete deletes an organization
func (s *Service) Delete(ctx context.Context, id string) error {
	org, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if org == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists organizations
func (s *Service) List(ctx context.Context, opts *ListOpts) ([]*Organization, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.List(ctx, limit, offset)
}

// AddMember adds a member to an organization
func (s *Service) AddMember(ctx context.Context, orgID, userID string, role string) error {
	// Validate role
	if role != RoleOwner && role != RoleAdmin && role != RoleMember {
		return ErrInvalidInput
	}

	// Check if already a member
	existing, err := s.store.GetMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrMemberExists
	}

	member := &Member{
		ID:        ulid.New(),
		OrgID:     orgID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
	}

	return s.store.AddMember(ctx, member)
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, orgID, userID string) error {
	member, err := s.store.GetMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrMemberNotFound
	}

	// Check if this is the last owner
	if member.Role == RoleOwner {
		count, err := s.store.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	return s.store.RemoveMember(ctx, orgID, userID)
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID string, role string) error {
	// Validate role
	if role != RoleOwner && role != RoleAdmin && role != RoleMember {
		return ErrInvalidInput
	}

	member, err := s.store.GetMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrMemberNotFound
	}

	// Check if demoting the last owner
	if member.Role == RoleOwner && role != RoleOwner {
		count, err := s.store.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	member.Role = role
	return s.store.UpdateMember(ctx, member)
}

// GetMember gets a member
func (s *Service) GetMember(ctx context.Context, orgID, userID string) (*Member, error) {
	member, err := s.store.GetMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrMemberNotFound
	}
	return member, nil
}

// ListMembers lists members of an organization
func (s *Service) ListMembers(ctx context.Context, orgID string, opts *ListOpts) ([]*Member, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListMembers(ctx, orgID, limit, offset)
}

// ListUserOrgs lists organizations a user belongs to
func (s *Service) ListUserOrgs(ctx context.Context, userID string) ([]*Organization, error) {
	return s.store.ListByUser(ctx, userID)
}

// IsMember checks if a user is a member of an organization
func (s *Service) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	member, err := s.store.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, err
	}
	return member != nil, nil
}

// IsOwner checks if a user is an owner of an organization
func (s *Service) IsOwner(ctx context.Context, orgID, userID string) (bool, error) {
	member, err := s.store.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, err
	}
	return member != nil && member.Role == RoleOwner, nil
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
