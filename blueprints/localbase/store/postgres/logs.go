package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogsStore implements store.LogsStore using PostgreSQL.
type LogsStore struct {
	pool *pgxpool.Pool
}

// CreateLog creates a new log entry.
func (s *LogsStore) CreateLog(ctx context.Context, entry *store.LogEntry) error {
	requestHeaders, err := json.Marshal(entry.RequestHeaders)
	if err != nil {
		requestHeaders = []byte("{}")
	}
	responseHeaders, err := json.Marshal(entry.ResponseHeaders)
	if err != nil {
		responseHeaders = []byte("{}")
	}
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	// Default severity to INFO if not set
	severity := entry.Severity
	if severity == "" {
		severity = "INFO"
	}

	sql := `
		INSERT INTO analytics.logs (
			timestamp, event_message, request_id, method, path, status_code,
			source, severity, user_id, user_agent, apikey, request_headers, response_headers,
			duration_ms, metadata, search
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) RETURNING id`

	var id string
	err = s.pool.QueryRow(ctx, sql,
		entry.Timestamp, entry.EventMessage, entry.RequestID, entry.Method, entry.Path,
		entry.StatusCode, entry.Source, severity, entry.UserID, entry.UserAgent, entry.APIKey,
		requestHeaders, responseHeaders, entry.DurationMs, metadata, entry.Search,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}

	entry.ID = id
	return nil
}

// GetLog retrieves a log entry by ID.
func (s *LogsStore) GetLog(ctx context.Context, id string) (*store.LogEntry, error) {
	sql := `
		SELECT id, timestamp, event_message, request_id, method, path, status_code,
			   source, severity, user_id, user_agent, apikey, request_headers, response_headers,
			   duration_ms, metadata, search
		FROM analytics.logs
		WHERE id = $1`

	var entry store.LogEntry
	var requestHeaders, responseHeaders, metadata []byte
	var severity *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&entry.ID, &entry.Timestamp, &entry.EventMessage, &entry.RequestID,
		&entry.Method, &entry.Path, &entry.StatusCode, &entry.Source,
		&severity, &entry.UserID, &entry.UserAgent, &entry.APIKey, &requestHeaders,
		&responseHeaders, &entry.DurationMs, &metadata, &entry.Search,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	if severity != nil {
		entry.Severity = *severity
	}
	if len(requestHeaders) > 0 {
		json.Unmarshal(requestHeaders, &entry.RequestHeaders)
	}
	if len(responseHeaders) > 0 {
		json.Unmarshal(responseHeaders, &entry.ResponseHeaders)
	}
	if len(metadata) > 0 {
		json.Unmarshal(metadata, &entry.Metadata)
	}

	return &entry, nil
}

