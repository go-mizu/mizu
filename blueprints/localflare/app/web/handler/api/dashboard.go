package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/blueprints/localflare/store"
)

// Dashboard handles dashboard requests.
type Dashboard struct {
	store store.Store
}

// NewDashboard creates a new Dashboard handler.
func NewDashboard(st store.Store) *Dashboard {
	return &Dashboard{store: st}
}

// GetStats returns aggregated stats from all services.
func (h *Dashboard) GetStats(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	// Aggregate stats from all services
	stats := map[string]any{
		"durable_objects": map[string]any{
			"namespaces": 0,
			"objects":    0,
		},
		"queues": map[string]any{
			"count":          0,
			"total_messages": 0,
		},
		"vectorize": map[string]any{
			"indexes":       0,
			"total_vectors": 0,
		},
		"analytics": map[string]any{
			"datasets":    0,
			"data_points": 0,
		},
		"ai": map[string]any{
			"requests_today": 0,
			"tokens_today":   0,
		},
		"ai_gateway": map[string]any{
			"gateways":       0,
			"requests_today": 0,
		},
		"hyperdrive": map[string]any{
			"configs":            0,
			"active_connections": 0,
		},
		"cron": map[string]any{
			"triggers":         0,
			"executions_today": 0,
		},
	}

	// Get Durable Objects stats
	if namespaces, err := h.store.DurableObjects().ListNamespaces(ctx); err == nil {
		objectCount := 0
		for _, ns := range namespaces {
			if instances, err := h.store.DurableObjects().ListInstances(ctx, ns.ID); err == nil {
				objectCount += len(instances)
			}
		}
		stats["durable_objects"] = map[string]any{
			"namespaces": len(namespaces),
			"objects":    objectCount,
		}
	}

	// Get Queues stats
	if queues, err := h.store.Queues().ListQueues(ctx); err == nil {
		totalMessages := int64(0)
		for _, q := range queues {
			if qStats, err := h.store.Queues().GetQueueStats(ctx, q.ID); err == nil {
				totalMessages += qStats.Messages
			}
		}
		stats["queues"] = map[string]any{
			"count":          len(queues),
			"total_messages": totalMessages,
		}
	}

	// Get Vectorize stats
	if indexes, err := h.store.Vectorize().ListIndexes(ctx); err == nil {
		totalVectors := int64(0)
		for _, idx := range indexes {
			totalVectors += idx.VectorCount
		}
		stats["vectorize"] = map[string]any{
			"indexes":       len(indexes),
			"total_vectors": totalVectors,
		}
	}

	// Get Analytics Engine stats
	if datasets, err := h.store.AnalyticsEngine().ListDatasets(ctx); err == nil {
		stats["analytics"] = map[string]any{
			"datasets":    len(datasets),
			"data_points": 0, // Would need to query each dataset
		}
	}

	// Get AI Gateway stats
	if gateways, err := h.store.AIGateway().ListGateways(ctx); err == nil {
		requestsToday := 0
		for _, gw := range gateways {
			if logs, err := h.store.AIGateway().GetLogs(ctx, gw.ID, 1000, 0); err == nil {
				today := time.Now().Truncate(24 * time.Hour)
				for _, log := range logs {
					if log.CreatedAt.After(today) {
						requestsToday++
					}
				}
			}
		}
		stats["ai_gateway"] = map[string]any{
			"gateways":       len(gateways),
			"requests_today": requestsToday,
		}
	}

	// Get Hyperdrive stats
	if configs, err := h.store.Hyperdrive().ListConfigs(ctx); err == nil {
		totalActive := 0
		for _, cfg := range configs {
			if hStats, err := h.store.Hyperdrive().GetStats(ctx, cfg.ID); err == nil {
				totalActive += hStats.ActiveConnections
			}
		}
		stats["hyperdrive"] = map[string]any{
			"configs":            len(configs),
			"active_connections": totalActive,
		}
	}

	// Get Cron stats
	if triggers, err := h.store.Cron().ListTriggers(ctx); err == nil {
		executionsToday := 0
		today := time.Now().Truncate(24 * time.Hour)
		for _, t := range triggers {
			if execs, err := h.store.Cron().GetRecentExecutions(ctx, t.ID, 100); err == nil {
				for _, exec := range execs {
					if exec.StartedAt.After(today) {
						executionsToday++
					}
				}
			}
		}
		stats["cron"] = map[string]any{
			"triggers":         len(triggers),
			"executions_today": executionsToday,
		}
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  stats,
	})
}

