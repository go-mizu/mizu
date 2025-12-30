// Package query provides query enhancement utilities for collections.
package query

import (
	"context"

	"github.com/go-mizu/blueprints/cms/config"
)

// PopulateOptions holds options for relationship population.
type PopulateOptions struct {
	Depth          int             // Max depth to populate (0 = no population)
	Select         map[string]bool // Fields to include
	Populate       map[string]bool // Relationships to populate
	Locale         string          // Locale for localized fields
	FallbackLocale string          // Fallback locale
}

// SelectOptions holds options for field selection.
type SelectOptions struct {
	Include map[string]bool // Fields to include (true = include)
	Exclude map[string]bool // Fields to exclude (true = exclude)
}

// Populator defines the relationship population interface.
type Populator interface {
	// Populate populates relationships in a document.
	Populate(ctx context.Context, doc map[string]any, fields []config.Field, opts *PopulateOptions) (map[string]any, error)

	// PopulateField populates a specific relationship field.
	PopulateField(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error)

	// PopulateDocs populates relationships in multiple documents.
	PopulateDocs(ctx context.Context, docs []map[string]any, fields []config.Field, opts *PopulateOptions) ([]map[string]any, error)
}

// Selector defines the field selection interface.
type Selector interface {
	// ApplySelect filters document fields based on selection options.
	ApplySelect(doc map[string]any, opts *SelectOptions) map[string]any

	// ApplySelectDocs filters fields for multiple documents.
	ApplySelectDocs(docs []map[string]any, opts *SelectOptions) []map[string]any

	// BuildSelectColumns builds SQL column list from selection options.
	BuildSelectColumns(fields []config.Field, opts *SelectOptions) []string
}

// WhereBuilder defines the WHERE clause building interface.
type WhereBuilder interface {
	// Build builds a WHERE clause from a query object.
	// Supports nested AND/OR operations.
	Build(where map[string]any) (string, []any)
}
