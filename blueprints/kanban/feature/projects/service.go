package projects

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrKeyExists = errors.New("project key already exists")
	ErrNotFound  = errors.New("project not found")
)

// Service implements the projects API.
type Service struct {
	store Store
}

// NewService creates a new projects service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, workspaceID string, in *CreateIn) (*Project, error) {
	key := strings.ToUpper(in.Key)

	// Check if key exists
	existing, err := s.store.GetByKey(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrKeyExists
	}

	color := in.Color
	if color == "" {
		color = "#6366f1" // Default indigo
	}

	now := time.Now()
	project := &Project{
		ID:           ulid.New(),
		WorkspaceID:  workspaceID,
		Key:          key,
		Name:         in.Name,
		Description:  in.Description,
		Color:        color,
		LeadID:       in.LeadID,
		Status:       StatusActive,
		IssueCounter: 0,
		StartDate:    in.StartDate,
		TargetDate:   in.TargetDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Project, error) {
	p, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) GetByKey(ctx context.Context, workspaceID, key string) (*Project, error) {
	p, err := s.store.GetByKey(ctx, workspaceID, strings.ToUpper(key))
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Project, error) {
	return s.store.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Project, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) GetStats(ctx context.Context, id string) (*Stats, error) {
	return s.store.GetStats(ctx, id)
}

func (s *Service) NextIssueNumber(ctx context.Context, id string) (int, error) {
	return s.store.IncrementIssueCounter(ctx, id)
}
