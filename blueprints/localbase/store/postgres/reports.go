package postgres

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportsStore implements store.ReportsStore using PostgreSQL.
type ReportsStore struct {
	pool *pgxpool.Pool
}

// Default report configurations
var defaultReportConfigs = map[string]*store.ReportConfig{
	"database": {
		ID:          "database-default",
		Name:        "Database Report",
		Description: "PostgreSQL performance metrics",
		ReportType:  "database",
		IsDefault:   true,
		Charts: []store.ChartConfig{
			{ID: "requests", Title: "Database Requests", Type: store.ChartTypeLine, Metric: "database.requests", Unit: ""},
			{ID: "response_time", Title: "Response Time (p95)", Type: store.ChartTypeLine, Metric: "database.response_time_p95", Unit: "ms"},
			{ID: "errors", Title: "Errors", Type: store.ChartTypeLine, Metric: "database.errors", Unit: ""},
			{ID: "connections", Title: "Active Queries", Type: store.ChartTypeLine, Metric: "database.connections", Unit: ""},
		},
	},
	"auth": {
		ID:          "auth-default",
		Name:        "Auth Report",
		Description: "Authentication metrics",
		ReportType:  "auth",
		IsDefault:   true,
		Charts: []store.ChartConfig{
			{ID: "signins", Title: "Sign-ins", Type: store.ChartTypeBar, Metric: "auth.signins", Unit: ""},
			{ID: "signups", Title: "New Registrations", Type: store.ChartTypeBar, Metric: "auth.signups", Unit: ""},
			{ID: "errors", Title: "Auth Errors", Type: store.ChartTypeLine, Metric: "auth.errors", Unit: ""},
			{ID: "requests", Title: "Total Requests", Type: store.ChartTypeLine, Metric: "auth.requests", Unit: ""},
		},
	},
	"storage": {
		ID:          "storage-default",
		Name:        "Storage Report",
		Description: "Object storage metrics",
		ReportType:  "storage",
		IsDefault:   true,
		Charts: []store.ChartConfig{
			{ID: "requests", Title: "Total Requests", Type: store.ChartTypeLine, Metric: "storage.requests", Unit: ""},
			{ID: "uploads", Title: "Uploads", Type: store.ChartTypeBar, Metric: "storage.uploads", Unit: ""},
			{ID: "downloads", Title: "Downloads", Type: store.ChartTypeBar, Metric: "storage.downloads", Unit: ""},
			{ID: "response_time", Title: "Response Time (p95)", Type: store.ChartTypeLine, Metric: "storage.response_time_p95", Unit: "ms"},
		},
	},
	"realtime": {
		ID:          "realtime-default",
		Name:        "Realtime Report",
		Description: "WebSocket metrics",
		ReportType:  "realtime",
		IsDefault:   true,
		Charts: []store.ChartConfig{
			{ID: "connections", Title: "WebSocket Events", Type: store.ChartTypeLine, Metric: "realtime.connections", Unit: ""},
			{ID: "messages", Title: "Messages", Type: store.ChartTypeLine, Metric: "realtime.messages", Unit: ""},
		},
	},
	"functions": {
		ID:          "functions-default",
		Name:        "Edge Functions Report",
		Description: "Serverless function metrics",
		ReportType:  "functions",
		IsDefault:   true,
		Charts: []store.ChartConfig{
			{ID: "invocations", Title: "Invocations", Type: store.ChartTypeLine, Metric: "functions.invocations", Unit: ""},
			{ID: "duration", Title: "Duration (p95)", Type: store.ChartTypeLine, Metric: "functions.duration_p95", Unit: "ms"},
			{ID: "errors", Title: "Errors", Type: store.ChartTypeLine, Metric: "functions.errors", Unit: ""},
			{ID: "success_rate", Title: "Success Rate", Type: store.ChartTypeLine, Metric: "functions.success_rate", Unit: "%"},
		},
	},
	"api": {
		ID:          "api-default",
		Name:        "API Gateway Report",
		Description: "HTTP traffic metrics",
		ReportType:  "api",
		IsDefault:   true,
		Charts: []store.ChartConfig{
			{ID: "requests", Title: "Total Requests", Type: store.ChartTypeLine, Metric: "api.requests", Unit: ""},
			{ID: "response_time", Title: "Response Time", Type: store.ChartTypeLine, Metrics: []string{"api.response_time_p50", "api.response_time_p95"}, Unit: "ms"},
			{ID: "errors", Title: "Error Rate", Type: store.ChartTypeArea, Metrics: []string{"api.4xx", "api.5xx"}, Unit: ""},
			{ID: "methods", Title: "By Method", Type: store.ChartTypeStackedBar, Metrics: []string{"api.get", "api.post", "api.put", "api.delete"}, Unit: ""},
		},
	},
}