// ListLogs retrieves log entries with optional filtering.
func (s *LogsStore) ListLogs(ctx context.Context, filter *store.LogFilter) ([]*store.LogEntry, int, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []any{}
	argNum := 1

	// Apply time filter from TimeRange if From/To not set
	if filter.TimeRange != "" && filter.From == nil {
		from := parseTimeRange(filter.TimeRange)
		filter.From = &from
	}

	if filter.Source != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argNum))
		args = append(args, filter.Source)
		argNum++
	}

	// Severity filtering
	if filter.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argNum))
		args = append(args, filter.Severity)
		argNum++
	}

	if len(filter.Severities) > 0 {
		conditions = append(conditions, fmt.Sprintf("severity = ANY($%d)", argNum))
		args = append(args, filter.Severities)
		argNum++
	}

	if filter.StatusMin > 0 {
		conditions = append(conditions, fmt.Sprintf("status_code >= $%d", argNum))
		args = append(args, filter.StatusMin)
		argNum++
	}

	if filter.StatusMax > 0 {
		conditions = append(conditions, fmt.Sprintf("status_code <= $%d", argNum))
		args = append(args, filter.StatusMax)
		argNum++
	}

	if len(filter.Methods) > 0 {
		conditions = append(conditions, fmt.Sprintf("method = ANY($%d)", argNum))
		args = append(args, filter.Methods)
		argNum++
	}

	if filter.PathPattern != "" {
		conditions = append(conditions, fmt.Sprintf("path LIKE $%d", argNum))
		args = append(args, "%"+filter.PathPattern+"%")
		argNum++
	}

	if filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(event_message ILIKE $%d OR path ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+filter.Query+"%")
		argNum++
	}

	// Regex filtering for event_message
	if filter.Regex != "" {
		conditions = append(conditions, fmt.Sprintf("event_message ~ $%d", argNum))
		args = append(args, filter.Regex)
		argNum++
	}

	// User ID filtering
	if filter.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
		args = append(args, filter.UserID)
		argNum++
	}

	// Request ID filtering for tracing
	if filter.RequestID != "" {
		conditions = append(conditions, fmt.Sprintf("request_id = $%d", argNum))
		args = append(args, filter.RequestID)
		argNum++
	}

	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		args = append(args, *filter.From)
		argNum++
	}

	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		args = append(args, *filter.To)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM analytics.logs %s", whereClause)
	var total int
	err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count logs: %w", err)
	}

	// Set defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Get logs
	sql := fmt.Sprintf(`
		SELECT id, timestamp, event_message, request_id, method, path, status_code,
			   source, severity, user_id, user_agent, apikey, request_headers, response_headers,
			   duration_ms, metadata, search
		FROM analytics.logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list logs: %w", err)
	}
	defer rows.Close()

	logs := make([]*store.LogEntry, 0)
	for rows.Next() {
		var entry store.LogEntry
		var requestHeaders, responseHeaders, metadata []byte
		var severity *string

		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.EventMessage, &entry.RequestID,
			&entry.Method, &entry.Path, &entry.StatusCode, &entry.Source,
			&severity, &entry.UserID, &entry.UserAgent, &entry.APIKey, &requestHeaders,
			&responseHeaders, &entry.DurationMs, &metadata, &entry.Search,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan log: %w", err)
		}

		if severity != nil {
			entry.Severity = *severity
		}
		if len(requestHeaders) > 0 {
			json.Unmarshal(requestHeaders, &entry.RequestHeaders)
		}
		if len(responseHeaders) > 0 {
			json.Unmarshal(responseHeaders, &entry.ResponseHeaders)
		}
		if len(metadata) > 0 {
			json.Unmarshal(metadata, &entry.Metadata)
		}

		logs = append(logs, &entry)
	}

	return logs, total, nil
}

// GetHistogram retrieves log counts grouped by time interval.
func (s *LogsStore) GetHistogram(ctx context.Context, filter *store.LogFilter, interval string) ([]store.LogHistogramBucket, error) {
	// Parse interval (1m, 5m, 15m, 1h, 6h, 1d)
	var truncUnit string
	switch interval {
	case "1m":
		truncUnit = "minute"
	case "5m":
		truncUnit = "5 minute"
	case "15m":
		truncUnit = "15 minute"
	case "1h":
		truncUnit = "hour"
	case "6h":
		truncUnit = "6 hour"
	case "1d":
		truncUnit = "day"
	default:
		truncUnit = "5 minute"
	}

	// Build WHERE clause
	conditions := []string{}
	args := []any{}
	argNum := 1

	// Apply time filter
	if filter.TimeRange != "" && filter.From == nil {
		from := parseTimeRange(filter.TimeRange)
		filter.From = &from
	}

	if filter.Source != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argNum))
		args = append(args, filter.Source)
		argNum++
	}

	// Severity filtering for histogram
	if filter.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argNum))
		args = append(args, filter.Severity)
		argNum++
	}

	if len(filter.Severities) > 0 {
		conditions = append(conditions, fmt.Sprintf("severity = ANY($%d)", argNum))
		args = append(args, filter.Severities)
		argNum++
	}

	if filter.StatusMin > 0 {
		conditions = append(conditions, fmt.Sprintf("status_code >= $%d", argNum))
		args = append(args, filter.StatusMin)
		argNum++
	}

	if filter.StatusMax > 0 {
		conditions = append(conditions, fmt.Sprintf("status_code <= $%d", argNum))
		args = append(args, filter.StatusMax)
		argNum++
	}

	if len(filter.Methods) > 0 {
		conditions = append(conditions, fmt.Sprintf("method = ANY($%d)", argNum))
		args = append(args, filter.Methods)
		argNum++
	}

	if filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(event_message ILIKE $%d OR path ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+filter.Query+"%")
		argNum++
	}

	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		args = append(args, *filter.From)
		argNum++
	}

	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		args = append(args, *filter.To)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Use date_trunc for standard intervals, date_bin for custom intervals
	var sql string
	if truncUnit == "5 minute" || truncUnit == "15 minute" || truncUnit == "6 hour" {
		// Use date_bin for non-standard intervals
		intervalStr := strings.Replace(truncUnit, " minute", " minutes", 1)
		intervalStr = strings.Replace(intervalStr, " hour", " hours", 1)
		sql = fmt.Sprintf(`
			SELECT date_bin('%s', timestamp, '2000-01-01') as bucket, COUNT(*) as count
			FROM analytics.logs
			%s
			GROUP BY bucket
			ORDER BY bucket ASC`, intervalStr, whereClause)
	} else {
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*) as count
			FROM analytics.logs
			%s
			GROUP BY bucket
			ORDER BY bucket ASC`, truncUnit, whereClause)
	}

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get histogram: %w", err)
	}
	defer rows.Close()

	buckets := make([]store.LogHistogramBucket, 0)
	for rows.Next() {
		var bucket store.LogHistogramBucket
		err := rows.Scan(&bucket.Timestamp, &bucket.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan histogram bucket: %w", err)
		}
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// CreateSavedQuery creates a new saved query.
func (s *LogsStore) CreateSavedQuery(ctx context.Context, query *store.SavedQuery) error {
	params, err := json.Marshal(query.QueryParams)
	if err != nil {
		return fmt.Errorf("failed to marshal query params: %w", err)
	}

	sql := `
		INSERT INTO analytics.saved_queries (name, description, query_params)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	err = s.pool.QueryRow(ctx, sql, query.Name, query.Description, params).
		Scan(&query.ID, &query.CreatedAt, &query.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create saved query: %w", err)
	}

	return nil
}

// GetSavedQuery retrieves a saved query by ID.
func (s *LogsStore) GetSavedQuery(ctx context.Context, id string) (*store.SavedQuery, error) {
	sql := `
		SELECT id, name, description, query_params, created_at, updated_at
		FROM analytics.saved_queries
		WHERE id = $1`

	var query store.SavedQuery
	var params []byte

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&query.ID, &query.Name, &query.Description, &params,
		&query.CreatedAt, &query.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get saved query: %w", err)
	}

	if len(params) > 0 {
		json.Unmarshal(params, &query.QueryParams)
	}

	return &query, nil
}

// ListSavedQueries retrieves all saved queries.
func (s *LogsStore) ListSavedQueries(ctx context.Context) ([]*store.SavedQuery, error) {
	sql := `
		SELECT id, name, description, query_params, created_at, updated_at
		FROM analytics.saved_queries
		ORDER BY updated_at DESC`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to list saved queries: %w", err)
	}
	defer rows.Close()

	queries := make([]*store.SavedQuery, 0)
	for rows.Next() {
		var query store.SavedQuery
		var params []byte

		err := rows.Scan(
			&query.ID, &query.Name, &query.Description, &params,
			&query.CreatedAt, &query.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan saved query: %w", err)
		}

		if len(params) > 0 {
			json.Unmarshal(params, &query.QueryParams)
		}

		queries = append(queries, &query)
	}

	return queries, nil
}

// UpdateSavedQuery updates an existing saved query.
func (s *LogsStore) UpdateSavedQuery(ctx context.Context, query *store.SavedQuery) error {
	params, err := json.Marshal(query.QueryParams)
	if err != nil {
		return fmt.Errorf("failed to marshal query params: %w", err)
	}

	sql := `
		UPDATE analytics.saved_queries
		SET name = $2, description = $3, query_params = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err = s.pool.QueryRow(ctx, sql, query.ID, query.Name, query.Description, params).
		Scan(&query.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update saved query: %w", err)
	}

	return nil
}

// DeleteSavedQuery deletes a saved query by ID.
func (s *LogsStore) DeleteSavedQuery(ctx context.Context, id string) error {
	sql := `DELETE FROM analytics.saved_queries WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("failed to delete saved query: %w", err)
	}
	return nil
}

// ListQueryTemplates retrieves all predefined query templates.
func (s *LogsStore) ListQueryTemplates(ctx context.Context) ([]*store.QueryTemplate, error) {
	sql := `
		SELECT id, name, description, query_params, category
		FROM analytics.query_templates
		ORDER BY category, name`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to list query templates: %w", err)
	}
	defer rows.Close()

	templates := make([]*store.QueryTemplate, 0)
	for rows.Next() {
		var tpl store.QueryTemplate
		var params []byte

		err := rows.Scan(&tpl.ID, &tpl.Name, &tpl.Description, &params, &tpl.Category)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query template: %w", err)
		}

		if len(params) > 0 {
			json.Unmarshal(params, &tpl.QueryParams)
		}

		templates = append(templates, &tpl)
	}

	return templates, nil
}

// parseTimeRange converts a time range string to a time.Time.
func parseTimeRange(tr string) time.Time {
	now := time.Now()
	switch tr {
	case "5m":
		return now.Add(-5 * time.Minute)
	case "15m":
		return now.Add(-15 * time.Minute)
	case "1h":
		return now.Add(-1 * time.Hour)
	case "3h":
		return now.Add(-3 * time.Hour)
	case "24h":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-1 * time.Hour)
	}
}
