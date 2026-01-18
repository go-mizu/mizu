package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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

// ExecuteQuery executes a SQL query with enhanced options.
func (h *DatabaseHandler) ExecuteQuery(c *mizu.Ctx) error {
	var req struct {
		Query   string `json:"query"`
		Params  []any  `json:"params"`
		Role    string `json:"role,omitempty"`
		Timeout int    `json:"timeout,omitempty"`
		Explain bool   `json:"explain,omitempty"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	// Record start time
	startTime := time.Now()
	queryID := fmt.Sprintf("q_%d", time.Now().UnixNano())

	// Add EXPLAIN if requested
	queryToRun := req.Query
	if req.Explain {
		queryToRun = "EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT JSON) " + req.Query
	}

	// Use role-specific execution context
	role := req.Role
	if role == "" {
		role = "postgres"
	}

	// Check if it's a SELECT query
	trimmed := strings.TrimSpace(strings.ToUpper(queryToRun))
	isSelect := strings.HasPrefix(trimmed, "SELECT") || strings.HasPrefix(trimmed, "WITH") || strings.HasPrefix(trimmed, "EXPLAIN")

	var result *store.QueryResult
	var err error
	var rowsAffected int64

	if isSelect {
		// For role-based execution with RLS
		if role != "postgres" && role != "service_role" {
			rlsCtx := &postgres.RLSContext{
				Role:   role,
				UserID: middleware.GetUserID(c),
				Email:  middleware.GetUserEmail(c),
			}
			result, err = h.store.DatabaseRLS().QueryWithRLS(c.Context(), rlsCtx, queryToRun, req.Params...)
		} else {
			result, err = h.store.Database().Query(c.Context(), queryToRun, req.Params...)
		}
	} else {
		// Execute non-SELECT query
		if role != "postgres" && role != "service_role" {
			rlsCtx := &postgres.RLSContext{
				Role:   role,
				UserID: middleware.GetUserID(c),
				Email:  middleware.GetUserEmail(c),
			}
			rowsAffected, err = h.store.DatabaseRLS().ExecWithRLS(c.Context(), rlsCtx, queryToRun, req.Params...)
		} else {
			rowsAffected, err = h.store.Database().Exec(c.Context(), queryToRun, req.Params...)
		}
	}

	duration := time.Since(startTime).Seconds() * 1000

	// Record in query history
	historyEntry := &store.QueryHistoryEntry{
		ID:         queryID,
		Query:      req.Query, // Original query without EXPLAIN
		ExecutedAt: startTime,
		DurationMs: duration,
		Role:       role,
		Success:    err == nil,
	}
	if err != nil {
		historyEntry.Error = err.Error()
	} else if result != nil {
		historyEntry.RowCount = result.RowCount
	} else {
		historyEntry.RowCount = int(rowsAffected)
	}
	// Record history asynchronously
	go func() {
		_ = h.store.Database().AddQueryHistory(context.Background(), historyEntry)
	}()

	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	if isSelect {
		// Enhanced result with query ID and column types
		response := map[string]any{
			"query_id":    queryID,
			"columns":     result.Columns,
			"rows":        result.Rows,
			"row_count":   result.RowCount,
			"duration_ms": duration,
		}
		return c.JSON(200, response)
	}

	return c.JSON(200, map[string]any{
		"query_id":      queryID,
		"rows_affected": rowsAffected,
		"duration_ms":   duration,
	})
}

// ========================================
// SQL Editor - Query History Endpoints
// ========================================

// ListQueryHistory returns query history.
func (h *DatabaseHandler) ListQueryHistory(c *mizu.Ctx) error {
	limit := parseIntQueryDB(c, "limit", 100)
	offset := parseIntQueryDB(c, "offset", 0)

	entries, err := h.store.Database().ListQueryHistory(c.Context(), limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, entries)
}

// ClearQueryHistory clears all query history.
func (h *DatabaseHandler) ClearQueryHistory(c *mizu.Ctx) error {
	if err := h.store.Database().ClearQueryHistory(c.Context()); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.NoContent()
}

// ========================================
// SQL Editor - Snippets Endpoints
// ========================================

// ListSnippets returns all SQL snippets.
func (h *DatabaseHandler) ListSnippets(c *mizu.Ctx) error {
	snippets, err := h.store.Database().ListSnippets(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, snippets)
}

// GetSnippet returns a SQL snippet by ID.
func (h *DatabaseHandler) GetSnippet(c *mizu.Ctx) error {
	id := c.Param("id")
	snippet, err := h.store.Database().GetSnippet(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, snippet)
}

// CreateSnippet creates a new SQL snippet.
func (h *DatabaseHandler) CreateSnippet(c *mizu.Ctx) error {
	var snippet store.SQLSnippet
	if err := c.BindJSON(&snippet, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if snippet.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}
	if snippet.Query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	if err := h.store.Database().CreateSnippet(c.Context(), &snippet); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, snippet)
}

// UpdateSnippet updates a SQL snippet.
func (h *DatabaseHandler) UpdateSnippet(c *mizu.Ctx) error {
	id := c.Param("id")

	var snippet store.SQLSnippet
	if err := c.BindJSON(&snippet, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	snippet.ID = id
	if err := h.store.Database().UpdateSnippet(c.Context(), &snippet); err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, snippet)
}

// DeleteSnippet deletes a SQL snippet.
func (h *DatabaseHandler) DeleteSnippet(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Database().DeleteSnippet(c.Context(), id); err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.NoContent()
}

// ========================================
// SQL Editor - Folders Endpoints
// ========================================

// ListFolders returns all SQL folders.
func (h *DatabaseHandler) ListFolders(c *mizu.Ctx) error {
	folders, err := h.store.Database().ListFolders(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, folders)
}

// CreateFolder creates a new SQL folder.
func (h *DatabaseHandler) CreateFolder(c *mizu.Ctx) error {
	var folder store.SQLFolder
	if err := c.BindJSON(&folder, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if folder.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}

	if err := h.store.Database().CreateFolder(c.Context(), &folder); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, folder)
}

// UpdateFolder updates a SQL folder.
func (h *DatabaseHandler) UpdateFolder(c *mizu.Ctx) error {
	id := c.Param("id")

	var folder store.SQLFolder
	if err := c.BindJSON(&folder, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	folder.ID = id
	if err := h.store.Database().UpdateFolder(c.Context(), &folder); err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, folder)
}

// DeleteFolder deletes a SQL folder.
func (h *DatabaseHandler) DeleteFolder(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Database().DeleteFolder(c.Context(), id); err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.NoContent()
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

// ========================================
// Enhanced Table Editor API handlers
// ========================================

// GetTableData returns table data with enhanced filtering and count.
func (h *DatabaseHandler) GetTableData(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")

	// Build query with filtering and pagination
	limit := parseIntQueryDB(c, "limit", 100)
	offset := parseIntQueryDB(c, "offset", 0)
	orderBy := c.Query("order")
	selectCols := c.Query("select")
	includeCount := c.Query("count") == "true"

	// Build SELECT query
	var colList string
	if selectCols != "" {
		colList = selectCols
	} else {
		colList = "*"
	}

	query := "SELECT " + colList + " FROM " + schema + "." + table

	// Parse filters from query params
	filters, _, _, err := h.parseFilters(c)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Build WHERE clause from filters
	if len(filters) > 0 {
		query += " WHERE "
		conditions := make([]string, 0, len(filters))
		for _, f := range filters {
			condition := buildFilterCondition(f)
			if condition != "" {
				conditions = append(conditions, condition)
			}
		}
		query += strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	if orderBy != "" {
		parts := strings.Split(orderBy, ".")
		if len(parts) == 2 {
			col := parts[0]
			dir := strings.ToUpper(parts[1])
			if dir != "ASC" && dir != "DESC" {
				dir = "ASC"
			}
			query += " ORDER BY " + col + " " + dir
		} else {
			query += " ORDER BY " + orderBy
		}
	}

	// Add pagination
	query += " LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)

	// Execute query
	result, err := h.store.Database().Query(c.Context(), query)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Get total count if requested
	if includeCount {
		countQuery := "SELECT COUNT(*) as count FROM " + schema + "." + table
		if len(filters) > 0 {
			countQuery += " WHERE "
			conditions := make([]string, 0, len(filters))
			for _, f := range filters {
				condition := buildFilterCondition(f)
				if condition != "" {
					conditions = append(conditions, condition)
				}
			}
			countQuery += strings.Join(conditions, " AND ")
		}
		countResult, err := h.store.Database().Query(c.Context(), countQuery)
		if err == nil && len(countResult.Rows) > 0 {
			if count, ok := countResult.Rows[0]["count"].(int64); ok {
				c.Header().Set("X-Total-Count", strconv.FormatInt(count, 10))
			}
		}
	}

	return c.JSON(200, result.Rows)
}

// ExportTableData exports table data in various formats.
func (h *DatabaseHandler) ExportTableData(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	selectCols := c.Query("select")
	var colList string
	if selectCols != "" {
		colList = selectCols
	} else {
		colList = "*"
	}

	// Build query
	query := "SELECT " + colList + " FROM " + schema + "." + table

	// Parse filters
	filters, _, _, err := h.parseFilters(c)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Build WHERE clause
	if len(filters) > 0 {
		query += " WHERE "
		conditions := make([]string, 0, len(filters))
		for _, f := range filters {
			condition := buildFilterCondition(f)
			if condition != "" {
				conditions = append(conditions, condition)
			}
		}
		query += strings.Join(conditions, " AND ")
	}

	// Execute query (no limit for export)
	result, err := h.store.Database().Query(c.Context(), query)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Set download headers
	filename := table + "_export"
	switch format {
	case "csv":
		c.Header().Set("Content-Type", "text/csv; charset=utf-8")
		c.Header().Set("Content-Disposition", "attachment; filename=\""+filename+".csv\"")
		return exportCSV(c, result)
	case "sql":
		c.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Header().Set("Content-Disposition", "attachment; filename=\""+filename+".sql\"")
		return exportSQL(c, schema, table, result)
	default: // json
		c.Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Header().Set("Content-Disposition", "attachment; filename=\""+filename+".json\"")
		return c.JSON(200, result.Rows)
	}
}

// BulkTableOperation performs bulk operations on table rows.
func (h *DatabaseHandler) BulkTableOperation(c *mizu.Ctx) error {
	schema := c.Param("schema")
	table := c.Param("name")

	var req struct {
		Operation string                   `json:"operation"` // delete, update
		IDs       []interface{}            `json:"ids"`       // Primary key values
		Column    string                   `json:"column"`    // Primary key column name
		Data      map[string]interface{}   `json:"data"`      // For update operation
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if len(req.IDs) == 0 {
		return c.JSON(400, map[string]string{"error": "ids required"})
	}

	if req.Column == "" {
		req.Column = "id"
	}

	switch req.Operation {
	case "delete":
		// Build DELETE query with IN clause
		placeholders := make([]string, len(req.IDs))
		args := make([]interface{}, len(req.IDs))
		for i, id := range req.IDs {
			placeholders[i] = "$" + strconv.Itoa(i+1)
			args[i] = id
		}
		query := "DELETE FROM " + schema + "." + table + " WHERE " + req.Column + " IN (" + strings.Join(placeholders, ", ") + ")"

		rows, err := h.store.Database().Exec(c.Context(), query, args...)
		if err != nil {
			return c.JSON(400, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, map[string]interface{}{
			"operation":     "delete",
			"rows_affected": rows,
		})

	case "update":
		if req.Data == nil || len(req.Data) == 0 {
			return c.JSON(400, map[string]string{"error": "data required for update"})
		}

		// Build UPDATE query
		setClauses := make([]string, 0, len(req.Data))
		args := make([]interface{}, 0, len(req.Data)+len(req.IDs))
		argIdx := 1

		for col, val := range req.Data {
			setClauses = append(setClauses, col+" = $"+strconv.Itoa(argIdx))
			args = append(args, val)
			argIdx++
		}

		// Add ID placeholders
		idPlaceholders := make([]string, len(req.IDs))
		for i, id := range req.IDs {
			idPlaceholders[i] = "$" + strconv.Itoa(argIdx)
			args = append(args, id)
			argIdx++
		}

		query := "UPDATE " + schema + "." + table + " SET " + strings.Join(setClauses, ", ") + " WHERE " + req.Column + " IN (" + strings.Join(idPlaceholders, ", ") + ")"

		rows, err := h.store.Database().Exec(c.Context(), query, args...)
		if err != nil {
			return c.JSON(400, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, map[string]interface{}{
			"operation":     "update",
			"rows_affected": rows,
		})

	default:
		return c.JSON(400, map[string]string{"error": "unsupported operation: " + req.Operation})
	}
}

// buildFilterCondition builds a SQL condition from a filter.
func buildFilterCondition(f postgrest.Filter) string {
	// Convert value to string
	valStr := valueToString(f.Value)

	// Handle different operators
	switch f.Operator {
	case "eq":
		return f.Column + " = '" + escapeSQL(valStr) + "'"
	case "neq":
		return f.Column + " != '" + escapeSQL(valStr) + "'"
	case "gt":
		return f.Column + " > '" + escapeSQL(valStr) + "'"
	case "gte":
		return f.Column + " >= '" + escapeSQL(valStr) + "'"
	case "lt":
		return f.Column + " < '" + escapeSQL(valStr) + "'"
	case "lte":
		return f.Column + " <= '" + escapeSQL(valStr) + "'"
	case "like":
		return f.Column + " LIKE '" + escapeSQL(valStr) + "'"
	case "ilike":
		return f.Column + " ILIKE '" + escapeSQL(valStr) + "'"
	case "is":
		if valStr == "null" {
			return f.Column + " IS NULL"
		} else if valStr == "true" {
			return f.Column + " IS TRUE"
		} else if valStr == "false" {
			return f.Column + " IS FALSE"
		}
		return ""
	case "in":
		// Value should be comma-separated list in parens
		return f.Column + " IN " + valStr
	default:
		return f.Column + " = '" + escapeSQL(valStr) + "'"
	}
}

// valueToString converts an interface{} value to string.
func valueToString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return formatValue(val)
	case float32, float64:
		return formatValue(val)
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return ""
	}
}

// escapeSQL escapes single quotes in SQL strings.
func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// exportCSV exports query results as CSV.
func exportCSV(c *mizu.Ctx, result *store.QueryResult) error {
	var sb strings.Builder

	// Write header
	sb.WriteString(strings.Join(result.Columns, ",") + "\n")

	// Write rows
	for _, row := range result.Rows {
		values := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			val := row[col]
			if val == nil {
				values[i] = ""
			} else {
				str := formatCSVValue(val)
				values[i] = str
			}
		}
		sb.WriteString(strings.Join(values, ",") + "\n")
	}

	return c.Text(200, sb.String())
}

// formatCSVValue formats a value for CSV output.
func formatCSVValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		// Escape quotes and wrap in quotes if contains comma, quote, or newline
		if strings.ContainsAny(v, ",\"\n") {
			return "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(strings.Trim(strings.Trim(formatValue(val), "["), "]")), "\"", "\"\""), "\n", " ")
	}
}

// formatValue formats a value for output.
func formatValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case nil:
		return "null"
	default:
		// Use JSON encoding for complex types
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return ""
	}
}

// exportSQL exports query results as SQL INSERT statements.
func exportSQL(c *mizu.Ctx, schema, table string, result *store.QueryResult) error {
	var sb strings.Builder

	sb.WriteString("-- Exported from " + schema + "." + table + "\n")
	sb.WriteString("-- Generated at " + time.Now().Format(time.RFC3339) + "\n\n")

	for _, row := range result.Rows {
		cols := make([]string, 0, len(result.Columns))
		vals := make([]string, 0, len(result.Columns))

		for _, col := range result.Columns {
			cols = append(cols, col)
			val := row[col]
			vals = append(vals, formatSQLValue(val))
		}

		sb.WriteString("INSERT INTO " + schema + "." + table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(vals, ", ") + ");\n")
	}

	return c.Text(200, sb.String())
}

// formatSQLValue formats a value for SQL INSERT.
func formatSQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case string:
		return "'" + escapeSQL(v) + "'"
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return formatValue(val)
	default:
		// JSON or complex types
		if b, err := json.Marshal(v); err == nil {
			return "'" + escapeSQL(string(b)) + "'"
		}
		return "NULL"
	}
}
