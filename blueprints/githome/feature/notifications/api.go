package notifications

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("notification not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Notification types
const (
	TypeIssue       = "issue"
	TypePullRequest = "pull_request"
	TypeRelease     = "release"
	TypeCommit      = "commit"
	TypeMention     = "mention"
)

// Reasons for notification
const (
	ReasonAssigned       = "assigned"
	ReasonMentioned      = "mentioned"
	ReasonSubscribed     = "subscribed"
	ReasonAuthor         = "author"
	ReasonReviewRequested = "review_requested"
	ReasonTeamMention    = "team_mention"
	ReasonComment        = "comment"
	ReasonStateChange    = "state_change"
)

// Notification represents a user notification
type Notification struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	RepoID     string     `json:"repo_id,omitempty"`
	Type       string     `json:"type"`
	ActorID    string     `json:"actor_id,omitempty"`
	TargetType string     `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Title      string     `json:"title"`
	Reason     string     `json:"reason"`
	Unread     bool       `json:"unread"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastReadAt *time.Time `json:"last_read_at,omitempty"`
}

// CreateIn is the input for creating a notification
type CreateIn struct {
	UserID     string `json:"user_id"`
	RepoID     string `json:"repo_id,omitempty"`
	Type       string `json:"type"`
	ActorID    string `json:"actor_id,omitempty"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Title      string `json:"title"`
	Reason     string `json:"reason"`
}

// ListOpts are options for listing notifications
type ListOpts struct {
	All        bool // Include read notifications
	Limit      int
	Offset     int
}

// API is the notifications service interface
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Notification, error)
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, userID string, opts *ListOpts) ([]*Notification, int, error)
	MarkAsRead(ctx context.Context, id string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	MarkRepoAsRead(ctx context.Context, userID, repoID string) error
	Delete(ctx context.Context, id string) error
	GetUnreadCount(ctx context.Context, userID string) (int, error)
}

// Store is the notifications data store interface
type Store interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]*Notification, int, error)
	MarkAsRead(ctx context.Context, id string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	MarkRepoAsRead(ctx context.Context, userID, repoID string) error
	Delete(ctx context.Context, id string) error
	CountUnread(ctx context.Context, userID string) (int, error)
}
