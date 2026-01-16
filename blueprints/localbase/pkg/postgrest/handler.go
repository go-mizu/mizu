package postgrest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Querier interface for database operations.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (*QueryResult, error)
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
	TableExists(ctx context.Context, schema, table string) (bool, error)
	GetForeignKeys(ctx context.Context, schema, table string) ([]ForeignKey, error)
}

// RLSContext holds JWT claims for RLS enforcement.
// This is passed to queries to set database-level context.
type RLSContext struct {
	Role       string // anon, authenticated, service_role
	UserID     string // auth.uid() - the sub claim
	Email      string // auth.email() - the email claim
	ClaimsJSON string // Full JWT claims as JSON
}

// RLSQuerier extends Querier with RLS-aware query methods.
type RLSQuerier interface {
	Querier
	QueryWithRLS(ctx context.Context, rlsCtx *RLSContext, sql string, args ...any) (*QueryResult, error)
	ExecWithRLS(ctx context.Context, rlsCtx *RLSContext, sql string, args ...any) (int64, error)
}

// QueryResult represents the result of a query.
type QueryResult struct {
	Columns []string
	Rows    []map[string]any
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	ConstraintName string
	ColumnName     string
	ForeignSchema  string
	ForeignTable   string
	ForeignColumn  string
	// M2M junction table info (set only for many-to-many relationships)
	JunctionTable  string
	JunctionMainFK string // FK column in junction pointing to main table
	JunctionFK     string // FK column in junction pointing to target table
}

// Handler provides PostgREST-compatible REST API handling.
type Handler struct {
	querier       Querier
	rlsQuerier    RLSQuerier // Optional: querier with RLS support
	defaultSchema string
	maxRows       int
}

// NewHandler creates a new PostgREST handler.
func NewHandler(querier Querier) *Handler {
	h := &Handler{
		querier:       querier,
		defaultSchema: "public",
		maxRows:       1000,
	}
	// If the querier supports RLS, store it for RLS-aware operations
	if rls, ok := querier.(RLSQuerier); ok {
		h.rlsQuerier = rls
	}
	return h
}

// Request represents a parsed PostgREST request.
type Request struct {
	Schema          string
	Table           string
	Select          []SelectColumn
	Filters         []Filter
	EmbeddedFilters map[string][]Filter // Filters keyed by embedded resource name
	LogicalOp       string              // "AND" or "OR" for top-level logical
	Order           []OrderClause
	Limit           int
	Offset          int
	Prefs           *Preferences
	Body            any
	OnConflict      string
	HasRange        bool        // True if Range header was specified
	RLSContext      *RLSContext // JWT claims for RLS enforcement (optional)
}

// Response represents a PostgREST response.
type Response struct {
	Status       int
	Body         any
	Headers      map[string]string
	ContentRange string
	Count        *int
}

// query executes a query with RLS context if available.
func (h *Handler) query(ctx context.Context, rlsCtx *RLSContext, sql string, params ...any) (*QueryResult, error) {
	if h.rlsQuerier != nil && rlsCtx != nil {
		return h.rlsQuerier.QueryWithRLS(ctx, rlsCtx, sql, params...)
	}
	return h.querier.Query(ctx, sql, params...)
}

// exec executes a statement with RLS context if available.
func (h *Handler) exec(ctx context.Context, rlsCtx *RLSContext, sql string, params ...any) (int64, error) {
	if h.rlsQuerier != nil && rlsCtx != nil {
		return h.rlsQuerier.ExecWithRLS(ctx, rlsCtx, sql, params...)
	}
	return h.querier.Exec(ctx, sql, params...)
}

// Select handles GET requests to a table.
func (h *Handler) Select(ctx context.Context, req *Request) (*Response, error) {
	// Check table exists
	exists, err := h.querier.TableExists(ctx, req.Schema, req.Table)
	if err != nil {
		return nil, ParsePGError(err)
	}
	if !exists {
		return nil, ErrPGRST205(req.Table)
	}

	// Build SQL
	sql, params, err := h.buildSelectSQL(ctx, req)
	if err != nil {
		return nil, err
	}

	// Execute query with RLS context
	result, err := h.query(ctx, req.RLSContext, sql, params...)
	if err != nil {
		return nil, ParsePGError(err)
	}

	// Handle count
	var count *int
	if req.Prefs.Count != CountNone {
		countVal, err := h.getCount(ctx, req)
		if err == nil {
			count = &countVal
		}
	}

	// Build response
	resp := &Response{
		Status:  http.StatusOK,
		Body:    result.Rows,
		Headers: make(map[string]string),
		Count:   count,
	}

	// Set Content-Range header
	if count != nil {
		start := req.Offset
		end := start + len(result.Rows) - 1
		if end < start {
			end = start
		}
		resp.ContentRange = fmt.Sprintf("%d-%d/%d", start, end, *count)
		resp.Headers["Content-Range"] = resp.ContentRange

		// Return 206 Partial Content only if Range header was explicitly specified
		// and the result is a partial result
		if req.HasRange && len(result.Rows) < *count {
			resp.Status = http.StatusPartialContent
		}
	}

	// Set Preference-Applied header
	if applied := req.Prefs.PreferenceApplied(); applied != "" {
		resp.Headers["Preference-Applied"] = applied
	}

	return resp, nil
}

