// Package databases provides database (structured collection) management.
package databases

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
)

// PropertyType represents the type of a database property.
type PropertyType string

const (
	PropTitle        PropertyType = "title"
	PropRichText     PropertyType = "rich_text"
	PropNumber       PropertyType = "number"
	PropSelect       PropertyType = "select"
	PropMultiSelect  PropertyType = "multi_select"
	PropDate         PropertyType = "date"
	PropPerson       PropertyType = "person"
	PropFiles        PropertyType = "files"
	PropCheckbox     PropertyType = "checkbox"
	PropURL          PropertyType = "url"
	PropEmail        PropertyType = "email"
	PropPhone        PropertyType = "phone_number"
	PropFormula      PropertyType = "formula"
	PropRelation     PropertyType = "relation"
	PropRollup       PropertyType = "rollup"
	PropCreatedTime  PropertyType = "created_time"
	PropCreatedBy    PropertyType = "created_by"
	PropLastEditTime PropertyType = "last_edited_time"
	PropLastEditBy   PropertyType = "last_edited_by"
	PropStatus       PropertyType = "status"
)

// Database represents a structured collection.
type Database struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	PageID      string          `json:"page_id"`
	Title       string          `json:"title"`
	Description []blocks.RichText `json:"description,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	Cover       string          `json:"cover,omitempty"`
	IsInline    bool            `json:"is_inline"`
	Properties  []Property      `json:"properties"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedBy   string          `json:"updated_by"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Property represents a database column/field.
type Property struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Type   PropertyType `json:"type"`
	Config interface{}  `json:"config,omitempty"`
}

// SelectConfig holds configuration for select properties.
type SelectConfig struct {
	Options []SelectOption `json:"options"`
}

// SelectOption represents an option in a select property.
type SelectOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// NumberConfig holds configuration for number properties.
type NumberConfig struct {
	Format string `json:"format"` // number, percent, dollar, euro, etc.
}

// FormulaConfig holds configuration for formula properties.
type FormulaConfig struct {
	Expression string `json:"expression"`
}

// RelationConfig holds configuration for relation properties.
type RelationConfig struct {
	DatabaseID string `json:"database_id"`
	Type       string `json:"type"` // single_property, dual_property
}

// RollupConfig holds configuration for rollup properties.
type RollupConfig struct {
	RelationPropertyID string `json:"relation_property_id"`
	RollupPropertyID   string `json:"rollup_property_id"`
	Function           string `json:"function"` // count, sum, average, etc.
}

// StatusConfig holds configuration for status properties.
type StatusConfig struct {
	Options []StatusOption `json:"options"`
	Groups  []StatusGroup  `json:"groups"`
}

// StatusOption represents a status option.
type StatusOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// StatusGroup groups status options.
type StatusGroup struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Color     string   `json:"color"`
	OptionIDs []string `json:"option_ids"`
}

// CreateIn contains input for creating a database.
type CreateIn struct {
	WorkspaceID string     `json:"workspace_id"`
	PageID      string     `json:"page_id"`
	Title       string     `json:"title"`
	Icon        string     `json:"icon,omitempty"`
	IsInline    bool       `json:"is_inline"`
	Properties  []Property `json:"properties,omitempty"`
	CreatedBy   string     `json:"-"`
}

// UpdateIn contains input for updating a database.
type UpdateIn struct {
	Title *string `json:"title,omitempty"`
	Icon  *string `json:"icon,omitempty"`
	Cover *string `json:"cover,omitempty"`
}

// API defines the databases service contract.
type API interface {
	// CRUD
	Create(ctx context.Context, in *CreateIn) (*Database, error)
	GetByID(ctx context.Context, id string) (*Database, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Database, error)
	Delete(ctx context.Context, id string) error

	// List
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Database, error)
	ListByPage(ctx context.Context, pageID string) ([]*Database, error)

	// Properties
	AddProperty(ctx context.Context, dbID string, prop Property) (*Database, error)
	UpdateProperty(ctx context.Context, dbID, propID string, prop Property) error
	DeleteProperty(ctx context.Context, dbID, propID string) error
	ReorderProperties(ctx context.Context, dbID string, propIDs []string) error

	// Duplicate
	Duplicate(ctx context.Context, id string, targetPageID string, userID string) (*Database, error)
}

// Store defines the data access contract for databases.
type Store interface {
	Create(ctx context.Context, db *Database) error
	GetByID(ctx context.Context, id string) (*Database, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Database, error)
	ListByPage(ctx context.Context, pageID string) ([]*Database, error)
	UpdateProperties(ctx context.Context, id string, props []Property) error
}
