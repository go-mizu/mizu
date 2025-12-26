// Package labels provides label management functionality.
package labels

import (
	"context"
)

// Label represents a project label.
type Label struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// CreateIn contains input for creating a label.
type CreateIn struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// UpdateIn contains input for updating a label.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

// API defines the labels service contract.
type API interface {
	Create(ctx context.Context, projectID string, in *CreateIn) (*Label, error)
	GetByID(ctx context.Context, id string) (*Label, error)
	ListByProject(ctx context.Context, projectID string) ([]*Label, error)
	GetByIssue(ctx context.Context, issueID string) ([]*Label, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Label, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for labels.
type Store interface {
	Create(ctx context.Context, l *Label) error
	GetByID(ctx context.Context, id string) (*Label, error)
	ListByProject(ctx context.Context, projectID string) ([]*Label, error)
	GetByIssue(ctx context.Context, issueID string) ([]*Label, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
}
