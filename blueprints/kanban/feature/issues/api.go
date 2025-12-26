// Package issues provides issue (card) management functionality.
package issues

import (
	"context"
	"time"
)

// Issue represents an issue (card) within a project board.
type Issue struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Number    int       `json:"number"`
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	ColumnID  string    `json:"column_id"`
	Position  int       `json:"position"`
	CreatorID string    `json:"creator_id"`
	CycleID   string    `json:"cycle_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateIn contains input for creating an issue.
type CreateIn struct {
	Title    string `json:"title"`
	ColumnID string `json:"column_id,omitempty"` // optional, uses default column if not set
	CycleID  string `json:"cycle_id,omitempty"`
}

// UpdateIn contains input for updating an issue.
type UpdateIn struct {
	Title   *string `json:"title,omitempty"`
	CycleID *string `json:"cycle_id,omitempty"`
}

// MoveIn contains input for moving an issue.
type MoveIn struct {
	ColumnID string `json:"column_id"`
	Position int    `json:"position"`
}

// API defines the issues service contract.
type API interface {
	Create(ctx context.Context, projectID, creatorID string, in *CreateIn) (*Issue, error)
	GetByID(ctx context.Context, id string) (*Issue, error)
	GetByKey(ctx context.Context, key string) (*Issue, error)
	ListByProject(ctx context.Context, projectID string) ([]*Issue, error)
	ListByColumn(ctx context.Context, columnID string) ([]*Issue, error)
	ListByCycle(ctx context.Context, cycleID string) ([]*Issue, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error)
	Move(ctx context.Context, id string, in *MoveIn) (*Issue, error)
	AttachCycle(ctx context.Context, id, cycleID string) error
	DetachCycle(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, projectID, query string, limit int) ([]*Issue, error)
}

// Store defines the data access contract for issues.
type Store interface {
	Create(ctx context.Context, i *Issue) error
	GetByID(ctx context.Context, id string) (*Issue, error)
	GetByKey(ctx context.Context, key string) (*Issue, error)
	ListByProject(ctx context.Context, projectID string) ([]*Issue, error)
	ListByColumn(ctx context.Context, columnID string) ([]*Issue, error)
	ListByCycle(ctx context.Context, cycleID string) ([]*Issue, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Move(ctx context.Context, id, columnID string, position int) error
	AttachCycle(ctx context.Context, id, cycleID string) error
	DetachCycle(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, projectID, query string, limit int) ([]*Issue, error)
}