func (h *Handler) buildSelectSQL(ctx context.Context, req *Request) (string, []any, error) {
	var params []any
	paramIdx := 1

	// Check if there are embedded resources
	hasEmbedded := false
	for _, col := range req.Select {
		if col.Embedded != nil {
			hasEmbedded = true
			break
		}
	}

	// If there are embedded resources, use a different approach
	if hasEmbedded {
		return h.buildSelectWithEmbedding(ctx, req)
	}

	// Build SELECT columns
	selectCols := BuildSelectColumns(req.Select, "")
	if selectCols == "" {
		selectCols = "*"
	}

	// Build FROM clause
	sql := fmt.Sprintf("SELECT %s FROM %s.%s",
		selectCols,
		QuoteIdent(req.Schema),
		QuoteIdent(req.Table))

	// Build WHERE clause
	whereClauses, whereParams, err := h.buildWhereClause(req.Filters, &paramIdx)
	if err != nil {
		return "", nil, err
	}
	params = append(params, whereParams...)

	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build ORDER BY
	if len(req.Order) > 0 {
		sql += " " + OrderToSQL(req.Order)
	}

	// Build LIMIT/OFFSET
	limit := req.Limit
	if limit <= 0 || limit > h.maxRows {
		limit = h.maxRows
	}
	sql += fmt.Sprintf(" LIMIT %d", limit)

	if req.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", req.Offset)
	}

	return sql, params, nil
}