// GetDefaultReportConfig returns the default configuration for a report type.
func (s *ReportsStore) GetDefaultReportConfig(ctx context.Context, reportType string) (*store.ReportConfig, error) {
	config, ok := defaultReportConfigs[reportType]
	if !ok {
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
	return config, nil
}

// ListReportConfigs returns all available report configurations.
func (s *ReportsStore) ListReportConfigs(ctx context.Context) ([]*store.ReportConfig, error) {
	configs := make([]*store.ReportConfig, 0, len(defaultReportConfigs))
	for _, config := range defaultReportConfigs {
		configs = append(configs, config)
	}
	// Sort by name for consistent ordering
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})
	return configs, nil
}

// GetMetricTimeSeries retrieves time series data for a single metric.
func (s *ReportsStore) GetMetricTimeSeries(ctx context.Context, metric string, from, to time.Time, interval string) ([]store.MetricDataPoint, error) {
	// Parse metric name: source.metric_name
	parts := strings.SplitN(metric, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid metric name: %s", metric)
	}
	source := parts[0]
	metricName := parts[1]

	// Build the aggregation query based on metric
	var sql string
	var args []any

	truncInterval := intervalToTrunc(interval)

	switch {
	case metricName == "requests":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = $3
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to, mapSourceName(source)}

	case metricName == "errors":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = $3 AND status_code >= 400
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to, mapSourceName(source)}

	case metricName == "signins":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = 'auth' AND path LIKE '%%/token%%' AND method = 'POST'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "signups":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = 'auth' AND path LIKE '%%/signup%%' AND method = 'POST'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "uploads":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = 'storage' AND method = 'POST'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "downloads":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = 'storage' AND method = 'GET'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "invocations":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = 'functions'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "connections" || metricName == "messages":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = 'realtime'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case strings.HasPrefix(metricName, "response_time_p") || metricName == "duration_p95":
		percentile := 0.95
		if strings.HasSuffix(metricName, "p50") {
			percentile = 0.50
		} else if strings.HasSuffix(metricName, "p99") {
			percentile = 0.99
		}
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket,
				   COALESCE(percentile_cont($4) WITHIN GROUP (ORDER BY duration_ms), 0)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = $3 AND duration_ms IS NOT NULL
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to, mapSourceName(source), percentile}

	case metricName == "success_rate":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket,
				   CASE WHEN COUNT(*) > 0
				        THEN (COUNT(*) FILTER (WHERE status_code < 400))::float / COUNT(*)::float * 100
				        ELSE 100 END as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = $3
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to, mapSourceName(source)}

	case metricName == "4xx":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND status_code >= 400 AND status_code < 500
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "5xx":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND status_code >= 500
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "get":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND method = 'GET'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "post":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND method = 'POST'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "put":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND method = 'PUT'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	case metricName == "delete":
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND method = 'DELETE'
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to}

	default:
		// Default to counting all logs for the source
		sql = fmt.Sprintf(`
			SELECT date_trunc('%s', timestamp) as bucket, COUNT(*)::float as value
			FROM analytics.logs
			WHERE timestamp >= $1 AND timestamp <= $2 AND source = $3
			GROUP BY bucket ORDER BY bucket`, truncInterval)
		args = []any{from, to, mapSourceName(source)}
	}

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metric %s: %w", metric, err)
	}
	defer rows.Close()

	var dataPoints []store.MetricDataPoint
	for rows.Next() {
		var dp store.MetricDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.Value); err != nil {
			return nil, fmt.Errorf("failed to scan metric data: %w", err)
		}
		// Round to 2 decimal places
		dp.Value = math.Round(dp.Value*100) / 100
		dataPoints = append(dataPoints, dp)
	}

	// Fill in missing buckets with zero values
	dataPoints = fillMissingBuckets(dataPoints, from, to, interval)

	return dataPoints, nil
}

