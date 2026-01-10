// Package views provides view management functionality.
package views

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("view not found")
)

// View types
const (
	TypeGrid     = "grid"
	TypeKanban   = "kanban"
	TypeCalendar = "calendar"
	TypeGallery  = "gallery"
	TypeTimeline = "timeline"
	TypeForm     = "form"
)

// View represents a saved view of a table.
type View struct {
	ID          string            `json:"id"`
	TableID     string            `json:"table_id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Config      json.RawMessage   `json:"config,omitempty"`
	Filters     []Filter          `json:"filters,omitempty"`
	Sorts       []SortSpec        `json:"sorts,omitempty"`
	Groups      []GroupSpec       `json:"groups,omitempty"`
	FieldConfig []FieldViewConfig `json:"field_config,omitempty"`
	Position    int               `json:"position"`
	IsDefault   bool              `json:"is_default"`
	IsLocked    bool              `json:"is_locked"`
	CreatedBy   string            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Filter defines a filter condition.
type Filter struct {
	FieldID  string      `json:"field_id"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// SortSpec defines a sort specification.
type SortSpec struct {
	FieldID   string `json:"field_id"`
	Direction string `json:"direction"` // asc, desc
}

// GroupSpec defines a grouping specification.
type GroupSpec struct {
	FieldID   string `json:"field_id"`
	Direction string `json:"direction"` // asc, desc
	Collapsed bool   `json:"collapsed"`
}

// FieldViewConfig defines field-specific view configuration.
type FieldViewConfig struct {
	FieldID  string `json:"field_id"`
	Visible  bool   `json:"visible"`
	Width    int    `json:"width"`
	Position int    `json:"position"`
}

// CreateIn contains input for creating a view.
type CreateIn struct {
	TableID   string `json:"table_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsDefault bool   `json:"is_default"`
}

// UpdateIn contains input for updating a view.
type UpdateIn struct {
	Name     *string `json:"name,omitempty"`
	IsLocked *bool   `json:"is_locked,omitempty"`
}

// API defines the views service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*View, error)
	GetByID(ctx context.Context, id string) (*View, error)
	Update(ctx context.Context, id string, in UpdateIn) (*View, error)
	Delete(ctx context.Context, id string) error
	Duplicate(ctx context.Context, id string, newName string) (*View, error)
	ListByTable(ctx context.Context, tableID string) ([]*View, error)

	// Configuration
	SetFilters(ctx context.Context, viewID string, filters []Filter) error
	SetSorts(ctx context.Context, viewID string, sorts []SortSpec) error
	SetGroups(ctx context.Context, viewID string, groups []GroupSpec) error
	SetFieldConfig(ctx context.Context, viewID string, config []FieldViewConfig) error
	SetConfig(ctx context.Context, viewID string, config map[string]interface{}) error
}

// Store defines the views data access interface.
type Store interface {
	Create(ctx context.Context, view *View) error
	GetByID(ctx context.Context, id string) (*View, error)
	Update(ctx context.Context, view *View) error
	Delete(ctx context.Context, id string) error
	ListByTable(ctx context.Context, tableID string) ([]*View, error)
}
