package repos

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/slug"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the repos API
type Service struct {
	store    Store
	reposDir string
}

// NewService creates a new repos service
func NewService(store Store, reposDir string) *Service {
	return &Service{
		store:    store,
		reposDir: reposDir,
	}
}

// Create creates a new repository
func (s *Service) Create(ctx context.Context, ownerID string, in *CreateIn) (*Repository, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	// Generate slug
	repoSlug := slug.Make(in.Name)
	if repoSlug == "" {
		return nil, ErrInvalidInput
	}

	// Check if exists
	existing, _ := s.store.GetByOwnerAndName(ctx, ownerID, "user", repoSlug)
	if existing != nil {
		return nil, ErrExists
	}

	defaultBranch := in.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	now := time.Now()
	repo := &Repository{
		ID:            ulid.New(),
		OwnerID:       ownerID,
		OwnerType:     "user",
		Name:          in.Name,
		Slug:          repoSlug,
		Description:   in.Description,
		DefaultBranch: defaultBranch,
		IsPrivate:     in.IsPrivate,
		License:       in.License,
		Topics:        in.Topics,
		HasIssues:     true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Create(ctx, repo); err != nil {
		return nil, err
	}

	// Create git repository directory
	repoPath := s.getRepoPath(ownerID, repoSlug)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		// Cleanup on failure
		s.store.Delete(ctx, repo.ID)
		return nil, err
	}

	return repo, nil
}

// GetByID retrieves a repository by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Repository, error) {
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	return repo, nil
}

// GetByOwnerAndName retrieves a repository by owner and name
func (s *Service) GetByOwnerAndName(ctx context.Context, ownerID, ownerType, name string) (*Repository, error) {
	repoSlug := strings.ToLower(name)
	repo, err := s.store.GetByOwnerAndName(ctx, ownerID, ownerType, repoSlug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	return repo, nil
}

// Update updates a repository
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Repository, error) {
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		repo.Name = *in.Name
		repo.Slug = slug.Make(*in.Name)
	}
	if in.Description != nil {
		repo.Description = *in.Description
	}
	if in.Website != nil {
		repo.Website = *in.Website
	}
	if in.IsPrivate != nil {
		repo.IsPrivate = *in.IsPrivate
	}
	if in.IsArchived != nil {
		repo.IsArchived = *in.IsArchived
	}
	if in.IsTemplate != nil {
		repo.IsTemplate = *in.IsTemplate
	}
	if in.DefaultBranch != nil {
		repo.DefaultBranch = *in.DefaultBranch
	}
	if in.HasIssues != nil {
		repo.HasIssues = *in.HasIssues
	}
	if in.HasWiki != nil {
		repo.HasWiki = *in.HasWiki
	}
	if in.HasProjects != nil {
		repo.HasProjects = *in.HasProjects
	}
	if in.Topics != nil {
		repo.Topics = *in.Topics
	}

	if err := s.store.Update(ctx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// Delete deletes a repository
func (s *Service) Delete(ctx context.Context, id string) error {
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrNotFound
	}

	// Delete git repository directory
	repoPath := s.getRepoPath(repo.OwnerID, repo.Slug)
	os.RemoveAll(repoPath)

	return s.store.Delete(ctx, id)
}

// ListByOwner lists repositories by owner
func (s *Service) ListByOwner(ctx context.Context, ownerID, ownerType string, opts *ListOpts) ([]*Repository, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByOwner(ctx, ownerID, ownerType, limit, offset)
}

// ListPublic lists public repositories
func (s *Service) ListPublic(ctx context.Context, opts *ListOpts) ([]*Repository, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListPublic(ctx, limit, offset)
}

// ListAccessible lists repositories accessible to a user
func (s *Service) ListAccessible(ctx context.Context, userID string, opts *ListOpts) ([]*Repository, error) {
	// For now, just list user's own repos
	// TODO: Include collaborated repos
	return s.ListByOwner(ctx, userID, "user", opts)
}

// AddCollaborator adds a collaborator to a repository
func (s *Service) AddCollaborator(ctx context.Context, repoID, userID string, perm Permission) error {
	collab := &Collaborator{
		RepoID:     repoID,
		UserID:     userID,
		Permission: perm,
		CreatedAt:  time.Now(),
	}
	return s.store.AddCollaborator(ctx, collab)
}

// RemoveCollaborator removes a collaborator from a repository
func (s *Service) RemoveCollaborator(ctx context.Context, repoID, userID string) error {
	return s.store.RemoveCollaborator(ctx, repoID, userID)
}

