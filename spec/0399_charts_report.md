# Spec 0399: Charts & Reports Dashboard

## Overview

This specification describes the Charts & Reports Dashboard feature for Localbase, modeled after Supabase's Reports & Metrics system. The feature provides comprehensive monitoring dashboards that visualize key metrics across database, auth, storage, realtime, edge functions, and API gateway systems.

## Research Summary

### Supabase Reports & Metrics

Based on research from [Supabase Reports Documentation](https://supabase.com/docs/guides/telemetry/reports) and [Supabase Features](https://supabase.com/features/reports-and-metrics):

#### Available Report Types

1. **Database Report** - Tracks PostgreSQL performance metrics:
   - Memory usage (RAM consumption)
   - CPU usage (average percentage)
   - Disk IOPS (read/write operations with limits)
   - Database connections (pooler activity)
   - Disk throughput (bytes per second)
   - Disk size breakdown (database, WAL, system)
   - Memory component analysis (used, cache+buffers, free)

2. **Auth Report** - Tracks authentication patterns:
   - Active users count
   - Sign-in attempts by method type
   - New user registrations
   - Error rates by status code
   - Password reset request volumes

3. **Storage Report** - Monitors object storage:
   - Request volume
   - Response speed
   - Ingress/egress traffic
   - Cache hit rates
   - Frequently accessed paths

4. **Realtime Report** - Covers WebSocket activity:
   - WebSocket connection counts
   - Broadcast/presence/database change events
   - Channel join frequency
   - Message payload sizes
   - RLS policy execution performance

5. **Edge Functions Report** - Displays serverless function metrics:
   - Execution status codes
   - Duration metrics (p50, p95, p99)
   - Regional invocation distribution
   - Cold start times

6. **API Gateway Report** - Analyzes traffic patterns:
   - Total requests count
   - Error rates (4XX/5XX)
   - Response times (p50, p95, p99)
   - Network traffic (ingress/egress)
   - Top endpoint usage

#### Time Range Support

| Duration | Free | Pro | Team | Enterprise |
|----------|------|-----|------|------------|
| Up to 24 hours | ✅ | ✅ | ✅ | ✅ |
| Up to 7 days | ❌ | ✅ | ✅ | ✅ |
| Up to 14 days | ❌ | ❌ | ✅ | ✅ |
| Up to 28 days | ❌ | ❌ | ✅ | ✅ |

#### Metrics API

Supabase exposes a Prometheus-compatible Metrics API endpoint at:
`https://<project-ref>.supabase.co/customer/v1/privileged/metrics`

- Uses HTTP Basic Auth (service_role credentials)
- Exposes ~200 Postgres performance and health series
- Recommended scrape interval: 1 minute
- Compatible with Grafana, Datadog, Prometheus

#### Chart Implementation

Supabase uses **Recharts** (MIT-licensed React chart library) for visualizations:
- Line charts for time series data
- Area charts for stacked metrics
- Bar charts for categorical data
- Configurable chart headers and tooltips

### Integration with Existing Localbase Architecture

The existing `analytics.logs` table and logging middleware already capture:
- HTTP request/response metrics (status, method, path, duration)
- Source classification (edge, postgres, auth, storage, realtime, functions)
- User and API key tracking
- Timestamp-based querying with histogram support

This provides a solid foundation for building aggregated reports.

## Goals

1. Provide 6 dedicated monitoring reports matching Supabase's feature set
2. Support time-series visualization with configurable intervals
3. Enable metric aggregation (count, sum, avg, p50, p95, p99)
4. Implement Prometheus-compatible metrics endpoint for external integrations
5. Match Supabase's chart UI/UX using Recharts
6. Support synchronized tooltips across multiple charts
7. Enable custom report configurations

## Non-Goals

1. Real-time streaming dashboards (WebSocket-based) - future enhancement
2. Alerting system - future enhancement
3. External metrics ingestion (e.g., from pg_stat_statements) - simplified for Localbase
4. Multi-project comparison views

## Design

### Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                         Localbase Server                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Metrics Collection Layer                   │ │
│  │                                                               │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │ │
│  │  │ Database │ │   Auth   │ │ Storage  │ │ Realtime │        │ │
│  │  │ Metrics  │ │ Metrics  │ │ Metrics  │ │ Metrics  │  ...   │ │
│  │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘        │ │
│  │       └────────────┼───────────┼────────────┘               │ │
│  │                    │           │                              │ │
│  │                    ▼           ▼                              │ │
│  │  ┌──────────────────────────────────────────────────────────┐│ │
│  │  │              Metrics Aggregation Service                  ││ │
│  │  │  - Aggregates from analytics.logs                        ││ │
│  │  │  - Computes statistics (count, sum, avg, percentiles)    ││ │
│  │  │  - Caches frequently requested aggregations              ││ │
│  │  └──────────────────────────────────────────────────────────┘│ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                       Reports API                             │ │
│  │   /api/reports/database                                       │ │
│  │   /api/reports/auth                                           │ │
│  │   /api/reports/storage                                        │ │
│  │   /api/reports/realtime                                       │ │
│  │   /api/reports/functions                                      │ │
│  │   /api/reports/api                                            │ │
│  │   /customer/v1/privileged/metrics (Prometheus format)         │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Create reports schema for aggregated metrics
CREATE SCHEMA IF NOT EXISTS reports;

-- Aggregated metrics table for hourly rollups
CREATE TABLE reports.metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(100) NOT NULL,
    source VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    interval VARCHAR(10) NOT NULL, -- '1m', '5m', '1h', '1d'

    -- Aggregation values
    count BIGINT DEFAULT 0,
    sum_value DOUBLE PRECISION DEFAULT 0,
    avg_value DOUBLE PRECISION DEFAULT 0,
    min_value DOUBLE PRECISION,
    max_value DOUBLE PRECISION,
    p50_value DOUBLE PRECISION,
    p95_value DOUBLE PRECISION,
    p99_value DOUBLE PRECISION,

    -- Dimensional breakdown
    dimensions JSONB DEFAULT '{}',

    -- Constraints
    UNIQUE(metric_name, source, timestamp, interval, dimensions)
);

