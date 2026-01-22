// Package sqlite provides a SQLite database driver.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-mizu/blueprints/bi/drivers"
)

func init() {
	drivers.Register("sqlite", func() drivers.Driver {
		return &Driver{}
	})
}

// Driver implements the SQLite database driver.
type Driver struct {
	db     *sql.DB
	config drivers.Config
}

// Name returns "sqlite".
func (d *Driver) Name() string {
	return "sqlite"
}

// Open opens a SQLite database connection.
func (d *Driver) Open(ctx context.Context, config drivers.Config) error {
	d.config = config

	// SQLite uses the Database field as the file path
	dsn := config.Database
	if dsn == "" {
		return fmt.Errorf("database path is required for SQLite")
	}

	// Add connection options
	if len(config.Options) > 0 {
		params := make([]string, 0, len(config.Options))
		for k, v := range config.Options {
			params = append(params, fmt.Sprintf("_%s=%s", k, v))
		}
		dsn += "?" + strings.Join(params, "&")
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	// Configure connection pool (SQLite is single-writer, so keep it small)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Enable foreign keys and WAL mode for better performance
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		// WAL might not be available, ignore error
	}

	d.db = db
	return nil
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

// ListSchemas returns schemas (SQLite doesn't support schemas).
func (d *Driver) ListSchemas(ctx context.Context) ([]string, error) {
	// SQLite doesn't have schemas, return main and temp
	return []string{"main", "temp"}, nil
}

// ListTables returns all tables in the database.
func (d *Driver) ListTables(ctx context.Context, schema string) ([]drivers.Table, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	query := `
		SELECT name, type
		FROM sqlite_master
		WHERE type IN ('table', 'view')
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []drivers.Table
	for rows.Next() {
		var t drivers.Table
		if err := rows.Scan(&t.Name, &t.Type); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}

		// Get row count for tables (not views)
		if t.Type == "table" {
			var count int64
			// Using fmt.Sprintf is safe here because t.Name comes from sqlite_master
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", d.QuoteIdentifier(t.Name))
			if err := d.db.QueryRowContext(ctx, countQuery).Scan(&count); err == nil {
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

	query := fmt.Sprintf("PRAGMA table_info(%s)", d.QuoteIdentifier(table))
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("table info: %w", err)
	}
	defer rows.Close()

	var columns []drivers.Column
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}

		col := drivers.Column{
			Name:       name,
			Type:       colType,
			MappedType: drivers.MapColumnType(colType),
			Nullable:   notNull == 0,
			PrimaryKey: primaryKey > 0,
			Position:   cid,
		}

		if dfltValue.Valid {
			col.DefaultValue = dfltValue.String
		}

		columns = append(columns, col)
	}

	// Check for foreign keys
	fkQuery := fmt.Sprintf("PRAGMA foreign_key_list(%s)", d.QuoteIdentifier(table))
	fkRows, err := d.db.QueryContext(ctx, fkQuery)
	if err == nil {
		defer fkRows.Close()
		fkColumns := make(map[string]bool)
		for fkRows.Next() {
			var (
				id, seq           int
				refTable, from    string
				to, onUpdate      string
				onDelete, match   string
			)
			if err := fkRows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err == nil {
				fkColumns[from] = true
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
			// Convert []byte to string for better JSON serialization
			if b, ok := values[i].([]byte); ok {
				row[name] = string(b)
			} else {
				row[name] = values[i]
			}
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

// QuoteIdentifier quotes an identifier for SQLite.
func (d *Driver) QuoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// SupportsSchemas returns false for SQLite.
func (d *Driver) SupportsSchemas() bool {
	return false
}

// Version returns the SQLite version.
func (d *Driver) Version(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not opened")
	}

	var version string
	err := d.db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	return version, err
}
