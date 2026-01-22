package api

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/bi/store"
	"github.com/go-mizu/blueprints/bi/store/sqlite"
)

// Query handles query execution API endpoints.
type Query struct {
	store *sqlite.Store
}

// NewQuery creates a new Query handler.
func NewQuery(store *sqlite.Store) *Query {
	return &Query{store: store}
}

// ExecuteRequest represents a query execution request.
type ExecuteRequest struct {
	DataSourceID string                 `json:"datasource_id"`
	Query        map[string]any `json:"query"`
}

// NativeQueryRequest represents a native SQL query request.
type NativeQueryRequest struct {
	DataSourceID string `json:"datasource_id"`
	Query        string `json:"query"`
	Params       []any  `json:"params,omitempty"`
}

// Execute executes a structured query.
func (h *Query) Execute(c *mizu.Ctx) error {
	var req ExecuteRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), req.DataSourceID)
	if err != nil || ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	result, err := executeQuery(ds, req.Query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result)
}

// ExecuteNative executes a native SQL query.
func (h *Query) ExecuteNative(c *mizu.Ctx) error {
	var req NativeQueryRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	ds, err := h.store.DataSources().GetByID(c.Request().Context(), req.DataSourceID)
	if err != nil || ds == nil {
		return c.JSON(404, map[string]string{"error": "Data source not found"})
	}

	start := time.Now()
	result, err := executeNativeQuery(ds, req.Query, req.Params)
	duration := time.Since(start).Seconds() * 1000

	// Record in history
	qh := &store.QueryHistory{
		UserID:       "anonymous", // TODO: Get from session
		DataSourceID: req.DataSourceID,
		Query:        req.Query,
		Duration:     duration,
		RowCount:     int64(len(result.Rows)),
	}
	if err != nil {
		qh.Error = err.Error()
	}
	h.store.QueryHistory().Create(c.Request().Context(), qh)

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result.Duration = duration
	return c.JSON(200, result)
}

// History returns query history for the current user.
func (h *Query) History(c *mizu.Ctx) error {
	userID := "anonymous" // TODO: Get from session
	history, err := h.store.QueryHistory().List(c.Request().Context(), userID, 50)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, history)
}

// executeQuery executes a structured query and returns results.
func executeQuery(ds *store.DataSource, query map[string]any) (*store.QueryResult, error) {
	// Build SQL from structured query with parameterized values
	sqlQuery, params, err := buildSQLFromQuery(query)
	if err != nil {
		return nil, err
	}
	return executeNativeQuery(ds, sqlQuery, params)
}

// executeNativeQuery executes a native SQL query.
func executeNativeQuery(ds *store.DataSource, query string, params []any) (*store.QueryResult, error) {
	switch ds.Engine {
	case "sqlite":
		return executeSQLiteQuery(ds.Database, query, params)
	default:
		return nil, fmt.Errorf("unsupported engine: %s", ds.Engine)
	}
}

// executeSQLiteQuery executes a query against SQLite.
func executeSQLiteQuery(dbPath, query string, params []any) (*store.QueryResult, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &store.QueryResult{
		Columns: make([]store.ResultColumn, len(columns)),
	}
	for i, col := range columns {
		result.Columns[i] = store.ResultColumn{
			Name:        col,
			DisplayName: col,
			Type:        "string",
		}
	}

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		result.Rows = append(result.Rows, row)
	}

	result.RowCount = int64(len(result.Rows))
	return result, nil
}

// identifierRegex validates SQL identifiers (table names, column names)
// Only allows alphanumeric characters, underscores, and dots (for schema.table)
var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

// validateIdentifier checks if a string is a valid SQL identifier
func validateIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(s) > 128 {
		return fmt.Errorf("identifier too long: %s", s)
	}
	if !identifierRegex.MatchString(s) {
		return fmt.Errorf("invalid identifier: %s", s)
	}
	// Check for SQL keywords that could be used for injection
	upper := strings.ToUpper(s)
	forbidden := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "EXEC", "EXECUTE", "UNION", "SCRIPT"}
	for _, keyword := range forbidden {
		if upper == keyword {
			return fmt.Errorf("identifier cannot be a SQL keyword: %s", s)
		}
	}
	return nil
}

