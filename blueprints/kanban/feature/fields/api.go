// Package fields provides custom field management functionality.
package fields

import (
	"context"
)

// Field represents a custom field definition for a project.
type Field struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Kind         string `json:"kind"` // text, number, bool, date, ts, select, user, json
	Position     int    `json:"position"`
	IsRequired   bool   `json:"is_required"`
	IsArchived   bool   `json:"is_archived"`
	SettingsJSON string `json:"settings_json,omitempty"`
}

// Kind constants
const (
	KindText   = "text"
	KindNumber = "number"
	KindBool   = "bool"
	KindDate   = "date"
	KindTS     = "ts"
	KindSelect = "select"
	KindUser   = "user"
	KindJSON   = "json"
)

// CreateIn contains input for creating a field.
type CreateIn struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Position     int    `json:"position,omitempty"`
	IsRequired   bool   `json:"is_required,omitempty"`
	SettingsJSON string `json:"settings_json,omitempty"`
}

// UpdateIn contains input for updating a field.
type UpdateIn struct {
	Name         *string `json:"name,omitempty"`
	IsRequired   *bool   `json:"is_required,omitempty"`
	SettingsJSON *string `json:"settings_json,omitempty"`
}

// API defines the fields service contract.
type API interface {
	Create(ctx context.Context, projectID string, in *CreateIn) (*Field, error)
	GetByID(ctx context.Context, id string) (*Field, error)
	GetByKey(ctx context.Context, projectID, key string) (*Field, error)
	ListByProject(ctx context.Context, projectID string) ([]*Field, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Field, error)
	UpdatePosition(ctx context.Context, id string, position int) error
	Archive(ctx context.Context, id string) error
	Unarchive(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for fields.
type Store interface {
	Create(ctx context.Context, f *Field) error
	GetByID(ctx context.Context, id string) (*Field, error)
	GetByKey(ctx context.Context, projectID, key string) (*Field, error)
	ListByProject(ctx context.Context, projectID string) ([]*Field, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePosition(ctx context.Context, id string, position int) error
	Archive(ctx context.Context, id string) error
	Unarchive(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}