// GetPermission gets a user's permission level for a repository
func (s *Service) GetPermission(ctx context.Context, repoID, userID string) (Permission, error) {
	repo, err := s.store.GetByID(ctx, repoID)
	if err != nil {
		return "", err
	}
	if repo == nil {
		return "", ErrNotFound
	}

	// Owner has admin access
	if repo.OwnerID == userID {
		return PermissionAdmin, nil
	}

	// Check collaborator
	collab, err := s.store.GetCollaborator(ctx, repoID, userID)
	if err != nil {
		return "", err
	}
	if collab != nil {
		return collab.Permission, nil
	}

	// Public repos have read access for everyone
	if !repo.IsPrivate {
		return PermissionRead, nil
	}

	return "", nil
}

// CanAccess checks if a user has the required permission level
func (s *Service) CanAccess(ctx context.Context, repoID, userID string, required Permission) bool {
	perm, err := s.GetPermission(ctx, repoID, userID)
	if err != nil {
		return false
	}
	return s.permissionAtLeast(perm, required)
}

// ListCollaborators lists collaborators for a repository
func (s *Service) ListCollaborators(ctx context.Context, repoID string) ([]*Collaborator, error) {
	return s.store.ListCollaborators(ctx, repoID)
}

// Star stars a repository
func (s *Service) Star(ctx context.Context, userID, repoID string) error {
	// Check if already starred
	starred, err := s.store.IsStarred(ctx, userID, repoID)
	if err != nil {
		return err
	}
	if starred {
		return nil
	}

	star := &Star{
		UserID:    userID,
		RepoID:    repoID,
		CreatedAt: time.Now(),
	}

	if err := s.store.Star(ctx, star); err != nil {
		return err
	}

	// Update star count
	repo, _ := s.store.GetByID(ctx, repoID)
	if repo != nil {
		repo.StarCount++
		s.store.Update(ctx, repo)
	}

	return nil
}

// Unstar unstars a repository
func (s *Service) Unstar(ctx context.Context, userID, repoID string) error {
	if err := s.store.Unstar(ctx, userID, repoID); err != nil {
		return err
	}

	// Update star count
	repo, _ := s.store.GetByID(ctx, repoID)
	if repo != nil && repo.StarCount > 0 {
		repo.StarCount--
		s.store.Update(ctx, repo)
	}

	return nil
}

// IsStarred checks if a user has starred a repository
func (s *Service) IsStarred(ctx context.Context, userID, repoID string) (bool, error) {
	return s.store.IsStarred(ctx, userID, repoID)
}

// ListStarred lists repositories starred by a user
func (s *Service) ListStarred(ctx context.Context, userID string, opts *ListOpts) ([]*Repository, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListStarredByUser(ctx, userID, limit, offset)
}

// Fork forks a repository
func (s *Service) Fork(ctx context.Context, userID, repoID string, in *ForkIn) (*Repository, error) {
	original, err := s.store.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, ErrNotFound
	}

	name := in.Name
	if name == "" {
		name = original.Name
	}

	now := time.Now()
	fork := &Repository{
		ID:            ulid.New(),
		OwnerID:       userID,
		OwnerType:     "user",
		Name:          name,
		Slug:          slug.Make(name),
		Description:   original.Description,
		DefaultBranch: original.DefaultBranch,
		IsPrivate:     original.IsPrivate,
		IsFork:        true,
		ForkedFromID:  original.ID,
		HasIssues:     original.HasIssues,
		HasWiki:       original.HasWiki,
		HasProjects:   original.HasProjects,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Create(ctx, fork); err != nil {
		return nil, err
	}

	// Update fork count on original
	original.ForkCount++
	s.store.Update(ctx, original)

	return fork, nil
}

// ListForks lists forks of a repository
func (s *Service) ListForks(ctx context.Context, repoID string, opts *ListOpts) ([]*Repository, error) {
	// TODO: Implement fork listing
	return nil, nil
}

func (s *Service) getRepoPath(ownerID, slug string) string {
	return filepath.Join(s.reposDir, ownerID, slug+".git")
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

func (s *Service) permissionAtLeast(have, need Permission) bool {
	levels := map[Permission]int{
		PermissionRead:     1,
		PermissionTriage:   2,
		PermissionWrite:    3,
		PermissionMaintain: 4,
		PermissionAdmin:    5,
	}
	return levels[have] >= levels[need]
}
