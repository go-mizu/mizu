package api

import (
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/mizu"
)

// Observability handles observability requests.
type Observability struct {
	store store.Store
}

// NewObservability creates a new Observability handler.
func NewObservability(st store.Store) *Observability {
	return &Observability{store: st}
}

// LogEntryResponse represents a log entry response.
type LogEntryResponse struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Worker    string         `json:"worker,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// TraceResponse represents a distributed trace response.
type TraceResponse struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	StartTime string              `json:"start_time"`
	Duration  int                 `json:"duration_ms"`
	Status    string              `json:"status"`
	Spans     []TraceSpanResponse `json:"spans"`
}

// TraceSpanResponse represents a span within a trace response.
type TraceSpanResponse struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	StartTime string         `json:"start_time"`
	Duration  int            `json:"duration_ms"`
	Status    string         `json:"status"`
	Tags      map[string]any `json:"tags,omitempty"`
}

// GetLogs retrieves logs.
func (h *Observability) GetLogs(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	workerID := c.Query("worker_id")
	level := c.Query("level")

	logs, err := h.store.Observability().QueryLogs(ctx, workerID, level, 100, 0)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []LogEntryResponse
	for _, log := range logs {
		data := map[string]any{}
		if log.RequestID != "" {
			data["request_id"] = log.RequestID
		}
		for k, v := range log.Metadata {
			data[k] = v
		}

		result = append(result, LogEntryResponse{
			ID:        log.ID,
			Timestamp: log.Timestamp.Format(time.RFC3339),
			Level:     log.Level,
			Message:   log.Message,
			Worker:    log.WorkerName,
			Data:      data,
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"logs": result,
		},
	})
}

// GetTraces retrieves distributed traces.
func (h *Observability) GetTraces(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	traces, err := h.store.Observability().ListTraces(ctx, 50, 0)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []TraceResponse
	for _, trace := range traces {
		// Get spans for each trace
		spans, _ := h.store.Observability().GetSpansByTraceID(ctx, trace.TraceID)

		var spanResponses []TraceSpanResponse
		for _, span := range spans {
			tags := map[string]any{}
			for k, v := range span.Attributes {
				tags[k] = v
			}

			spanResponses = append(spanResponses, TraceSpanResponse{
				ID:        span.SpanID,
				Name:      span.Name,
				StartTime: span.StartTime.Format(time.RFC3339),
				Duration:  span.DurationMs,
				Status:    span.Status,
				Tags:      tags,
			})
		}

		result = append(result, TraceResponse{
			ID:        trace.TraceID,
			Name:      trace.RootService,
			StartTime: trace.StartedAt.Format(time.RFC3339),
			Duration:  trace.DurationMs,
			Status:    trace.Status,
			Spans:     spanResponses,
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"traces": result,
		},
	})
}

// GetMetrics retrieves metrics data.
func (h *Observability) GetMetrics(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	timeRange := c.Query("range")
	metricName := c.Query("name")

	if metricName == "" {
		metricName = "requests"
	}

	if timeRange == "" {
		timeRange = "24h"
	}

	// Determine time range
	var duration time.Duration
	switch timeRange {
	case "1h":
		duration = time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	end := time.Now()
	start := end.Add(-duration)

	metrics, err := h.store.Observability().QueryMetrics(ctx, metricName, start, end, 1000)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var data []map[string]any
	for _, m := range metrics {
		data = append(data, map[string]any{
			"timestamp": m.Timestamp.Format(time.RFC3339),
			"value":     m.Value,
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"data": data,
		},
	})
}
