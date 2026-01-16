package api

import (
	"fmt"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// DatabaseHandler handles database endpoints.
type DatabaseHandler struct {
	store *postgres.Store
}

// NewDatabaseHandler creates a new database handler.
func NewDatabaseHandler(store *postgres.Store) *DatabaseHandler {
	return &DatabaseHandler{store: store}
}

// ListTables lists all tables.
func (h *DatabaseHandler) ListTables(c *mizu.Ctx) error {
	schema := c.Query("schema")
	if schema == "" {
		schema = "public"
	}

	tables, err := h.store.Database().ListTables(c.Context(), schema)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list tables"})
	}

	return c.JSON(200, tables)
}

// GetTable gets a table with columns.
func (h *DatabaseHandler) GetTable(c *mizu.Ctx) error {
	schema := c.Param("schema")
	name := c.Param("name")

	table, err := h.store.Database().GetTable(c.Context(), schema, name)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "table not found"})
	}

	return c.JSON(200, table)
}

// CreateTable creates a new table.
func (h *DatabaseHandler) CreateTable(c *mizu.Ctx) error {
	var req struct {
		Schema  string          `json:"schema"`
		Name    string          `json:"name"`
		Columns []*store.Column `json:"columns"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}

	if len(req.Columns) == 0 {
		return c.JSON(400, map[string]string{"error": "at least one column required"})
	}

	schema := req.Schema
	if schema == "" {
		schema = "public"
	}

	if err := h.store.Database().CreateTable(c.Context(), schema, req.Name, req.Columns); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create table: " + err.Error()})
	}

	return c.JSON(201, map[string]string{"message": "table created"})
}

// DropTable drops a table.
func (h *DatabaseHandler) DropTable(c *mizu.Ctx) error {
	schema := c.Param("schema")
	name := c.Param("name")

	if err := h.store.Database().DropTable(c.Context(), schema, name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop table"})
	}

	return c.NoContent()
}

// ListColumns lists columns in a table.
func (h *DatabaseHandler) ListColumns(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")

	columns, err := h.store.Database().ListColumns(c.Context(), schema, table)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list columns"})
	}

	return c.JSON(200, columns)
}

// AddColumn adds a column to a table.
func (h *DatabaseHandler) AddColumn(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")

	var column store.Column
	if err := c.BindJSON(&column, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.Database().AddColumn(c.Context(), schema, table, &column); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to add column: " + err.Error()})
	}

	return c.JSON(201, map[string]string{"message": "column added"})
}

// AlterColumn alters a column.
func (h *DatabaseHandler) AlterColumn(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")
	columnName := c.Param("column")

	var column store.Column
	if err := c.BindJSON(&column, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	column.Name = columnName

	if err := h.store.Database().AlterColumn(c.Context(), schema, table, &column); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to alter column: " + err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "column altered"})
}

// DropColumn drops a column.
func (h *DatabaseHandler) DropColumn(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")
	column := c.Param("column")

	if err := h.store.Database().DropColumn(c.Context(), schema, table, column); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop column"})
	}

	return c.NoContent()
}

// ListSchemas lists all schemas.
func (h *DatabaseHandler) ListSchemas(c *mizu.Ctx) error {
	schemas, err := h.store.Database().ListSchemas(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list schemas"})
	}

	return c.JSON(200, schemas)
}

// CreateSchema creates a new schema.
func (h *DatabaseHandler) CreateSchema(c *mizu.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.Database().CreateSchema(c.Context(), req.Name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create schema"})
	}

	return c.JSON(201, map[string]string{"message": "schema created"})
}

// ListExtensions lists all extensions.
func (h *DatabaseHandler) ListExtensions(c *mizu.Ctx) error {
	extensions, err := h.store.Database().ListExtensions(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list extensions"})
	}

	return c.JSON(200, extensions)
}

// EnableExtension enables an extension.
func (h *DatabaseHandler) EnableExtension(c *mizu.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.Database().EnableExtension(c.Context(), req.Name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to enable extension: " + err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "extension enabled"})
}

// ListPolicies lists RLS policies.
func (h *DatabaseHandler) ListPolicies(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("table")

	policies, err := h.store.Database().ListPolicies(c.Context(), schema, table)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list policies"})
	}

	return c.JSON(200, policies)
}

// CreatePolicy creates an RLS policy.
func (h *DatabaseHandler) CreatePolicy(c *mizu.Ctx) error {
	var policy store.Policy
	if err := c.BindJSON(&policy, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if err := h.store.Database().CreatePolicy(c.Context(), &policy); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create policy: " + err.Error()})
	}

	return c.JSON(201, map[string]string{"message": "policy created"})
}

// DropPolicy drops an RLS policy.
func (h *DatabaseHandler) DropPolicy(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("table")
	name := c.Param("name")

	if err := h.store.Database().DropPolicy(c.Context(), schema, table, name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop policy"})
	}

	return c.NoContent()
}

// ExecuteQuery executes a SQL query.
func (h *DatabaseHandler) ExecuteQuery(c *mizu.Ctx) error {
	var req struct {
		Query  string        `json:"query"`
		Params []interface{} `json:"params"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	// Check if it's a SELECT query
	trimmed := strings.TrimSpace(strings.ToUpper(req.Query))
	if strings.HasPrefix(trimmed, "SELECT") || strings.HasPrefix(trimmed, "WITH") {
		result, err := h.store.Database().Query(c.Context(), req.Query, req.Params...)
		if err != nil {
			return c.JSON(400, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, result)
	}

	// Execute non-SELECT query
	rows, err := h.store.Database().Exec(c.Context(), req.Query, req.Params...)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"rows_affected": rows,
	})
}