// GetMultiMetricTimeSeries retrieves time series data for multiple metrics.
func (s *ReportsStore) GetMultiMetricTimeSeries(ctx context.Context, metrics []string, from, to time.Time, interval string) (map[string][]store.MetricDataPoint, error) {
	result := make(map[string][]store.MetricDataPoint)
	for _, metric := range metrics {
		data, err := s.GetMetricTimeSeries(ctx, metric, from, to, interval)
		if err != nil {
			return nil, err
		}
		result[metric] = data
	}
	return result, nil
}

// GetPrometheusMetrics returns metrics in Prometheus text format.
func (s *ReportsStore) GetPrometheusMetrics(ctx context.Context) (string, error) {
	var b strings.Builder

	// Get current metrics from logs
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// Total requests by source
	b.WriteString("# HELP localbase_http_requests_total Total HTTP requests\n")
	b.WriteString("# TYPE localbase_http_requests_total counter\n")

	rows, err := s.pool.Query(ctx, `
		SELECT source, method, status_code, COUNT(*)
		FROM analytics.logs
		WHERE timestamp >= $1
		GROUP BY source, method, status_code`, oneHourAgo)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source, method string
			var statusCode *int
			var count int64
			if err := rows.Scan(&source, &method, &statusCode, &count); err == nil {
				status := "0"
				if statusCode != nil {
					status = fmt.Sprintf("%d", *statusCode)
				}
				if method == "" {
					method = "UNKNOWN"
				}
				b.WriteString(fmt.Sprintf("localbase_http_requests_total{source=\"%s\",method=\"%s\",status=\"%s\"} %d\n", source, method, status, count))
			}
		}
	}

	// Response time percentiles
	b.WriteString("\n# HELP localbase_http_request_duration_ms HTTP request duration in milliseconds\n")
	b.WriteString("# TYPE localbase_http_request_duration_ms gauge\n")

	rows, err = s.pool.Query(ctx, `
		SELECT source,
			   percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_ms) as p50,
			   percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95,
			   percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99
		FROM analytics.logs
		WHERE timestamp >= $1 AND duration_ms IS NOT NULL
		GROUP BY source`, oneHourAgo)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source string
			var p50, p95, p99 *float64
			if err := rows.Scan(&source, &p50, &p95, &p99); err == nil {
				if p50 != nil {
					b.WriteString(fmt.Sprintf("localbase_http_request_duration_ms{source=\"%s\",quantile=\"0.5\"} %.2f\n", source, *p50))
				}
				if p95 != nil {
					b.WriteString(fmt.Sprintf("localbase_http_request_duration_ms{source=\"%s\",quantile=\"0.95\"} %.2f\n", source, *p95))
				}
				if p99 != nil {
					b.WriteString(fmt.Sprintf("localbase_http_request_duration_ms{source=\"%s\",quantile=\"0.99\"} %.2f\n", source, *p99))
				}
			}
		}
	}

	// Error rate
	b.WriteString("\n# HELP localbase_http_errors_total Total HTTP errors\n")
	b.WriteString("# TYPE localbase_http_errors_total counter\n")

	rows, err = s.pool.Query(ctx, `
		SELECT source, COUNT(*)
		FROM analytics.logs
		WHERE timestamp >= $1 AND status_code >= 400
		GROUP BY source`, oneHourAgo)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source string
			var count int64
			if err := rows.Scan(&source, &count); err == nil {
				b.WriteString(fmt.Sprintf("localbase_http_errors_total{source=\"%s\"} %d\n", source, count))
			}
		}
	}

	return b.String(), nil
}

