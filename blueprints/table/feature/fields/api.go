// Package fields provides field (column) management functionality.
package fields

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound      = errors.New("field not found")
	ErrChoiceNotFound = errors.New("choice not found")
	ErrInvalidType   = errors.New("invalid field type")
)

// Field types - Airtable compatible
const (
	TypeSingleLineText    = "single_line_text"
	TypeLongText          = "long_text"
	TypeRichText          = "rich_text"
	TypeNumber            = "number"
	TypeCurrency          = "currency"
	TypePercent           = "percent"
	TypeDuration          = "duration"
	TypeRating            = "rating"
	TypeSingleSelect      = "single_select"
	TypeMultiSelect       = "multi_select"
	TypeCheckbox          = "checkbox"
	TypeDate              = "date"
	TypeDateTime          = "date_time"
	TypeCreatedTime       = "created_time"
	TypeLastModifiedTime  = "last_modified_time"
	TypeLink              = "link"
	TypeLookup            = "lookup"
	TypeRollup            = "rollup"
	TypeCount             = "count"
	TypeCollaborator      = "collaborator"
	TypeCollaborators     = "collaborators"
	TypeCreatedBy         = "created_by"
	TypeLastModifiedBy    = "last_modified_by"
	TypeAttachment        = "attachment"
	TypeBarcode           = "barcode"
	TypeAutoNumber        = "auto_number"
	TypeFormula           = "formula"
	TypeButton            = "button"
	TypeEmail             = "email"
	TypeURL               = "url"
	TypePhone             = "phone"
)

// Field represents a column in a table.
type Field struct {
	ID          string          `json:"id"`
	TableID     string          `json:"table_id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
	Position    int             `json:"position"`
	IsPrimary   bool            `json:"is_primary"`
	IsComputed  bool            `json:"is_computed"`
	IsHidden    bool            `json:"is_hidden"`
	Width       int             `json:"width"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// SelectChoice represents a choice for single/multi select fields.
type SelectChoice struct {
	ID       string `json:"id"`
	FieldID  string `json:"field_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

// CreateIn contains input for creating a field.
type CreateIn struct {
	TableID     string          `json:"table_id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
}

// UpdateIn contains input for updating a field.
type UpdateIn struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Options     *json.RawMessage `json:"options,omitempty"`
	Width       *int             `json:"width,omitempty"`
	IsHidden    *bool            `json:"is_hidden,omitempty"`
}

// UpdateChoiceIn contains input for updating a select choice.
type UpdateChoiceIn struct {
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
}

// API defines the fields service interface.
type API interface {
	Create(ctx context.Context, userID string, in CreateIn) (*Field, error)
	GetByID(ctx context.Context, id string) (*Field, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Field, error)
	Delete(ctx context.Context, id string) error
	ListByTable(ctx context.Context, tableID string) ([]*Field, error)
	Reorder(ctx context.Context, tableID string, fieldIDs []string) error

	// Select options
	AddSelectChoice(ctx context.Context, fieldID string, choice *SelectChoice) error
	UpdateSelectChoice(ctx context.Context, fieldID, choiceID string, in UpdateChoiceIn) error
	DeleteSelectChoice(ctx context.Context, fieldID, choiceID string) error
	ListSelectChoices(ctx context.Context, fieldID string) ([]*SelectChoice, error)
}

// Store defines the fields data access interface.
type Store interface {
	Create(ctx context.Context, field *Field) error
	GetByID(ctx context.Context, id string) (*Field, error)
	Update(ctx context.Context, field *Field) error
	Delete(ctx context.Context, id string) error
	ListByTable(ctx context.Context, tableID string) ([]*Field, error)
	Reorder(ctx context.Context, tableID string, fieldIDs []string) error

	// Select options
	AddSelectChoice(ctx context.Context, choice *SelectChoice) error
	UpdateSelectChoice(ctx context.Context, choiceID string, in UpdateChoiceIn) error
	DeleteSelectChoice(ctx context.Context, choiceID string) error
	ListSelectChoices(ctx context.Context, fieldID string) ([]*SelectChoice, error)
	GetSelectChoice(ctx context.Context, choiceID string) (*SelectChoice, error)
}
