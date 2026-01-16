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

// GetStats returns dashboard statistics.
func (h *DashboardHandler) GetStats(c *mizu.Ctx) error {
	ctx := c.Context()

	// Get user count
	_, total, _ := h.store.Auth().ListUsers(ctx, 1, 1)

	// Get storage stats
	buckets, _ := h.store.Storage().ListBuckets(ctx)

	// Get function stats
	functions, _ := h.store.Functions().ListFunctions(ctx)

	// Get table stats
	tables, _ := h.store.Database().ListTables(ctx, "public")

	return c.JSON(200, map[string]any{
		"users": map[string]any{
			"total": total,
		},
		"storage": map[string]any{
			"buckets": len(buckets),
		},
		"functions": map[string]any{
			"total":  len(functions),
			"active": countActiveFunctions(functions),
		},
		"database": map[string]any{
			"tables": len(tables),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetHealth returns health check status.
func (h *DashboardHandler) GetHealth(c *mizu.Ctx) error {
	// Check database connection
	_, err := h.store.Database().ListSchemas(c.Context())
	dbHealthy := err == nil

	status := "healthy"
	if !dbHealthy {
		status = "unhealthy"
	}

	return c.JSON(200, map[string]any{
		"status": status,
		"services": map[string]bool{
			"database": dbHealthy,
			"auth":     true, // Simplified
			"storage":  true, // Simplified
			"realtime": true, // Simplified
		},
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
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
