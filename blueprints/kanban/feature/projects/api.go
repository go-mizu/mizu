// Package projects provides project (board) management functionality.
package projects

import (
	"context"
)

// Project represents a project (board) within a team.
type Project struct {
	ID           string `json:"id"`
	TeamID       string `json:"team_id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	IssueCounter int    `json:"issue_counter"`
}

// CreateIn contains input for creating a project.
type CreateIn struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// UpdateIn contains input for updating a project.
type UpdateIn struct {
	Key  *string `json:"key,omitempty"`
	Name *string `json:"name,omitempty"`
}

// API defines the projects service contract.
type API interface {
	Create(ctx context.Context, teamID string, in *CreateIn) (*Project, error)
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByKey(ctx context.Context, teamID, key string) (*Project, error)
	ListByTeam(ctx context.Context, teamID string) ([]*Project, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Project, error)
	Delete(ctx context.Context, id string) error
	NextIssueNumber(ctx context.Context, id string) (int, error)
}

// Store defines the data access contract for projects.
type Store interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByKey(ctx context.Context, teamID, key string) (*Project, error)
	ListByTeam(ctx context.Context, teamID string) ([]*Project, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	IncrementIssueCounter(ctx context.Context, id string) (int, error)
}
