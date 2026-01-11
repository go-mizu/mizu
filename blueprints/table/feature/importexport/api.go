// Package importexport provides base import/export functionality.
package importexport

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
)

// Version is the current schema version for exports.
const Version = "1.0"

// Errors
var (
	ErrInvalidMeta   = errors.New("invalid or corrupt metadata")
	ErrMissingData   = errors.New("CSV data file not found")
	ErrFieldMismatch = errors.New("CSV columns don't match field names")
	ErrInvalidValue  = errors.New("value doesn't match field type")
	ErrDirNotExist   = errors.New("directory does not exist")
	ErrBaseNotFound  = errors.New("base not found")
)

// Meta contains all metadata for a base export/import.
type Meta struct {
	Version    string      `json:"version"`
	ExportedAt time.Time   `json:"exported_at"`
	Base       bases.Base  `json:"base"`
	Tables     []TableMeta `json:"tables"`
}

// TableMeta contains metadata for a single table.
type TableMeta struct {
	Table   tables.Table                      `json:"table"`
	Fields  []*fields.Field                   `json:"fields"`
	Choices map[string][]*fields.SelectChoice `json:"choices"` // fieldID -> choices
	Views   []*views.View                     `json:"views"`
}

// API defines the import/export service interface.
type API interface {
	// Export exports a base to a directory.
	// Creates base.json and data/*.csv files.
	Export(ctx context.Context, baseID, dir string) error

	// Import imports a base from a directory.
	// Creates base, tables, fields, views, and records.
	// Returns the created base.
	Import(ctx context.Context, workspaceID, userID, dir string) (*bases.Base, error)

	// ExportMeta exports only the metadata (no data).
	ExportMeta(ctx context.Context, baseID string) (*Meta, error)

	// ImportMeta imports only the metadata (no data).
	ImportMeta(ctx context.Context, workspaceID, userID string, meta *Meta) (*bases.Base, error)
}