// buildSelectWithEmbedding builds a SELECT query with embedded resources using lateral joins.
func (h *Handler) buildSelectWithEmbedding(ctx context.Context, req *Request) (string, []any, error) {
	var params []any
	paramIdx := 1

	mainAlias := "_main"

	// Get FK relationships for the main table
	fks, err := h.querier.GetForeignKeys(ctx, req.Schema, req.Table)
	if err != nil {
		return "", nil, err
	}

	// Also get reverse FKs (where other tables point to this one)
	reverseFKs, err := h.getReverseForeignKeys(ctx, req.Schema, req.Table)
	if err != nil {
		return "", nil, err
	}

	// Build base columns (non-embedded)
	var baseCols []string
	var embeddedCols []SelectColumn

	for _, col := range req.Select {
		if col.Embedded != nil {
			embeddedCols = append(embeddedCols, col)
		} else if col.Name == "*" {
			baseCols = append(baseCols, mainAlias+".*")
		} else {
			part := mainAlias + "." + QuoteIdent(col.Name)
			if col.JSONPath != "" {
				part += col.JSONPath
			}
			if col.Cast != "" {
				part = "(" + part + ")::" + col.Cast
			}
			if col.Alias != "" {
				part += " AS " + QuoteIdent(col.Alias)
			}
			baseCols = append(baseCols, part)
		}
	}

	if len(baseCols) == 0 {
		baseCols = append(baseCols, mainAlias+".*")
	}

	// Build embedded column subqueries
	var embeddedSubqueries []string
	for _, col := range embeddedCols {
		embedded := col.Embedded
		embeddedAlias := embedded.Table
		if col.Alias != "" {
			embeddedAlias = col.Alias
		}

		// Find the FK relationship
		fk, isReverse := h.findFKRelation(fks, reverseFKs, embedded.Table, embedded.Hint)
		if fk == nil {
			// Try to find a many-to-many relationship through a junction table
			junctionFK := h.findManyToManyRelation(ctx, req.Schema, req.Table, embedded.Table)
			if junctionFK != nil {
				fk = junctionFK
				isReverse = true // M2M is handled like one-to-many
			} else {
				return "", nil, ErrPGRST200(fmt.Sprintf("could not find relationship for '%s'", embedded.Table))
			}
		}

		// Build the embedded columns
		embeddedSelectCols := "*"
		if len(embedded.Columns) > 0 {
			// Check if it's just "*"
			if len(embedded.Columns) == 1 && embedded.Columns[0].Name == "*" {
				embeddedSelectCols = "*"
			} else {
				var cols []string
				for _, ec := range embedded.Columns {
					if ec.Embedded != nil {
						// Nested embedding - not supported in first pass
						continue
					}
					if ec.Name == "*" {
						cols = append(cols, "*")
					} else {
						cols = append(cols, QuoteIdent(ec.Name))
					}
				}
				if len(cols) > 0 {
					embeddedSelectCols = strings.Join(cols, ", ")
				}
			}
		}

		// Build extra WHERE conditions for embedded filters
		var extraWhere string
		allEmbeddedFilters := append(req.EmbeddedFilters[embedded.Table], req.EmbeddedFilters[embeddedAlias]...)
		if len(allEmbeddedFilters) > 0 {
			var clauses []string
			for _, f := range allEmbeddedFilters {
				clause := buildFilterLiteral(&f)
				if clause != "" {
					clauses = append(clauses, clause)
				}
			}
			if len(clauses) > 0 {
				extraWhere = " AND " + strings.Join(clauses, " AND ")
			}
		}

		var subquery string
		if fk.JunctionTable != "" {
			// Many-to-many: Join through junction table
			subquery = fmt.Sprintf(
				"(SELECT COALESCE(json_agg(row_to_json(_sub)), '[]'::json) FROM (SELECT %s FROM %s.%s _t JOIN %s.%s _j ON _j.%s = _t.%s WHERE _j.%s = %s.%s%s) _sub) AS %s",
				embeddedSelectCols,
				QuoteIdent(fk.ForeignSchema),
				QuoteIdent(fk.ForeignTable),
				QuoteIdent(fk.ForeignSchema),
				QuoteIdent(fk.JunctionTable),
				QuoteIdent(fk.JunctionFK),
				QuoteIdent(fk.ForeignColumn),
				QuoteIdent(fk.JunctionMainFK),
				mainAlias,
				QuoteIdent(fk.ColumnName),
				extraWhere,
				QuoteIdent(embeddedAlias),
			)
		} else if isReverse {
			// One-to-many: The embedded table has FK pointing to main table
			subquery = fmt.Sprintf(
				"(SELECT COALESCE(json_agg(row_to_json(_sub)), '[]'::json) FROM (SELECT %s FROM %s.%s WHERE %s.%s = %s.%s%s) _sub) AS %s",
				embeddedSelectCols,
				QuoteIdent(fk.ForeignSchema),
				QuoteIdent(fk.ForeignTable),
				QuoteIdent(fk.ForeignTable),
				QuoteIdent(fk.ForeignColumn),
				mainAlias,
				QuoteIdent(fk.ColumnName),
				extraWhere,
				QuoteIdent(embeddedAlias),
			)
		} else {
			// Many-to-one: Main table has FK pointing to embedded table
			subquery = fmt.Sprintf(
				"(SELECT row_to_json(_sub) FROM (SELECT %s FROM %s.%s WHERE %s.%s = %s.%s%s) _sub) AS %s",
				embeddedSelectCols,
				QuoteIdent(fk.ForeignSchema),
				QuoteIdent(fk.ForeignTable),
				QuoteIdent(fk.ForeignTable),
				QuoteIdent(fk.ForeignColumn),
				mainAlias,
				QuoteIdent(fk.ColumnName),
				extraWhere,
				QuoteIdent(embeddedAlias),
			)
		}

		embeddedSubqueries = append(embeddedSubqueries, subquery)
	}

	// Build the final SELECT
	allCols := append(baseCols, embeddedSubqueries...)
	sql := fmt.Sprintf("SELECT %s FROM %s.%s AS %s",
		strings.Join(allCols, ", "),
		QuoteIdent(req.Schema),
		QuoteIdent(req.Table),
		mainAlias)

	// Build WHERE clause
	whereClauses, whereParams, err := h.buildWhereClauseWithAlias(req.Filters, &paramIdx, mainAlias)
	if err != nil {
		return "", nil, err
	}
	params = append(params, whereParams...)

	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build ORDER BY
	if len(req.Order) > 0 {
		sql += " " + h.orderToSQLWithAlias(req.Order, mainAlias)
	}

	// Build LIMIT/OFFSET
	limit := req.Limit
	if limit <= 0 || limit > h.maxRows {
		limit = h.maxRows
	}
	sql += fmt.Sprintf(" LIMIT %d", limit)

	if req.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", req.Offset)
	}

	return sql, params, nil
}

