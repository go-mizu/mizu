// Package notifications provides notification delivery and management.
package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// NotificationType is the type of notification.
type NotificationType string

const (
	TypeFollow  NotificationType = "follow"
	TypeLike    NotificationType = "like"
	TypeRepost  NotificationType = "repost"
	TypeMention NotificationType = "mention"
	TypeReply   NotificationType = "reply"
	TypePoll    NotificationType = "poll"
	TypeUpdate  NotificationType = "update"
)

// Notification represents a user notification.
type Notification struct {
	ID        string            `json:"id"`
	Type      NotificationType  `json:"type"`
	AccountID string            `json:"account_id"`
	ActorID   string            `json:"actor_id,omitempty"`
	PostID    string            `json:"post_id,omitempty"`
	Read      bool              `json:"read"`
	CreatedAt time.Time         `json:"created_at"`

	// Loaded relations
	Actor *accounts.Account `json:"actor,omitempty"`
}

// API defines the notifications service contract.
type API interface {
	List(ctx context.Context, accountID string, types []NotificationType, limit int, maxID, sinceID string, excludeTypes []NotificationType) ([]*Notification, error)
	Get(ctx context.Context, id, accountID string) (*Notification, error)
	MarkAsRead(ctx context.Context, id, accountID string) error
	MarkAllAsRead(ctx context.Context, accountID string) error
	Dismiss(ctx context.Context, id, accountID string) error
	DismissAll(ctx context.Context, accountID string) error
	CountUnread(ctx context.Context, accountID string) (int, error)
	CleanOld(ctx context.Context, olderThan time.Duration) (int64, error)
}

// Store defines the data access contract for notifications.
type Store interface {
	List(ctx context.Context, accountID string, types, excludeTypes []NotificationType, limit int, maxID, sinceID string) ([]*Notification, error)
	Get(ctx context.Context, id, accountID string) (*Notification, error)
	MarkAsRead(ctx context.Context, id, accountID string) error
	MarkAllAsRead(ctx context.Context, accountID string) error
	Dismiss(ctx context.Context, id, accountID string) error
	DismissAll(ctx context.Context, accountID string) error
	CountUnread(ctx context.Context, accountID string) (int, error)
	CleanOld(ctx context.Context, olderThan time.Duration) (int64, error)
}
