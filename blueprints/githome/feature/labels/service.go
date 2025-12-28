package labels

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the labels API
type Service struct {
	store Store
}

// NewService creates a new labels service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new label
func (s *Service) Create(ctx context.Context, repoID string, in *CreateIn) (*Label, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	// Normalize name
	name := strings.TrimSpace(in.Name)

	// Check if exists
	existing, _ := s.store.GetByName(ctx, repoID, name)
	if existing != nil {
		return nil, ErrExists
	}

	// Default color if not provided
	color := in.Color
	if color == "" {
		color = "0366d6"
	}
	// Remove # prefix if present
	color = strings.TrimPrefix(color, "#")

	label := &Label{
		ID:          ulid.New(),
		RepoID:      repoID,
		Name:        name,
		Color:       color,
		Description: in.Description,
		CreatedAt:   time.Now(),
	}

	if err := s.store.Create(ctx, label); err != nil {
		return nil, err
	}

	return label, nil
}

// GetByID retrieves a label by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Label, error) {
	label, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrNotFound
	}
	return label, nil
}

// GetByName retrieves a label by repository ID and name
func (s *Service) GetByName(ctx context.Context, repoID, name string) (*Label, error) {
	label, err := s.store.GetByName(ctx, repoID, name)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrNotFound
	}
	return label, nil
}

// Update updates a label
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Label, error) {
	label, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		// Check if new name conflicts with existing label
		if name != label.Name {
			existing, _ := s.store.GetByName(ctx, label.RepoID, name)
			if existing != nil {
				return nil, ErrExists
			}
		}
		label.Name = name
	}
	if in.Color != nil {
		label.Color = strings.TrimPrefix(*in.Color, "#")
	}
	if in.Description != nil {
		label.Description = *in.Description
	}

	if err := s.store.Update(ctx, label); err != nil {
		return nil, err
	}

	return label, nil
}

// Delete deletes a label
func (s *Service) Delete(ctx context.Context, id string) error {
	label, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if label == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists all labels for a repository
func (s *Service) List(ctx context.Context, repoID string) ([]*Label, error) {
	return s.store.List(ctx, repoID)
}
