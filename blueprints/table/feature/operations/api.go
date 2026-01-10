// Package operations provides operation logging for history and undo/redo.
package operations

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/table/feature/records"
)

// Errors
var (
	ErrNotFound = errors.New("operation not found")
)

// Operation types
const (
	OpCreateRecord   = "create_record"
	OpUpdateRecord   = "update_record"
	OpDeleteRecord   = "delete_record"
	OpUpdateCell     = "update_cell"
	OpCreateField    = "create_field"
	OpUpdateField    = "update_field"
	OpDeleteField    = "delete_field"
	OpCreateView     = "create_view"
	OpUpdateView     = "update_view"
	OpDeleteView     = "delete_view"
	OpReorderRecords = "reorder_records"
	OpReorderFields  = "reorder_fields"
)

// Operation represents a single operation for history tracking.
type Operation struct {
	ID        string          `json:"id"`
	TableID   string          `json:"table_id,omitempty"`
	RecordID  string          `json:"record_id,omitempty"`
	FieldID   string          `json:"field_id,omitempty"`
	ViewID    string          `json:"view_id,omitempty"`
	OpType    string          `json:"op_type"`
	OldValue  json.RawMessage `json:"old_value,omitempty"`
	NewValue  json.RawMessage `json:"new_value,omitempty"`
	UserID    string          `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
}

// ListOpts contains options for listing operations.
type ListOpts struct {
	Since  time.Time
	Until  time.Time
	Limit  int
	Cursor string
}

// API defines the operations service interface.
type API interface {
	// Recording
	Record(ctx context.Context, op *Operation) error
	RecordBatch(ctx context.Context, ops []*Operation) error

	// Querying
	GetByID(ctx context.Context, id string) (*Operation, error)
	ListByTable(ctx context.Context, tableID string, opts ListOpts) ([]*Operation, error)
	ListByRecord(ctx context.Context, recordID string, opts ListOpts) ([]*Operation, error)
	ListByUser(ctx context.Context, userID string, opts ListOpts) ([]*Operation, error)

	// Time travel
	GetRecordAtTime(ctx context.Context, recordID string, t time.Time) (*records.Record, error)
	GetTableAtTime(ctx context.Context, tableID string, t time.Time) ([]*records.Record, error)

	// Undo/Redo
	Undo(ctx context.Context, opID string) (*Operation, error)
	Redo(ctx context.Context, opID string) (*Operation, error)
}

// Store defines the operations data access interface.
type Store interface {
	Create(ctx context.Context, op *Operation) error
	CreateBatch(ctx context.Context, ops []*Operation) error
	GetByID(ctx context.Context, id string) (*Operation, error)
	ListByTable(ctx context.Context, tableID string, opts ListOpts) ([]*Operation, error)
	ListByRecord(ctx context.Context, recordID string, opts ListOpts) ([]*Operation, error)
	ListByUser(ctx context.Context, userID string, opts ListOpts) ([]*Operation, error)
}
