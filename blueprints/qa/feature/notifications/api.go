package notifications

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("notification not found")
)

// Notification represents a notification.
type Notification struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// API defines the notifications service interface.
type API interface {
	Create(ctx context.Context, notification *Notification) (*Notification, error)
	ListByAccount(ctx context.Context, accountID string, limit int) ([]*Notification, error)
	GetUnreadCount(ctx context.Context, accountID string) (int64, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, accountID string) error
}

// Store defines the data storage interface for notifications.
type Store interface {
	Create(ctx context.Context, notification *Notification) error
	ListByAccount(ctx context.Context, accountID string, limit int) ([]*Notification, error)
	GetUnreadCount(ctx context.Context, accountID string) (int64, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, accountID string) error
}
