// Package query provides query execution and management.
package query

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNoDataSource   = errors.New("no data source specified")
	ErrNoTable        = errors.New("no table specified")
	ErrInvalidQuery   = errors.New("invalid query")
	ErrQueryFailed    = errors.New("query execution failed")
	ErrInvalidColumn  = errors.New("invalid column name")
	ErrInvalidFilter  = errors.New("invalid filter")
	ErrDirectSQLDeny  = errors.New("direct SQL queries not allowed")
	ErrTimeout        = errors.New("query timeout")
)

// ResultColumn represents a column in query results.
type ResultColumn struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// QueryResult represents the result of a query execution.
type QueryResult struct {
	Columns  []ResultColumn   `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	RowCount int64            `json:"row_count"`
	Duration float64          `json:"duration_ms"`
	Cached   bool             `json:"cached"`
}

// StructuredQuery represents a structured query from the query builder.
type StructuredQuery struct {
	DataSourceID string         `json:"datasource_id"`
	Table        string         `json:"table"`
	Columns      []string       `json:"columns,omitempty"`
	Filters      []Filter       `json:"filters,omitempty"`
	GroupBy      []string       `json:"group_by,omitempty"`
	OrderBy      []OrderBy      `json:"order_by,omitempty"`
	Limit        int            `json:"limit,omitempty"`
	Joins        []Join         `json:"joins,omitempty"`
	Aggregations []Aggregation  `json:"aggregations,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
}

// Filter represents a query filter condition.
type Filter struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// OrderBy represents a sort specification.
type OrderBy struct {
	Column    string `json:"column"`
	Direction string `json:"direction"` // ASC or DESC
}

// Join represents a table join.
type Join struct {
	Table       string `json:"table"`
	Type        string `json:"type"` // INNER, LEFT, RIGHT
	LeftColumn  string `json:"left_column"`
	RightColumn string `json:"right_column"`
}

// Aggregation represents an aggregation specification.
type Aggregation struct {
	Function string `json:"function"` // COUNT, SUM, AVG, MIN, MAX
	Column   string `json:"column"`
	Alias    string `json:"alias,omitempty"`
}

// NativeQuery represents a native SQL query.
type NativeQuery struct {
	DataSourceID string         `json:"datasource_id"`
	Query        string         `json:"query"`
	Params       []any          `json:"params,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
}

// QueryHistory represents a query execution history entry.
type QueryHistory struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	DataSourceID string    `json:"datasource_id"`
	Query        string    `json:"query"`
	Duration     float64   `json:"duration_ms"`
	RowCount     int64     `json:"row_count"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Expression represents a custom expression/calculation.
type Expression struct {
	Formula string         `json:"formula"`
	Alias   string         `json:"alias"`
	Params  map[string]any `json:"params,omitempty"`
}

// RelativeDateFilter represents a relative date filter.
type RelativeDateFilter struct {
	Column   string `json:"column"`
	Unit     string `json:"unit"`    // day, week, month, quarter, year
	Value    int    `json:"value"`   // -1 = last, -7 = last 7
	Included bool   `json:"include"` // include current period
}

// ExecuteIn contains input for executing a structured query.
type ExecuteIn struct {
	Query   StructuredQuery `json:"query"`
	UserID  string          `json:"-"`
	Timeout time.Duration   `json:"-"`
}

// ExecuteNativeIn contains input for executing a native query.
type ExecuteNativeIn struct {
	Query   NativeQuery   `json:"query"`
	UserID  string        `json:"-"`
	Timeout time.Duration `json:"-"`
}

// HistoryListOpts specifies options for listing query history.
type HistoryListOpts struct {
	UserID string
	Limit  int
	Offset int
}

// API defines the Query service contract.
type API interface {
	// Execute executes a structured query.
	Execute(ctx context.Context, in *ExecuteIn) (*QueryResult, error)

	// ExecuteNative executes a native SQL query.
	ExecuteNative(ctx context.Context, in *ExecuteNativeIn) (*QueryResult, error)

	// ValidateQuery validates a structured query without executing.
	ValidateQuery(ctx context.Context, query *StructuredQuery) error

	// BuildSQL builds SQL from a structured query (for preview).
	BuildSQL(ctx context.Context, query *StructuredQuery) (string, []any, error)

	// ListHistory returns query history for a user.
	ListHistory(ctx context.Context, opts HistoryListOpts) ([]*QueryHistory, error)

	// GetCachedResult returns a cached result if available.
	GetCachedResult(ctx context.Context, queryHash string) (*QueryResult, error)
}

// DataSourceStore defines data access for data sources.
type DataSourceStore interface {
	GetByID(ctx context.Context, id string) (*DataSource, error)
}

// DataSource represents a data source connection.
type DataSource struct {
	ID       string `json:"id"`
	Engine   string `json:"engine"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"-"`
	SSL      bool   `json:"ssl"`
}

// HistoryStore defines data access for query history.
type HistoryStore interface {
	Create(ctx context.Context, h *QueryHistory) error
	List(ctx context.Context, userID string, limit int) ([]*QueryHistory, error)
}

// CacheStore defines data access for query cache.
type CacheStore interface {
	Get(ctx context.Context, key string) (*QueryResult, error)
	Set(ctx context.Context, key string, result *QueryResult, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Driver defines the database driver interface.
type Driver interface {
	Execute(ctx context.Context, query string, params ...any) (*QueryResult, error)
	Close() error
}

// DriverFactory creates drivers for data sources.
type DriverFactory interface {
	Open(ctx context.Context, ds *DataSource) (Driver, error)
}
