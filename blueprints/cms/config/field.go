package config

import "encoding/json"

// FieldType represents the type of a field.
type FieldType string

const (
	// Data fields
	FieldTypeText         FieldType = "text"
	FieldTypeTextarea     FieldType = "textarea"
	FieldTypeEmail        FieldType = "email"
	FieldTypeNumber       FieldType = "number"
	FieldTypeCheckbox     FieldType = "checkbox"
	FieldTypeDate         FieldType = "date"
	FieldTypeCode         FieldType = "code"
	FieldTypeJSON         FieldType = "json"
	FieldTypeRichText     FieldType = "richText"
	FieldTypeSelect       FieldType = "select"
	FieldTypeRadio        FieldType = "radio"
	FieldTypePoint        FieldType = "point"
	FieldTypeRelationship FieldType = "relationship"
	FieldTypeUpload       FieldType = "upload"
	FieldTypeArray        FieldType = "array"
	FieldTypeBlocks       FieldType = "blocks"
	FieldTypeGroup        FieldType = "group"
	FieldTypeTabs         FieldType = "tabs"

	// Presentational fields
	FieldTypeRow         FieldType = "row"
	FieldTypeCollapsible FieldType = "collapsible"
	FieldTypeUI          FieldType = "ui"

	// Virtual fields
	FieldTypeJoin    FieldType = "join"
	FieldTypeVirtual FieldType = "virtual"
)

// Field defines a field in a collection or global.
type Field struct {
	Type     FieldType
	Name     string
	Label    string
	Required bool
	Unique   bool
	Index    bool
	Localized bool
	DefaultValue any

	// Validation
	Validate  ValidateFn
	MinLength *int
	MaxLength *int
	Min       *float64
	Max       *float64

	// Select/Radio options
	Options  []SelectOption
	HasMany  bool // For select with multiple values

	// Relationship
	RelationTo []string // Collection slugs
	FilterOptions map[string]any

	// Upload
	MimeTypes []string

	// Array/Blocks/Group
	Fields    []Field
	Blocks    []Block
	MinRows   *int
	MaxRows   *int

	// Tabs
	Tabs []Tab

	// Admin UI
	Admin *FieldAdmin

	// Hooks
	Hooks *FieldHooks

	// Access Control
	Access *FieldAccess

	// Conditional logic
	Condition ConditionFn

	// Virtual field path
	Virtual string
}

// SelectOption represents an option for select/radio fields.
type SelectOption struct {
	Label string
	Value string
}

// Block represents a block type for blocks fields.
type Block struct {
	Slug       string
	Labels     Labels
	Fields     []Field
	ImageURL   string
	ImageAltText string
}

// Tab represents a tab in a tabs field.
type Tab struct {
	Name        string // Named tab (stores data)
	Label       string
	Description string
	Fields      []Field
}

// FieldAdmin holds field-specific admin configuration.
type FieldAdmin struct {
	Position    string // "sidebar" or empty for main
	Width       string // "50%", "100%", etc.
	Description string
	Placeholder string
	Hidden      bool
	ReadOnly    bool
	DisableBulkEdit bool
	Condition   ConditionFn
	Style       map[string]string
	ClassName   string
}

// FieldHooks defines field-level hooks.
type FieldHooks struct {
	BeforeValidate []FieldHookFn
	BeforeChange   []FieldHookFn
	AfterChange    []FieldHookFn
	AfterRead      []FieldHookFn
}

// FieldHookFn is a field hook function.
type FieldHookFn func(ctx *FieldHookContext) (any, error)

// FieldHookContext provides context for field hooks.
type FieldHookContext struct {
	Value        any
	OriginalDoc  map[string]any
	Data         map[string]any
	SiblingData  map[string]any
	Field        *Field
	Collection   string
	Operation    string
	Req          any // *http.Request
	User         map[string]any
}

// FieldAccess defines field-level access control.
type FieldAccess struct {
	Create AccessFn
	Read   AccessFn
	Update AccessFn
}

// ValidateFn is a validation function.
type ValidateFn func(value any, ctx *ValidationContext) error

// ValidationContext provides context for validation.
type ValidationContext struct {
	Data        map[string]any
	SiblingData map[string]any
	Operation   string
	User        map[string]any
}

// ConditionFn determines if a field should be shown.
type ConditionFn func(data map[string]any, siblingData map[string]any) bool

// IsDataField returns true if the field stores data.
func (f *Field) IsDataField() bool {
	switch f.Type {
	case FieldTypeRow, FieldTypeCollapsible, FieldTypeUI:
		return false
	case FieldTypeTabs:
		// Named tabs store data, unnamed don't
		return f.Name != ""
	default:
		return true
	}
}

// IsRelational returns true if the field references other documents.
func (f *Field) IsRelational() bool {
	return f.Type == FieldTypeRelationship || f.Type == FieldTypeUpload
}

// HasNestedFields returns true if the field contains nested fields.
func (f *Field) HasNestedFields() bool {
	return f.Type == FieldTypeArray || f.Type == FieldTypeBlocks ||
		f.Type == FieldTypeGroup || f.Type == FieldTypeTabs ||
		f.Type == FieldTypeRow || f.Type == FieldTypeCollapsible
}

// GetNestedFields returns all nested fields.
func (f *Field) GetNestedFields() []Field {
	switch f.Type {
	case FieldTypeArray, FieldTypeGroup, FieldTypeRow, FieldTypeCollapsible:
		return f.Fields
	case FieldTypeBlocks:
		var fields []Field
		for _, block := range f.Blocks {
			fields = append(fields, block.Fields...)
		}
		return fields
	case FieldTypeTabs:
		var fields []Field
		for _, tab := range f.Tabs {
			fields = append(fields, tab.Fields...)
		}
		return fields
	default:
		return nil
	}
}

// SQLType returns the SQL type for this field.
func (f *Field) SQLType() string {
	switch f.Type {
	case FieldTypeText, FieldTypeTextarea, FieldTypeEmail, FieldTypeCode:
		return "TEXT"
	case FieldTypeNumber:
		return "DOUBLE"
	case FieldTypeCheckbox:
		return "BOOLEAN"
	case FieldTypeDate:
		return "TIMESTAMP"
	case FieldTypeJSON, FieldTypeRichText, FieldTypeArray, FieldTypeBlocks, FieldTypeGroup:
		return "TEXT" // JSON stored as text
	case FieldTypeSelect:
		if f.HasMany {
			return "TEXT" // JSON array
		}
		return "VARCHAR(255)"
	case FieldTypeRadio:
		return "VARCHAR(255)"
	case FieldTypePoint:
		return "TEXT" // JSON [lng, lat]
	case FieldTypeRelationship, FieldTypeUpload:
		if f.HasMany {
			return "TEXT" // JSON array of IDs
		}
		return "VARCHAR(26)" // ULID
	default:
		return "TEXT"
	}
}

// MarshalJSON implements custom JSON marshaling.
func (f Field) MarshalJSON() ([]byte, error) {
	type Alias Field
	return json.Marshal(struct {
		Alias
		Type string `json:"type"`
	}{
		Alias: Alias(f),
		Type:  string(f.Type),
	})
}
