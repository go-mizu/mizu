package api

import (
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/middleware"
	"github.com/go-mizu/mizu/blueprints/localbase/pkg/postgrest"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// DatabaseHandler handles database endpoints.
type DatabaseHandler struct {
	store     *postgres.Store
	pgHandler *postgrest.Handler
}

// NewDatabaseHandler creates a new database handler.
func NewDatabaseHandler(store *postgres.Store) *DatabaseHandler {
	return &DatabaseHandler{
		store:     store,
		pgHandler: postgrest.NewHandler(&dbQuerier{store: store}),
	}
}

// dbQuerier adapts the store to the postgrest.Querier interface.
// It supports RLS context propagation for Supabase-compatible row-level security.
type dbQuerier struct {
	store *postgres.Store
}

// QueryWithRLS executes a query with RLS context from JWT claims.
func (q *dbQuerier) QueryWithRLS(ctx context.Context, rlsCtx *postgres.RLSContext, sql string, args ...any) (*postgrest.QueryResult, error) {
	result, err := q.store.DatabaseRLS().QueryWithRLS(ctx, rlsCtx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &postgrest.QueryResult{
		Columns: result.Columns,
		Rows:    result.Rows,
	}, nil
}

// ExecWithRLS executes a statement with RLS context from JWT claims.
func (q *dbQuerier) ExecWithRLS(ctx context.Context, rlsCtx *postgres.RLSContext, sql string, args ...any) (int64, error) {
	return q.store.DatabaseRLS().ExecWithRLS(ctx, rlsCtx, sql, args...)
}

// Query implements the postgrest.Querier interface (without RLS context).
// This is used for backward compatibility and internal queries.
func (q *dbQuerier) Query(ctx context.Context, sql string, args ...any) (*postgrest.QueryResult, error) {
	result, err := q.store.Database().Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &postgrest.QueryResult{
		Columns: result.Columns,
		Rows:    result.Rows,
	}, nil
}

// Exec implements the postgrest.Querier interface (without RLS context).
func (q *dbQuerier) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	return q.store.Database().Exec(ctx, sql, args...)
}

func (q *dbQuerier) TableExists(ctx context.Context, schema, table string) (bool, error) {
	return q.store.Database().TableExists(ctx, schema, table)
}

func (q *dbQuerier) GetForeignKeys(ctx context.Context, schema, table string) ([]postgrest.ForeignKey, error) {
	fks, err := q.store.Database().GetForeignKeysForEmbedding(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	result := make([]postgrest.ForeignKey, 0, len(fks))
	for _, fk := range fks {
		result = append(result, postgrest.ForeignKey{
			ConstraintName: fk.ConstraintName,
			ColumnName:     fk.ColumnName,
			ForeignSchema:  fk.ForeignSchema,
			ForeignTable:   fk.ForeignTable,
			ForeignColumn:  fk.ForeignColumn,
		})
	}
	return result, nil
}

// getRLSContext extracts RLS context from the mizu.Ctx request headers.
// This is populated by the API key middleware from JWT claims.
func getRLSContext(c *mizu.Ctx) *postgres.RLSContext {
	return &postgres.RLSContext{
		Role:       middleware.GetRole(c),
		UserID:     middleware.GetUserID(c),
		Email:      middleware.GetUserEmail(c),
		ClaimsJSON: middleware.GetJWTClaimsJSON(c),
	}
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
		Query  string `json:"query"`
		Params []any  `json:"params"`
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

// ========================================
// REST API handlers (PostgREST compatible)
// ========================================

// SelectTable handles GET requests for PostgREST.
func (h *DatabaseHandler) SelectTable(c *mizu.Ctx) error {
	req, err := h.parseRequest(c)
	if err != nil {
		return h.sendError(c, err)
	}

	resp, err := h.pgHandler.Select(c.Context(), req)
	if err != nil {
		return h.sendError(c, err)
	}

	return h.sendResponse(c, resp)
}

// InsertTable handles POST requests for PostgREST.
func (h *DatabaseHandler) InsertTable(c *mizu.Ctx) error {
	req, err := h.parseRequest(c)
	if err != nil {
		return h.sendError(c, err)
	}

	// Parse body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return h.sendError(c, postgrest.ErrPGRST102("failed to read body"))
	}

	req.Body, err = postgrest.ParseRequestBody(body)
	if err != nil {
		return h.sendError(c, postgrest.ErrPGRST102(err.Error()))
	}

	// Check for upsert
	req.OnConflict = c.Query("on_conflict")

	resp, err := h.pgHandler.Insert(c.Context(), req)
	if err != nil {
		return h.sendError(c, err)
	}

	return h.sendResponse(c, resp)
}

// UpdateTable handles PATCH requests for PostgREST.
func (h *DatabaseHandler) UpdateTable(c *mizu.Ctx) error {
	req, err := h.parseRequest(c)
	if err != nil {
		return h.sendError(c, err)
	}

	// Parse body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return h.sendError(c, postgrest.ErrPGRST102("failed to read body"))
	}

	req.Body, err = postgrest.ParseRequestBody(body)
	if err != nil {
		return h.sendError(c, postgrest.ErrPGRST102(err.Error()))
	}

	resp, err := h.pgHandler.Update(c.Context(), req)
	if err != nil {
		return h.sendError(c, err)
	}

	return h.sendResponse(c, resp)
}

// DeleteTable handles DELETE requests for PostgREST.
func (h *DatabaseHandler) DeleteTable(c *mizu.Ctx) error {
	req, err := h.parseRequest(c)
	if err != nil {
		return h.sendError(c, err)
	}

	resp, err := h.pgHandler.Delete(c.Context(), req)
	if err != nil {
		return h.sendError(c, err)
	}

	return h.sendResponse(c, resp)
}

// CallFunction calls a database function (RPC).
func (h *DatabaseHandler) CallFunction(c *mizu.Ctx) error {
	fnName := c.Param("function")

	req, err := h.parseRequest(c)
	if err != nil {
		return h.sendError(c, err)
	}

	// Parse body for function parameters
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return h.sendError(c, postgrest.ErrPGRST102("failed to read body"))
	}

	if len(body) > 0 {
		req.Body, err = postgrest.ParseRequestBody(body)
		if err != nil {
			return h.sendError(c, postgrest.ErrPGRST102(err.Error()))
		}
	} else {
		req.Body = make(map[string]any)
	}

	resp, err := h.pgHandler.RPC(c.Context(), fnName, req)
	if err != nil {
		return h.sendError(c, err)
	}

	return h.sendResponse(c, resp)
}

