// Package drivers provides database driver abstractions for the BI tool.
// It supports multiple database backends including SQLite, PostgreSQL, and MySQL.
package drivers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Config holds database connection configuration.
type Config struct {
	Engine   string            `json:"engine"`
	Host     string            `json:"host,omitempty"`
	Port     int               `json:"port,omitempty"`
	Database string            `json:"database"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	SSL      bool              `json:"ssl,omitempty"`
	Options  map[string]string `json:"options,omitempty"`

	// SSL/TLS Configuration
	SSLMode       string `json:"ssl_mode,omitempty"`        // disable, allow, prefer, require, verify-ca, verify-full
	SSLRootCert   string `json:"ssl_root_cert,omitempty"`   // PEM certificate content or path
	SSLClientCert string `json:"ssl_client_cert,omitempty"` // Client certificate
	SSLClientKey  string `json:"ssl_client_key,omitempty"`  // Client private key

	// SSH Tunnel Configuration
	Tunnel *TunnelConfig `json:"tunnel,omitempty"`

	// Connection Pool Configuration
	MaxOpenConns    int           `json:"max_open_conns,omitempty"`
	MaxIdleConns    int           `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime,omitempty"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time,omitempty"`

	// Schema Filtering
	SchemaFilterType     string   `json:"schema_filter_type,omitempty"` // inclusion, exclusion
	SchemaFilterPatterns []string `json:"schema_filter_patterns,omitempty"`
}

// TunnelConfig holds SSH tunnel configuration.
type TunnelConfig struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"` // default 22
	User       string `json:"user"`
	AuthMethod string `json:"auth_method"` // password, ssh-key
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// DriverCapabilities describes what features a driver supports.
type DriverCapabilities struct {
	SupportsSSH          bool          `json:"supports_ssh"`
	SupportsSSL          bool          `json:"supports_ssl"`
	SupportsSchemas      bool          `json:"supports_schemas"`
	SupportsCTEs         bool          `json:"supports_ctes"`
	SupportsJSON         bool          `json:"supports_json"`
	SupportsArrays       bool          `json:"supports_arrays"`
	SupportsWindowFuncs  bool          `json:"supports_window_funcs"`
	SupportsTransactions bool          `json:"supports_transactions"`
	MaxQueryTimeout      time.Duration `json:"max_query_timeout"`
	DefaultPort          int           `json:"default_port"`
}

// Table represents a database table.
type Table struct {
	Schema      string `json:"schema,omitempty"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "table", "view", "materialized_view"
	RowCount    int64  `json:"row_count,omitempty"`
	Description string `json:"description,omitempty"`
}

// Column represents a table column.
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`           // Original database type
	MappedType   string `json:"mapped_type"`    // Normalized type: string, number, boolean, datetime
	Nullable     bool   `json:"nullable"`
	PrimaryKey   bool   `json:"primary_key"`
	ForeignKey   bool   `json:"foreign_key"`
	DefaultValue string `json:"default_value,omitempty"`
	Description  string `json:"description,omitempty"`
	Position     int    `json:"position"`
}

// QueryResult represents the result of a query execution.
type QueryResult struct {
	Columns    []ResultColumn   `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	RowCount   int64            `json:"row_count"`
	Duration   float64          `json:"duration_ms"`
	Truncated  bool             `json:"truncated"`
	Cached     bool             `json:"cached"`
	NativeSQL  string           `json:"native_sql,omitempty"`
}

// ResultColumn represents a column in query results.
type ResultColumn struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	MappedType  string `json:"mapped_type"`
}

// Driver defines the interface for database drivers.
type Driver interface {
	// Name returns the driver name (e.g., "sqlite", "postgres").
	Name() string

	// Open opens a database connection.
	Open(ctx context.Context, config Config) error

	// Close closes the database connection.
	Close() error

	// Ping verifies the connection is alive.
	Ping(ctx context.Context) error

	// DB returns the underlying sql.DB for advanced operations.
	DB() *sql.DB

	// Schema Discovery

	// ListSchemas returns available schemas (for databases that support them).
	ListSchemas(ctx context.Context) ([]string, error)

	// ListTables returns tables in the database or schema.
	ListTables(ctx context.Context, schema string) ([]Table, error)

	// ListColumns returns columns for a table.
	ListColumns(ctx context.Context, schema, table string) ([]Column, error)

	// Query Execution

	// Execute runs a query and returns results.
	Execute(ctx context.Context, query string, params ...any) (*QueryResult, error)

	// SQL Helpers

	// QuoteIdentifier quotes an identifier (table/column name) for safe use in SQL.
	QuoteIdentifier(s string) string

	// SupportsSchemas returns whether this driver supports database schemas.
	SupportsSchemas() bool

	// Capabilities returns driver capabilities.
	Capabilities() DriverCapabilities
}

// Registry holds registered drivers.
var registry = make(map[string]func() Driver)

// Register adds a driver to the registry.
func Register(name string, factory func() Driver) {
	registry[name] = factory
}

// Get returns a driver by name.
func Get(name string) (Driver, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
	return factory(), nil
}

// Open opens a connection using the appropriate driver.
func Open(ctx context.Context, config Config) (Driver, error) {
	driver, err := Get(config.Engine)
	if err != nil {
		return nil, err
	}

	if err := driver.Open(ctx, config); err != nil {
		return nil, fmt.Errorf("open %s: %w", config.Engine, err)
	}

	return driver, nil
}

// ListDrivers returns all registered driver names.
func ListDrivers() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// MapColumnType converts a database-specific type to a normalized type.
func MapColumnType(dbType string) string {
	// Normalize to uppercase for comparison
	upperType := strings.ToUpper(dbType)

	// Common type mappings
	switch upperType {
	// Integer types (PostgreSQL, MySQL, SQLite)
	case "INTEGER", "INT", "SMALLINT", "BIGINT", "TINYINT", "MEDIUMINT",
		"SERIAL", "BIGSERIAL", "SMALLSERIAL",
		"INT2", "INT4", "INT8", "SERIAL4", "SERIAL8",
		"INT UNSIGNED", "BIGINT UNSIGNED", "SMALLINT UNSIGNED",
		"TINYINT UNSIGNED", "MEDIUMINT UNSIGNED":
		return "number"

	// Floating point types
	case "REAL", "FLOAT", "DOUBLE", "DOUBLE PRECISION", "NUMERIC", "DECIMAL",
		"FLOAT4", "FLOAT8", "MONEY",
		"FLOAT UNSIGNED", "DOUBLE UNSIGNED":
		return "number"

	// Boolean types
	case "BOOLEAN", "BOOL", "BIT":
		return "boolean"

	// Date/time types
	case "DATE", "TIME", "DATETIME", "TIMESTAMP", "TIMESTAMPTZ",
		"TIMETZ", "INTERVAL", "YEAR":
		return "datetime"

	// Text types
	case "TEXT", "VARCHAR", "CHAR", "CHARACTER VARYING", "CHARACTER",
		"BPCHAR", "CITEXT", "UUID", "NAME",
		"TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"ENUM", "SET":
		return "string"

	// JSON types
	case "JSON", "JSONB":
		return "string"

	// Binary types
	case "BLOB", "BYTEA", "BINARY", "VARBINARY",
		"TINYBLOB", "MEDIUMBLOB", "LONGBLOB":
		return "string"
	}

	// Handle types with parameters like VARCHAR(255)
	if strings.HasPrefix(upperType, "VARCHAR") ||
		strings.HasPrefix(upperType, "CHAR") ||
		strings.HasPrefix(upperType, "TEXT") {
		return "string"
	}
	if strings.HasPrefix(upperType, "INT") ||
		strings.HasPrefix(upperType, "DECIMAL") ||
		strings.HasPrefix(upperType, "NUMERIC") ||
		strings.HasPrefix(upperType, "FLOAT") ||
		strings.HasPrefix(upperType, "DOUBLE") {
		return "number"
	}

	return "string"
}

// ExecuteWithTimeout wraps query execution with a timeout.
func ExecuteWithTimeout(ctx context.Context, driver Driver, query string, timeout time.Duration, params ...any) (*QueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return driver.Execute(ctx, query, params...)
}