// getReverseForeignKeys gets FKs from other tables that point to this table.
func (h *Handler) getReverseForeignKeys(ctx context.Context, schema, table string) ([]ForeignKey, error) {
	sql := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_schema AS foreign_schema,
			kcu.table_name AS foreign_table,
			ccu.column_name AS foreign_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND ccu.table_schema = $1
			AND ccu.table_name = $2
	`

	result, err := h.querier.Query(ctx, sql, schema, table)
	if err != nil {
		return nil, err
	}

	var fks []ForeignKey
	for _, row := range result.Rows {
		fk := ForeignKey{
			ConstraintName: getString(row, "constraint_name"),
			ColumnName:     getString(row, "foreign_column"), // The column in the main table
			ForeignSchema:  getString(row, "foreign_schema"),
			ForeignTable:   getString(row, "foreign_table"),  // The table with the FK
			ForeignColumn:  getString(row, "column_name"),    // The FK column
		}
		fks = append(fks, fk)
	}

	return fks, nil
}

func getString(row map[string]any, key string) string {
	if v, ok := row[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// findFKRelation finds the FK relationship for an embedded table.
// Returns the FK and whether it's a reverse relationship (one-to-many).
func (h *Handler) findFKRelation(fks, reverseFKs []ForeignKey, tableName, hint string) (*ForeignKey, bool) {
	// First check direct FKs (many-to-one)
	for _, fk := range fks {
		if fk.ForeignTable == tableName {
			if hint == "" || fk.ConstraintName == hint || fk.ColumnName == hint {
				return &fk, false
			}
		}
	}

	// Check reverse FKs (one-to-many)
	for _, fk := range reverseFKs {
		if fk.ForeignTable == tableName {
			if hint == "" || fk.ConstraintName == hint {
				return &fk, true
			}
		}
	}

	return nil, false
}

// findManyToManyRelation finds a many-to-many relationship through a junction table.
// For example: posts -> post_tags -> tags
// Returns a synthetic FK that represents the junction relationship.
func (h *Handler) findManyToManyRelation(ctx context.Context, schema, mainTable, targetTable string) *ForeignKey {
	// Look for a junction table that has FKs to both tables
	// Common patterns: main_target, target_main, maintarget, targetmain
	// Also try singular/plural variations
	junctionPatterns := []string{
		mainTable + "_" + targetTable,
		targetTable + "_" + mainTable,
		mainTable + targetTable,
		targetTable + mainTable,
		// Try singular versions (post_tags for posts->tags)
		strings.TrimSuffix(mainTable, "s") + "_" + targetTable,
		strings.TrimSuffix(targetTable, "s") + "_" + mainTable,
	}

	for _, junctionTable := range junctionPatterns {
		// Check if junction table exists
		exists, err := h.querier.TableExists(ctx, schema, junctionTable)
		if err != nil || !exists {
			continue
		}

		// Get FKs from junction table
		junctionFKs, err := h.querier.GetForeignKeys(ctx, schema, junctionTable)
		if err != nil {
			continue
		}

		// Look for FKs pointing to both main and target tables
		var mainFK, targetFK *ForeignKey
		for i := range junctionFKs {
			fk := &junctionFKs[i]
			if fk.ForeignTable == mainTable {
				mainFK = fk
			}
			if fk.ForeignTable == targetTable {
				targetFK = fk
			}
		}

		if mainFK != nil && targetFK != nil {
			// Found a valid junction! Create a synthetic FK for the M2M relationship
			return &ForeignKey{
				ConstraintName: junctionTable + "_m2m",
				ColumnName:     mainFK.ForeignColumn, // PK column of main table
				ForeignSchema:  schema,
				ForeignTable:   targetTable,
				ForeignColumn:  targetFK.ForeignColumn, // PK column of target table
				JunctionTable:  junctionTable,
				JunctionMainFK: mainFK.ColumnName, // FK column in junction pointing to main table
				JunctionFK:     targetFK.ColumnName, // FK column in junction pointing to target table
			}
		}
	}

	return nil
}

func (h *Handler) buildWhereClauseWithAlias(filters []Filter, paramIdx *int, alias string) ([]string, []any, error) {
	var clauses []string
	var params []any

	for _, f := range filters {
		clause, filterParams, err := filterToSQLWithAlias(&f, paramIdx, alias)
		if err != nil {
			return nil, nil, ErrPGRST100(err.Error())
		}
		clauses = append(clauses, clause)
		params = append(params, filterParams...)
	}

	return clauses, params, nil
}

func filterToSQLWithAlias(f *Filter, paramIdx *int, alias string) (string, []interface{}, error) {
	// Create a copy and prefix the column with the alias
	fCopy := *f
	fCopy.Column = alias + "." + f.Column
	return FilterToSQL(&fCopy, paramIdx)
}

// buildFilterLiteral builds a filter SQL with literal values (for embedded subqueries).
// Note: This should only be used for embedded resource filters where parameterization is difficult.
func buildFilterLiteral(f *Filter) string {
	col := QuoteIdent(f.Column)
	val := fmt.Sprintf("%v", f.Value)

	// Quote string values
	quotedVal := "'" + strings.ReplaceAll(val, "'", "''") + "'"

	var sql string
	switch f.Operator {
	case "eq":
		sql = fmt.Sprintf("%s = %s", col, quotedVal)
	case "neq":
		sql = fmt.Sprintf("%s != %s", col, quotedVal)
	case "gt":
		sql = fmt.Sprintf("%s > %s", col, quotedVal)
	case "gte":
		sql = fmt.Sprintf("%s >= %s", col, quotedVal)
	case "lt":
		sql = fmt.Sprintf("%s < %s", col, quotedVal)
	case "lte":
		sql = fmt.Sprintf("%s <= %s", col, quotedVal)
	case "like":
		likeVal := strings.ReplaceAll(val, "*", "%")
		sql = fmt.Sprintf("%s LIKE '%s'", col, strings.ReplaceAll(likeVal, "'", "''"))
	case "ilike":
		likeVal := strings.ReplaceAll(val, "*", "%")
		sql = fmt.Sprintf("%s ILIKE '%s'", col, strings.ReplaceAll(likeVal, "'", "''"))
	case "is":
		upperVal := strings.ToUpper(val)
		switch upperVal {
		case "NULL":
			sql = fmt.Sprintf("%s IS NULL", col)
		case "TRUE":
			sql = fmt.Sprintf("%s IS TRUE", col)
		case "FALSE":
			sql = fmt.Sprintf("%s IS FALSE", col)
		default:
			sql = fmt.Sprintf("%s IS %s", col, upperVal)
		}
	default:
		// For other operators, fall back to simple equality
		sql = fmt.Sprintf("%s = %s", col, quotedVal)
	}

	if f.Negated {
		sql = "NOT (" + sql + ")"
	}

	return sql
}

func (h *Handler) orderToSQLWithAlias(clauses []OrderClause, alias string) string {
	if len(clauses) == 0 {
		return ""
	}

	var parts []string
	for _, c := range clauses {
		part := alias + "." + QuoteIdent(c.Column)
		if c.Descending {
			part += " DESC"
		} else {
			part += " ASC"
		}
		if c.NullsFirst != nil {
			if *c.NullsFirst {
				part += " NULLS FIRST"
			} else {
				part += " NULLS LAST"
			}
		}
		parts = append(parts, part)
	}

	return "ORDER BY " + strings.Join(parts, ", ")
}

func (h *Handler) buildWhereClause(filters []Filter, paramIdx *int) ([]string, []any, error) {
	var clauses []string
	var params []any

	for _, f := range filters {
		clause, filterParams, err := FilterToSQL(&f, paramIdx)
		if err != nil {
			return nil, nil, ErrPGRST100(err.Error())
		}
		clauses = append(clauses, clause)
		params = append(params, filterParams...)
	}

	return clauses, params, nil
}

func (h *Handler) getCount(ctx context.Context, req *Request) (int, error) {
	paramIdx := 1

	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s",
		QuoteIdent(req.Schema),
		QuoteIdent(req.Table))

	// Apply filters
	whereClauses, params, err := h.buildWhereClause(req.Filters, &paramIdx)
	if err != nil {
		return 0, err
	}

	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	result, err := h.querier.Query(ctx, sql, params...)
	if err != nil {
		return 0, err
	}

	if len(result.Rows) > 0 {
		if count, ok := result.Rows[0]["count"]; ok {
			switch v := count.(type) {
			case int64:
				return int(v), nil
			case int:
				return v, nil
			case float64:
				return int(v), nil
			}
		}
	}

	return 0, nil
}

// Insert handles POST requests to a table.
func (h *Handler) Insert(ctx context.Context, req *Request) (*Response, error) {
	// Check table exists
	exists, err := h.querier.TableExists(ctx, req.Schema, req.Table)
	if err != nil {
		return nil, ParsePGError(err)
	}
	if !exists {
		return nil, ErrPGRST205(req.Table)
	}

	// Parse body
	rows, err := h.parseInsertBody(req.Body)
	if err != nil {
		return nil, ErrPGRST102(err.Error())
	}

	if len(rows) == 0 {
		return nil, ErrPGRST102("no data to insert")
	}

	// Build INSERT SQL
	sql, params, err := h.buildInsertSQL(req, rows)
	if err != nil {
		return nil, err
	}

	// Execute query with RLS context
	result, err := h.query(ctx, req.RLSContext, sql, params...)
	if err != nil {
		return nil, ParsePGError(err)
	}

	// Build response based on preferences
	// PostgREST default is to return representation with 201
	// For upserts with resolution preference, return 200 (OK) as rows may be updated
	status := http.StatusCreated
	if req.Prefs.Resolution != ResolutionNone && req.OnConflict != "" {
		status = http.StatusOK
	}
	resp := &Response{
		Status:  status,
		Headers: make(map[string]string),
	}

	// Default behavior is to return representation (matches Supabase behavior)
	// Only return no body if explicitly requested via Prefer: return=minimal
	switch req.Prefs.Return {
	case ReturnMinimal:
		// Return 201 with no body when explicitly requested
		// Note: Some clients expect 204, but Supabase returns 201
		resp.Body = nil
	case ReturnHeadersOnly:
		// Only headers, no body
		resp.Body = nil
	default:
		// Default and ReturnRepresentation - return the inserted data
		if len(result.Rows) == 1 && !isArray(req.Body) {
			resp.Body = result.Rows[0]
		} else {
			resp.Body = result.Rows
		}
	}

	// Set Preference-Applied header
	if applied := req.Prefs.PreferenceApplied(); applied != "" {
		resp.Headers["Preference-Applied"] = applied
	}

	return resp, nil
}

func (h *Handler) parseInsertBody(body any) ([]map[string]any, error) {
	if body == nil {
		return nil, fmt.Errorf("empty body")
	}

	switch v := body.(type) {
	case []any:
		var rows []map[string]any
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rows = append(rows, m)
			} else {
				return nil, fmt.Errorf("invalid array item type")
			}
		}
		return rows, nil
	case map[string]any:
		return []map[string]any{v}, nil
	default:
		return nil, fmt.Errorf("invalid body type")
	}
}

func isArray(body any) bool {
	_, ok := body.([]any)
	return ok
}

func (h *Handler) buildInsertSQL(req *Request, rows []map[string]any) (string, []any, error) {
	if len(rows) == 0 {
		return "", nil, fmt.Errorf("no rows to insert")
	}

	// Get columns from first row
	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}

	var quotedCols []string
	for _, col := range columns {
		quotedCols = append(quotedCols, QuoteIdent(col))
	}

	var valueRows []string
	var params []any
	paramIdx := 1

	for _, row := range rows {
		var placeholders []string
		for _, col := range columns {
			params = append(params, row[col])
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramIdx))
			paramIdx++
		}
		valueRows = append(valueRows, "("+strings.Join(placeholders, ", ")+")")
	}

	sql := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES %s",
		QuoteIdent(req.Schema),
		QuoteIdent(req.Table),
		strings.Join(quotedCols, ", "),
		strings.Join(valueRows, ", "))

	// Handle ON CONFLICT for upsert
	if req.OnConflict != "" && req.Prefs.Resolution != ResolutionNone {
		onConflictCols := strings.Split(req.OnConflict, ",")
		var quotedConflictCols []string
		for _, col := range onConflictCols {
			quotedConflictCols = append(quotedConflictCols, QuoteIdent(strings.TrimSpace(col)))
		}

		switch req.Prefs.Resolution {
		case ResolutionIgnoreDuplicates:
			sql += fmt.Sprintf(" ON CONFLICT (%s) DO NOTHING", strings.Join(quotedConflictCols, ", "))
		case ResolutionMergeDuplicates:
			// Build UPDATE SET clause
			var setClauses []string
			for _, col := range columns {
				setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s", QuoteIdent(col), QuoteIdent(col)))
			}
			sql += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s",
				strings.Join(quotedConflictCols, ", "),
				strings.Join(setClauses, ", "))
		}
	}

	// Add RETURNING
	sql += " RETURNING *"

	return sql, params, nil
}

// Update handles PATCH requests to a table.
func (h *Handler) Update(ctx context.Context, req *Request) (*Response, error) {
	// Check table exists
	exists, err := h.querier.TableExists(ctx, req.Schema, req.Table)
	if err != nil {
		return nil, ParsePGError(err)
	}
	if !exists {
		return nil, ErrPGRST205(req.Table)
	}

	// Block mass updates without filters
	if len(req.Filters) == 0 {
		return nil, MassOperationError()
	}

	// Parse body
	data, ok := req.Body.(map[string]any)
	if !ok {
		return nil, ErrPGRST102("body must be a JSON object")
	}

	if len(data) == 0 {
		return nil, ErrPGRST102("no data to update")
	}

	// Build UPDATE SQL
	sql, params, err := h.buildUpdateSQL(req, data)
	if err != nil {
		return nil, err
	}

	// Execute query with RLS context
	result, err := h.query(ctx, req.RLSContext, sql, params...)
	if err != nil {
		return nil, ParsePGError(err)
	}

	// Check max-affected preference
	if req.Prefs.MaxAffected != nil && len(result.Rows) > *req.Prefs.MaxAffected {
		return nil, ErrPGRST124(len(result.Rows), *req.Prefs.MaxAffected)
	}

	// Build response based on preferences
	// PostgREST default for PATCH is 204 No Content unless return=representation is specified
	resp := &Response{
		Headers: make(map[string]string),
	}

	switch req.Prefs.Return {
	case ReturnRepresentation:
		// Return the updated rows
		resp.Status = http.StatusOK
		resp.Body = result.Rows
	case ReturnHeadersOnly:
		resp.Status = http.StatusNoContent
	default:
		// Default for PATCH is 204 No Content (matches PostgREST behavior)
		resp.Status = http.StatusNoContent
	}

	// Set Preference-Applied header
	if applied := req.Prefs.PreferenceApplied(); applied != "" {
		resp.Headers["Preference-Applied"] = applied
	}

	return resp, nil
}

func (h *Handler) buildUpdateSQL(req *Request, data map[string]any) (string, []any, error) {
	var params []any
	paramIdx := 1

	// Build SET clause
	var setClauses []string
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", QuoteIdent(col), paramIdx))
		params = append(params, val)
		paramIdx++
	}

	sql := fmt.Sprintf("UPDATE %s.%s SET %s",
		QuoteIdent(req.Schema),
		QuoteIdent(req.Table),
		strings.Join(setClauses, ", "))

	// Build WHERE clause
	whereClauses, whereParams, err := h.buildWhereClause(req.Filters, &paramIdx)
	if err != nil {
		return "", nil, err
	}
	params = append(params, whereParams...)

	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	sql += " RETURNING *"

	return sql, params, nil
}

// Delete handles DELETE requests to a table.
func (h *Handler) Delete(ctx context.Context, req *Request) (*Response, error) {
	// Check table exists
	exists, err := h.querier.TableExists(ctx, req.Schema, req.Table)
	if err != nil {
		return nil, ParsePGError(err)
	}
	if !exists {
		return nil, ErrPGRST205(req.Table)
	}

	// Block mass deletes without filters
	if len(req.Filters) == 0 {
		return nil, MassOperationError()
	}

	// Build DELETE SQL
	sql, params, err := h.buildDeleteSQL(req)
	if err != nil {
		return nil, err
	}

	// Execute query with RLS context
	result, err := h.query(ctx, req.RLSContext, sql, params...)
	if err != nil {
		return nil, ParsePGError(err)
	}

	// Check max-affected preference
	if req.Prefs.MaxAffected != nil && len(result.Rows) > *req.Prefs.MaxAffected {
		return nil, ErrPGRST124(len(result.Rows), *req.Prefs.MaxAffected)
	}

	// Build response based on preferences
	// PostgREST default for DELETE is 204 No Content unless return=representation is specified
	resp := &Response{
		Headers: make(map[string]string),
	}

	switch req.Prefs.Return {
	case ReturnRepresentation:
		// Return the deleted rows
		resp.Status = http.StatusOK
		resp.Body = result.Rows
	case ReturnHeadersOnly:
		resp.Status = http.StatusNoContent
	default:
		// Default for DELETE is 204 No Content (matches PostgREST behavior)
		resp.Status = http.StatusNoContent
	}

	// Set Preference-Applied header
	if applied := req.Prefs.PreferenceApplied(); applied != "" {
		resp.Headers["Preference-Applied"] = applied
	}

	return resp, nil
}

func (h *Handler) buildDeleteSQL(req *Request) (string, []any, error) {
	var params []any
	paramIdx := 1

	sql := fmt.Sprintf("DELETE FROM %s.%s",
		QuoteIdent(req.Schema),
		QuoteIdent(req.Table))

	// Build WHERE clause
	whereClauses, whereParams, err := h.buildWhereClause(req.Filters, &paramIdx)
	if err != nil {
		return "", nil, err
	}
	params = append(params, whereParams...)

	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	sql += " RETURNING *"

	return sql, params, nil
}

// RPC handles POST requests to /rpc/{function}.
func (h *Handler) RPC(ctx context.Context, fnName string, req *Request) (*Response, error) {
	// Build function call
	var args []string
	var params []any
	paramIdx := 1

	if data, ok := req.Body.(map[string]any); ok {
		for name, val := range data {
			args = append(args, fmt.Sprintf("%s := $%d", name, paramIdx))
			params = append(params, val)
			paramIdx++
		}
	}

	sql := fmt.Sprintf("SELECT * FROM %s.%s(%s)",
		QuoteIdent(req.Schema),
		QuoteIdent(fnName),
		strings.Join(args, ", "))

	// Apply filters to result if any
	if len(req.Filters) > 0 {
		// Wrap in subquery to apply filters
		subquery := sql
		sql = fmt.Sprintf("SELECT * FROM (%s) AS _result", subquery)

		whereClauses, whereParams, err := h.buildWhereClause(req.Filters, &paramIdx)
		if err != nil {
			return nil, err
		}
		params = append(params, whereParams...)

		if len(whereClauses) > 0 {
			sql += " WHERE " + strings.Join(whereClauses, " AND ")
		}
	}

	// Apply ordering
	if len(req.Order) > 0 {
		sql += " " + OrderToSQL(req.Order)
	}

	// Apply limit/offset
	if req.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", req.Limit)
	}
	if req.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", req.Offset)
	}

	// Execute with RLS context
	result, err := h.query(ctx, req.RLSContext, sql, params...)
	if err != nil {
		pgErr := ParsePGError(err)
		// Check if function doesn't exist
		if strings.Contains(err.Error(), "does not exist") {
			return nil, ErrPGRST202(fnName)
		}
		return nil, pgErr
	}

	// Build response
	resp := &Response{
		Status:  http.StatusOK,
		Headers: make(map[string]string),
	}

	// For void functions or empty results, return 204
	// Void functions return one row with a single column containing null
	if len(result.Rows) == 0 {
		resp.Status = http.StatusNoContent
	} else if isVoidFunctionResult(result.Rows, fnName) {
		resp.Status = http.StatusNoContent
	} else {
		resp.Body = result.Rows
	}

	return resp, nil
}

// isVoidFunctionResult checks if the result is from a void function.
// Void functions return one row with a single column (named after the function) containing null or empty string.
func isVoidFunctionResult(rows []map[string]any, fnName string) bool {
	if len(rows) != 1 {
		return false
	}

	row := rows[0]
	if len(row) != 1 {
		return false
	}

	// Check if the single value is null or empty string (void representation)
	for key, v := range row {
		// Check if the column name matches the function name (void result pattern)
		if key == fnName {
			if v == nil {
				return true
			}
			if s, ok := v.(string); ok && s == "" {
				return true
			}
		}
	}

	return false
}

// ParseRequestBody parses a request body into a generic type.
func ParseRequestBody(body []byte) (any, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ParseRange parses a Range header value.
func ParseRange(rangeHeader string) (start, end int, hasRange bool) {
	if rangeHeader == "" {
		return 0, 0, false
	}

	// Format: 0-9 or 0- or -10
	rangeHeader = strings.TrimSpace(rangeHeader)

	if strings.HasPrefix(rangeHeader, "-") {
		// Last N items
		if n, err := strconv.Atoi(rangeHeader[1:]); err == nil {
			return -n, 0, true
		}
		return 0, 0, false
	}

	parts := strings.Split(rangeHeader, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}

	if parts[1] == "" {
		// Open-ended: 50-
		return start, -1, true
	}

	end, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}

	return start, end, true
}
