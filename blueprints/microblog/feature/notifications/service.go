package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// Service handles notification operations.
// Implements API interface.
type Service struct {
	store    Store
	accounts accounts.API
}

// NewService creates a new notifications service.
func NewService(store Store, accounts accounts.API) *Service {
	return &Service{store: store, accounts: accounts}
}

// List returns notifications for an account.
func (s *Service) List(ctx context.Context, accountID string, types []NotificationType, limit int, maxID, sinceID string, excludeTypes []NotificationType) ([]*Notification, error) {
	notifications, err := s.store.List(ctx, accountID, types, excludeTypes, limit, maxID, sinceID)
	if err != nil {
		return nil, err
	}

	// Load actors
	for _, n := range notifications {
		if n.ActorID != "" {
			n.Actor, _ = s.accounts.GetByID(ctx, n.ActorID)
		}
	}

	return notifications, nil
}

// Get returns a single notification.
func (s *Service) Get(ctx context.Context, id, accountID string) (*Notification, error) {
	n, err := s.store.Get(ctx, id, accountID)
	if err != nil {
		return nil, err
	}

	if n.ActorID != "" {
		n.Actor, _ = s.accounts.GetByID(ctx, n.ActorID)
	}

	return n, nil
}

// MarkAsRead marks a notification as read.
func (s *Service) MarkAsRead(ctx context.Context, id, accountID string) error {
	return s.store.MarkAsRead(ctx, id, accountID)
}

// MarkAllAsRead marks all notifications as read for an account.
func (s *Service) MarkAllAsRead(ctx context.Context, accountID string) error {
	return s.store.MarkAllAsRead(ctx, accountID)
}

// Dismiss removes a notification.
func (s *Service) Dismiss(ctx context.Context, id, accountID string) error {
	return s.store.Dismiss(ctx, id, accountID)
}

// DismissAll removes all notifications for an account.
func (s *Service) DismissAll(ctx context.Context, accountID string) error {
	return s.store.DismissAll(ctx, accountID)
}

// CountUnread returns the number of unread notifications.
func (s *Service) CountUnread(ctx context.Context, accountID string) (int, error) {
	return s.store.CountUnread(ctx, accountID)
}

// CleanOld removes notifications older than the given duration.
func (s *Service) CleanOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.store.CleanOld(ctx, olderThan)
}