// GenerateReport generates a complete report with data for all charts.
func (s *ReportsStore) GenerateReport(ctx context.Context, reportType string, from, to time.Time, interval string) (*store.Report, error) {
	config, err := s.GetDefaultReportConfig(ctx, reportType)
	if err != nil {
		return nil, err
	}

	report := &store.Report{
		ReportType: reportType,
		From:       from,
		To:         to,
		Interval:   interval,
		Charts:     make([]store.ChartData, 0, len(config.Charts)),
	}

	for _, chartConfig := range config.Charts {
		chartData := store.ChartData{
			ChartConfig: chartConfig,
		}

		if len(chartConfig.Metrics) > 0 {
			// Multi-metric chart
			multiData, err := s.GetMultiMetricTimeSeries(ctx, chartConfig.Metrics, from, to, interval)
			if err != nil {
				return nil, fmt.Errorf("failed to get multi-metric data for chart %s: %w", chartConfig.ID, err)
			}

			// Combine into data points with Values map
			if len(multiData) > 0 {
				// Get timestamps from first metric
				var firstMetric string
				for m := range multiData {
					firstMetric = m
					break
				}
				for i, dp := range multiData[firstMetric] {
					combinedDP := store.MetricDataPoint{
						Timestamp: dp.Timestamp,
						Values:    make(map[string]float64),
					}
					for metric, points := range multiData {
						if i < len(points) {
							// Use short metric name as key
							parts := strings.SplitN(metric, ".", 2)
							key := metric
							if len(parts) == 2 {
								key = parts[1]
							}
							combinedDP.Values[key] = points[i].Value
						}
					}
					chartData.Data = append(chartData.Data, combinedDP)
				}
			}
		} else if chartConfig.Metric != "" {
			// Single metric chart
			data, err := s.GetMetricTimeSeries(ctx, chartConfig.Metric, from, to, interval)
			if err != nil {
				return nil, fmt.Errorf("failed to get metric data for chart %s: %w", chartConfig.ID, err)
			}
			chartData.Data = data
		}

		report.Charts = append(report.Charts, chartData)
	}

	return report, nil
}

// Helper functions

func mapSourceName(source string) string {
	switch source {
	case "database":
		return "postgrest"
	case "api":
		return "edge"
	default:
		return source
	}
}

func intervalToTrunc(interval string) string {
	switch interval {
	case "1m":
		return "minute"
	case "5m":
		return "minute" // Will need date_bin for 5m
	case "15m":
		return "minute"
	case "1h":
		return "hour"
	case "6h":
		return "hour"
	case "1d":
		return "day"
	default:
		return "hour"
	}
}

func fillMissingBuckets(dataPoints []store.MetricDataPoint, from, to time.Time, interval string) []store.MetricDataPoint {
	if len(dataPoints) == 0 {
		// Generate empty buckets
		return generateEmptyBuckets(from, to, interval)
	}

	// Create a map of existing timestamps
	existing := make(map[int64]store.MetricDataPoint)
	for _, dp := range dataPoints {
		existing[dp.Timestamp.Unix()] = dp
	}

	// Generate all expected buckets
	var step time.Duration
	switch interval {
	case "1m":
		step = time.Minute
	case "5m":
		step = 5 * time.Minute
	case "15m":
		step = 15 * time.Minute
	case "1h":
		step = time.Hour
	case "6h":
		step = 6 * time.Hour
	case "1d":
		step = 24 * time.Hour
	default:
		step = time.Hour
	}

	var result []store.MetricDataPoint
	current := from.Truncate(step)
	for current.Before(to) || current.Equal(to) {
		if dp, ok := existing[current.Unix()]; ok {
			result = append(result, dp)
		} else {
			result = append(result, store.MetricDataPoint{
				Timestamp: current,
				Value:     0,
			})
		}
		current = current.Add(step)
	}

	return result
}

func generateEmptyBuckets(from, to time.Time, interval string) []store.MetricDataPoint {
	var step time.Duration
	switch interval {
	case "1m":
		step = time.Minute
	case "5m":
		step = 5 * time.Minute
	case "15m":
		step = 15 * time.Minute
	case "1h":
		step = time.Hour
	case "6h":
		step = 6 * time.Hour
	case "1d":
		step = 24 * time.Hour
	default:
		step = time.Hour
	}

	var result []store.MetricDataPoint
	current := from.Truncate(step)
	for current.Before(to) || current.Equal(to) {
		result = append(result, store.MetricDataPoint{
			Timestamp: current,
			Value:     0,
		})
		current = current.Add(step)
	}
	return result
}