// parseRequest parses an HTTP request into a PostgREST request.
func (h *DatabaseHandler) parseRequest(c *mizu.Ctx) (*postgrest.Request, error) {
	req := &postgrest.Request{
		Table:  c.Param("table"),
		Schema: c.Query("schema"),
		Prefs:  postgrest.ParsePrefer(c.Request().Header.Get("Prefer")),
	}

	if req.Schema == "" {
		req.Schema = "public"
	}

	// Parse select
	selectStr := c.Query("select")
	if selectStr != "" {
		cols, err := postgrest.ParseSelect(selectStr)
		if err != nil {
			return nil, postgrest.ErrPGRST100("invalid select: " + err.Error())
		}
		req.Select = cols
	}

	// Parse order
	orderStr := c.Query("order")
	if orderStr != "" {
		order, err := postgrest.ParseOrder(orderStr)
		if err != nil {
			return nil, postgrest.ErrPGRST100("invalid order: " + err.Error())
		}
		req.Order = order
	}

	// Parse limit/offset
	req.Limit = parseIntQueryDB(c, "limit", 100)
	req.Offset = parseIntQueryDB(c, "offset", 0)

	// Parse Range header
	rangeHeader := c.Request().Header.Get("Range")
	if rangeHeader != "" {
		start, end, hasRange := postgrest.ParseRange(rangeHeader)
		if hasRange {
			req.Offset = start
			req.HasRange = true
			if end > 0 {
				req.Limit = end - start + 1
			}
		}
	}

	// Parse filters from query parameters
	filters, embeddedFilters, logicalOp, err := h.parseFilters(c)
	if err != nil {
		return nil, err
	}
	req.Filters = filters
	req.EmbeddedFilters = embeddedFilters
	req.LogicalOp = logicalOp

	// Set RLS context from JWT claims (extracted by API key middleware)
	req.RLSContext = convertRLSContext(getRLSContext(c))

	return req, nil
}

