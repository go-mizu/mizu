package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Query API.
type Service struct {
	datasources   DataSourceStore
	history       HistoryStore
	cache         CacheStore
	driverFactory DriverFactory
	cacheTTL      time.Duration
}

// NewService creates a new Query service.
func NewService(
	datasources DataSourceStore,
	history HistoryStore,
	cache CacheStore,
	driverFactory DriverFactory,
) *Service {
	return &Service{
		datasources:   datasources,
		history:       history,
		cache:         cache,
		driverFactory: driverFactory,
		cacheTTL:      5 * time.Minute,
	}
}

// Execute executes a structured query.
func (s *Service) Execute(ctx context.Context, in *ExecuteIn) (*QueryResult, error) {
	if in.Query.DataSourceID == "" {
		return nil, ErrNoDataSource
	}
	if in.Query.Table == "" {
		return nil, ErrNoTable
	}

	// Validate the query
	if err := s.ValidateQuery(ctx, &in.Query); err != nil {
		return nil, err
	}

	// Get data source
	ds, err := s.datasources.GetByID(ctx, in.Query.DataSourceID)
	if err != nil {
		return nil, fmt.Errorf("get datasource: %w", err)
	}
	if ds == nil {
		return nil, fmt.Errorf("datasource not found: %s", in.Query.DataSourceID)
	}

	// Build SQL
	sqlQuery, params, err := s.BuildSQL(ctx, &in.Query)
	if err != nil {
		return nil, fmt.Errorf("build sql: %w", err)
	}

	// Check cache
	queryHash := s.hashQuery(sqlQuery, params)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, queryHash); err == nil && cached != nil {
			cached.Cached = true
			return cached, nil
		}
	}

	// Execute query
	start := time.Now()
	result, err := s.executeOnDriver(ctx, ds, sqlQuery, params, in.Timeout)
	duration := time.Since(start).Milliseconds()

	// Record in history
	h := &QueryHistory{
		ID:           ulid.Make().String(),
		UserID:       in.UserID,
		DataSourceID: in.Query.DataSourceID,
		Query:        sqlQuery,
		Duration:     float64(duration),
		CreatedAt:    time.Now(),
	}
	if err != nil {
		h.Error = err.Error()
	} else {
		h.RowCount = result.RowCount
	}
	if s.history != nil {
		s.history.Create(ctx, h)
	}

	if err != nil {
		return nil, err
	}

	// Cache result
	if s.cache != nil && result.RowCount < 10000 {
		s.cache.Set(ctx, queryHash, result, s.cacheTTL)
	}

	result.Duration = float64(duration)
	return result, nil
}

// ExecuteNative executes a native SQL query.
func (s *Service) ExecuteNative(ctx context.Context, in *ExecuteNativeIn) (*QueryResult, error) {
	if in.Query.DataSourceID == "" {
		return nil, ErrNoDataSource
	}
	if in.Query.Query == "" {
		return nil, ErrInvalidQuery
	}

	// Basic SQL injection protection for native queries
	if containsDangerousSQL(in.Query.Query) {
		return nil, fmt.Errorf("query contains potentially dangerous operations")
	}

	// Get data source
	ds, err := s.datasources.GetByID(ctx, in.Query.DataSourceID)
	if err != nil {
		return nil, fmt.Errorf("get datasource: %w", err)
	}
	if ds == nil {
		return nil, fmt.Errorf("datasource not found: %s", in.Query.DataSourceID)
	}

	// Replace parameters in query
	query, params := s.replaceParameters(in.Query.Query, in.Query.Params, in.Query.Parameters)

	// Execute query
	start := time.Now()
	result, err := s.executeOnDriver(ctx, ds, query, params, in.Timeout)
	duration := time.Since(start).Milliseconds()

	// Record in history
	h := &QueryHistory{
		ID:           ulid.Make().String(),
		UserID:       in.UserID,
		DataSourceID: in.Query.DataSourceID,
		Query:        query,
		Duration:     float64(duration),
		CreatedAt:    time.Now(),
	}
	if err != nil {
		h.Error = err.Error()
	} else {
		h.RowCount = result.RowCount
	}
	if s.history != nil {
		s.history.Create(ctx, h)
	}

	if err != nil {
		return nil, err
	}

	result.Duration = float64(duration)
	return result, nil
}

