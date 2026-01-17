package api

import (
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// LogsHandler handles logs explorer API endpoints.
type LogsHandler struct {
	logs    []LogEntry
	logsMu  sync.RWMutex
	maxLogs int
}

// LogEntry represents a single log entry.
type LogEntry struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	Timestamp time.Time      `json:"timestamp"`
}

// LogType represents an available log type.
type LogType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NewLogsHandler creates a new logs handler.
func NewLogsHandler() *LogsHandler {
	return &LogsHandler{
		logs:    make([]LogEntry, 0),
		maxLogs: 10000,
	}
}

// AddLog adds a log entry (called internally by other handlers).
func (h *LogsHandler) AddLog(logType, level, message string, metadata map[string]any) {
	h.logsMu.Lock()
	defer h.logsMu.Unlock()

	entry := LogEntry{
		ID:        generateLogID(),
		Type:      logType,
		Level:     level,
		Message:   message,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	h.logs = append(h.logs, entry)

	// Keep only last maxLogs entries
	if len(h.logs) > h.maxLogs {
		h.logs = h.logs[len(h.logs)-h.maxLogs:]
	}
}

// ListLogs returns log entries with optional filtering.
func (h *LogsHandler) ListLogs(c *mizu.Ctx) error {
	h.logsMu.RLock()
	defer h.logsMu.RUnlock()

	logType := c.Query("type")
	level := c.Query("level")
	query := c.Query("query")
	from := c.Query("from")
	to := c.Query("to")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	limit := 100
	offset := 0
	if limitStr != "" {
		if l, err := parsePositiveInt(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := parsePositiveInt(offsetStr); err == nil {
			offset = o
		}
	}

	var fromTime, toTime time.Time
	if from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			fromTime = t
		}
	}
	if to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			toTime = t
		}
	}

	// Filter logs
	filtered := make([]LogEntry, 0)
	for _, log := range h.logs {
		if logType != "" && log.Type != logType {
			continue
		}
		if level != "" && log.Level != level {
			continue
		}
		if query != "" && !containsIgnoreCase(log.Message, query) {
			continue
		}
		if !fromTime.IsZero() && log.Timestamp.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && log.Timestamp.After(toTime) {
			continue
		}
		filtered = append(filtered, log)
	}

	// Apply pagination
	total := len(filtered)
	if offset >= len(filtered) {
		filtered = []LogEntry{}
	} else {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[offset:end]
	}

	return c.JSON(200, map[string]any{
		"logs":   filtered,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// SearchLogs searches logs with advanced filters.
func (h *LogsHandler) SearchLogs(c *mizu.Ctx) error {
	var req struct {
		Type   string   `json:"type"`
		Levels []string `json:"levels"`
		From   string   `json:"from"`
		To     string   `json:"to"`
		Query  string   `json:"query"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	h.logsMu.RLock()
	defer h.logsMu.RUnlock()

	var fromTime, toTime time.Time
	if req.From != "" {
		if t, err := time.Parse(time.RFC3339, req.From); err == nil {
			fromTime = t
		}
	}
	if req.To != "" {
		if t, err := time.Parse(time.RFC3339, req.To); err == nil {
			toTime = t
		}
	}

	levelSet := make(map[string]bool)
	for _, l := range req.Levels {
		levelSet[l] = true
	}

	// Filter logs
	filtered := make([]LogEntry, 0)
	for _, log := range h.logs {
		if req.Type != "" && log.Type != req.Type {
			continue
		}
		if len(levelSet) > 0 && !levelSet[log.Level] {
			continue
		}
		if req.Query != "" && !containsIgnoreCase(log.Message, req.Query) {
			continue
		}
		if !fromTime.IsZero() && log.Timestamp.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && log.Timestamp.After(toTime) {
			continue
		}
		filtered = append(filtered, log)
	}

	// Apply pagination
	total := len(filtered)
	if req.Offset >= len(filtered) {
		filtered = []LogEntry{}
	} else {
		end := req.Offset + req.Limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[req.Offset:end]
	}

	return c.JSON(200, map[string]any{
		"logs":   filtered,
		"total":  total,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

// ListLogTypes returns available log types.
func (h *LogsHandler) ListLogTypes(c *mizu.Ctx) error {
	types := []LogType{
		{ID: "postgres", Name: "PostgreSQL", Description: "Database server logs"},
		{ID: "auth", Name: "Authentication", Description: "Auth service logs"},
		{ID: "storage", Name: "Storage", Description: "Storage service logs"},
		{ID: "functions", Name: "Edge Functions", Description: "Edge function invocation logs"},
		{ID: "realtime", Name: "Realtime", Description: "Realtime connection logs"},
		{ID: "api", Name: "REST API", Description: "API request logs"},
	}
	return c.JSON(200, types)
}

// ExportLogs exports logs as JSON or CSV.
func (h *LogsHandler) ExportLogs(c *mizu.Ctx) error {
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	logType := c.Query("type")
	from := c.Query("from")
	to := c.Query("to")

	h.logsMu.RLock()
	defer h.logsMu.RUnlock()

	var fromTime, toTime time.Time
	if from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			fromTime = t
		}
	}
	if to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			toTime = t
		}
	}

	// Filter logs
	filtered := make([]LogEntry, 0)
	for _, log := range h.logs {
		if logType != "" && log.Type != logType {
			continue
		}
		if !fromTime.IsZero() && log.Timestamp.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && log.Timestamp.After(toTime) {
			continue
		}
		filtered = append(filtered, log)
	}

	if format == "csv" {
		c.Writer().Header().Set("Content-Type", "text/csv")
		c.Writer().Header().Set("Content-Disposition", "attachment; filename=logs.csv")

		var csv string
		csv = "id,type,level,message,timestamp\n"
		for _, log := range filtered {
			csv += log.ID + "," + log.Type + "," + log.Level + "," + escapeCSV(log.Message) + "," + log.Timestamp.Format(time.RFC3339) + "\n"
		}
		return c.Text(200, csv)
	}

	return c.JSON(200, filtered)
}

// Helper functions

func generateLogID() string {
	return "log_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsIgnoreCaseImpl(s, substr))
}

func containsIgnoreCaseImpl(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		result[i] = c
	}
	return string(result)
}

func escapeCSV(s string) string {
	// Simple CSV escaping
	needsQuotes := false
	for i := 0; i < len(s); i++ {
		if s[i] == ',' || s[i] == '"' || s[i] == '\n' {
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

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
