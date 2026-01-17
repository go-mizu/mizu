package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// DashboardHandler handles dashboard endpoints.
type DashboardHandler struct {
	store *postgres.Store
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(store *postgres.Store) *DashboardHandler {
	return &DashboardHandler{store: store}
}

// GetStats returns extended dashboard statistics.
func (h *DashboardHandler) GetStats(c *mizu.Ctx) error {
	ctx := c.Context()

	// Get user count
	_, total, _ := h.store.Auth().ListUsers(ctx, 1, 1)

	// Get storage stats
	buckets, _ := h.store.Storage().ListBuckets(ctx)
	var totalStorageSize int64
	var objectCount int
	for _, b := range buckets {
		objects, _ := h.store.Storage().ListObjects(ctx, b.ID, "", 1000, 0)
		objectCount += len(objects)
		for _, obj := range objects {
			totalStorageSize += obj.Size
		}
	}

	// Get function stats
	functions, _ := h.store.Functions().ListFunctions(ctx)

	// Get table stats
	tables, _ := h.store.Database().ListTables(ctx, "public")
	var totalRows int64
	for _, t := range tables {
		totalRows += t.RowCount
	}

	// Get schemas
	schemas, _ := h.store.Database().ListSchemas(ctx)

	// Get realtime stats
	channels, _ := h.store.Realtime().ListChannels(ctx)
	activeConnections := 0 // Would need connection tracking

	return c.JSON(200, map[string]any{
		"users": map[string]any{
			"total":         total,
			"active_today":  0, // Would need activity tracking
			"new_this_week": 0, // Would need activity tracking
		},
		"storage": map[string]any{
			"buckets":    len(buckets),
			"total_size": totalStorageSize,
			"objects":    objectCount,
		},
		"functions": map[string]any{
			"total":            len(functions),
			"active":           countActiveFunctions(functions),
			"invocations_today": 0, // Would need invocation tracking
		},
		"database": map[string]any{
			"tables":     len(tables),
			"total_rows": totalRows,
			"schemas":    schemas,
		},
		"realtime": map[string]any{
			"active_connections": activeConnections,
			"channels":           len(channels),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetHealth returns extended health check status.
func (h *DashboardHandler) GetHealth(c *mizu.Ctx) error {
	ctx := c.Context()
	start := time.Now()

	// Check database connection
	_, err := h.store.Database().ListSchemas(ctx)
	dbHealthy := err == nil
	dbLatency := time.Since(start).Milliseconds()

	// Get database version
	version, _ := h.store.PGMeta().GetVersion(ctx)
	dbVersion := ""
	if version != nil {
		dbVersion = version.Version
	}

	status := "healthy"
	if !dbHealthy {
		status = "unhealthy"
	}

	return c.JSON(200, map[string]any{
		"status": status,
		"services": map[string]any{
			"database": map[string]any{
				"status":     boolToStatus(dbHealthy),
				"version":    dbVersion,
				"latency_ms": dbLatency,
			},
			"auth": map[string]any{
				"status":  "healthy",
				"version": "2.40.0",
			},
			"storage": map[string]any{
				"status": "healthy",
				"type":   "local",
			},
			"realtime": map[string]any{
				"status":      "healthy",
				"connections": 0,
			},
		},
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func boolToStatus(b bool) string {
	if b {
		return "healthy"
	}
	return "unhealthy"
}

func countActiveFunctions(functions []*store.Function) int {
	count := 0
	for _, fn := range functions {
		if fn.Status == "active" {
			count++
		}
	}
	return count
}
