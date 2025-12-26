// Package notifications provides notification functionality.
package notifications

import (
	"context"
	"time"
)

// Notification represents a user notification.
type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"` // mention, assignment, due_date, comment
	IssueID   string     `json:"issue_id,omitempty"`
	ActorID   string     `json:"actor_id,omitempty"`
	Content   string     `json:"content"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Type constants
const (
	TypeMention    = "mention"
	TypeAssignment = "assignment"
	TypeDueDate    = "due_date"
	TypeComment    = "comment"
)

// CreateIn contains input for creating a notification.
type CreateIn struct {
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
	IssueID string `json:"issue_id,omitempty"`
	ActorID string `json:"actor_id,omitempty"`
	Content string `json:"content"`
}

// API defines the notifications service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Notification, error)
	GetByID(ctx context.Context, id string) (*Notification, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]*Notification, error)
	ListUnread(ctx context.Context, userID string) ([]*Notification, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	CountUnread(ctx context.Context, userID string) (int, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for notifications.
type Store interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]*Notification, error)
	ListUnread(ctx context.Context, userID string) ([]*Notification, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	CountUnread(ctx context.Context, userID string) (int, error)
	Delete(ctx context.Context, id string) error
}
