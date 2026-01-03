// Package views provides database view management.
package views

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// ViewType represents the type of view.
type ViewType string

const (
	ViewTable    ViewType = "table"
	ViewBoard    ViewType = "board"
	ViewList     ViewType = "list"
	ViewCalendar ViewType = "calendar"
	ViewGallery  ViewType = "gallery"
	ViewTimeline ViewType = "timeline"
)

// View represents a database view.
type View struct {
	ID         string     `json:"id"`
	DatabaseID string     `json:"database_id"`
	Name       string     `json:"name"`
	Type       ViewType   `json:"type"`
	Filter     *Filter    `json:"filter,omitempty"`
	Sorts      []Sort     `json:"sorts,omitempty"`
	Properties []ViewProp `json:"properties,omitempty"`
	GroupBy    string     `json:"group_by,omitempty"`
	CalendarBy string     `json:"calendar_by,omitempty"`
	Position   int        `json:"position"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ViewProp holds view-specific property configuration.
type ViewProp struct {
	PropertyID string `json:"property_id"`
	Visible    bool   `json:"visible"`
	Width      int    `json:"width,omitempty"`
}

// Filter represents a filter condition.
type Filter struct {
	And []Filter `json:"and,omitempty"`
	Or  []Filter `json:"or,omitempty"`

	PropertyID string      `json:"property_id,omitempty"`
	Operator   string      `json:"operator,omitempty"`
	Value      interface{} `json:"value,omitempty"`
}

// Sort represents a sort configuration.
type Sort struct {
	PropertyID string `json:"property_id"`
	Direction  string `json:"direction"` // asc, desc
}

// QueryResult holds the result of a view query.
type QueryResult struct {
	Items      []*pages.Page `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

// CreateIn contains input for creating a view.
type CreateIn struct {
	DatabaseID string     `json:"database_id"`
	Name       string     `json:"name"`
	Type       ViewType   `json:"type"`
	Filter     *Filter    `json:"filter,omitempty"`
	Sorts      []Sort     `json:"sorts,omitempty"`
	GroupBy    string     `json:"group_by,omitempty"`
	CalendarBy string     `json:"calendar_by,omitempty"`
	CreatedBy  string     `json:"-"`
}

// UpdateIn contains input for updating a view.
type UpdateIn struct {
	Name       *string    `json:"name,omitempty"`
	Filter     *Filter    `json:"filter,omitempty"`
	Sorts      []Sort     `json:"sorts,omitempty"`
	Properties []ViewProp `json:"properties,omitempty"`
	GroupBy    *string    `json:"group_by,omitempty"`
	CalendarBy *string    `json:"calendar_by,omitempty"`
}

// API defines the views service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*View, error)
	GetByID(ctx context.Context, id string) (*View, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*View, error)
	Delete(ctx context.Context, id string) error

	ListByDatabase(ctx context.Context, databaseID string) ([]*View, error)
	Reorder(ctx context.Context, databaseID string, viewIDs []string) error
	Duplicate(ctx context.Context, id string, userID string) (*View, error)

	// Query
	Query(ctx context.Context, viewID string, cursor string, limit int) (*QueryResult, error)
}

// Store defines the data access contract for views.
type Store interface {
	Create(ctx context.Context, v *View) error
	GetByID(ctx context.Context, id string) (*View, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByDatabase(ctx context.Context, databaseID string) ([]*View, error)
	Reorder(ctx context.Context, databaseID string, viewIDs []string) error
}
