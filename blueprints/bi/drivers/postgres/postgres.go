// Package postgres provides a PostgreSQL database driver using pgx/v5.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-mizu/blueprints/bi/drivers"
)

func init() {
	drivers.Register("postgres", func() drivers.Driver {
		return &Driver{}
	})
	// Also register as postgresql for compatibility
	drivers.Register("postgresql", func() drivers.Driver {
		return &Driver{}
	})
}

// Driver implements the PostgreSQL database driver using pgx.
type Driver struct {
	db     *sql.DB
	config drivers.Config
}

// Name returns "postgres".
func (d *Driver) Name() string {
	return "postgres"
}

// Open opens a PostgreSQL database connection using pgx.
func (d *Driver) Open(ctx context.Context, config drivers.Config) error {
	d.config = config

	// Build connection string in pgx format
	dsn := d.buildDSN(config)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping postgres: %w", err)
	}

	d.db = db
	return nil
}

// buildDSN builds a PostgreSQL connection string (pgx format).
func (d *Driver) buildDSN(config drivers.Config) string {
	var parts []string

	host := config.Host
	if host == "" {
		host = "localhost"
	}
	parts = append(parts, fmt.Sprintf("host=%s", host))

	port := config.Port
	if port <= 0 {
		port = 5432
	}
	parts = append(parts, fmt.Sprintf("port=%d", port))

	if config.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", config.Database))
	}

	if config.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.Username))
	}

	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}

	if config.SSL {
		parts = append(parts, "sslmode=require")
	} else {
		parts = append(parts, "sslmode=disable")
	}

	// Add custom options
	for k, v := range config.Options {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(parts, " ")
}

// Close closes the database connection.
func (d *Driver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping verifies the connection is alive.
func (d *Driver) Ping(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database not opened")
	}
	return d.db.PingContext(ctx)
}

// DB returns the underlying sql.DB.
func (d *Driver) DB() *sql.DB {
	return d.db
}

// ListSchemas returns all schemas in the database.
func (d *Driver) ListSchemas(ctx context.Context) ([]string, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'pg_toast', 'information_schema')
		ORDER BY schema_name
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema: %w", err)
		}
		schemas = append(schemas, name)
	}

	return schemas, rows.Err()
}

