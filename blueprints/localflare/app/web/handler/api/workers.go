package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Workers handles worker-related requests.
type Workers struct {
	store store.WorkerStore
}

// NewWorkers creates a new Workers handler.
func NewWorkers(store store.WorkerStore) *Workers {
	return &Workers{store: store}
}

// List lists all workers.
func (h *Workers) List(c *mizu.Ctx) error {
	workers, err := h.store.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  workers,
	})
}

// Get retrieves a worker by ID.
func (h *Workers) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	worker, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Worker not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  worker,
	})
}

// CreateWorkerInput is the input for creating a worker.
type CreateWorkerInput struct {
	Name     string            `json:"name"`
	Script   string            `json:"script"`
	Routes   []string          `json:"routes"`
	Bindings map[string]string `json:"bindings"`
	Enabled  bool              `json:"enabled"`
}

// Create creates a new worker.
func (h *Workers) Create(c *mizu.Ctx) error {
	var input CreateWorkerInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "Name is required"})
	}

	now := time.Now()
	worker := &store.Worker{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		Script:    input.Script,
		Routes:    input.Routes,
		Bindings:  input.Bindings,
		Enabled:   input.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if worker.Bindings == nil {
		worker.Bindings = make(map[string]string)
	}

	if err := h.store.Create(c.Request().Context(), worker); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  worker,
	})
}

// Update updates a worker.
func (h *Workers) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	worker, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Worker not found"})
	}

	var input CreateWorkerInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name != "" {
		worker.Name = input.Name
	}
	if input.Script != "" {
		worker.Script = input.Script
	}
	if input.Routes != nil {
		worker.Routes = input.Routes
	}
	if input.Bindings != nil {
		worker.Bindings = input.Bindings
	}
	worker.Enabled = input.Enabled
	worker.UpdatedAt = time.Now()

	if err := h.store.Update(c.Request().Context(), worker); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  worker,
	})
}

// Delete deletes a worker.
func (h *Workers) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// Logs returns worker logs.
func (h *Workers) Logs(c *mizu.Ctx) error {
	// In a real implementation, this would stream logs
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": []map[string]interface{}{
			{
				"timestamp": time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
				"level":     "info",
				"message":   "Worker started",
			},
			{
				"timestamp": time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
				"level":     "info",
				"message":   "Received request",
			},
			{
				"timestamp": time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
				"level":     "info",
				"message":   "Request completed",
			},
		},
	})
}

// Deploy deploys a worker.
func (h *Workers) Deploy(c *mizu.Ctx) error {
	id := c.Param("id")
	worker, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Worker not found"})
	}

	// In a real implementation, this would compile and deploy the worker
	worker.UpdatedAt = time.Now()
	if err := h.store.Update(c.Request().Context(), worker); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"id":          worker.ID,
			"name":        worker.Name,
			"deployed_at": worker.UpdatedAt,
		},
	})
}

// ListRoutes lists all worker routes for a zone.
func (h *Workers) ListRoutes(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	routes, err := h.store.ListRoutes(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  routes,
	})
}

// CreateRouteInput is the input for creating a worker route.
type CreateRouteInput struct {
	Pattern  string `json:"pattern"`
	WorkerID string `json:"worker_id"`
	Enabled  bool   `json:"enabled"`
}

// CreateRoute creates a new worker route.
func (h *Workers) CreateRoute(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateRouteInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Pattern == "" || input.WorkerID == "" {
		return c.JSON(400, map[string]string{"error": "Pattern and worker_id are required"})
	}

	route := &store.WorkerRoute{
		ID:       ulid.Make().String(),
		ZoneID:   zoneID,
		Pattern:  input.Pattern,
		WorkerID: input.WorkerID,
		Enabled:  input.Enabled,
	}

	if err := h.store.CreateRoute(c.Request().Context(), route); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  route,
	})
}

// DeleteRoute deletes a worker route.
func (h *Workers) DeleteRoute(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteRoute(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}
