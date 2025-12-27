package activities

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("activity not found")
)

// Service implements the activities API.
type Service struct {
	store Store
}

// NewService creates a new activities service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, issueID, actorID string, in *CreateIn) (*Activity, error) {
	activity := &Activity{
		ID:        ulid.New(),
		IssueID:   issueID,
		ActorID:   actorID,
		Action:    in.Action,
		OldValue:  in.OldValue,
		NewValue:  in.NewValue,
		Metadata:  in.Metadata,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, activity); err != nil {
		return nil, err
	}

	return activity, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Activity, error) {
	a, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *Service) ListByIssue(ctx context.Context, issueID string) ([]*Activity, error) {
	return s.store.ListByIssue(ctx, issueID)
}

func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*ActivityWithContext, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.store.ListByWorkspace(ctx, workspaceID, limit, offset)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
