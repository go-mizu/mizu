package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the notifications API
type Service struct {
	store Store
}

// NewService creates a new notifications service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new notification
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Notification, error) {
	if in.UserID == "" || in.TargetType == "" || in.TargetID == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	notification := &Notification{
		ID:         ulid.New(),
		UserID:     in.UserID,
		RepoID:     in.RepoID,
		Type:       in.Type,
		ActorID:    in.ActorID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Title:      in.Title,
		Reason:     in.Reason,
		Unread:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// GetByID retrieves a notification by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Notification, error) {
	notification, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if notification == nil {
		return nil, ErrNotFound
	}
	return notification, nil
}

// List lists notifications for a user
func (s *Service) List(ctx context.Context, userID string, opts *ListOpts) ([]*Notification, int, error) {
	limit := 30
	offset := 0
	unreadOnly := true

	if opts != nil {
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
		if opts.All {
			unreadOnly = false
		}
	}

	return s.store.List(ctx, userID, unreadOnly, limit, offset)
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, id string) error {
	notification, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notification == nil {
		return ErrNotFound
	}
	return s.store.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *Service) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.store.MarkAllAsRead(ctx, userID)
}

// MarkRepoAsRead marks all notifications for a repository as read
func (s *Service) MarkRepoAsRead(ctx context.Context, userID, repoID string) error {
	return s.store.MarkRepoAsRead(ctx, userID, repoID)
}

// Delete deletes a notification
func (s *Service) Delete(ctx context.Context, id string) error {
	notification, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notification == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// GetUnreadCount gets the count of unread notifications for a user
func (s *Service) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return s.store.CountUnread(ctx, userID)
}