// convertRLSContext converts postgres.RLSContext to postgrest.RLSContext
func convertRLSContext(ctx *postgres.RLSContext) *postgrest.RLSContext {
	if ctx == nil {
		return nil
	}
	return &postgrest.RLSContext{
		Role:       ctx.Role,
		UserID:     ctx.UserID,
		Email:      ctx.Email,
		ClaimsJSON: ctx.ClaimsJSON,
	}
}

// parseFilters parses filter parameters from the query string.
// Returns main filters, embedded filters (keyed by resource name), and logical operator.
func (h *DatabaseHandler) parseFilters(c *mizu.Ctx) ([]postgrest.Filter, map[string][]postgrest.Filter, string, error) {
	var filters []postgrest.Filter
	embeddedFilters := make(map[string][]postgrest.Filter)
	logicalOp := "AND"

	reservedParams := map[string]bool{
		"select": true, "order": true, "limit": true, "offset": true, "schema": true,
		"on_conflict": true, "columns": true,
	}

	for key, values := range c.QueryValues() {
		if reservedParams[key] {
			continue
		}

		// Check for logical operators
		if key == "and" || key == "or" {
			for _, value := range values {
				// ParseLogicalFilter expects "and(...)" or "or(...)" format
				// The URL gives us key="and" value="(...)" so we need to prepend the key
				fullValue := key + value
				op, logicalFilters, err := postgrest.ParseLogicalFilter(fullValue)
				if err == nil {
					logicalOp = op
					filters = append(filters, logicalFilters...)
				}
			}
			continue
		}

		// Check for embedded resource filter (e.g., posts.published)
		if dotIdx := strings.Index(key, "."); dotIdx > 0 {
			embeddedResource := key[:dotIdx]
			embeddedColumn := key[dotIdx+1:]

			for _, value := range values {
				filter, err := postgrest.ParseFilter(embeddedColumn, value)
				if err != nil {
					return nil, nil, "", postgrest.ErrPGRST100(err.Error())
				}
				// Skip nil filters (invalid/ignored params - Supabase compatibility)
				if filter != nil {
					embeddedFilters[embeddedResource] = append(embeddedFilters[embeddedResource], *filter)
				}
			}
			continue
		}

		// Regular filters
		for _, value := range values {
			filter, err := postgrest.ParseFilter(key, value)
			if err != nil {
				return nil, nil, "", postgrest.ErrPGRST100(err.Error())
			}
			// Skip nil filters (invalid/ignored params - Supabase compatibility)
			if filter != nil {
				filters = append(filters, *filter)
			}
		}
	}

	return filters, embeddedFilters, logicalOp, nil
}

// sendResponse sends a PostgREST response.
func (h *DatabaseHandler) sendResponse(c *mizu.Ctx, resp *postgrest.Response) error {
	// Set headers
	for key, value := range resp.Headers {
		c.Header().Set(key, value)
	}

	// Set content type
	c.Header().Set("Content-Type", "application/json")

	// Handle no content response (204)
	if resp.Status == 204 {
		return c.NoContent()
	}

	// Handle empty body - return status with empty JSON for consistency
	if resp.Body == nil {
		// For 201 with no body (return=minimal), Supabase returns 201 with empty body
		c.Writer().WriteHeader(resp.Status)
		return nil
	}

	return c.JSON(resp.Status, resp.Body)
}

// sendError sends a PostgREST error response.
func (h *DatabaseHandler) sendError(c *mizu.Ctx, err error) error {
	if pgErr, ok := err.(*postgrest.Error); ok {
		return c.JSON(pgErr.Status, map[string]any{
			"code":    pgErr.Code,
			"message": pgErr.Message,
			"details": pgErr.Details,
			"hint":    pgErr.Hint,
		})
	}

	return c.JSON(500, map[string]string{
		"error": err.Error(),
	})
}

// parseIntQueryDB parses an integer query parameter with a default value.
func parseIntQueryDB(c *mizu.Ctx, key string, def int) int {
	val := c.Query(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return n
}
