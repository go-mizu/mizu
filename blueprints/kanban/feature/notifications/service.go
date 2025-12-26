package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("notification not found")
)

// Service implements the notifications API.
type Service struct {
	store Store
}

// NewService creates a new notifications service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in *CreateIn) (*Notification, error) {
	notification := &Notification{
		ID:        ulid.New(),
		UserID:    in.UserID,
		Type:      in.Type,
		IssueID:   in.IssueID,
		ActorID:   in.ActorID,
		Content:   in.Content,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Notification, error) {
	n, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, ErrNotFound
	}
	return n, nil
}

func (s *Service) ListByUser(ctx context.Context, userID string, limit int) ([]*Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListByUser(ctx, userID, limit)
}

func (s *Service) ListUnread(ctx context.Context, userID string) ([]*Notification, error) {
	return s.store.ListUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.store.MarkRead(ctx, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.store.MarkAllRead(ctx, userID)
}

func (s *Service) CountUnread(ctx context.Context, userID string) (int, error) {
	return s.store.CountUnread(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
