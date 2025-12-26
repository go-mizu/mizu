// Package projects provides project management functionality.
package projects

import (
	"context"
	"time"
)

// Project represents a project within a workspace.
type Project struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspace_id"`
	Key          string     `json:"key"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	Color        string     `json:"color"`
	LeadID       string     `json:"lead_id,omitempty"`
	Status       string     `json:"status"` // active, archived, completed
	IssueCounter int        `json:"issue_counter"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	TargetDate   *time.Time `json:"target_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Stats contains project statistics.
type Stats struct {
	TotalIssues      int `json:"total_issues"`
	DoneIssues       int `json:"done_issues"`
	InProgressIssues int `json:"in_progress_issues"`
	BacklogIssues    int `json:"backlog_issues"`
}

// Status constants
const (
	StatusActive    = "active"
	StatusArchived  = "archived"
	StatusCompleted = "completed"
)

// CreateIn contains input for creating a project.
type CreateIn struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Color       string     `json:"color,omitempty"`
	LeadID      string     `json:"lead_id,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
}

// UpdateIn contains input for updating a project.
type UpdateIn struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	Color       *string    `json:"color,omitempty"`
	LeadID      *string    `json:"lead_id,omitempty"`
	Status      *string    `json:"status,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
}

// API defines the projects service contract.
type API interface {
	Create(ctx context.Context, workspaceID string, in *CreateIn) (*Project, error)
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByKey(ctx context.Context, workspaceID, key string) (*Project, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Project, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Project, error)
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context, id string) (*Stats, error)
	NextIssueNumber(ctx context.Context, id string) (int, error)
}

// Store defines the data access contract for projects.
type Store interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByKey(ctx context.Context, workspaceID, key string) (*Project, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Project, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	IncrementIssueCounter(ctx context.Context, id string) (int, error)
	GetStats(ctx context.Context, id string) (*Stats, error)
}
