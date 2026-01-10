// Package records provides record (row) management functionality.
package records

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound = errors.New("record not found")
)

// Record represents a row in a table.
type Record struct {
	ID        string                 `json:"id"`
	TableID   string                 `json:"table_id"`
	Cells     map[string]interface{} `json:"cells"`
	Position  int                    `json:"position"`
	CreatedBy string                 `json:"created_by"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	UpdatedBy string                 `json:"updated_by,omitempty"`
}

// RecordLink represents a link between two records.
type RecordLink struct {
	ID             string `json:"id"`
	SourceRecordID string `json:"source_record_id"`
	SourceFieldID  string `json:"source_field_id"`
	TargetRecordID string `json:"target_record_id"`
	Position       int    `json:"position"`
}

// ListOpts contains options for listing records.
type ListOpts struct {
	ViewID     string     // Apply view's filters/sorts
	FilterBy   string     // Filter formula
	SortBy     []SortSpec // Sort specification
	Fields     []string   // Only return these fields
	Offset     int        // Pagination offset
	Limit      int        // Max records to return
	CellFormat string     // "json" or "string"
}

// SortSpec defines a sort specification.
type SortSpec struct {
	FieldID   string `json:"field_id"`
	Direction string `json:"direction"` // asc, desc
}

// RecordList represents a paginated list of records.
type RecordList struct {
	Records []*Record `json:"records"`
	Offset  int       `json:"offset"`
	Total   int       `json:"total"`
}

// RecordUpdate contains an update for a single record.
type RecordUpdate struct {
	ID    string                 `json:"id"`
	Cells map[string]interface{} `json:"cells"`
}

// API defines the records service interface.
type API interface {
	Create(ctx context.Context, tableID string, cells map[string]interface{}, userID string) (*Record, error)
	CreateBatch(ctx context.Context, tableID string, records []map[string]interface{}, userID string) ([]*Record, error)
	GetByID(ctx context.Context, id string) (*Record, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Record, error)
	Update(ctx context.Context, id string, cells map[string]interface{}, userID string) (*Record, error)
	UpdateBatch(ctx context.Context, updates []RecordUpdate, userID string) ([]*Record, error)
	Delete(ctx context.Context, id string) error
	DeleteBatch(ctx context.Context, ids []string) error

	// Queries
	List(ctx context.Context, tableID string, opts ListOpts) (*RecordList, error)
	Search(ctx context.Context, tableID, query string, opts ListOpts) (*RecordList, error)

	// Cell operations
	UpdateCell(ctx context.Context, recordID, fieldID string, value interface{}, userID string) error
	ClearCell(ctx context.Context, recordID, fieldID string, userID string) error

	// Bulk operations
	UpdateFieldValues(ctx context.Context, tableID, fieldID string, updates map[string]interface{}, userID string) error
}

// Store defines the records data access interface.
type Store interface {
	Create(ctx context.Context, record *Record) error
	CreateBatch(ctx context.Context, records []*Record) error
	GetByID(ctx context.Context, id string) (*Record, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Record, error)
	Update(ctx context.Context, record *Record) error
	Delete(ctx context.Context, id string) error
	DeleteBatch(ctx context.Context, ids []string) error
	List(ctx context.Context, tableID string, opts ListOpts) (*RecordList, error)

	// Cell operations
	UpdateCell(ctx context.Context, recordID, fieldID string, value interface{}) error
	ClearCell(ctx context.Context, recordID, fieldID string) error

	// Links
	CreateLink(ctx context.Context, link *RecordLink) error
	DeleteLink(ctx context.Context, id string) error
	DeleteLinksBySource(ctx context.Context, recordID, fieldID string) error
	ListLinksBySource(ctx context.Context, recordID, fieldID string) ([]*RecordLink, error)
	ListLinksByTarget(ctx context.Context, targetRecordID string) ([]*RecordLink, error)
}
