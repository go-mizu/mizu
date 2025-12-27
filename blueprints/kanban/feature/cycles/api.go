// Package cycles provides cycle (planning period) management functionality.
package cycles

import (
	"context"
	"time"
)

// Cycle represents a planning period within a team.
type Cycle struct {
	ID             string    `json:"id"`
	TeamID         string    `json:"team_id"`
	Number         int       `json:"number"`
	Name           string    `json:"name"`
	Status         string    `json:"status"` // planning, active, completed
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	TotalCount     int       `json:"total_count"`     // Total issues in cycle
	CompletedCount int       `json:"completed_count"` // Completed issues in cycle
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Status constants
const (
	StatusPlanning  = "planning"
	StatusActive    = "active"
	StatusCompleted = "completed"
)

// CreateIn contains input for creating a cycle.
type CreateIn struct {
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// UpdateIn contains input for updating a cycle.
type UpdateIn struct {
	Name      *string    `json:"name,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// API defines the cycles service contract.
type API interface {
	Create(ctx context.Context, teamID string, in *CreateIn) (*Cycle, error)
	GetByID(ctx context.Context, id string) (*Cycle, error)
	GetByNumber(ctx context.Context, teamID string, number int) (*Cycle, error)
	ListByTeam(ctx context.Context, teamID string) ([]*Cycle, error)
	GetActive(ctx context.Context, teamID string) (*Cycle, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Cycle, error)
	UpdateStatus(ctx context.Context, id, status string) error
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for cycles.
type Store interface {
	Create(ctx context.Context, c *Cycle) error
	GetByID(ctx context.Context, id string) (*Cycle, error)
	GetByNumber(ctx context.Context, teamID string, number int) (*Cycle, error)
	ListByTeam(ctx context.Context, teamID string) ([]*Cycle, error)
	GetActive(ctx context.Context, teamID string) (*Cycle, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdateStatus(ctx context.Context, id, status string) error
	Delete(ctx context.Context, id string) error
	GetNextNumber(ctx context.Context, teamID string) (int, error)
}