// ValidateQuery validates a structured query without executing.
func (s *Service) ValidateQuery(ctx context.Context, query *StructuredQuery) error {
	if query.Table == "" {
		return ErrNoTable
	}

	// Validate table name
	if err := validateIdentifier(query.Table); err != nil {
		return fmt.Errorf("invalid table: %w", err)
	}

	// Validate columns
	for _, col := range query.Columns {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column %q: %w", col, err)
		}
	}

	// Validate filters
	for _, f := range query.Filters {
		if err := validateIdentifier(f.Column); err != nil {
			return fmt.Errorf("invalid filter column %q: %w", f.Column, err)
		}
		if err := validateOperator(f.Operator); err != nil {
			return err
		}
	}

	// Validate group by
	for _, col := range query.GroupBy {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid group_by column %q: %w", col, err)
		}
	}

	// Validate order by
	for _, o := range query.OrderBy {
		if err := validateIdentifier(o.Column); err != nil {
			return fmt.Errorf("invalid order_by column %q: %w", o.Column, err)
		}
		if o.Direction != "" && o.Direction != "ASC" && o.Direction != "DESC" {
			return fmt.Errorf("invalid order direction: %s", o.Direction)
		}
	}

	return nil
}

// BuildSQL builds SQL from a structured query.
func (s *Service) BuildSQL(ctx context.Context, query *StructuredQuery) (string, []any, error) {
	var params []any
	var sql strings.Builder

	// SELECT
	sql.WriteString("SELECT ")
	if len(query.Columns) > 0 {
		for i, col := range query.Columns {
			if i > 0 {
				sql.WriteString(", ")
			}
			sql.WriteString(quoteIdentifier(col))
		}
	} else if len(query.Aggregations) > 0 {
		for i, agg := range query.Aggregations {
			if i > 0 {
				sql.WriteString(", ")
			}
			sql.WriteString(agg.Function)
			sql.WriteString("(")
			if agg.Column == "*" {
				sql.WriteString("*")
			} else {
				sql.WriteString(quoteIdentifier(agg.Column))
			}
			sql.WriteString(")")
			if agg.Alias != "" {
				sql.WriteString(" AS ")
				sql.WriteString(quoteIdentifier(agg.Alias))
			}
		}
	} else {
		sql.WriteString("*")
	}

	// FROM
	sql.WriteString(" FROM ")
	sql.WriteString(quoteIdentifier(query.Table))

	// JOINS
	for _, j := range query.Joins {
		sql.WriteString(" ")
		sql.WriteString(strings.ToUpper(j.Type))
		sql.WriteString(" JOIN ")
		sql.WriteString(quoteIdentifier(j.Table))
		sql.WriteString(" ON ")
		sql.WriteString(quoteIdentifier(query.Table))
		sql.WriteString(".")
		sql.WriteString(quoteIdentifier(j.LeftColumn))
		sql.WriteString(" = ")
		sql.WriteString(quoteIdentifier(j.Table))
		sql.WriteString(".")
		sql.WriteString(quoteIdentifier(j.RightColumn))
	}

	// WHERE
	if len(query.Filters) > 0 {
		sql.WriteString(" WHERE ")
		for i, f := range query.Filters {
			if i > 0 {
				sql.WriteString(" AND ")
			}

			upperOp := strings.ToUpper(f.Operator)
			switch upperOp {
			case "IS NULL", "IS NOT NULL":
				sql.WriteString(quoteIdentifier(f.Column))
				sql.WriteString(" ")
				sql.WriteString(upperOp)
			case "IN", "NOT IN":
				sql.WriteString(quoteIdentifier(f.Column))
				sql.WriteString(" ")
				sql.WriteString(upperOp)
				sql.WriteString(" (")
				if valSlice, ok := f.Value.([]any); ok {
					for j := range valSlice {
						if j > 0 {
							sql.WriteString(", ")
						}
						sql.WriteString("?")
						params = append(params, valSlice[j])
					}
				} else {
					sql.WriteString("?")
					params = append(params, f.Value)
				}
				sql.WriteString(")")
			case "BETWEEN":
				if valSlice, ok := f.Value.([]any); ok && len(valSlice) == 2 {
					sql.WriteString(quoteIdentifier(f.Column))
					sql.WriteString(" BETWEEN ? AND ?")
					params = append(params, valSlice[0], valSlice[1])
				}
			default:
				sql.WriteString(quoteIdentifier(f.Column))
				sql.WriteString(" ")
				sql.WriteString(f.Operator)
				sql.WriteString(" ?")
				params = append(params, f.Value)
			}
		}
	}

	// GROUP BY
	if len(query.GroupBy) > 0 {
		sql.WriteString(" GROUP BY ")
		for i, col := range query.GroupBy {
			if i > 0 {
				sql.WriteString(", ")
			}
			sql.WriteString(quoteIdentifier(col))
		}
	}

	// ORDER BY
	if len(query.OrderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		for i, o := range query.OrderBy {
			if i > 0 {
				sql.WriteString(", ")
			}
			sql.WriteString(quoteIdentifier(o.Column))
			if o.Direction != "" {
				sql.WriteString(" ")
				sql.WriteString(strings.ToUpper(o.Direction))
			}
		}
	}

	// LIMIT
	if query.Limit > 0 {
		if query.Limit > 10000 {
			query.Limit = 10000
		}
		sql.WriteString(fmt.Sprintf(" LIMIT %d", query.Limit))
	}

	return sql.String(), params, nil
}