// GetTimeSeries returns time series data for charts.
func (h *Dashboard) GetTimeSeries(c *mizu.Ctx) error {
	metric := c.Query("metric")
	timeRange := c.Query("range")

	if metric == "" {
		metric = "requests"
	}
	if timeRange == "" {
		timeRange = "24h"
	}

	// Determine time range
	var duration time.Duration
	var points int
	var interval time.Duration

	switch timeRange {
	case "1h":
		duration = time.Hour
		points = 60
		interval = time.Minute
	case "24h":
		duration = 24 * time.Hour
		points = 24
		interval = time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
		points = 7 * 24
		interval = time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
		points = 30
		interval = 24 * time.Hour
	default:
		duration = 24 * time.Hour
		points = 24
		interval = time.Hour
	}

	// Generate time series data points
	now := time.Now()
	start := now.Add(-duration)
	data := make([]map[string]any, 0, points)

	// For now, generate synthetic data based on actual activity
	// In a real implementation, this would query actual metrics
	ctx := c.Request().Context()

	for i := 0; i < points; i++ {
		timestamp := start.Add(time.Duration(i) * interval)
		value := 0

		// Count actual events in this time window
		windowEnd := timestamp.Add(interval)

		// Count cron executions in this window
		if triggers, err := h.store.Cron().ListTriggers(ctx); err == nil {
			for _, t := range triggers {
				if execs, err := h.store.Cron().GetRecentExecutions(ctx, t.ID, 1000); err == nil {
					for _, exec := range execs {
						if exec.StartedAt.After(timestamp) && exec.StartedAt.Before(windowEnd) {
							value++
						}
					}
				}
			}
		}

		// Count AI gateway logs in this window
		if gateways, err := h.store.AIGateway().ListGateways(ctx); err == nil {
			for _, gw := range gateways {
				if logs, err := h.store.AIGateway().GetLogs(ctx, gw.ID, 1000, 0); err == nil {
					for _, log := range logs {
						if log.CreatedAt.After(timestamp) && log.CreatedAt.Before(windowEnd) {
							value++
						}
					}
				}
			}
		}

		data = append(data, map[string]any{
			"timestamp": timestamp.Format(time.RFC3339),
			"value":     value,
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"data": data,
		},
	})
}

// GetActivity returns recent activity events.
func (h *Dashboard) GetActivity(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	limit := 10
	if l := c.Query("limit"); l != "" {
		if n := parseIntOrDefault(l, 10); n > 0 && n <= 100 {
			limit = n
		}
	}

	events := make([]map[string]any, 0, limit)

	// Collect recent cron executions
	if triggers, err := h.store.Cron().ListTriggers(ctx); err == nil {
		for _, t := range triggers {
			if execs, err := h.store.Cron().GetRecentExecutions(ctx, t.ID, 5); err == nil {
				for _, exec := range execs {
					eventType := "cron_success"
					message := "Cron trigger executed: " + t.ScriptName
					if exec.Status == "failed" {
						eventType = "cron_failed"
						message = "Cron trigger failed: " + t.ScriptName
					}
					events = append(events, map[string]any{
						"id":        exec.ID,
						"type":      eventType,
						"message":   message,
						"timestamp": exec.StartedAt.Format(time.RFC3339),
						"service":   "Cron",
					})
				}
			}
		}
	}

	// Collect recent AI gateway logs
	if gateways, err := h.store.AIGateway().ListGateways(ctx); err == nil {
		for _, gw := range gateways {
			if logs, err := h.store.AIGateway().GetLogs(ctx, gw.ID, 5, 0); err == nil {
				for _, log := range logs {
					eventType := "ai_request"
					message := "AI request to " + log.Model
					if log.Cached {
						message += " (cached)"
					}
					events = append(events, map[string]any{
						"id":        log.ID,
						"type":      eventType,
						"message":   message,
						"timestamp": log.CreatedAt.Format(time.RFC3339),
						"service":   "AI Gateway",
					})
				}
			}
		}
	}

	// Sort by timestamp descending and limit
	sortEventsByTimestamp(events)
	if len(events) > limit {
		events = events[:limit]
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"events": events,
		},
	})
}

// GetStatus returns system health status.
func (h *Dashboard) GetStatus(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	services := []map[string]any{}

	// Check each service
	serviceChecks := []struct {
		name  string
		check func() bool
	}{
		{"Durable Objects", func() bool {
			_, err := h.store.DurableObjects().ListNamespaces(ctx)
			return err == nil
		}},
		{"Queues", func() bool {
			_, err := h.store.Queues().ListQueues(ctx)
			return err == nil
		}},
		{"Vectorize", func() bool {
			_, err := h.store.Vectorize().ListIndexes(ctx)
			return err == nil
		}},
		{"Analytics Engine", func() bool {
			_, err := h.store.AnalyticsEngine().ListDatasets(ctx)
			return err == nil
		}},
		{"Workers AI", func() bool {
			_, err := h.store.AI().ListModels(ctx, "")
			return err == nil
		}},
		{"AI Gateway", func() bool {
			_, err := h.store.AIGateway().ListGateways(ctx)
			return err == nil
		}},
		{"Hyperdrive", func() bool {
			_, err := h.store.Hyperdrive().ListConfigs(ctx)
			return err == nil
		}},
		{"Cron", func() bool {
			_, err := h.store.Cron().ListTriggers(ctx)
			return err == nil
		}},
	}

	for _, svc := range serviceChecks {
		status := "online"
		if !svc.check() {
			status = "offline"
		}
		services = append(services, map[string]any{
			"service":    svc.name,
			"status":     status,
			"latency_ms": 1, // Would measure actual latency in production
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"services": services,
		},
	})
}

// Helper functions

func parseIntOrDefault(s string, def int) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n == 0 {
		return def
	}
	return n
}

func sortEventsByTimestamp(events []map[string]any) {
	// Simple bubble sort for small arrays
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			ti := events[i]["timestamp"].(string)
			tj := events[j]["timestamp"].(string)
			if ti < tj {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}
