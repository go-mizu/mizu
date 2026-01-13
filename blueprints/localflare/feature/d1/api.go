// Package d1 provides D1 database management.
package d1

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("database not found")
	ErrNameRequired = errors.New("name is required")
)

// Database represents a D1 database.
type Database struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	NumTables int       `json:"num_tables"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// QueryResult represents query results.
type QueryResult struct {
	Results []map[string]interface{} `json:"results"`
	Success bool                     `json:"success"`
	Meta    *QueryMeta               `json:"meta,omitempty"`
}

// QueryMeta contains query metadata.
type QueryMeta struct {
	ChangedDB   bool  `json:"changed_db"`
	Changes     int64 `json:"changes"`
	Duration    int64 `json:"duration_ms"`
	LastRowID   int64 `json:"last_row_id"`
	RowsRead    int64 `json:"rows_read"`
	RowsWritten int64 `json:"rows_written"`
}

// CreateDatabaseIn contains input for creating a database.
type CreateDatabaseIn struct {
	Name string `json:"name"`
}

// QueryIn contains input for running a query.
type QueryIn struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params,omitempty"`
}

// API defines the D1 service contract.
type API interface {
	CreateDatabase(ctx context.Context, in *CreateDatabaseIn) (*Database, error)
	GetDatabase(ctx context.Context, id string) (*Database, error)
	ListDatabases(ctx context.Context) ([]*Database, error)
	DeleteDatabase(ctx context.Context, id string) error
	Query(ctx context.Context, dbID string, in *QueryIn) (*QueryResult, error)
}

// Store defines the data access contract.
type Store interface {
	CreateDatabase(ctx context.Context, db *Database) error
	GetDatabase(ctx context.Context, id string) (*Database, error)
	ListDatabases(ctx context.Context) ([]*Database, error)
	DeleteDatabase(ctx context.Context, id string) error
	Query(ctx context.Context, dbID, sql string, params []interface{}) ([]map[string]interface{}, error)
	Exec(ctx context.Context, dbID, sql string, params []interface{}) (int64, error)
}
