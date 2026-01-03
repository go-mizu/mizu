// Package notifications provides notification management.
package notifications

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// NotificationType represents the type of notification.
type NotificationType string

const (
	NotifyMention  NotificationType = "mention"
	NotifyComment  NotificationType = "comment"
	NotifyReply    NotificationType = "reply"
	NotifyShare    NotificationType = "share"
	NotifyInvite   NotificationType = "invite"
	NotifyAssign   NotificationType = "assign"
	NotifyReminder NotificationType = "reminder"
)

// Notification represents a notification.
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Body      string           `json:"body,omitempty"`
	PageID    string           `json:"page_id,omitempty"`
	ActorID   string           `json:"actor_id,omitempty"`
	IsRead    bool             `json:"is_read"`
	CreatedAt time.Time        `json:"created_at"`

	// Enriched
	Page  *pages.PageRef `json:"page,omitempty"`
	Actor *users.User    `json:"actor,omitempty"`
}

// CreateIn contains input for creating a notification.
type CreateIn struct {
	UserID  string           `json:"user_id"`
	Type    NotificationType `json:"type"`
	Title   string           `json:"title"`
	Body    string           `json:"body,omitempty"`
	PageID  string           `json:"page_id,omitempty"`
	ActorID string           `json:"actor_id,omitempty"`
}

// ListOpts contains options for listing notifications.
type ListOpts struct {
	UnreadOnly bool
	Limit      int
	Cursor     string
}

// API defines the notifications service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Notification, error)
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, userID string, opts ListOpts) ([]*Notification, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	Delete(ctx context.Context, id string) error
	CountUnread(ctx context.Context, userID string) (int, error)
}

// Store defines the data access contract for notifications.
type Store interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, userID string, opts ListOpts) ([]*Notification, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	Delete(ctx context.Context, id string) error
	CountUnread(ctx context.Context, userID string) (int, error)
}