// ListTables returns all tables in a schema.
func (d *Driver) ListTables(ctx context.Context, schema string) ([]drivers.Table, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT
			t.table_schema,
			t.table_name,
			t.table_type,
			COALESCE(pg_catalog.obj_description(c.oid, 'pg_class'), '') as description
		FROM information_schema.tables t
		LEFT JOIN pg_catalog.pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		WHERE t.table_schema = $1
		AND t.table_type IN ('BASE TABLE', 'VIEW')
		ORDER BY t.table_name
	`

	rows, err := d.db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []drivers.Table
	for rows.Next() {
		var t drivers.Table
		var tableType string
		if err := rows.Scan(&t.Schema, &t.Name, &tableType, &t.Description); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}

		// Map PostgreSQL table types
		switch tableType {
		case "BASE TABLE":
			t.Type = "table"
		case "VIEW":
			t.Type = "view"
		default:
			t.Type = strings.ToLower(tableType)
		}

		// Get row count estimate for tables (fast method using pg_class)
		if t.Type == "table" {
			countQuery := `
				SELECT reltuples::bigint
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relname = $1 AND n.nspname = $2
			`
			var count int64
			if err := d.db.QueryRowContext(ctx, countQuery, t.Name, schema).Scan(&count); err == nil && count >= 0 {
				t.RowCount = count
			}
		}

		tables = append(tables, t)
	}

	return tables, rows.Err()
}

// ListColumns returns columns for a table.
func (d *Driver) ListColumns(ctx context.Context, schema, table string) ([]drivers.Column, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.udt_name,
			c.is_nullable = 'YES' as nullable,
			c.column_default,
			COALESCE(pg_catalog.col_description(
				(SELECT oid FROM pg_class WHERE relname = c.table_name AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = c.table_schema)),
				c.ordinal_position
			), '') as description,
			c.ordinal_position - 1 as position
		FROM information_schema.columns c
		WHERE c.table_schema = $1
		AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := d.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()

	var columns []drivers.Column
	for rows.Next() {
		var col drivers.Column
		var dataType, udtName string
		var dfltValue sql.NullString

		if err := rows.Scan(&col.Name, &dataType, &udtName, &col.Nullable, &dfltValue, &col.Description, &col.Position); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}

		// Use udt_name for more accurate type mapping
		col.Type = udtName
		col.MappedType = drivers.MapColumnType(udtName)

		if dfltValue.Valid {
			col.DefaultValue = dfltValue.String
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get primary key information
	pkQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = $1
		AND tc.table_name = $2
		AND tc.constraint_type = 'PRIMARY KEY'
	`

	pkRows, err := d.db.QueryContext(ctx, pkQuery, schema, table)
	if err == nil {
		defer pkRows.Close()
		pkColumns := make(map[string]bool)
		for pkRows.Next() {
			var colName string
			if err := pkRows.Scan(&colName); err == nil {
				pkColumns[colName] = true
			}
		}

		for i := range columns {
			if pkColumns[columns[i].Name] {
				columns[i].PrimaryKey = true
			}
		}
	}

	// Get foreign key information
	fkQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = $1
		AND tc.table_name = $2
		AND tc.constraint_type = 'FOREIGN KEY'
	`

	fkRows, err := d.db.QueryContext(ctx, fkQuery, schema, table)
	if err == nil {
		defer fkRows.Close()
		fkColumns := make(map[string]bool)
		for fkRows.Next() {
			var colName string
			if err := fkRows.Scan(&colName); err == nil {
				fkColumns[colName] = true
			}
		}

		for i := range columns {
			if fkColumns[columns[i].Name] {
				columns[i].ForeignKey = true
			}
		}
	}

	return columns, nil
}

// Execute runs a query and returns results.
func (d *Driver) Execute(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	start := time.Now()

	rows, err := d.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column info
	colNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("get columns: %w", err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("get column types: %w", err)
	}

	result := &drivers.QueryResult{
		Columns:   make([]drivers.ResultColumn, len(colNames)),
		NativeSQL: query,
	}

	for i, name := range colNames {
		dbType := ""
		if colTypes[i] != nil {
			dbType = colTypes[i].DatabaseTypeName()
		}
		result.Columns[i] = drivers.ResultColumn{
			Name:        name,
			DisplayName: name,
			Type:        dbType,
			MappedType:  drivers.MapColumnType(dbType),
		}
	}

	// Scan rows
	const maxRows = 10000
	for rows.Next() {
		if len(result.Rows) >= maxRows {
			result.Truncated = true
			break
		}

		values := make([]any, len(colNames))
		valuePtrs := make([]any, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		row := make(map[string]any)
		for i, name := range colNames {
			row[name] = d.convertValue(values[i])
		}
		result.Rows = append(result.Rows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	result.RowCount = int64(len(result.Rows))
	result.Duration = float64(time.Since(start).Microseconds()) / 1000

	return result, nil
}

// convertValue converts PostgreSQL-specific types to JSON-serializable values.
func (d *Driver) convertValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return v
	}
}

// QuoteIdentifier quotes an identifier for PostgreSQL.
func (d *Driver) QuoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// SupportsSchemas returns true for PostgreSQL.
func (d *Driver) SupportsSchemas() bool {
	return true
}

// Version returns the PostgreSQL server version.
func (d *Driver) Version(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not opened")
	}

	var version string
	err := d.db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	return version, err
}

// CurrentDatabase returns the current database name.
func (d *Driver) CurrentDatabase(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not opened")
	}

	var name string
	err := d.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name)
	return name, err
}

// CurrentSchema returns the current schema (search_path).
func (d *Driver) CurrentSchema(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not opened")
	}

	var name string
	err := d.db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&name)
	return name, err
}