// quoteIdentifier safely quotes an identifier for SQL
func quoteIdentifier(s string) string {
	// Double any existing double quotes and wrap in double quotes
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// validateOperator checks if an operator is valid
func validateOperator(op string) error {
	validOps := map[string]bool{
		"=": true, "!=": true, "<>": true,
		">": true, ">=": true, "<": true, "<=": true,
		"LIKE": true, "like": true,
		"IN": true, "in": true,
		"NOT IN": true, "not in": true,
		"IS NULL": true, "is null": true,
		"IS NOT NULL": true, "is not null": true,
		"BETWEEN": true, "between": true,
	}
	if !validOps[op] {
		return fmt.Errorf("invalid operator: %s", op)
	}
	return nil
}

// buildSQLFromQuery builds SQL from a structured query using parameterized queries.
// Returns the SQL string, parameters slice, and any error.
func buildSQLFromQuery(query map[string]any) (string, []any, error) {
	var params []any
	var sqlBuilder strings.Builder

	// Handle direct SQL (only for trusted internal queries)
	if _, ok := query["sql"].(string); ok {
		// Direct SQL is dangerous - we should not support this for user queries
		// For now, return an error for direct SQL to prevent injection
		return "", nil, fmt.Errorf("direct SQL queries are not supported for security reasons")
	}

	// Get and validate table name
	table, ok := query["table"].(string)
	if !ok || table == "" {
		return "", nil, fmt.Errorf("no table specified")
	}
	if err := validateIdentifier(table); err != nil {
		return "", nil, fmt.Errorf("invalid table name: %w", err)
	}

	// Build SELECT clause
	sqlBuilder.WriteString("SELECT ")

	if cols, ok := query["columns"].([]any); ok && len(cols) > 0 {
		for i, col := range cols {
			colStr, ok := col.(string)
			if !ok {
				return "", nil, fmt.Errorf("column must be a string")
			}
			if err := validateIdentifier(colStr); err != nil {
				return "", nil, fmt.Errorf("invalid column name: %w", err)
			}
			if i > 0 {
				sqlBuilder.WriteString(", ")
			}
			sqlBuilder.WriteString(quoteIdentifier(colStr))
		}
	} else {
		sqlBuilder.WriteString("*")
	}

	// Add FROM clause
	sqlBuilder.WriteString(" FROM ")
	sqlBuilder.WriteString(quoteIdentifier(table))

	// Add WHERE clause with parameterized values
	if filters, ok := query["filters"].([]any); ok && len(filters) > 0 {
		sqlBuilder.WriteString(" WHERE ")
		for i, f := range filters {
			filter, ok := f.(map[string]any)
			if !ok {
				return "", nil, fmt.Errorf("filter must be an object")
			}

			if i > 0 {
				sqlBuilder.WriteString(" AND ")
			}

			col, ok := filter["column"].(string)
			if !ok {
				return "", nil, fmt.Errorf("filter column must be a string")
			}
			if err := validateIdentifier(col); err != nil {
				return "", nil, fmt.Errorf("invalid filter column: %w", err)
			}

			op, ok := filter["operator"].(string)
			if !ok {
				op = "=" // default operator
			}
			if err := validateOperator(op); err != nil {
				return "", nil, err
			}

			val := filter["value"]

			// Handle special operators
			upperOp := strings.ToUpper(op)
			switch upperOp {
			case "IS NULL", "IS NOT NULL":
				sqlBuilder.WriteString(quoteIdentifier(col))
				sqlBuilder.WriteString(" ")
				sqlBuilder.WriteString(upperOp)
			case "IN", "NOT IN":
				sqlBuilder.WriteString(quoteIdentifier(col))
				sqlBuilder.WriteString(" ")
				sqlBuilder.WriteString(upperOp)
				sqlBuilder.WriteString(" (")
				if valSlice, ok := val.([]any); ok {
					for j, v := range valSlice {
						if j > 0 {
							sqlBuilder.WriteString(", ")
						}
						sqlBuilder.WriteString("?")
						params = append(params, v)
					}
				} else {
					sqlBuilder.WriteString("?")
					params = append(params, val)
				}
				sqlBuilder.WriteString(")")
			case "BETWEEN":
				if valSlice, ok := val.([]any); ok && len(valSlice) == 2 {
					sqlBuilder.WriteString(quoteIdentifier(col))
					sqlBuilder.WriteString(" BETWEEN ? AND ?")
					params = append(params, valSlice[0], valSlice[1])
				} else {
					return "", nil, fmt.Errorf("BETWEEN requires an array of two values")
				}
			default:
				// Standard comparison operators
				sqlBuilder.WriteString(quoteIdentifier(col))
				sqlBuilder.WriteString(" ")
				sqlBuilder.WriteString(op)
				sqlBuilder.WriteString(" ?")
				params = append(params, val)
			}
		}
	}

	// Add GROUP BY clause
	if groupBy, ok := query["group_by"].([]any); ok && len(groupBy) > 0 {
		sqlBuilder.WriteString(" GROUP BY ")
		for i, col := range groupBy {
			colStr, ok := col.(string)
			if !ok {
				return "", nil, fmt.Errorf("group_by column must be a string")
			}
			if err := validateIdentifier(colStr); err != nil {
				return "", nil, fmt.Errorf("invalid group_by column: %w", err)
			}
			if i > 0 {
				sqlBuilder.WriteString(", ")
			}
			sqlBuilder.WriteString(quoteIdentifier(colStr))
		}
	}

	// Add ORDER BY clause
	if orderBy, ok := query["order_by"].([]any); ok && len(orderBy) > 0 {
		sqlBuilder.WriteString(" ORDER BY ")
		for i, o := range orderBy {
			order, ok := o.(map[string]any)
			if !ok {
				return "", nil, fmt.Errorf("order_by item must be an object")
			}

			if i > 0 {
				sqlBuilder.WriteString(", ")
			}

			col, ok := order["column"].(string)
			if !ok {
				return "", nil, fmt.Errorf("order_by column must be a string")
			}
			if err := validateIdentifier(col); err != nil {
				return "", nil, fmt.Errorf("invalid order_by column: %w", err)
			}

			sqlBuilder.WriteString(quoteIdentifier(col))

			if dir, ok := order["direction"].(string); ok {
				dirUpper := strings.ToUpper(dir)
				if dirUpper != "ASC" && dirUpper != "DESC" {
					return "", nil, fmt.Errorf("invalid order direction: %s", dir)
				}
				sqlBuilder.WriteString(" ")
				sqlBuilder.WriteString(dirUpper)
			}
		}
	}

	// Add LIMIT clause (safely convert to integer)
	if limit, ok := query["limit"].(float64); ok {
		if limit > 0 && limit <= 10000 {
			sqlBuilder.WriteString(fmt.Sprintf(" LIMIT %d", int(limit)))
		} else if limit > 10000 {
			sqlBuilder.WriteString(" LIMIT 10000")
		}
	}

	return sqlBuilder.String(), params, nil
}
