package notifications

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("notification not found")
)

// Notification represents a notification
type Notification struct {
	ID              string             `json:"id"`
	Unread          bool               `json:"unread"`
	Reason          string             `json:"reason"` // assign, author, comment, invitation, manual, mention, review_requested, security_alert, state_change, subscribed, team_mention
	UpdatedAt       time.Time          `json:"updated_at"`
	LastReadAt      *time.Time         `json:"last_read_at"`
	Subject         *Subject           `json:"subject"`
	Repository      *Repository        `json:"repository"`
	URL             string             `json:"url"`
	SubscriptionURL string             `json:"subscription_url"`
}

// Subject represents the subject of a notification
type Subject struct {
	Title            string `json:"title"`
	URL              string `json:"url"`
	LatestCommentURL string `json:"latest_comment_url,omitempty"`
	Type             string `json:"type"` // Issue, PullRequest, Commit, Release, etc.
}

// Repository is a minimal repo reference
type Repository struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// ThreadSubscription represents a thread subscription
type ThreadSubscription struct {
	Subscribed bool       `json:"subscribed"`
	Ignored    bool       `json:"ignored"`
	Reason     string     `json:"reason,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	URL        string     `json:"url"`
	ThreadURL  string     `json:"thread_url"`
}

// ListOpts contains options for listing notifications
type ListOpts struct {
	Page          int       `json:"page,omitempty"`
	PerPage       int       `json:"per_page,omitempty"`
	All           bool      `json:"all,omitempty"`
	Participating bool      `json:"participating,omitempty"`
	Since         time.Time `json:"since,omitempty"`
	Before        time.Time `json:"before,omitempty"`
}

// API defines the notifications service interface
type API interface {
	// List returns notifications for the authenticated user
	List(ctx context.Context, userID int64, opts *ListOpts) ([]*Notification, error)

	// ListForRepo returns notifications for a repository
	ListForRepo(ctx context.Context, userID int64, owner, repo string, opts *ListOpts) ([]*Notification, error)

	// MarkAsRead marks all notifications as read
	MarkAsRead(ctx context.Context, userID int64, lastReadAt time.Time) error

	// MarkRepoAsRead marks all notifications for a repo as read
	MarkRepoAsRead(ctx context.Context, userID int64, owner, repo string, lastReadAt time.Time) error

	// GetThread retrieves a notification thread
	GetThread(ctx context.Context, userID int64, threadID string) (*Notification, error)

	// MarkThreadAsRead marks a thread as read
	MarkThreadAsRead(ctx context.Context, userID int64, threadID string) error

	// MarkThreadAsDone marks a thread as done (removes it)
	MarkThreadAsDone(ctx context.Context, userID int64, threadID string) error

	// GetThreadSubscription returns the thread subscription
	GetThreadSubscription(ctx context.Context, userID int64, threadID string) (*ThreadSubscription, error)

	// SetThreadSubscription sets the thread subscription
	SetThreadSubscription(ctx context.Context, userID int64, threadID string, ignored bool) (*ThreadSubscription, error)

	// DeleteThreadSubscription removes the thread subscription
	DeleteThreadSubscription(ctx context.Context, userID int64, threadID string) error

	// Create a notification (internal use)
	Create(ctx context.Context, userID, repoID int64, reason, subjectType, subjectTitle, subjectURL string) (*Notification, error)
}

// Store defines the data access interface for notifications
type Store interface {
	Create(ctx context.Context, n *Notification, userID int64) error
	GetByID(ctx context.Context, id string, userID int64) (*Notification, error)
	MarkAsRead(ctx context.Context, userID int64, lastReadAt time.Time) error
	MarkRepoAsRead(ctx context.Context, userID, repoID int64, lastReadAt time.Time) error
	MarkThreadAsRead(ctx context.Context, id string, userID int64) error
	Delete(ctx context.Context, id string, userID int64) error
	List(ctx context.Context, userID int64, opts *ListOpts) ([]*Notification, error)
	ListForRepo(ctx context.Context, userID, repoID int64, opts *ListOpts) ([]*Notification, error)

	// Thread subscriptions
	GetSubscription(ctx context.Context, id string, userID int64) (*ThreadSubscription, error)
	SetSubscription(ctx context.Context, id string, userID int64, ignored bool) error
	DeleteSubscription(ctx context.Context, id string, userID int64) error
}
