package workspaces

import (
	"context"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the workspaces API.
type Service struct {
	store Store
}

// NewService creates a new workspaces service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new workspace.
func (s *Service) Create(ctx context.Context, ownerID string, in CreateIn) (*Workspace, error) {
	// Check for existing slug
	existing, _ := s.store.GetBySlug(ctx, in.Slug)
	if existing != nil {
		return nil, ErrSlugTaken
	}

	ws := &Workspace{
		ID:      ulid.New(),
		Name:    in.Name,
		Slug:    in.Slug,
		Icon:    in.Icon,
		Plan:    "free",
		OwnerID: ownerID,
	}

	if err := s.store.Create(ctx, ws); err != nil {
		return nil, err
	}

	// Add owner as member
	s.store.AddMember(ctx, &Member{
		ID:          ulid.New(),
		WorkspaceID: ws.ID,
		UserID:      ownerID,
		Role:        RoleOwner,
	})

	return ws, nil
}

// GetByID retrieves a workspace by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Workspace, error) {
	return s.store.GetByID(ctx, id)
}

// GetBySlug retrieves a workspace by slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	return s.store.GetBySlug(ctx, slug)
}

// Update updates a workspace.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Workspace, error) {
	ws, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		ws.Name = *in.Name
	}
	if in.Slug != nil {
		// Check for existing slug
		existing, _ := s.store.GetBySlug(ctx, *in.Slug)
		if existing != nil && existing.ID != id {
			return nil, ErrSlugTaken
		}
		ws.Slug = *in.Slug
	}
	if in.Icon != nil {
		ws.Icon = *in.Icon
	}

	if err := s.store.Update(ctx, ws); err != nil {
		return nil, err
	}

	return ws, nil
}

// Delete deletes a workspace.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// AddMember adds a member to a workspace.
func (s *Service) AddMember(ctx context.Context, workspaceID, userID, role string) error {
	if !isValidRole(role) {
		return ErrInvalidRole
	}

	return s.store.AddMember(ctx, &Member{
		ID:          ulid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	})
}

// RemoveMember removes a member from a workspace.
func (s *Service) RemoveMember(ctx context.Context, workspaceID, userID string) error {
	return s.store.RemoveMember(ctx, workspaceID, userID)
}

// UpdateMemberRole updates a member's role.
func (s *Service) UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error {
	if !isValidRole(role) {
		return ErrInvalidRole
	}
	return s.store.UpdateMemberRole(ctx, workspaceID, userID, role)
}

// GetMember retrieves a specific member.
func (s *Service) GetMember(ctx context.Context, workspaceID, userID string) (*Member, error) {
	return s.store.GetMember(ctx, workspaceID, userID)
}

// ListMembers lists all members of a workspace.
func (s *Service) ListMembers(ctx context.Context, workspaceID string) ([]*Member, error) {
	return s.store.ListMembers(ctx, workspaceID)
}

// ListByUser lists all workspaces a user belongs to.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Workspace, error) {
	return s.store.ListByUser(ctx, userID)
}

func isValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleMember, RoleReadonly:
		return true
	}
	return false
}
