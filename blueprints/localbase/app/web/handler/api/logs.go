package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// LogsHandler handles logs explorer API endpoints.
type LogsHandler struct {
	store *postgres.Store
}

// NewLogsHandler creates a new logs handler.
func NewLogsHandler(store *postgres.Store) *LogsHandler {
	return &LogsHandler{store: store}
}

// ListLogs returns log entries with optional filtering.
// GET /api/logs
func (h *LogsHandler) ListLogs(c *mizu.Ctx) error {
	filter := parseLogFilter(c)

	logs, total, err := h.store.Logs().ListLogs(c.Context(), filter)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"logs":   logs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetLog returns a single log entry by ID.
// GET /api/logs/{id}
func (h *LogsHandler) GetLog(c *mizu.Ctx) error {
	id := c.Param("id")

	log, err := h.store.Logs().GetLog(c.Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if log == nil {
		return c.JSON(404, map[string]string{"error": "log not found"})
	}

	return c.JSON(200, log)
}

// GetHistogram returns log counts grouped by time interval.
// GET /api/logs/histogram
func (h *LogsHandler) GetHistogram(c *mizu.Ctx) error {
	filter := parseLogFilter(c)
	interval := c.Query("interval")
	if interval == "" {
		interval = "5m"
	}

	buckets, err := h.store.Logs().GetHistogram(c.Context(), filter, interval)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Calculate total
	total := 0
	for _, b := range buckets {
		total += b.Count
	}

	return c.JSON(200, map[string]any{
		"buckets": buckets,
		"total":   total,
	})
}

// ListLogSources returns available log sources/collections.
// GET /api/logs/sources
func (h *LogsHandler) ListLogSources(c *mizu.Ctx) error {
	sources := []store.LogSource{
		{ID: "edge", Name: "API Gateway", Description: "HTTP request/response logs from the API gateway"},
		{ID: "postgres", Name: "Postgres", Description: "PostgreSQL database server logs"},
		{ID: "postgrest", Name: "PostgREST", Description: "PostgREST API request logs"},
		{ID: "pooler", Name: "Pooler", Description: "Connection pooler logs"},
		{ID: "auth", Name: "Auth", Description: "Authentication and authorization logs"},
		{ID: "storage", Name: "Storage", Description: "Object storage operation logs"},
		{ID: "realtime", Name: "Realtime", Description: "WebSocket connection and subscription logs"},
		{ID: "functions", Name: "Edge Functions", Description: "Edge function invocation logs"},
		{ID: "cron", Name: "Cron", Description: "Scheduled job execution logs"},
	}
	return c.JSON(200, sources)
}

// SearchLogs searches logs with advanced filters.
// POST /api/logs/search
func (h *LogsHandler) SearchLogs(c *mizu.Ctx) error {
	var req struct {
		Source      string   `json:"source"`
		StatusMin   int      `json:"status_min"`
		StatusMax   int      `json:"status_max"`
		Methods     []string `json:"methods"`
		PathPattern string   `json:"path_pattern"`
		Query       string   `json:"query"`
		From        string   `json:"from"`
		To          string   `json:"to"`
		TimeRange   string   `json:"time_range"`
		Limit       int      `json:"limit"`
		Offset      int      `json:"offset"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	filter := &store.LogFilter{
		Source:      req.Source,
		StatusMin:   req.StatusMin,
		StatusMax:   req.StatusMax,
		Methods:     req.Methods,
		PathPattern: req.PathPattern,
		Query:       req.Query,
		TimeRange:   req.TimeRange,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}

	if req.From != "" {
		if t, err := time.Parse(time.RFC3339, req.From); err == nil {
			filter.From = &t
		}
	}
	if req.To != "" {
		if t, err := time.Parse(time.RFC3339, req.To); err == nil {
			filter.To = &t
		}
	}

	logs, total, err := h.store.Logs().ListLogs(c.Context(), filter)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"logs":   logs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// ListSavedQueries returns all saved queries.
// GET /api/logs/queries
func (h *LogsHandler) ListSavedQueries(c *mizu.Ctx) error {
	queries, err := h.store.Logs().ListSavedQueries(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, queries)
}

// CreateSavedQuery creates a new saved query.
// POST /api/logs/queries
func (h *LogsHandler) CreateSavedQuery(c *mizu.Ctx) error {
	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		QueryParams map[string]any `json:"query_params"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	query := &store.SavedQuery{
		Name:        req.Name,
		Description: req.Description,
		QueryParams: req.QueryParams,
	}

	if err := h.store.Logs().CreateSavedQuery(c.Context(), query); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, query)
}

// GetSavedQuery returns a saved query by ID.
// GET /api/logs/queries/{id}
func (h *LogsHandler) GetSavedQuery(c *mizu.Ctx) error {
	id := c.Param("id")

	query, err := h.store.Logs().GetSavedQuery(c.Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if query == nil {
		return c.JSON(404, map[string]string{"error": "query not found"})
	}

	return c.JSON(200, query)
}

// UpdateSavedQuery updates a saved query.
// PUT /api/logs/queries/{id}
func (h *LogsHandler) UpdateSavedQuery(c *mizu.Ctx) error {
	id := c.Param("id")

	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		QueryParams map[string]any `json:"query_params"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	query := &store.SavedQuery{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		QueryParams: req.QueryParams,
	}

	if err := h.store.Logs().UpdateSavedQuery(c.Context(), query); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, query)
}

// DeleteSavedQuery deletes a saved query.
// DELETE /api/logs/queries/{id}
func (h *LogsHandler) DeleteSavedQuery(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Logs().DeleteSavedQuery(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// ListQueryTemplates returns all predefined query templates.
// GET /api/logs/templates
func (h *LogsHandler) ListQueryTemplates(c *mizu.Ctx) error {
	templates, err := h.store.Logs().ListQueryTemplates(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, templates)
}

// ExportLogs exports logs as JSON or CSV.
// GET /api/logs/export
func (h *LogsHandler) ExportLogs(c *mizu.Ctx) error {
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	filter := parseLogFilter(c)
	// Increase limit for export
	if filter.Limit == 0 || filter.Limit == 25 {
		filter.Limit = 1000
	}

	logs, _, err := h.store.Logs().ListLogs(c.Context(), filter)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	if format == "csv" {
		c.Writer().Header().Set("Content-Type", "text/csv")
		c.Writer().Header().Set("Content-Disposition", "attachment; filename=logs.csv")

		csv := "id,timestamp,source,method,status_code,path,event_message,user_agent,duration_ms\n"
		for _, log := range logs {
			csv += log.ID + "," +
				log.Timestamp.Format(time.RFC3339) + "," +
				log.Source + "," +
				log.Method + "," +
				strconv.Itoa(log.StatusCode) + "," +
				escapeCSV(log.Path) + "," +
				escapeCSV(log.EventMessage) + "," +
				escapeCSV(log.UserAgent) + "," +
				strconv.Itoa(log.DurationMs) + "\n"
		}
		return c.Text(200, csv)
	}

	return c.JSON(200, logs)
}

// AddLog adds a log entry (called internally by middleware or other handlers).
func (h *LogsHandler) AddLog(entry *store.LogEntry) error {
	return h.store.Logs().CreateLog(nil, entry)
}

// parseLogFilter parses log filter from query parameters.
func parseLogFilter(c *mizu.Ctx) *store.LogFilter {
	filter := &store.LogFilter{
		Source:      c.Query("source"),
		PathPattern: c.Query("path"),
		Query:       c.Query("query"),
		TimeRange:   c.Query("time_range"),
	}

	if statusMin := c.Query("status_min"); statusMin != "" {
		if n, err := strconv.Atoi(statusMin); err == nil {
			filter.StatusMin = n
		}
	}

	if statusMax := c.Query("status_max"); statusMax != "" {
		if n, err := strconv.Atoi(statusMax); err == nil {
			filter.StatusMax = n
		}
	}

	if methods := c.Query("methods"); methods != "" {
		filter.Methods = strings.Split(methods, ",")
	}

	if method := c.Query("method"); method != "" {
		filter.Methods = []string{method}
	}

	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}

	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil {
			filter.Limit = n
		}
	}
	if filter.Limit <= 0 {
		filter.Limit = 25
	}

	if offset := c.Query("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil {
			filter.Offset = n
		}
	}

	return filter
}

// escapeCSV escapes a string for CSV output.
func escapeCSV(s string) string {
	needsQuotes := false
	for i := 0; i < len(s); i++ {
		if s[i] == ',' || s[i] == '"' || s[i] == '\n' || s[i] == '\r' {
			needsQuotes = true
			break
		}
	}
	if !needsQuotes {
		return s
	}

	var result []byte
	result = append(result, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			result = append(result, '"', '"')
		} else {
			result = append(result, s[i])
		}
	}
	result = append(result, '"')
	return string(result)
}