// REST API handlers (PostgREST compatible)

// SelectTable handles GET requests for PostgREST.
func (h *DatabaseHandler) SelectTable(c *mizu.Ctx) error {
	table := c.Param("table")
	schema := c.Query("schema")
	if schema == "" {
		schema = "public"
	}

	// Build SELECT query from query params
	selectCols := c.Query("select")
	if selectCols == "" {
		selectCols = "*"
	}
	orderBy := c.Query("order")
	limit := queryInt(c, "limit", 100)
	offset := queryInt(c, "offset", 0)

	sql := fmt.Sprintf("SELECT %s FROM %s.%s", selectCols, quoteIdent(schema), quoteIdent(table))

	// Handle filters
	where := []string{}
	for key, values := range c.QueryValues() {
		if key == "select" || key == "order" || key == "limit" || key == "offset" || key == "schema" {
			continue
		}
		for _, value := range values {
			// Parse filter operator
			op, val := parseFilter(value)
			where = append(where, fmt.Sprintf("%s %s '%s'", quoteIdent(key), op, val))
		}
	}

	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}

	if orderBy != "" {
		sql += " ORDER BY " + orderBy
	}

	sql += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	result, err := h.store.Database().Query(c.Context(), sql)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result.Rows)
}

// InsertTable handles POST requests for PostgREST.
func (h *DatabaseHandler) InsertTable(c *mizu.Ctx) error {
	table := c.Param("table")
	schema := c.Query("schema")
	if schema == "" {
		schema = "public"
	}

	// Parse body as generic interface to handle both array and single object
	var body interface{}
	if err := c.BindJSON(&body, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	var rows []map[string]interface{}
	switch v := body.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, m)
			}
		}
	case map[string]interface{}:
		rows = []map[string]interface{}{v}
	default:
		return c.JSON(400, map[string]string{"error": "invalid request body format"})
	}

	if len(rows) == 0 {
		return c.JSON(400, map[string]string{"error": "no data to insert"})
	}

	// Build INSERT query
	columns := []string{}
	for col := range rows[0] {
		columns = append(columns, quoteIdent(col))
	}

	var valueRows []string
	var params []interface{}
	paramIdx := 1

	for _, row := range rows {
		var placeholders []string
		for _, col := range columns {
			colName := strings.Trim(col, `"`)
			params = append(params, row[colName])
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramIdx))
			paramIdx++
		}
		valueRows = append(valueRows, "("+strings.Join(placeholders, ", ")+")")
	}

	sql := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES %s RETURNING *",
		quoteIdent(schema),
		quoteIdent(table),
		strings.Join(columns, ", "),
		strings.Join(valueRows, ", "),
	)

	result, err := h.store.Database().Query(c.Context(), sql, params...)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	if len(result.Rows) == 1 {
		return c.JSON(201, result.Rows[0])
	}
	return c.JSON(201, result.Rows)
}

