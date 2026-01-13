package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Observability handles observability requests.
type Observability struct{}

// NewObservability creates a new Observability handler.
func NewObservability() *Observability {
	return &Observability{}
}

// LogEntry represents a log entry.
type LogEntry struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Worker    string         `json:"worker,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// Trace represents a distributed trace.
type Trace struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	StartTime string      `json:"start_time"`
	Duration  int         `json:"duration_ms"`
	Status    string      `json:"status"`
	Spans     []TraceSpan `json:"spans"`
}

// TraceSpan represents a span within a trace.
type TraceSpan struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	StartTime string         `json:"start_time"`
	Duration  int            `json:"duration_ms"`
	Status    string         `json:"status"`
	Tags      map[string]any `json:"tags,omitempty"`
}

// GetLogs retrieves logs.
func (h *Observability) GetLogs(c *mizu.Ctx) error {
	now := time.Now()
	workers := []string{"api-router", "auth-middleware", "image-optimizer", "analytics-tracker"}
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"Request processed successfully",
		"Cache hit for key: user:1234",
		"Rate limit exceeded for IP: 192.168.1.1",
		"Database query completed in 45ms",
		"Worker started",
		"Authentication successful for user",
		"Failed to connect to upstream",
		"Retry attempt 2 of 3",
	}

	logs := make([]LogEntry, 0, 20)
	for i := 0; i < 20; i++ {
		logs = append(logs, LogEntry{
			ID:        "log-" + ulid.Make().String()[:8],
			Timestamp: now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			Level:     levels[i%len(levels)],
			Message:   messages[i%len(messages)],
			Worker:    workers[i%len(workers)],
			Data: map[string]any{
				"request_id": ulid.Make().String()[:8],
				"duration":   10 + (i * 5),
			},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"logs": logs,
		},
	})
}

// GetTraces retrieves distributed traces.
func (h *Observability) GetTraces(c *mizu.Ctx) error {
	now := time.Now()
	traces := []Trace{
		{
			ID:        "trace-" + ulid.Make().String()[:8],
			Name:      "POST /api/users",
			StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339),
			Duration:  245,
			Status:    "success",
			Spans: []TraceSpan{
				{
					ID:        "span-" + ulid.Make().String()[:8],
					Name:      "api-router.handleRequest",
					StartTime: now.Add(-5 * time.Minute).Format(time.RFC3339),
					Duration:  245,
					Status:    "success",
					Tags:      map[string]any{"http.method": "POST", "http.path": "/api/users"},
				},
				{
					ID:        "span-" + ulid.Make().String()[:8],
					Name:      "auth-middleware.verify",
					StartTime: now.Add(-5*time.Minute + 10*time.Millisecond).Format(time.RFC3339),
					Duration:  35,
					Status:    "success",
					Tags:      map[string]any{"user.id": "user-123"},
				},
				{
					ID:        "span-" + ulid.Make().String()[:8],
					Name:      "d1.query",
					StartTime: now.Add(-5*time.Minute + 50*time.Millisecond).Format(time.RFC3339),
					Duration:  120,
					Status:    "success",
					Tags:      map[string]any{"db.statement": "INSERT INTO users"},
				},
			},
		},
		{
			ID:        "trace-" + ulid.Make().String()[:8],
			Name:      "GET /api/products",
			StartTime: now.Add(-10 * time.Minute).Format(time.RFC3339),
			Duration:  89,
			Status:    "success",
			Spans: []TraceSpan{
				{
					ID:        "span-" + ulid.Make().String()[:8],
					Name:      "api-router.handleRequest",
					StartTime: now.Add(-10 * time.Minute).Format(time.RFC3339),
					Duration:  89,
					Status:    "success",
					Tags:      map[string]any{"http.method": "GET", "http.path": "/api/products"},
				},
				{
					ID:        "span-" + ulid.Make().String()[:8],
					Name:      "kv.get",
					StartTime: now.Add(-10*time.Minute + 5*time.Millisecond).Format(time.RFC3339),
					Duration:  12,
					Status:    "success",
					Tags:      map[string]any{"cache.hit": true},
				},
			},
		},
		{
			ID:        "trace-" + ulid.Make().String()[:8],
			Name:      "POST /api/checkout",
			StartTime: now.Add(-15 * time.Minute).Format(time.RFC3339),
			Duration:  1234,
			Status:    "error",
			Spans: []TraceSpan{
				{
					ID:        "span-" + ulid.Make().String()[:8],
					Name:      "api-router.handleRequest",
					StartTime: now.Add(-15 * time.Minute).Format(time.RFC3339),
					Duration:  1234,
					Status:    "error",
					Tags:      map[string]any{"http.method": "POST", "http.path": "/api/checkout", "error": "payment failed"},
				},
			},
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"traces": traces,
		},
	})
}

// GetMetrics retrieves metrics data.
func (h *Observability) GetMetrics(c *mizu.Ctx) error {
	timeRange := c.Query("range")
	if timeRange == "" {
		timeRange = "24h"
	}

	// Determine time range
	var points int
	var interval time.Duration
	switch timeRange {
	case "1h":
		points = 60
		interval = time.Minute
	case "24h":
		points = 24
		interval = time.Hour
	case "7d":
		points = 168
		interval = time.Hour
	default:
		points = 24
		interval = time.Hour
	}

	now := time.Now()
	data := make([]map[string]any, 0, points)
	for i := 0; i < points; i++ {
		data = append(data, map[string]any{
			"timestamp": now.Add(-time.Duration(points-i) * interval).Format(time.RFC3339),
			"value":     500 + (i * 10) + (i % 5 * 50),
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"data": data,
		},
	})
}