// ListHistory returns query history for a user.
func (s *Service) ListHistory(ctx context.Context, opts HistoryListOpts) ([]*QueryHistory, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	return s.history.List(ctx, opts.UserID, opts.Limit)
}

// GetCachedResult returns a cached result if available.
func (s *Service) GetCachedResult(ctx context.Context, queryHash string) (*QueryResult, error) {
	if s.cache == nil {
		return nil, nil
	}
	return s.cache.Get(ctx, queryHash)
}

// executeOnDriver executes a query on the data source driver.
func (s *Service) executeOnDriver(ctx context.Context, ds *DataSource, query string, params []any, timeout time.Duration) (*QueryResult, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	driver, err := s.driverFactory.Open(ctx, ds)
	if err != nil {
		return nil, fmt.Errorf("open driver: %w", err)
	}
	defer driver.Close()

	return driver.Execute(ctx, query, params...)
}

// hashQuery creates a hash of the query for caching.
func (s *Service) hashQuery(query string, params []any) string {
	data, _ := json.Marshal(map[string]any{
		"query":  query,
		"params": params,
	})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// replaceParameters replaces named parameters in a query.
func (s *Service) replaceParameters(query string, positional []any, named map[string]any) (string, []any) {
	params := positional

	// Replace {{param}} style parameters
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	result := re.ReplaceAllStringFunc(query, func(match string) string {
		name := strings.Trim(match, "{}")
		if val, ok := named[name]; ok {
			params = append(params, val)
			return "?"
		}
		return match
	})

	return result, params
}

// Validation helpers

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

func validateIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(s) > 128 {
		return fmt.Errorf("identifier too long")
	}
	if !identifierRegex.MatchString(s) {
		return fmt.Errorf("invalid identifier: %s", s)
	}

	upper := strings.ToUpper(s)
	forbidden := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "EXEC", "EXECUTE", "UNION"}
	for _, keyword := range forbidden {
		if upper == keyword {
			return fmt.Errorf("identifier cannot be a SQL keyword")
		}
	}
	return nil
}

func validateOperator(op string) error {
	validOps := map[string]bool{
		"=": true, "!=": true, "<>": true,
		">": true, ">=": true, "<": true, "<=": true,
		"LIKE": true, "like": true,
		"ILIKE": true, "ilike": true,
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

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func containsDangerousSQL(query string) bool {
	upper := strings.ToUpper(query)
	dangerous := []string{
		"DROP ", "DELETE ", "TRUNCATE ", "ALTER ", "CREATE ",
		"INSERT ", "UPDATE ", "GRANT ", "REVOKE ", "EXEC ", "EXECUTE ",
	}
	for _, d := range dangerous {
		if strings.Contains(upper, d) {
			return true
		}
	}
	return false
}
