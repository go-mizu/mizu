package issues

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("issue not found")
)

// Service implements the issues API.
type Service struct {
	store    Store
	projects projects.API
}

// NewService creates a new issues service.
func NewService(store Store, projects projects.API) *Service {
	return &Service{store: store, projects: projects}
}

func (s *Service) Create(ctx context.Context, projectID, creatorID string, in *CreateIn) (*Issue, error) {
	// Get project to generate issue key
	project, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Get next issue number
	number, err := s.projects.NextIssueNumber(ctx, projectID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	issue := &Issue{
		ID:        ulid.New(),
		ProjectID: projectID,
		Number:    number,
		Key:       fmt.Sprintf("%s-%d", project.Key, number),
		Title:     in.Title,
		ColumnID:  in.ColumnID,
		CycleID:   in.CycleID,
		CreatorID: creatorID,
		Position:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Issue, error) {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}
	return issue, nil
}

func (s *Service) GetByKey(ctx context.Context, key string) (*Issue, error) {
	issue, err := s.store.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}
	return issue, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Issue, error) {
	return s.store.ListByProject(ctx, projectID)
}

func (s *Service) ListByColumn(ctx context.Context, columnID string) ([]*Issue, error) {
	return s.store.ListByColumn(ctx, columnID)
}

func (s *Service) ListByCycle(ctx context.Context, cycleID string) ([]*Issue, error) {
	return s.store.ListByCycle(ctx, cycleID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Service) Move(ctx context.Context, id string, in *MoveIn) (*Issue, error) {
	if err := s.store.Move(ctx, id, in.ColumnID, in.Position); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Service) AttachCycle(ctx context.Context, id, cycleID string) error {
	return s.store.AttachCycle(ctx, id, cycleID)
}

func (s *Service) DetachCycle(ctx context.Context, id string) error {
	return s.store.DetachCycle(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) Search(ctx context.Context, projectID, query string, limit int) ([]*Issue, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.Search(ctx, projectID, query, limit)
}
