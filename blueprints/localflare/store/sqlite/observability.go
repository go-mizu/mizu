package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// ObservabilityStoreImpl implements store.ObservabilityStore.
type ObservabilityStoreImpl struct {
	db *sql.DB
}

// WriteLog writes a log entry.
func (s *ObservabilityStoreImpl) WriteLog(ctx context.Context, log *store.Log) error {
	metadata, _ := json.Marshal(log.Metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO observability_logs (id, worker_id, worker_name, level, message, request_id, trace_id, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.WorkerID, log.WorkerName, log.Level, log.Message, log.RequestID, log.TraceID,
		log.Timestamp, string(metadata))
	return err
}

// WriteLogs writes multiple log entries.
func (s *ObservabilityStoreImpl) WriteLogs(ctx context.Context, logs []*store.Log) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO observability_logs (id, worker_id, worker_name, level, message, request_id, trace_id, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, log := range logs {
		metadata, _ := json.Marshal(log.Metadata)
		_, err := stmt.ExecContext(ctx, log.ID, log.WorkerID, log.WorkerName, log.Level, log.Message,
			log.RequestID, log.TraceID, log.Timestamp, string(metadata))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// QueryLogs queries logs with filtering.
func (s *ObservabilityStoreImpl) QueryLogs(ctx context.Context, workerID string, level string, limit, offset int) ([]*store.Log, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, worker_id, worker_name, level, message, request_id, trace_id, timestamp, metadata
		FROM observability_logs WHERE 1=1`
	args := []any{}

	if workerID != "" {
		query += ` AND worker_id = ?`
		args = append(args, workerID)
	}
	if level != "" {
		query += ` AND level = ?`
		args = append(args, level)
	}

	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*store.Log
	for rows.Next() {
		log, err := s.scanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// GetLogsByTraceID retrieves logs by trace ID.
func (s *ObservabilityStoreImpl) GetLogsByTraceID(ctx context.Context, traceID string) ([]*store.Log, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, worker_id, worker_name, level, message, request_id, trace_id, timestamp, metadata
		FROM observability_logs WHERE trace_id = ? ORDER BY timestamp`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*store.Log
	for rows.Next() {
		log, err := s.scanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// CreateTrace creates a new trace.
func (s *ObservabilityStoreImpl) CreateTrace(ctx context.Context, trace *store.Trace) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO observability_traces (id, trace_id, root_service, status, duration_ms, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		trace.ID, trace.TraceID, trace.RootService, trace.Status, trace.DurationMs, trace.StartedAt, trace.FinishedAt)
	return err
}

// GetTrace retrieves a trace by ID.
func (s *ObservabilityStoreImpl) GetTrace(ctx context.Context, traceID string) (*store.Trace, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, trace_id, root_service, status, duration_ms, started_at, finished_at
		FROM observability_traces WHERE trace_id = ?`, traceID)
	return s.scanTrace(row)
}

// ListTraces lists traces with pagination.
func (s *ObservabilityStoreImpl) ListTraces(ctx context.Context, limit, offset int) ([]*store.Trace, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, trace_id, root_service, status, duration_ms, started_at, finished_at
		FROM observability_traces ORDER BY started_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []*store.Trace
	for rows.Next() {
		trace, err := s.scanTrace(rows)
		if err != nil {
			return nil, err
		}
		traces = append(traces, trace)
	}
	return traces, rows.Err()
}

// UpdateTrace updates a trace.
func (s *ObservabilityStoreImpl) UpdateTrace(ctx context.Context, trace *store.Trace) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE observability_traces SET status = ?, duration_ms = ?, finished_at = ? WHERE trace_id = ?`,
		trace.Status, trace.DurationMs, trace.FinishedAt, trace.TraceID)
	return err
}

// CreateSpan creates a new trace span.
func (s *ObservabilityStoreImpl) CreateSpan(ctx context.Context, span *store.TraceSpan) error {
	attributes, _ := json.Marshal(span.Attributes)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO observability_trace_spans (id, trace_id, span_id, parent_span_id, name, service, start_time, duration_ms, status, attributes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		span.ID, span.TraceID, span.SpanID, span.ParentSpanID, span.Name, span.Service,
		span.StartTime, span.DurationMs, span.Status, string(attributes))
	return err
}

// GetSpansByTraceID retrieves all spans for a trace.
func (s *ObservabilityStoreImpl) GetSpansByTraceID(ctx context.Context, traceID string) ([]*store.TraceSpan, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, trace_id, span_id, parent_span_id, name, service, start_time, duration_ms, status, attributes
		FROM observability_trace_spans WHERE trace_id = ? ORDER BY start_time`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*store.TraceSpan
	for rows.Next() {
		span, err := s.scanSpan(rows)
		if err != nil {
			return nil, err
		}
		spans = append(spans, span)
	}
	return spans, rows.Err()
}

// WriteMetric writes a metric data point.
func (s *ObservabilityStoreImpl) WriteMetric(ctx context.Context, metric *store.Metric) error {
	tags, _ := json.Marshal(metric.Tags)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO observability_metrics (id, name, value, tags, timestamp)
		VALUES (?, ?, ?, ?, ?)`,
		metric.ID, metric.Name, metric.Value, string(tags), metric.Timestamp)
	return err
}

// WriteMetrics writes multiple metric data points.
func (s *ObservabilityStoreImpl) WriteMetrics(ctx context.Context, metrics []*store.Metric) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO observability_metrics (id, name, value, tags, timestamp) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, metric := range metrics {
		tags, _ := json.Marshal(metric.Tags)
		_, err := stmt.ExecContext(ctx, metric.ID, metric.Name, metric.Value, string(tags), metric.Timestamp)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// QueryMetrics queries metrics by name and time range.
func (s *ObservabilityStoreImpl) QueryMetrics(ctx context.Context, name string, start, end time.Time, limit int) ([]*store.Metric, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, value, tags, timestamp FROM observability_metrics
		WHERE name = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp LIMIT ?`,
		name, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*store.Metric
	for rows.Next() {
		metric, err := s.scanMetric(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, rows.Err()
}

// AggregateMetrics aggregates metrics by name and time range.
func (s *ObservabilityStoreImpl) AggregateMetrics(ctx context.Context, name string, start, end time.Time, aggregation string) (float64, error) {
	var query string
	switch aggregation {
	case "sum":
		query = `SELECT COALESCE(SUM(value), 0) FROM observability_metrics WHERE name = ? AND timestamp >= ? AND timestamp <= ?`
	case "avg":
		query = `SELECT COALESCE(AVG(value), 0) FROM observability_metrics WHERE name = ? AND timestamp >= ? AND timestamp <= ?`
	case "min":
		query = `SELECT COALESCE(MIN(value), 0) FROM observability_metrics WHERE name = ? AND timestamp >= ? AND timestamp <= ?`
	case "max":
		query = `SELECT COALESCE(MAX(value), 0) FROM observability_metrics WHERE name = ? AND timestamp >= ? AND timestamp <= ?`
	case "count":
		query = `SELECT COUNT(*) FROM observability_metrics WHERE name = ? AND timestamp >= ? AND timestamp <= ?`
	default:
		query = `SELECT COALESCE(AVG(value), 0) FROM observability_metrics WHERE name = ? AND timestamp >= ? AND timestamp <= ?`
	}

	var result float64
	err := s.db.QueryRowContext(ctx, query, name, start, end).Scan(&result)
	return result, err
}

func (s *ObservabilityStoreImpl) scanLog(row scanner) (*store.Log, error) {
	var log store.Log
	var requestID, traceID, metadata sql.NullString
	if err := row.Scan(&log.ID, &log.WorkerID, &log.WorkerName, &log.Level, &log.Message,
		&requestID, &traceID, &log.Timestamp, &metadata); err != nil {
		return nil, err
	}
	log.RequestID = requestID.String
	log.TraceID = traceID.String
	if metadata.Valid && metadata.String != "" {
		json.Unmarshal([]byte(metadata.String), &log.Metadata)
	}
	return &log, nil
}

func (s *ObservabilityStoreImpl) scanTrace(row scanner) (*store.Trace, error) {
	var trace store.Trace
	var finishedAt sql.NullTime
	if err := row.Scan(&trace.ID, &trace.TraceID, &trace.RootService, &trace.Status,
		&trace.DurationMs, &trace.StartedAt, &finishedAt); err != nil {
		return nil, err
	}
	if finishedAt.Valid {
		trace.FinishedAt = &finishedAt.Time
	}
	return &trace, nil
}

func (s *ObservabilityStoreImpl) scanSpan(row scanner) (*store.TraceSpan, error) {
	var span store.TraceSpan
	var parentSpanID, status, attributes sql.NullString
	if err := row.Scan(&span.ID, &span.TraceID, &span.SpanID, &parentSpanID, &span.Name,
		&span.Service, &span.StartTime, &span.DurationMs, &status, &attributes); err != nil {
		return nil, err
	}
	span.ParentSpanID = parentSpanID.String
	span.Status = status.String
	if attributes.Valid && attributes.String != "" {
		json.Unmarshal([]byte(attributes.String), &span.Attributes)
	}
	return &span, nil
}

func (s *ObservabilityStoreImpl) scanMetric(row scanner) (*store.Metric, error) {
	var metric store.Metric
	var tags sql.NullString
	if err := row.Scan(&metric.ID, &metric.Name, &metric.Value, &tags, &metric.Timestamp); err != nil {
		return nil, err
	}
	if tags.Valid && tags.String != "" {
		json.Unmarshal([]byte(tags.String), &metric.Tags)
	}
	return &metric, nil
}
