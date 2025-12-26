// Package sprints provides sprint management functionality.
package sprints

import (
	"context"
	"time"
)

// Sprint represents a project sprint.
type Sprint struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Goal      string     `json:"goal,omitempty"`
	Status    string     `json:"status"` // planning, active, completed
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Status constants
const (
	StatusPlanning  = "planning"
	StatusActive    = "active"
	StatusCompleted = "completed"
)

// CreateIn contains input for creating a sprint.
type CreateIn struct {
	Name      string     `json:"name"`
	Goal      string     `json:"goal,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// UpdateIn contains input for updating a sprint.
type UpdateIn struct {
	Name      *string    `json:"name,omitempty"`
	Goal      *string    `json:"goal,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// API defines the sprints service contract.
type API interface {
	Create(ctx context.Context, projectID string, in *CreateIn) (*Sprint, error)
	GetByID(ctx context.Context, id string) (*Sprint, error)
	GetActive(ctx context.Context, projectID string) (*Sprint, error)
	ListByProject(ctx context.Context, projectID string) ([]*Sprint, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Sprint, error)
	Start(ctx context.Context, id string) (*Sprint, error)
	Complete(ctx context.Context, id string) (*Sprint, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for sprints.
type Store interface {
	Create(ctx context.Context, sp *Sprint) error
	GetByID(ctx context.Context, id string) (*Sprint, error)
	GetActive(ctx context.Context, projectID string) (*Sprint, error)
	ListByProject(ctx context.Context, projectID string) ([]*Sprint, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdateStatus(ctx context.Context, id, status string) error
	Delete(ctx context.Context, id string) error
	GetIssueCount(ctx context.Context, sprintID string) (int, error)
	GetDoneIssueCount(ctx context.Context, sprintID string) (int, error)
}
