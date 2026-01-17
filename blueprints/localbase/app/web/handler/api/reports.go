package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// ReportsHandler handles reports API endpoints.
type ReportsHandler struct {
	store *postgres.Store
}

// NewReportsHandler creates a new reports handler.
func NewReportsHandler(store *postgres.Store) *ReportsHandler {
	return &ReportsHandler{store: store}
}

// ListReportTypes returns available report types.
// GET /api/reports
func (h *ReportsHandler) ListReportTypes(c *mizu.Ctx) error {
	reportTypes := []store.ReportType{
		{ID: "database", Name: "Database", Description: "PostgreSQL performance metrics"},
		{ID: "auth", Name: "Auth", Description: "Authentication metrics"},
		{ID: "storage", Name: "Storage", Description: "Object storage metrics"},
		{ID: "realtime", Name: "Realtime", Description: "WebSocket metrics"},
		{ID: "functions", Name: "Edge Functions", Description: "Serverless function metrics"},
		{ID: "api", Name: "API Gateway", Description: "HTTP traffic metrics"},
	}
	return c.JSON(200, reportTypes)
}

// GetReport returns a complete report with data.
// GET /api/reports/{type}
func (h *ReportsHandler) GetReport(c *mizu.Ctx) error {
	reportType := c.Param("type")
	if reportType == "" {
		return c.JSON(400, map[string]string{"error": "report type is required"})
	}

	// Parse time range
	from, to := parseTimeRange(c)
	interval := c.Query("interval")
	if interval == "" {
		interval = autoInterval(from, to)
	}

	report, err := h.store.Reports().GenerateReport(c.Context(), reportType, from, to, interval)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, report)
}

// GetReportChart returns a single chart's data.
// GET /api/reports/{type}/chart/{chartId}
func (h *ReportsHandler) GetReportChart(c *mizu.Ctx) error {
	reportType := c.Param("type")
	chartID := c.Param("chartId")

	config, err := h.store.Reports().GetDefaultReportConfig(c.Context(), reportType)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "report type not found"})
	}

	// Find chart config
	var chartConfig *store.ChartConfig
	for _, chart := range config.Charts {
		if chart.ID == chartID {
			chartConfig = &chart
			break
		}
	}
	if chartConfig == nil {
		return c.JSON(404, map[string]string{"error": "chart not found"})
	}

	from, to := parseTimeRange(c)
	interval := c.Query("interval")
	if interval == "" {
		interval = autoInterval(from, to)
	}

	chartData := store.ChartData{
		ChartConfig: *chartConfig,
	}

	if len(chartConfig.Metrics) > 0 {
		multiData, err := h.store.Reports().GetMultiMetricTimeSeries(c.Context(), chartConfig.Metrics, from, to, interval)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		// Combine data points
		if len(multiData) > 0 {
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
						combinedDP.Values[metric] = points[i].Value
					}
				}
				chartData.Data = append(chartData.Data, combinedDP)
			}
		}
	} else {
		data, err := h.store.Reports().GetMetricTimeSeries(c.Context(), chartConfig.Metric, from, to, interval)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		chartData.Data = data
	}

	return c.JSON(200, chartData)
}

// GetMetrics returns Prometheus-compatible metrics.
// GET /customer/v1/privileged/metrics
func (h *ReportsHandler) GetMetrics(c *mizu.Ctx) error {
	metrics, err := h.store.Reports().GetPrometheusMetrics(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	c.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	return c.Text(200, metrics)
}

// ListReportConfigs returns all available report configurations.
// GET /api/reports/configs
func (h *ReportsHandler) ListReportConfigs(c *mizu.Ctx) error {
	configs, err := h.store.Reports().ListReportConfigs(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, configs)
}

// GetReportConfig returns a specific report configuration.
// GET /api/reports/configs/{type}
func (h *ReportsHandler) GetReportConfig(c *mizu.Ctx) error {
	reportType := c.Param("type")
	config, err := h.store.Reports().GetDefaultReportConfig(c.Context(), reportType)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, config)
}

// Helper functions

func parseTimeRange(c *mizu.Ctx) (time.Time, time.Time) {
	now := time.Now()
	to := now

	// Check for explicit to parameter
	if toParam := c.Query("to"); toParam != "" {
		if parsed, err := time.Parse(time.RFC3339, toParam); err == nil {
			to = parsed
		}
	}

	// Check for explicit from parameter
	from := now.Add(-24 * time.Hour) // Default to 24 hours
	if fromParam := c.Query("from"); fromParam != "" {
		if parsed, err := time.Parse(time.RFC3339, fromParam); err == nil {
			from = parsed
		}
	} else if timeRange := c.Query("time_range"); timeRange != "" {
		// Parse time range shorthand
		switch timeRange {
		case "1h":
			from = now.Add(-1 * time.Hour)
		case "3h":
			from = now.Add(-3 * time.Hour)
		case "6h":
			from = now.Add(-6 * time.Hour)
		case "12h":
			from = now.Add(-12 * time.Hour)
		case "24h":
			from = now.Add(-24 * time.Hour)
		case "7d":
			from = now.Add(-7 * 24 * time.Hour)
		case "30d":
			from = now.Add(-30 * 24 * time.Hour)
		}
	}

	return from, to
}

func autoInterval(from, to time.Time) string {
	duration := to.Sub(from)
	switch {
	case duration <= 1*time.Hour:
		return "1m"
	case duration <= 6*time.Hour:
		return "5m"
	case duration <= 24*time.Hour:
		return "15m"
	case duration <= 7*24*time.Hour:
		return "1h"
	default:
		return "6h"
	}
}
