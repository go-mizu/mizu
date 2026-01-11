// Package tables provides table management functionality.
package tables

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("table not found")
)

// Table represents a table within a base.
type Table struct {
	ID             string    `json:"id"`
	BaseID         string    `json:"base_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Icon           string    `json:"icon,omitempty"`
	Position       int       `json:"position"`
	PrimaryFieldID string    `json:"primary_field_id,omitempty"`
	AutoNumberSeq  int64     `json:"-"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a table.
type CreateIn struct {
	BaseID      string `json:"base_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// UpdateIn contains input for updating a table.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
}

// API defines the tables service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*Table, error)
	GetByID(ctx context.Context, id string) (*Table, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Table, error)
	Delete(ctx context.Context, id string) error
	Duplicate(ctx context.Context, id string, newName string) (*Table, error)
	ListByBase(ctx context.Context, baseID string) ([]*Table, error)
	Reorder(ctx context.Context, baseID string, tableIDs []string) error

	// Schema
	SetPrimaryField(ctx context.Context, tableID, fieldID string) error
	NextAutoNumber(ctx context.Context, tableID string) (int64, error)
}

// Store defines the tables data access interface.
type Store interface {
	Create(ctx context.Context, tbl *Table) error
	GetByID(ctx context.Context, id string) (*Table, error)
	Update(ctx context.Context, tbl *Table) error
	Delete(ctx context.Context, id string) error
	ListByBase(ctx context.Context, baseID string) ([]*Table, error)

	SetPrimaryField(ctx context.Context, tableID, fieldID string) error
	NextAutoNumber(ctx context.Context, tableID string) (int64, error)
}
