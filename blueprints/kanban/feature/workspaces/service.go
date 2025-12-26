package workspaces

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrSlugExists = errors.New("workspace slug already exists")
	ErrNotFound   = errors.New("workspace not found")
	ErrForbidden  = errors.New("forbidden")
)

// Service implements the workspaces API.
type Service struct {
	store Store
}

// NewService creates a new workspaces service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, userID string, in *CreateIn) (*Workspace, error) {
	// Check if slug exists
	existing, err := s.store.GetBySlug(ctx, in.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugExists
	}

	now := time.Now()
	workspace := &Workspace{
		ID:   ulid.New(),
		Slug: in.Slug,
		Name: in.Name,
	}

	if err := s.store.Create(ctx, workspace); err != nil {
		return nil, err
	}

	// Add creator as owner
	member := &Member{
		WorkspaceID: workspace.ID,
		UserID:      userID,
		Role:        RoleOwner,
		JoinedAt:    now,
	}

	if err := s.store.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	ws, err := s.store.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrNotFound
	}
	return ws, nil
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Workspace, error) {
	return s.store.ListByUser(ctx, userID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Workspace, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) AddMember(ctx context.Context, workspaceID, userID, role string) (*Member, error) {
	member := &Member{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		JoinedAt:    time.Now(),
	}

	if err := s.store.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

func (s *Service) GetMember(ctx context.Context, workspaceID, userID string) (*Member, error) {
	return s.store.GetMember(ctx, workspaceID, userID)
}

func (s *Service) ListMembers(ctx context.Context, workspaceID string) ([]*Member, error) {
	return s.store.ListMembers(ctx, workspaceID)
}

func (s *Service) RemoveMember(ctx context.Context, workspaceID, userID string) error {
	return s.store.RemoveMember(ctx, workspaceID, userID)
}
