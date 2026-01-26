// Package metrics provides observability for LLM operations.
package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// Collector collects and reports LLM metrics.
type Collector struct {
	mu       sync.RWMutex
	requests []llm.RequestMetrics
	sessions map[string]*SessionStats

	// Configuration
	enableLogging    bool
	enablePrometheus bool
	enableOTel       bool
	logger           *slog.Logger
}

// SessionStats tracks cumulative stats per session.
type SessionStats struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	RequestCount int
}

// Config configures the metrics collector.
type Config struct {
	EnableLogging    bool
	EnablePrometheus bool
	EnableOTel       bool
	Logger           *slog.Logger
}

// NewCollector creates a new metrics collector.
func NewCollector(cfg Config) *Collector {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	c := &Collector{
		sessions:         make(map[string]*SessionStats),
		enableLogging:    cfg.EnableLogging,
		enablePrometheus: cfg.EnablePrometheus,
		enableOTel:       cfg.EnableOTel,
		logger:           logger,
	}

	if cfg.EnablePrometheus {
		registerPrometheusMetrics()
	}

	return c
}

// RecordRequest records metrics for an LLM request.
func (c *Collector) RecordRequest(ctx context.Context, m llm.RequestMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store for history
	c.requests = append(c.requests, m)

	// Keep only last 1000 requests
	if len(c.requests) > 1000 {
		c.requests = c.requests[1:]
	}

	// Update session stats if we have a session context
	if sessionID := getSessionID(ctx); sessionID != "" {
		stats, ok := c.sessions[sessionID]
		if !ok {
			stats = &SessionStats{}
			c.sessions[sessionID] = stats
		}
		stats.InputTokens += m.InputTokens
		stats.OutputTokens += m.OutputTokens
		stats.CostUSD += m.CostUSD
		stats.RequestCount++
	}

	// Log metrics
	if c.enableLogging {
		c.logRequest(m)
	}

	// Record to Prometheus
	if c.enablePrometheus {
		recordPrometheusMetrics(m)
	}

	// Record to OpenTelemetry
	if c.enableOTel {
		recordOTelMetrics(ctx, m)
	}
}

// GetSessionTotals returns cumulative token usage for a session.
func (c *Collector) GetSessionTotals(sessionID string) (inputTokens, outputTokens int, costUSD float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats, ok := c.sessions[sessionID]
	if !ok {
		return 0, 0, 0
	}
	return stats.InputTokens, stats.OutputTokens, stats.CostUSD
}

// GetRecentRequests returns the most recent requests.
func (c *Collector) GetRecentRequests(limit int) []llm.RequestMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.requests) {
		limit = len(c.requests)
	}

	result := make([]llm.RequestMetrics, limit)
	copy(result, c.requests[len(c.requests)-limit:])
	return result
}

// logRequest logs a request to the configured logger.
func (c *Collector) logRequest(m llm.RequestMetrics) {
	attrs := []any{
		"provider", m.Provider,
		"model", m.Model,
		"request_id", m.RequestID,
		"input_tokens", m.InputTokens,
		"output_tokens", m.OutputTokens,
		"duration_ms", m.TotalDuration.Milliseconds(),
		"tokens_per_sec", m.TokensPerSecond,
		"success", m.Success,
	}

	if m.CostUSD > 0 {
		attrs = append(attrs, "cost_usd", m.CostUSD)
	}
	if m.CacheReadTokens > 0 {
		attrs = append(attrs, "cache_read_tokens", m.CacheReadTokens)
	}
	if m.CacheWriteTokens > 0 {
		attrs = append(attrs, "cache_write_tokens", m.CacheWriteTokens)
	}
	if m.ToolCalls > 0 {
		attrs = append(attrs, "tool_calls", m.ToolCalls)
	}
	if m.TimeToFirstToken > 0 {
		attrs = append(attrs, "ttft_ms", m.TimeToFirstToken.Milliseconds())
	}
	if m.Error != "" {
		attrs = append(attrs, "error", m.Error)
	}

	if m.Success {
		c.logger.Info("llm_request", attrs...)
	} else {
		c.logger.Error("llm_request", attrs...)
	}
}

// Context keys for session tracking.
type contextKey string

const sessionIDKey contextKey = "llm_session_id"

// WithSessionID adds a session ID to the context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// getSessionID retrieves the session ID from context.
func getSessionID(ctx context.Context) string {
	if v := ctx.Value(sessionIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// RequestTracker helps track metrics for a single request.
type RequestTracker struct {
	metrics   llm.RequestMetrics
	collector *Collector
	startTime time.Time
	firstTokenTime time.Time
}

// StartRequest begins tracking a new request.
func (c *Collector) StartRequest(provider, model, requestID string) *RequestTracker {
	return &RequestTracker{
		metrics: llm.RequestMetrics{
			Provider:  provider,
			Model:     model,
			RequestID: requestID,
			StartTime: time.Now(),
		},
		collector: c,
		startTime: time.Now(),
	}
}

// RecordFirstToken records when the first token was received.
func (t *RequestTracker) RecordFirstToken() {
	if t.firstTokenTime.IsZero() {
		t.firstTokenTime = time.Now()
		t.metrics.TimeToFirstToken = t.firstTokenTime.Sub(t.startTime)
	}
}

// RecordToolCall increments the tool call counter.
func (t *RequestTracker) RecordToolCall() {
	t.metrics.ToolCalls++
}

// End completes the request tracking and records metrics.
func (t *RequestTracker) End(ctx context.Context, inputTokens, outputTokens int, costUSD float64, err error) {
	t.metrics.TotalDuration = time.Since(t.startTime)
	t.metrics.InputTokens = inputTokens
	t.metrics.OutputTokens = outputTokens
	t.metrics.CostUSD = costUSD
	t.metrics.Success = err == nil

	if err != nil {
		t.metrics.Error = err.Error()
	}

	// Calculate tokens per second
	if t.metrics.TotalDuration > 0 && outputTokens > 0 {
		t.metrics.TokensPerSecond = float64(outputTokens) / t.metrics.TotalDuration.Seconds()
	}

	t.collector.RecordRequest(ctx, t.metrics)
}

// SetCacheTokens sets cache token counts.
func (t *RequestTracker) SetCacheTokens(read, write int) {
	t.metrics.CacheReadTokens = read
	t.metrics.CacheWriteTokens = write
}
