package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the notifications API.
type Service struct {
	store Store
}

// NewService creates a new notifications service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a notification.
func (s *Service) Create(ctx context.Context, notification *Notification) (*Notification, error) {
	notification.ID = ulid.New()
	notification.CreatedAt = time.Now()
	if err := s.store.Create(ctx, notification); err != nil {
		return nil, err
	}
	return notification, nil
}

// ListByAccount lists notifications.
func (s *Service) ListByAccount(ctx context.Context, accountID string, limit int) ([]*Notification, error) {
	return s.store.ListByAccount(ctx, accountID, limit)
}

// GetUnreadCount returns unread count.
func (s *Service) GetUnreadCount(ctx context.Context, accountID string) (int64, error) {
	return s.store.GetUnreadCount(ctx, accountID)
}

// MarkRead marks a notification read.
func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.store.MarkRead(ctx, id)
}

// MarkAllRead marks all notifications read.
func (s *Service) MarkAllRead(ctx context.Context, accountID string) error {
	return s.store.MarkAllRead(ctx, accountID)
}
