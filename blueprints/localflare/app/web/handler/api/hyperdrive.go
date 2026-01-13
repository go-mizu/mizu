package api

import (
	"github.com/go-mizu/mizu"

	hd "github.com/go-mizu/blueprints/localflare/feature/hyperdrive"
)

// Hyperdrive handles Hyperdrive requests.
type Hyperdrive struct {
	svc hd.API
}

// NewHyperdrive creates a new Hyperdrive handler.
func NewHyperdrive(svc hd.API) *Hyperdrive {
	return &Hyperdrive{svc: svc}
}

// ListConfigs lists all Hyperdrive configs.
func (h *Hyperdrive) ListConfigs(c *mizu.Ctx) error {
	configs, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"configs": configs,
		},
	})
}

// CreateConfig creates a new Hyperdrive config.
func (h *Hyperdrive) CreateConfig(c *mizu.Ctx) error {
	var input hd.CreateConfigIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	if input.Origin.Host == "" || input.Origin.Database == "" {
		return c.JSON(400, map[string]string{"error": "origin.host and origin.database are required"})
	}

	// Set defaults
	if input.Origin.Port == 0 {
		input.Origin.Port = 5432
	}
	if input.Origin.Scheme == "" {
		input.Origin.Scheme = "postgres"
	}
	if input.Caching.MaxAge == 0 {
		input.Caching.MaxAge = 60
	}
	if input.Caching.StaleWhileRevalidate == 0 {
		input.Caching.StaleWhileRevalidate = 15
	}

	cfg, err := h.svc.Create(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  cfg,
	})
}

// GetConfig retrieves a Hyperdrive config by ID.
func (h *Hyperdrive) GetConfig(c *mizu.Ctx) error {
	id := c.Param("id")
	cfg, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Config not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  cfg,
	})
}

// UpdateConfig updates a Hyperdrive config.
func (h *Hyperdrive) UpdateConfig(c *mizu.Ctx) error {
	id := c.Param("id")

	var input hd.UpdateConfigIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	cfg, err := h.svc.Update(c.Request().Context(), id, &input)
	if err != nil {
		if err == hd.ErrNotFound {
			return c.JSON(404, map[string]string{"error": "Config not found"})
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  cfg,
	})
}

// DeleteConfig deletes a Hyperdrive config.
func (h *Hyperdrive) DeleteConfig(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// GetStats retrieves stats for a Hyperdrive config.
func (h *Hyperdrive) GetStats(c *mizu.Ctx) error {
	id := c.Param("id")
	stats, err := h.svc.GetStats(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  stats,
	})
}