// UpdateTable handles PATCH requests for PostgREST.
func (h *DatabaseHandler) UpdateTable(c *mizu.Ctx) error {
	table := c.Param("table")
	schema := c.Query("schema")
	if schema == "" {
		schema = "public"
	}

	var data map[string]interface{}
	if err := c.BindJSON(&data, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if len(data) == 0 {
		return c.JSON(400, map[string]string{"error": "no data to update"})
	}

	// Build UPDATE query
	var setClauses []string
	var params []interface{}
	paramIdx := 1

	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdent(col), paramIdx))
		params = append(params, val)
		paramIdx++
	}

	sql := fmt.Sprintf("UPDATE %s.%s SET %s",
		quoteIdent(schema),
		quoteIdent(table),
		strings.Join(setClauses, ", "),
	)

	// Handle filters from query params
	where := []string{}
	for key, values := range c.QueryValues() {
		if key == "schema" {
			continue
		}
		for _, value := range values {
			op, val := parseFilter(value)
			where = append(where, fmt.Sprintf("%s %s '%s'", quoteIdent(key), op, val))
		}
	}

	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}

	sql += " RETURNING *"

	result, err := h.store.Database().Query(c.Context(), sql, params...)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result.Rows)
}

// DeleteTable handles DELETE requests for PostgREST.
func (h *DatabaseHandler) DeleteTable(c *mizu.Ctx) error {
	table := c.Param("table")
	schema := c.Query("schema")
	if schema == "" {
		schema = "public"
	}

	sql := fmt.Sprintf("DELETE FROM %s.%s", quoteIdent(schema), quoteIdent(table))

	// Handle filters from query params
	where := []string{}
	for key, values := range c.QueryValues() {
		if key == "schema" {
			continue
		}
		for _, value := range values {
			op, val := parseFilter(value)
			where = append(where, fmt.Sprintf("%s %s '%s'", quoteIdent(key), op, val))
		}
	}

	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}

	sql += " RETURNING *"

	result, err := h.store.Database().Query(c.Context(), sql)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result.Rows)
}

// CallFunction calls a database function (RPC).
func (h *DatabaseHandler) CallFunction(c *mizu.Ctx) error {
	fnName := c.Param("function")
	schema := c.Query("schema")
	if schema == "" {
		schema = "public"
	}

	var params map[string]interface{}
	if err := c.BindJSON(&params, 0); err != nil {
		params = make(map[string]interface{})
	}

	// Build function call
	var args []string
	var values []interface{}
	paramIdx := 1

	for name, val := range params {
		args = append(args, fmt.Sprintf("%s := $%d", name, paramIdx))
		values = append(values, val)
		paramIdx++
	}

	sql := fmt.Sprintf("SELECT * FROM %s.%s(%s)", quoteIdent(schema), quoteIdent(fnName), strings.Join(args, ", "))

	result, err := h.store.Database().Query(c.Context(), sql, values...)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result.Rows)
}

// Helper functions

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func parseFilter(value string) (string, string) {
	// PostgREST filter format: eq.value, gt.value, etc.
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "=", value
	}

	switch parts[0] {
	case "eq":
		return "=", parts[1]
	case "neq":
		return "!=", parts[1]
	case "gt":
		return ">", parts[1]
	case "gte":
		return ">=", parts[1]
	case "lt":
		return "<", parts[1]
	case "lte":
		return "<=", parts[1]
	case "like":
		return "LIKE", parts[1]
	case "ilike":
		return "ILIKE", parts[1]
	case "is":
		return "IS", parts[1]
	default:
		return "=", value
	}
}
