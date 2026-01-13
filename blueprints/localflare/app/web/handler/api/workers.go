package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/workers"
)

// Workers handles worker-related requests.
type Workers struct {
	svc workers.API
}

// NewWorkers creates a new Workers handler.
func NewWorkers(svc workers.API) *Workers {
	return &Workers{svc: svc}
}

// List lists all workers.
func (h *Workers) List(c *mizu.Ctx) error {
	result, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// Get retrieves a worker by ID.
func (h *Workers) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	worker, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Worker not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  worker,
	})
}

// Create creates a new worker.
func (h *Workers) Create(c *mizu.Ctx) error {
	var input workers.CreateIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	worker, err := h.svc.Create(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  worker,
	})
}

// Update updates a worker.
func (h *Workers) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var input workers.UpdateIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	worker, err := h.svc.Update(c.Request().Context(), id, &input)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  worker,
	})
}

// Delete deletes a worker.
func (h *Workers) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// Logs returns worker logs.
func (h *Workers) Logs(c *mizu.Ctx) error {
	id := c.Param("id")
	logs, err := h.svc.Logs(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  logs,
	})
}

// Deploy deploys a worker.
func (h *Workers) Deploy(c *mizu.Ctx) error {
	id := c.Param("id")
	result, err := h.svc.Deploy(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// ListRoutes lists all worker routes for a zone.
func (h *Workers) ListRoutes(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	routes, err := h.svc.ListRoutes(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  routes,
	})
}

// CreateRoute creates a new worker route.
func (h *Workers) CreateRoute(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input workers.CreateRouteIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}
	input.ZoneID = zoneID

	route, err := h.svc.CreateRoute(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  route,
	})
}

// DeleteRoute deletes a worker route.
func (h *Workers) DeleteRoute(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.DeleteRoute(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}
