// Package activities provides activity tracking functionality for issues.
package activities

import (
	"context"
	"time"
)

// Action types for activity tracking.
const (
	ActionIssueCreated    = "issue_created"
	ActionIssueUpdated    = "issue_updated"
	ActionStatusChanged   = "status_changed"
	ActionPriorityChanged = "priority_changed"
	ActionAssigneeAdded   = "assignee_added"
	ActionAssigneeRemoved = "assignee_removed"
	ActionCycleAttached   = "cycle_attached"
	ActionCycleDetached   = "cycle_detached"
	ActionStartDateSet    = "start_date_set"
	ActionStartDateCleared = "start_date_cleared"
	ActionDueDateSet      = "due_date_set"
	ActionDueDateCleared  = "due_date_cleared"
	ActionCommentAdded    = "comment_added"
	ActionTitleChanged    = "title_changed"
	ActionDescChanged     = "description_changed"
)

// Activity represents an activity log entry for an issue.
type Activity struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	ActorID   string    `json:"actor_id"`
	Action    string    `json:"action"`
	OldValue  string    `json:"old_value,omitempty"`
	NewValue  string    `json:"new_value,omitempty"`
	Metadata  string    `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ActivityWithContext includes additional context for display.
type ActivityWithContext struct {
	*Activity
	ActorName string `json:"actor_name,omitempty"`
	IssueKey  string `json:"issue_key,omitempty"`
}

// CreateIn contains input for creating an activity.
type CreateIn struct {
	Action   string `json:"action"`
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

// API defines the activities service contract.
type API interface {
	Create(ctx context.Context, issueID, actorID string, in *CreateIn) (*Activity, error)
	GetByID(ctx context.Context, id string) (*Activity, error)
	ListByIssue(ctx context.Context, issueID string) ([]*Activity, error)
	ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*ActivityWithContext, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for activities.
type Store interface {
	Create(ctx context.Context, a *Activity) error
	GetByID(ctx context.Context, id string) (*Activity, error)
	ListByIssue(ctx context.Context, issueID string) ([]*Activity, error)
	ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*ActivityWithContext, error)
	Delete(ctx context.Context, id string) error
	CountByIssue(ctx context.Context, issueID string) (int, error)
}