-- Indexes for efficient querying
CREATE INDEX idx_metrics_name_ts ON reports.metrics (metric_name, timestamp DESC);
CREATE INDEX idx_metrics_source_ts ON reports.metrics (source, timestamp DESC);
CREATE INDEX idx_metrics_interval ON reports.metrics (interval, timestamp DESC);
CREATE INDEX idx_metrics_dimensions ON reports.metrics USING GIN (dimensions);

-- Report configurations table
CREATE TABLE reports.configs (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    report_type VARCHAR(50) NOT NULL, -- 'database', 'auth', 'storage', etc.
    charts JSONB NOT NULL, -- Array of chart configurations
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default report configurations
INSERT INTO reports.configs (id, name, description, report_type, charts, is_default) VALUES
-- Database Report
('database-default', 'Database Report', 'PostgreSQL performance metrics', 'database', '[
    {"id": "cpu_usage", "title": "CPU Usage", "type": "line", "metric": "database.cpu_percent", "unit": "%"},
    {"id": "memory_usage", "title": "Memory Usage", "type": "area", "metric": "database.memory_percent", "unit": "%"},
    {"id": "disk_iops", "title": "Disk IOPS", "type": "line", "metric": "database.disk_iops", "unit": "ops/s"},
    {"id": "connections", "title": "Active Connections", "type": "line", "metric": "database.connections", "unit": ""},
    {"id": "disk_size", "title": "Disk Size", "type": "stacked_area", "metrics": ["database.disk_data", "database.disk_wal", "database.disk_system"], "unit": "bytes"},
    {"id": "query_time", "title": "Query Time (p95)", "type": "line", "metric": "database.query_time_p95", "unit": "ms"}
]', true),

-- Auth Report
('auth-default', 'Auth Report', 'Authentication metrics', 'auth', '[
    {"id": "active_users", "title": "Active Users", "type": "line", "metric": "auth.active_users", "unit": ""},
    {"id": "signins", "title": "Sign-ins", "type": "bar", "metric": "auth.signins", "unit": ""},
    {"id": "signups", "title": "New Registrations", "type": "bar", "metric": "auth.signups", "unit": ""},
    {"id": "errors", "title": "Auth Errors", "type": "line", "metric": "auth.errors", "unit": ""},
    {"id": "methods", "title": "Sign-in Methods", "type": "stacked_bar", "metrics": ["auth.email", "auth.oauth", "auth.phone"], "unit": ""}
]', true),

-- Storage Report
('storage-default', 'Storage Report', 'Object storage metrics', 'storage', '[
    {"id": "requests", "title": "Total Requests", "type": "line", "metric": "storage.requests", "unit": ""},
    {"id": "bandwidth", "title": "Bandwidth", "type": "area", "metrics": ["storage.ingress", "storage.egress"], "unit": "bytes"},
    {"id": "response_time", "title": "Response Time (p95)", "type": "line", "metric": "storage.response_time_p95", "unit": "ms"},
    {"id": "cache_hit", "title": "Cache Hit Rate", "type": "line", "metric": "storage.cache_hit_rate", "unit": "%"},
    {"id": "operations", "title": "Operations", "type": "stacked_bar", "metrics": ["storage.uploads", "storage.downloads", "storage.deletes"], "unit": ""}
]', true),

-- Realtime Report
('realtime-default', 'Realtime Report', 'WebSocket metrics', 'realtime', '[
    {"id": "connections", "title": "WebSocket Connections", "type": "line", "metric": "realtime.connections", "unit": ""},
    {"id": "messages", "title": "Messages", "type": "area", "metrics": ["realtime.broadcast", "realtime.presence", "realtime.db_changes"], "unit": ""},
    {"id": "channels", "title": "Active Channels", "type": "line", "metric": "realtime.channels", "unit": ""},
    {"id": "payload_size", "title": "Avg Payload Size", "type": "line", "metric": "realtime.payload_size_avg", "unit": "bytes"}
]', true),

-- Functions Report
('functions-default', 'Edge Functions Report', 'Serverless function metrics', 'functions', '[
    {"id": "invocations", "title": "Invocations", "type": "line", "metric": "functions.invocations", "unit": ""},
    {"id": "duration", "title": "Duration (p95)", "type": "line", "metric": "functions.duration_p95", "unit": "ms"},
    {"id": "status", "title": "Status Codes", "type": "stacked_bar", "metrics": ["functions.2xx", "functions.4xx", "functions.5xx"], "unit": ""},
    {"id": "cold_starts", "title": "Cold Starts", "type": "line", "metric": "functions.cold_starts", "unit": ""}
]', true),

-- API Report
('api-default', 'API Gateway Report', 'HTTP traffic metrics', 'api', '[
    {"id": "requests", "title": "Total Requests", "type": "line", "metric": "api.requests", "unit": ""},
    {"id": "response_time", "title": "Response Time", "type": "line", "metrics": ["api.response_time_p50", "api.response_time_p95", "api.response_time_p99"], "unit": "ms"},
    {"id": "errors", "title": "Error Rate", "type": "area", "metrics": ["api.4xx", "api.5xx"], "unit": ""},
    {"id": "bandwidth", "title": "Bandwidth", "type": "area", "metrics": ["api.ingress", "api.egress"], "unit": "bytes"},
    {"id": "top_endpoints", "title": "Top Endpoints", "type": "table", "metric": "api.endpoints", "unit": ""}
]', true);
```

### Metric Definitions

```go
// Metric names follow the pattern: {source}.{metric_name}

// Database metrics (aggregated from pg_stat_statements if available, otherwise simulated)
database.cpu_percent          // CPU usage percentage
database.memory_percent       // Memory usage percentage
database.memory_used          // Memory used in bytes
database.memory_cache         // Memory cache in bytes
database.disk_iops            // Disk I/O operations per second
database.disk_read_bytes      // Disk read bytes per second
database.disk_write_bytes     // Disk write bytes per second
database.connections          // Active connection count
database.disk_data            // Data file size
database.disk_wal             // WAL file size
database.disk_system          // System file size
database.query_time_p50       // Query time 50th percentile
database.query_time_p95       // Query time 95th percentile
database.query_time_p99       // Query time 99th percentile

// Auth metrics (aggregated from analytics.logs where source='auth')
auth.active_users             // Unique users with activity
auth.signins                  // Sign-in count (POST /auth/v1/token)
auth.signups                  // Sign-up count (POST /auth/v1/signup)
auth.errors                   // Error count (status >= 400)
auth.email                    // Email sign-ins
auth.oauth                    // OAuth sign-ins
auth.phone                    // Phone sign-ins

// Storage metrics (aggregated from analytics.logs where source='storage')
storage.requests              // Total request count
storage.ingress               // Upload bytes
storage.egress                // Download bytes
storage.response_time_p50     // Response time 50th percentile
storage.response_time_p95     // Response time 95th percentile
storage.cache_hit_rate        // Cache hit percentage
storage.uploads               // Upload count
storage.downloads             // Download count (GET)
storage.deletes               // Delete count

// Realtime metrics (from realtime tracking)
realtime.connections          // Active WebSocket connections
realtime.broadcast            // Broadcast message count
realtime.presence             // Presence event count
realtime.db_changes           // Database change event count
realtime.channels             // Active channel count
realtime.payload_size_avg     // Average message payload size

// Functions metrics (aggregated from analytics.logs where source='functions')
functions.invocations         // Total invocation count
functions.duration_p50        // Duration 50th percentile
functions.duration_p95        // Duration 95th percentile
functions.duration_p99        // Duration 99th percentile
functions.2xx                 // Success responses
functions.4xx                 // Client error responses
functions.5xx                 // Server error responses
functions.cold_starts         // Cold start count

// API metrics (aggregated from all analytics.logs)
api.requests                  // Total request count
api.response_time_p50         // Response time 50th percentile
api.response_time_p95         // Response time 95th percentile
api.response_time_p99         // Response time 99th percentile
api.4xx                       // 4xx error count
api.5xx                       // 5xx error count
api.ingress                   // Request bytes
api.egress                    // Response bytes
api.endpoints                 // Top endpoints by request count
```

### API Endpoints

#### Reports API

```
GET /api/reports
Response:
[
  {"id": "database", "name": "Database", "description": "PostgreSQL metrics"},
  {"id": "auth", "name": "Auth", "description": "Authentication metrics"},
  {"id": "storage", "name": "Storage", "description": "Object storage metrics"},
  {"id": "realtime", "name": "Realtime", "description": "WebSocket metrics"},
  {"id": "functions", "name": "Edge Functions", "description": "Serverless function metrics"},
  {"id": "api", "name": "API Gateway", "description": "HTTP traffic metrics"}
]
```

```
GET /api/reports/{type}
Query Parameters:
  - from: RFC3339 timestamp (default: 24 hours ago)
  - to: RFC3339 timestamp (default: now)
  - interval: string (1m, 5m, 1h, 1d - default: auto based on range)

Response:
{
  "report_type": "database",
  "from": "2026-01-16T21:00:00Z",
  "to": "2026-01-17T21:00:00Z",
  "interval": "1h",
  "charts": [
    {
      "id": "cpu_usage",
      "title": "CPU Usage",
      "type": "line",
      "unit": "%",
      "data": [
        {"timestamp": "2026-01-16T21:00:00Z", "value": 15.2},
        {"timestamp": "2026-01-16T22:00:00Z", "value": 18.7},
        ...
      ]
    },
    ...
  ]
}
```

```
GET /api/reports/{type}/chart/{chartId}
Query Parameters:
  - from: RFC3339 timestamp
  - to: RFC3339 timestamp
  - interval: string

Response:
{
  "id": "cpu_usage",
  "title": "CPU Usage",
  "type": "line",
  "unit": "%",
  "data": [...]
}
```

#### Metrics API (Prometheus-compatible)

```
GET /customer/v1/privileged/metrics
Authorization: Basic service_role:<service_role_key>

Response: (Prometheus text format)
# HELP localbase_database_connections_total Number of database connections
# TYPE localbase_database_connections_total gauge
localbase_database_connections_total 42

# HELP localbase_http_requests_total Total HTTP requests
# TYPE localbase_http_requests_total counter
localbase_http_requests_total{source="auth",method="POST",status="200"} 1523
localbase_http_requests_total{source="storage",method="GET",status="200"} 8934
...

# HELP localbase_http_request_duration_seconds HTTP request duration
# TYPE localbase_http_request_duration_seconds histogram
localbase_http_request_duration_seconds_bucket{source="auth",le="0.1"} 1200
localbase_http_request_duration_seconds_bucket{source="auth",le="0.5"} 1450
localbase_http_request_duration_seconds_bucket{source="auth",le="1.0"} 1520
localbase_http_request_duration_seconds_sum{source="auth"} 456.78
localbase_http_request_duration_seconds_count{source="auth"} 1523
...
```

#### Report Configuration API

```
GET /api/reports/configs
POST /api/reports/configs
GET /api/reports/configs/{id}
PUT /api/reports/configs/{id}
DELETE /api/reports/configs/{id}
```

### Go Types

```go
// store/store.go additions

// ChartType represents the type of chart visualization
type ChartType string

const (
    ChartTypeLine       ChartType = "line"
    ChartTypeArea       ChartType = "area"
    ChartTypeStackedArea ChartType = "stacked_area"
    ChartTypeBar        ChartType = "bar"
    ChartTypeStackedBar ChartType = "stacked_bar"
    ChartTypeTable      ChartType = "table"
)

// MetricDataPoint represents a single data point in a time series
type MetricDataPoint struct {
    Timestamp time.Time `json:"timestamp"`
    Value     float64   `json:"value"`
    // For multi-series charts
    Values    map[string]float64 `json:"values,omitempty"`
}

// ChartConfig represents configuration for a single chart
type ChartConfig struct {
    ID      string    `json:"id"`
    Title   string    `json:"title"`
    Type    ChartType `json:"type"`
    Metric  string    `json:"metric,omitempty"`
    Metrics []string  `json:"metrics,omitempty"` // For multi-series charts
    Unit    string    `json:"unit"`
}

// ChartData represents a chart with its data
type ChartData struct {
    ChartConfig
    Data []MetricDataPoint `json:"data"`
}

// ReportConfig represents a saved report configuration
type ReportConfig struct {
    ID          string        `json:"id"`
    Name        string        `json:"name"`
    Description string        `json:"description,omitempty"`
    ReportType  string        `json:"report_type"`
    Charts      []ChartConfig `json:"charts"`
    IsDefault   bool          `json:"is_default"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
}

// Report represents a complete report with data
type Report struct {
    ReportType string      `json:"report_type"`
    From       time.Time   `json:"from"`
    To         time.Time   `json:"to"`
    Interval   string      `json:"interval"`
    Charts     []ChartData `json:"charts"`
}

// MetricAggregation represents an aggregated metric value
type MetricAggregation struct {
    ID         string             `json:"id"`
    MetricName string             `json:"metric_name"`
    Source     string             `json:"source"`
    Timestamp  time.Time          `json:"timestamp"`
    Interval   string             `json:"interval"`
    Count      int64              `json:"count"`
    Sum        float64            `json:"sum"`
    Avg        float64            `json:"avg"`
    Min        *float64           `json:"min,omitempty"`
    Max        *float64           `json:"max,omitempty"`
    P50        *float64           `json:"p50,omitempty"`
    P95        *float64           `json:"p95,omitempty"`
    P99        *float64           `json:"p99,omitempty"`
    Dimensions map[string]string  `json:"dimensions,omitempty"`
}

// ReportsStore defines the interface for reports operations
type ReportsStore interface {
    // Report configs
    CreateReportConfig(ctx context.Context, config *ReportConfig) error
    GetReportConfig(ctx context.Context, id string) (*ReportConfig, error)
    GetDefaultReportConfig(ctx context.Context, reportType string) (*ReportConfig, error)
    ListReportConfigs(ctx context.Context) ([]*ReportConfig, error)
    UpdateReportConfig(ctx context.Context, config *ReportConfig) error
    DeleteReportConfig(ctx context.Context, id string) error

    // Metrics aggregation
    GetMetricTimeSeries(ctx context.Context, metric string, from, to time.Time, interval string) ([]MetricDataPoint, error)
    GetMultiMetricTimeSeries(ctx context.Context, metrics []string, from, to time.Time, interval string) (map[string][]MetricDataPoint, error)

    // Prometheus export
    GetPrometheusMetrics(ctx context.Context) (string, error)

    // Report generation
    GenerateReport(ctx context.Context, reportType string, from, to time.Time, interval string) (*Report, error)
}
```

### Frontend Components

#### Reports Page Structure

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Reports                                            [24 hours ▼] [⟳]     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │  Database   │ │    Auth     │ │   Storage   │ │  Realtime   │  ...  │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘       │
│                                                                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────────────────────┐  ┌──────────────────────────────┐    │
│  │ CPU Usage               [?]  │  │ Memory Usage            [?]  │    │
│  │                              │  │                              │    │
│  │     ╱╲    ╱╲                │  │   ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄   │    │
│  │    ╱  ╲  ╱  ╲               │  │   █████████████████████████   │    │
│  │   ╱    ╲╱    ╲              │  │   █████████████████████████   │    │
│  │  ────────────────────       │  │  ────────────────────────     │    │
│  │  12:00  14:00  16:00  18:00 │  │  12:00  14:00  16:00  18:00   │    │
│  │                      15.2%  │  │                       62.3%   │    │
│  └──────────────────────────────┘  └──────────────────────────────┘    │
│                                                                          │
│  ┌──────────────────────────────┐  ┌──────────────────────────────┐    │
│  │ Disk IOPS              [?]  │  │ Active Connections      [?]  │    │
│  │                              │  │                              │    │
│  │     ___    ___              │  │          ╱╲                  │    │
│  │    /   \  /   \             │  │         ╱  ╲                 │    │
│  │   /     \/     \            │  │   _____╱    ╲_______         │    │
│  │  ────────────────────       │  │  ────────────────────────     │    │
│  │                    450 ops/s │  │                          42  │    │
│  └──────────────────────────────┘  └──────────────────────────────┘    │
│                                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │ Query Time Distribution                                      [?]  │  │
│  │                                                                    │  │
│  │  p50 ─────────────────────────────────────────────   12ms         │  │
│  │  p95 ───────────────────────────────────────────────────  45ms    │  │
│  │  p99 ─────────────────────────────────────────────────────── 120ms│  │
│  │                                                                    │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Key Components

1. **ReportsPage** - Main container with report type tabs
2. **ReportHeader** - Title, time range selector, refresh button
3. **ChartGrid** - Responsive grid of charts
4. **LineChart** - Time series line chart (Recharts)
5. **AreaChart** - Stacked area chart
6. **BarChart** - Bar/histogram chart
7. **MetricCard** - Current metric value with sparkline
8. **ChartTooltip** - Synchronized tooltip across charts

### Implementation Plan

#### Phase 1: Backend Infrastructure
1. Add `ReportsStore` interface to `store/store.go`
2. Create `store/postgres/reports.go` with PostgreSQL implementation
3. Add schema creation for `reports` schema
4. Implement metric aggregation queries from `analytics.logs`

#### Phase 2: API Handlers
1. Create `app/web/handler/api/reports.go` with endpoints
2. Implement Prometheus metrics endpoint
3. Add routes to server configuration
4. Create background job for metric pre-aggregation (optional)

#### Phase 3: Frontend
1. Create `app/frontend/src/pages/reports/` components
2. Add `app/frontend/src/api/reports.ts` API client
3. Implement chart components using Recharts
4. Add synchronized tooltips feature
5. Integrate into sidebar navigation

#### Phase 4: Seeding & Testing
1. Add sample metrics data to seeder
2. Write integration tests for reports API
3. Add E2E tests for reports UI

## Testing Strategy

### Unit Tests
- Metric aggregation queries
- Time interval calculations
- Prometheus format generation

### Integration Tests
- Report generation with various time ranges
- Multi-metric queries
- Config CRUD operations

### E2E Tests
- Report type navigation
- Time range selection
- Chart rendering
- Tooltip synchronization

## Success Metrics

1. **Functionality**: All 6 report types render with accurate data
2. **Performance**: Report generation < 1s for 24h range
3. **Compatibility**: Prometheus endpoint works with Grafana
4. **UX**: Charts match Supabase design patterns

## Future Enhancements

1. Custom chart builder
2. Report scheduling and email delivery
3. Alerting based on metric thresholds
4. Comparison views (this week vs last week)
5. Dashboard embedding/sharing

## References

- [Supabase Reports Documentation](https://supabase.com/docs/guides/telemetry/reports)
- [Supabase Metrics API](https://supabase.com/docs/guides/telemetry/metrics)
- [Supabase Reports & Metrics Feature](https://supabase.com/features/reports-and-metrics)
- [Supabase Reports Blog Post](https://supabase.com/blog/supabase-reports-and-metrics)
- [Supabase Metrics API Blog](https://supabase.com/blog/metrics-api-observability)
- [Recharts Documentation](https://recharts.org/)
- [Prometheus Exposition Formats](https://prometheus.io/docs/instrumenting/exposition_formats/)
