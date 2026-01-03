package workspaces

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrInvalidName    = errors.New("invalid workspace name")
	ErrInvalidSlug    = errors.New("invalid workspace slug")
	ErrSlugExists     = errors.New("workspace slug already exists")
	ErrNotFound       = errors.New("workspace not found")
	ErrNotOwner       = errors.New("only the owner can perform this action")
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// Service implements the workspaces API.
type Service struct {
	store Store
}

// NewService creates a new workspaces service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new workspace.
func (s *Service) Create(ctx context.Context, ownerID string, in *CreateIn) (*Workspace, error) {
	if in.Name == "" {
		return nil, ErrInvalidName
	}
	if !slugRegex.MatchString(in.Slug) {
		return nil, ErrInvalidSlug
	}

	// Check if slug exists
	existing, _ := s.store.GetBySlug(ctx, in.Slug)
	if existing != nil {
		return nil, ErrSlugExists
	}

	now := time.Now()
	ws := &Workspace{
		ID:      ulid.New(),
		Name:    in.Name,
		Slug:    in.Slug,
		Icon:    in.Icon,
		Plan:    PlanFree,
		OwnerID: ownerID,
		Settings: Settings{
			AllowPublicPages:  true,
			AllowGuestInvites: true,
			DefaultPermission: "edit",
			ExportEnabled:     true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, ws); err != nil {
		return nil, err
	}

	return ws, nil
}

// GetByID retrieves a workspace by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Workspace, error) {
	ws, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return ws, nil
}

// GetBySlug retrieves a workspace by slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	ws, err := s.store.GetBySlug(ctx, slug)
	if err != nil {
		return nil, ErrNotFound
	}
	return ws, nil
}

// ListByUser lists all workspaces the user is a member of.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Workspace, error) {
	return s.store.ListByUser(ctx, userID)
}

// Update updates a workspace.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Workspace, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// UpdateSettings updates workspace settings.
func (s *Service) UpdateSettings(ctx context.Context, id string, settings Settings) error {
	return s.store.UpdateSettings(ctx, id, settings)
}

// Delete deletes a workspace.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Transfer transfers ownership of a workspace.
func (s *Service) Transfer(ctx context.Context, id, newOwnerID string) error {
	return s.store.UpdateOwner(ctx, id, newOwnerID)
}
