// Package issues provides issue management functionality.
package issues

import (
	"context"
	"time"
)

// Issue represents a project issue/task.
type Issue struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Number      int        `json:"number"`
	Key         string     `json:"key"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Type        string     `json:"type"`     // epic, story, task, bug, subtask
	Status      string     `json:"status"`   // backlog, todo, in_progress, in_review, done, cancelled
	Priority    string     `json:"priority"` // urgent, high, medium, low, none
	ParentID    string     `json:"parent_id,omitempty"`
	CreatorID   string     `json:"creator_id"`
	SprintID    string     `json:"sprint_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Estimate    *int       `json:"estimate,omitempty"`
	Position    int        `json:"position"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Loaded relations
	AssigneeIDs []string `json:"assignee_ids,omitempty"`
	LabelIDs    []string `json:"label_ids,omitempty"`
}

// Type constants
const (
	TypeEpic    = "epic"
	TypeStory   = "story"
	TypeTask    = "task"
	TypeBug     = "bug"
	TypeSubtask = "subtask"
)

// Status constants
const (
	StatusBacklog    = "backlog"
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusInReview   = "in_review"
	StatusDone       = "done"
	StatusCancelled  = "cancelled"
)

// Priority constants
const (
	PriorityUrgent = "urgent"
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
	PriorityNone   = "none"
)

// Statuses returns all available statuses in order.
func Statuses() []string {
	return []string{StatusBacklog, StatusTodo, StatusInProgress, StatusInReview, StatusDone, StatusCancelled}
}

// CreateIn contains input for creating an issue.
type CreateIn struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Type        string     `json:"type,omitempty"`
	Status      string     `json:"status,omitempty"`
	Priority    string     `json:"priority,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`
	SprintID    string     `json:"sprint_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Estimate    *int       `json:"estimate,omitempty"`
	AssigneeIDs []string   `json:"assignee_ids,omitempty"`
	LabelIDs    []string   `json:"label_ids,omitempty"`
}

// UpdateIn contains input for updating an issue.
type UpdateIn struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Type        *string    `json:"type,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	ParentID    *string    `json:"parent_id,omitempty"`
	SprintID    *string    `json:"sprint_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Estimate    *int       `json:"estimate,omitempty"`
}

// MoveIn contains input for moving an issue.
type MoveIn struct {
	Status   string `json:"status"`
	Position int    `json:"position"`
}

// Filter contains filter options for listing issues.
type Filter struct {
	Status     string `json:"status,omitempty"`
	Priority   string `json:"priority,omitempty"`
	Type       string `json:"type,omitempty"`
	AssigneeID string `json:"assignee_id,omitempty"`
	SprintID   string `json:"sprint_id,omitempty"`
	ParentID   string `json:"parent_id,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// API defines the issues service contract.
type API interface {
	Create(ctx context.Context, projectID, creatorID string, in *CreateIn) (*Issue, error)
	GetByID(ctx context.Context, id string) (*Issue, error)
	GetByKey(ctx context.Context, key string) (*Issue, error)
	ListByProject(ctx context.Context, projectID string, filter *Filter) ([]*Issue, error)
	ListByStatus(ctx context.Context, projectID string) (map[string][]*Issue, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error)
	Move(ctx context.Context, id string, in *MoveIn) (*Issue, error)
	Delete(ctx context.Context, id string) error
	AddAssignee(ctx context.Context, issueID, userID string) error
	RemoveAssignee(ctx context.Context, issueID, userID string) error
	AddLabel(ctx context.Context, issueID, labelID string) error
	RemoveLabel(ctx context.Context, issueID, labelID string) error
	Search(ctx context.Context, projectID, query string, limit int) ([]*Issue, error)
}

// Store defines the data access contract for issues.
type Store interface {
	Create(ctx context.Context, i *Issue) error
	GetByID(ctx context.Context, id string) (*Issue, error)
	GetByKey(ctx context.Context, key string) (*Issue, error)
	ListByProject(ctx context.Context, projectID string, filter *Filter) ([]*Issue, error)
	ListByStatus(ctx context.Context, projectID string) (map[string][]*Issue, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePosition(ctx context.Context, id, status string, position int) error
	Delete(ctx context.Context, id string) error
	AddAssignee(ctx context.Context, issueID, userID string) error
	RemoveAssignee(ctx context.Context, issueID, userID string) error
	GetAssignees(ctx context.Context, issueID string) ([]string, error)
	AddLabel(ctx context.Context, issueID, labelID string) error
	RemoveLabel(ctx context.Context, issueID, labelID string) error
	GetLabels(ctx context.Context, issueID string) ([]string, error)
	Search(ctx context.Context, projectID, query string, limit int) ([]*Issue, error)
}
