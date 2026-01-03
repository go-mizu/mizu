package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("notification not found")
)

// Service implements the notifications API.
type Service struct {
	store Store
	users users.API
	pages pages.API
}

// NewService creates a new notifications service.
func NewService(store Store, users users.API, pages pages.API) *Service {
	return &Service{store: store, users: users, pages: pages}
}

// Create creates a new notification.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Notification, error) {
	notification := &Notification{
		ID:        ulid.New(),
		UserID:    in.UserID,
		Type:      in.Type,
		Title:     in.Title,
		Body:      in.Body,
		PageID:    in.PageID,
		ActorID:   in.ActorID,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, notification); err != nil {
		return nil, err
	}

	return s.enrichNotification(ctx, notification)
}

// GetByID retrieves a notification by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Notification, error) {
	n, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.enrichNotification(ctx, n)
}

// List lists notifications for a user.
func (s *Service) List(ctx context.Context, userID string, opts ListOpts) ([]*Notification, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	notifications, err := s.store.List(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	return s.enrichNotifications(ctx, notifications)
}

// MarkRead marks a notification as read.
func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.store.MarkRead(ctx, id)
}

// MarkAllRead marks all notifications as read for a user.
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.store.MarkAllRead(ctx, userID)
}

// Delete deletes a notification.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// CountUnread counts unread notifications for a user.
func (s *Service) CountUnread(ctx context.Context, userID string) (int, error) {
	return s.store.CountUnread(ctx, userID)
}

// enrichNotification adds user and page data.
func (s *Service) enrichNotification(ctx context.Context, n *Notification) (*Notification, error) {
	if n.ActorID != "" {
		actor, _ := s.users.GetByID(ctx, n.ActorID)
		n.Actor = actor
	}
	if n.PageID != "" {
		if page, err := s.pages.GetByID(ctx, n.PageID); err == nil {
			n.Page = &pages.PageRef{
				ID:    page.ID,
				Title: page.Title,
				Icon:  page.Icon,
			}
		}
	}
	return n, nil
}

// enrichNotifications adds user and page data to multiple notifications.
func (s *Service) enrichNotifications(ctx context.Context, notifications []*Notification) ([]*Notification, error) {
	if len(notifications) == 0 {
		return notifications, nil
	}

	// Collect IDs
	actorIDs := make([]string, 0)
	for _, n := range notifications {
		if n.ActorID != "" {
			actorIDs = append(actorIDs, n.ActorID)
		}
	}

	// Batch fetch users
	usersMap, _ := s.users.GetByIDs(ctx, actorIDs)

	// Attach data
	for _, n := range notifications {
		if n.ActorID != "" {
			n.Actor = usersMap[n.ActorID]
		}
		if n.PageID != "" {
			if page, err := s.pages.GetByID(ctx, n.PageID); err == nil {
				n.Page = &pages.PageRef{
					ID:    page.ID,
					Title: page.Title,
					Icon:  page.Icon,
				}
			}
		}
	}

	return notifications, nil
}
