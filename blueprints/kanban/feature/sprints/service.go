package sprints

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound     = errors.New("sprint not found")
	ErrActiveExists = errors.New("an active sprint already exists")
)

// Service implements the sprints API.
type Service struct {
	store Store
}

// NewService creates a new sprints service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, projectID string, in *CreateIn) (*Sprint, error) {
	sprint := &Sprint{
		ID:        ulid.New(),
		ProjectID: projectID,
		Name:      in.Name,
		Goal:      in.Goal,
		Status:    StatusPlanning,
		StartDate: in.StartDate,
		EndDate:   in.EndDate,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, sprint); err != nil {
		return nil, err
	}

	return sprint, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Sprint, error) {
	sp, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrNotFound
	}
	return sp, nil
}

func (s *Service) GetActive(ctx context.Context, projectID string) (*Sprint, error) {
	return s.store.GetActive(ctx, projectID)
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Sprint, error) {
	return s.store.ListByProject(ctx, projectID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Sprint, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Start(ctx context.Context, id string) (*Sprint, error) {
	sprint, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint == nil {
		return nil, ErrNotFound
	}

	// Check if there's already an active sprint
	active, err := s.store.GetActive(ctx, sprint.ProjectID)
	if err != nil {
		return nil, err
	}
	if active != nil && active.ID != id {
		return nil, ErrActiveExists
	}

	if err := s.store.UpdateStatus(ctx, id, StatusActive); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, id)
}

func (s *Service) Complete(ctx context.Context, id string) (*Sprint, error) {
	if err := s.store.UpdateStatus(ctx, id, StatusCompleted); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
