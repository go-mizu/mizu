package collaborators

import (
	"context"
	"time"
)

// Service implements the collaborators API
type Service struct {
	store Store
}

// NewService creates a new collaborators service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Add adds a collaborator to a repository
func (s *Service) Add(ctx context.Context, repoID, userID string, permission string) error {
	if repoID == "" || userID == "" {
		return ErrInvalidInput
	}

	// Validate permission
	if !s.validPermission(permission) {
		return ErrInvalidInput
	}

	// Check if already exists
	existing, err := s.store.Get(ctx, repoID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrExists
	}

	collab := &Collaborator{
		RepoID:     repoID,
		UserID:     userID,
		Permission: permission,
		CreatedAt:  time.Now(),
	}

	return s.store.Create(ctx, collab)
}

// Remove removes a collaborator from a repository
func (s *Service) Remove(ctx context.Context, repoID, userID string) error {
	existing, err := s.store.Get(ctx, repoID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, repoID, userID)
}

// Update updates a collaborator's permission
func (s *Service) Update(ctx context.Context, repoID, userID string, permission string) error {
	// Validate permission
	if !s.validPermission(permission) {
		return ErrInvalidInput
	}

	existing, err := s.store.Get(ctx, repoID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}

	existing.Permission = permission
	return s.store.Update(ctx, existing)
}

// Get gets a collaborator
func (s *Service) Get(ctx context.Context, repoID, userID string) (*Collaborator, error) {
	collab, err := s.store.Get(ctx, repoID, userID)
	if err != nil {
		return nil, err
	}
	if collab == nil {
		return nil, ErrNotFound
	}
	return collab, nil
}

// List lists collaborators for a repository
func (s *Service) List(ctx context.Context, repoID string, opts *ListOpts) ([]*Collaborator, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.List(ctx, repoID, limit, offset)
}

// ListUserRepos lists repository IDs a user is a collaborator on
func (s *Service) ListUserRepos(ctx context.Context, userID string, opts *ListOpts) ([]string, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByUser(ctx, userID, limit, offset)
}

// GetPermission gets a collaborator's permission level
func (s *Service) GetPermission(ctx context.Context, repoID, userID string) (string, error) {
	collab, err := s.store.Get(ctx, repoID, userID)
	if err != nil {
		return "", err
	}
	if collab == nil {
		return "", nil
	}
	return collab.Permission, nil
}

// HasPermission checks if a collaborator has at least the required permission
func (s *Service) HasPermission(ctx context.Context, repoID, userID string, required string) (bool, error) {
	perm, err := s.GetPermission(ctx, repoID, userID)
	if err != nil {
		return false, err
	}
	if perm == "" {
		return false, nil
	}
	return s.permissionAtLeast(perm, required), nil
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

func (s *Service) permissionAtLeast(have, need string) bool {
	levels := map[string]int{
		PermissionRead:     1,
		PermissionTriage:   2,
		PermissionWrite:    3,
		PermissionMaintain: 4,
		PermissionAdmin:    5,
	}
	return levels[have] >= levels[need]
}
