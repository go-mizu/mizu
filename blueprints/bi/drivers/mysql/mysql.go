// Package mysql provides a MySQL/MariaDB database driver.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-mizu/blueprints/bi/drivers"
)

func init() {
	drivers.Register("mysql", func() drivers.Driver {
		return &Driver{}
	})
	// Also register as mariadb for compatibility
	drivers.Register("mariadb", func() drivers.Driver {
		return &Driver{}
	})
}

// Driver implements the MySQL/MariaDB database driver.
type Driver struct {
	db     *sql.DB
	config drivers.Config
}

// Name returns "mysql".
func (d *Driver) Name() string {
	return "mysql"
}

// Open opens a MySQL database connection.
func (d *Driver) Open(ctx context.Context, config drivers.Config) error {
	d.config = config

	// Build connection string in MySQL DSN format
	dsn := d.buildDSN(config)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}

	// Configure connection pool
	maxOpen := config.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := config.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	maxLifetime := config.ConnMaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = 5 * time.Minute
	}
	maxIdleTime := config.ConnMaxIdleTime
	if maxIdleTime <= 0 {
		maxIdleTime = time.Minute
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
	db.SetConnMaxIdleTime(maxIdleTime)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping mysql: %w", err)
	}

	d.db = db
	return nil
}

// buildDSN builds a MySQL connection string (DSN format).
// Format: [user[:password]@][net[(addr)]]/dbname[?param1=value1&paramN=valueN]
func (d *Driver) buildDSN(config drivers.Config) string {
	var dsn strings.Builder

	// User and password
	if config.Username != "" {
		dsn.WriteString(config.Username)
		if config.Password != "" {
			dsn.WriteString(":")
			dsn.WriteString(config.Password)
		}
		dsn.WriteString("@")
	}

	// Protocol and address
	host := config.Host
	if host == "" {
		host = "localhost"
	}
	port := config.Port
	if port <= 0 {
		port = 3306
	}
	dsn.WriteString(fmt.Sprintf("tcp(%s:%d)", host, port))

	// Database name
	dsn.WriteString("/")
	if config.Database != "" {
		dsn.WriteString(config.Database)
	}

	// Parameters
	params := make([]string, 0)

	// Always use utf8mb4 charset
	params = append(params, "charset=utf8mb4")

	// Parse time values to time.Time
	params = append(params, "parseTime=true")

	// Set location to UTC
	params = append(params, "loc=UTC")

	// SSL/TLS
	if config.SSL {
		sslMode := config.SSLMode
		if sslMode == "" {
			sslMode = "required"
		}
		// MySQL uses tls parameter: true, false, skip-verify, preferred, or custom config name
		switch sslMode {
		case "disable":
			params = append(params, "tls=false")
		case "allow", "prefer":
			params = append(params, "tls=preferred")
		case "require":
			params = append(params, "tls=true")
		case "verify-ca", "verify-full":
			params = append(params, "tls=skip-verify") // Would need custom TLS config for full verification
		default:
			params = append(params, "tls=true")
		}
	} else {
		params = append(params, "tls=false")
	}

	// Add custom options
	for k, v := range config.Options {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}

	if len(params) > 0 {
		dsn.WriteString("?")
		dsn.WriteString(strings.Join(params, "&"))
	}

	return dsn.String()
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

// ListSchemas returns all databases (MySQL treats databases as schemas).
func (d *Driver) ListSchemas(ctx context.Context) ([]string, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	// MySQL uses databases as schemas
	query := `
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME
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

// ListTables returns all tables in a database/schema.
func (d *Driver) ListTables(ctx context.Context, schema string) ([]drivers.Table, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not opened")
	}

	// Use current database if schema not specified
	if schema == "" {
		schema = d.config.Database
	}

	query := `
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			TABLE_TYPE,
			COALESCE(TABLE_COMMENT, '') as description,
			COALESCE(TABLE_ROWS, 0) as row_count
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		AND TABLE_TYPE IN ('BASE TABLE', 'VIEW')
		ORDER BY TABLE_NAME
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
		if err := rows.Scan(&t.Schema, &t.Name, &tableType, &t.Description, &t.RowCount); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}

		// Map MySQL table types
		switch tableType {
		case "BASE TABLE":
			t.Type = "table"
		case "VIEW":
			t.Type = "view"
		default:
			t.Type = strings.ToLower(tableType)
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

	// Use current database if schema not specified
	if schema == "" {
		schema = d.config.Database
	}

	query := `
		SELECT
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_TYPE,
			IS_NULLABLE = 'YES' as nullable,
			COLUMN_DEFAULT,
			COALESCE(COLUMN_COMMENT, '') as description,
			ORDINAL_POSITION - 1 as position,
			COLUMN_KEY = 'PRI' as is_primary
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := d.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()

	var columns []drivers.Column
	for rows.Next() {
		var col drivers.Column
		var dataType, columnType string
		var dfltValue sql.NullString

		if err := rows.Scan(&col.Name, &dataType, &columnType, &col.Nullable, &dfltValue, &col.Description, &col.Position, &col.PrimaryKey); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}

		// Use data_type for type mapping
		col.Type = columnType
		col.MappedType = drivers.MapColumnType(dataType)

		if dfltValue.Valid {
			col.DefaultValue = dfltValue.String
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get foreign key information
	fkQuery := `
		SELECT COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ?
		AND TABLE_NAME = ?
		AND REFERENCED_TABLE_NAME IS NOT NULL
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

// convertValue converts MySQL-specific types to JSON-serializable values.
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

// QuoteIdentifier quotes an identifier for MySQL.
func (d *Driver) QuoteIdentifier(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

// SupportsSchemas returns true for MySQL (databases act as schemas).
func (d *Driver) SupportsSchemas() bool {
	return true
}

// Capabilities returns MySQL driver capabilities.
func (d *Driver) Capabilities() drivers.DriverCapabilities {
	return drivers.DriverCapabilities{
		SupportsSSH:          true,
		SupportsSSL:          true,
		SupportsSchemas:      true,
		SupportsCTEs:         true, // MySQL 8.0+
		SupportsJSON:         true, // MySQL 5.7+
		SupportsArrays:       false,
		SupportsWindowFuncs:  true, // MySQL 8.0+
		SupportsTransactions: true,
		MaxQueryTimeout:      time.Hour,
		DefaultPort:          3306,
	}
}

// Version returns the MySQL server version.
func (d *Driver) Version(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not opened")
	}

	var version string
	err := d.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	return version, err
}

// CurrentDatabase returns the current database name.
func (d *Driver) CurrentDatabase(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not opened")
	}

	var name sql.NullString
	err := d.db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&name)
	if err != nil {
		return "", err
	}
	return name.String, nil
}
