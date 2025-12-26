package labels

import (
	"context"
	"errors"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("label not found")
)

// Service implements the labels API.
type Service struct {
	store Store
}

// NewService creates a new labels service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, projectID string, in *CreateIn) (*Label, error) {
	label := &Label{
		ID:          ulid.New(),
		ProjectID:   projectID,
		Name:        in.Name,
		Color:       in.Color,
		Description: in.Description,
	}

	if err := s.store.Create(ctx, label); err != nil {
		return nil, err
	}

	return label, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Label, error) {
	l, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrNotFound
	}
	return l, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Label, error) {
	return s.store.ListByProject(ctx, projectID)
}

func (s *Service) GetByIssue(ctx context.Context, issueID string) ([]*Label, error) {
	return s.store.GetByIssue(ctx, issueID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Label, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
