package issues

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("issue not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrAccessDenied = errors.New("access denied")
	ErrLocked       = errors.New("issue is locked")
	ErrMissingTitle = errors.New("issue title is required")
)

// Issue represents a GitHub issue
type Issue struct {
	ID             string     `json:"id"`
	RepoID         string     `json:"repo_id"`
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	AuthorID       string     `json:"author_id"`
	State          string     `json:"state"` // open, closed
	StateReason    string     `json:"state_reason,omitempty"`
	IsLocked       bool       `json:"is_locked"`
	LockReason     string     `json:"lock_reason,omitempty"`
	MilestoneID    string     `json:"milestone_id,omitempty"`
	CommentCount   int        `json:"comment_count"`
	ReactionsCount int        `json:"reactions_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	ClosedByID     string     `json:"closed_by_id,omitempty"`

	// Populated from joins
	Labels    []*Label `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"` // From issue_assignees table
}

// Label represents an issue label
type Label struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Milestone represents a milestone
type Milestone struct {
	ID          string     `json:"id"`
	RepoID      string     `json:"repo_id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"` // open, closed
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// Comment represents a comment on an issue
type Comment struct {
	ID         string    `json:"id"`
	TargetType string    `json:"target_type"` // issue, pull_request
	TargetID   string    `json:"target_id"`
	UserID     string    `json:"user_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// IssueLabel represents the association between an issue and a label
// Uses composite PK (issue_id, label_id) - no ID field
type IssueLabel struct {
	IssueID   string    `json:"issue_id"`
	LabelID   string    `json:"label_id"`
	CreatedAt time.Time `json:"created_at"`
}

// IssueAssignee represents the association between an issue and an assignee
// Uses composite PK (issue_id, user_id) - no ID field
type IssueAssignee struct {
	IssueID   string    `json:"issue_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateIn is the input for creating an issue
type CreateIn struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Assignees   []string `json:"assignees,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	MilestoneID string   `json:"milestone_id,omitempty"`
}

// UpdateIn is the input for updating an issue
type UpdateIn struct {
	Title       *string   `json:"title,omitempty"`
	Body        *string   `json:"body,omitempty"`
	State       *string   `json:"state,omitempty"`
	StateReason *string   `json:"state_reason,omitempty"`
	Assignees   *[]string `json:"assignees,omitempty"`
	Labels      *[]string `json:"labels,omitempty"`
	MilestoneID *string   `json:"milestone_id,omitempty"`
}

// ListOpts are options for listing issues
type ListOpts struct {
	State       string // open, closed, all
	Sort        string // created, updated, comments
	Direction   string // asc, desc
	Labels      []string
	Assignee    string
	MilestoneID string
	Limit       int
	Offset      int
}

// API is the issues service interface
type API interface {
	// CRUD
	Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*Issue, error)
	GetByID(ctx context.Context, id string) (*Issue, error)
	GetByNumber(ctx context.Context, repoID string, number int) (*Issue, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, opts *ListOpts) ([]*Issue, int, error)

	// State management
	Close(ctx context.Context, id, userID, reason string) error
	Reopen(ctx context.Context, id string) error
	Lock(ctx context.Context, id, reason string) error
	Unlock(ctx context.Context, id string) error

	// Labels
	AddLabels(ctx context.Context, id string, labelIDs []string) error
	RemoveLabel(ctx context.Context, id, labelID string) error
	SetLabels(ctx context.Context, id string, labelIDs []string) error

	// Assignees
	AddAssignees(ctx context.Context, id string, userIDs []string) error
	RemoveAssignees(ctx context.Context, id string, userIDs []string) error

	// Comments
	AddComment(ctx context.Context, issueID, userID, body string) (*Comment, error)
	UpdateComment(ctx context.Context, commentID, body string) (*Comment, error)
	DeleteComment(ctx context.Context, commentID string) error
	ListComments(ctx context.Context, issueID string) ([]*Comment, error)
}

// Store is the issues data store interface
type Store interface {
	Create(ctx context.Context, i interface{}) error
	GetByID(ctx context.Context, id string) (*Issue, error)
	GetByNumber(ctx context.Context, repoID string, number int) (*Issue, error)
	Update(ctx context.Context, i *Issue) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, repoID string, state string, limit, offset int) ([]*Issue, int, error)
	GetNextNumber(ctx context.Context, repoID string) (int, error)

	// Labels
	AddLabel(ctx context.Context, il *IssueLabel) error
	RemoveLabel(ctx context.Context, issueID, labelID string) error
	ListLabels(ctx context.Context, issueID string) ([]string, error)

	// Assignees
	AddAssignee(ctx context.Context, ia *IssueAssignee) error
	RemoveAssignee(ctx context.Context, issueID, userID string) error
	ListAssignees(ctx context.Context, issueID string) ([]string, error)
}
