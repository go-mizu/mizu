package api

import (
	"database/sql"
	"fmt"
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
	Query        map[string]interface{} `json:"query"`
}

// NativeQueryRequest represents a native SQL query request.
type NativeQueryRequest struct {
	DataSourceID string        `json:"datasource_id"`
	Query        string        `json:"query"`
	Params       []interface{} `json:"params,omitempty"`
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
func executeQuery(ds *store.DataSource, query map[string]interface{}) (*store.QueryResult, error) {
	// Build SQL from structured query
	sqlQuery, err := buildSQLFromQuery(query)
	if err != nil {
		return nil, err
	}
	return executeNativeQuery(ds, sqlQuery, nil)
}

// executeNativeQuery executes a native SQL query.
func executeNativeQuery(ds *store.DataSource, query string, params []interface{}) (*store.QueryResult, error) {
	switch ds.Engine {
	case "sqlite":
		return executeSQLiteQuery(ds.Database, query, params)
	default:
		return nil, fmt.Errorf("unsupported engine: %s", ds.Engine)
	}
}

// executeSQLiteQuery executes a query against SQLite.
func executeSQLiteQuery(dbPath, query string, params []interface{}) (*store.QueryResult, error) {
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
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		result.Rows = append(result.Rows, row)
	}

	result.RowCount = int64(len(result.Rows))
	return result, nil
}

// buildSQLFromQuery builds SQL from a structured query.
func buildSQLFromQuery(query map[string]interface{}) (string, error) {
	// Simple query builder - in a real implementation this would be more sophisticated
	table, ok := query["table"].(string)
	if !ok {
		// Try to get SQL directly
		if sql, ok := query["sql"].(string); ok {
			return sql, nil
		}
		return "", fmt.Errorf("no table specified")
	}

	selectCols := "*"
	if cols, ok := query["columns"].([]interface{}); ok && len(cols) > 0 {
		selectCols = ""
		for i, col := range cols {
			if i > 0 {
				selectCols += ", "
			}
			selectCols += col.(string)
		}
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", selectCols, table)

	// Add WHERE clause
	if filters, ok := query["filters"].([]interface{}); ok && len(filters) > 0 {
		sql += " WHERE "
		for i, f := range filters {
			filter := f.(map[string]interface{})
			if i > 0 {
				sql += " AND "
			}
			col := filter["column"].(string)
			op := filter["operator"].(string)
			val := filter["value"]
			sql += fmt.Sprintf("%s %s '%v'", col, op, val)
		}
	}

	// Add GROUP BY
	if groupBy, ok := query["group_by"].([]interface{}); ok && len(groupBy) > 0 {
		sql += " GROUP BY "
		for i, col := range groupBy {
			if i > 0 {
				sql += ", "
			}
			sql += col.(string)
		}
	}

	// Add ORDER BY
	if orderBy, ok := query["order_by"].([]interface{}); ok && len(orderBy) > 0 {
		sql += " ORDER BY "
		for i, o := range orderBy {
			order := o.(map[string]interface{})
			if i > 0 {
				sql += ", "
			}
			sql += order["column"].(string)
			if dir, ok := order["direction"].(string); ok {
				sql += " " + dir
			}
		}
	}

	// Add LIMIT
	if limit, ok := query["limit"].(float64); ok {
		sql += fmt.Sprintf(" LIMIT %d", int(limit))
	}

	return sql, nil
}
