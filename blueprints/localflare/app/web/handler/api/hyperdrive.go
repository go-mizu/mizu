package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Hyperdrive handles Hyperdrive requests.
type Hyperdrive struct {
	store store.HyperdriveStore
}

// NewHyperdrive creates a new Hyperdrive handler.
func NewHyperdrive(store store.HyperdriveStore) *Hyperdrive {
	return &Hyperdrive{store: store}
}

// ListConfigs lists all Hyperdrive configs.
func (h *Hyperdrive) ListConfigs(c *mizu.Ctx) error {
	configs, err := h.store.ListConfigs(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  configs,
	})
}

// CreateConfigInput is the input for creating a Hyperdrive config.
type CreateHyperdriveConfigInput struct {
	Name   string                      `json:"name"`
	Origin store.HyperdriveOrigin      `json:"origin"`
	Caching store.HyperdriveCaching    `json:"caching"`
}

// CreateConfig creates a new Hyperdrive config.
func (h *Hyperdrive) CreateConfig(c *mizu.Ctx) error {
	var input CreateHyperdriveConfigInput
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

	cfg := &store.HyperdriveConfig{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		Origin:    input.Origin,
		Caching:   input.Caching,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateConfig(c.Request().Context(), cfg); err != nil {
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
	cfg, err := h.store.GetConfig(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Config not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  cfg,
	})
}

// UpdateConfigInput is the input for updating a Hyperdrive config.
type UpdateHyperdriveConfigInput struct {
	Name    *string                    `json:"name"`
	Origin  *store.HyperdriveOrigin    `json:"origin"`
	Caching *store.HyperdriveCaching   `json:"caching"`
}

// UpdateConfig updates a Hyperdrive config.
func (h *Hyperdrive) UpdateConfig(c *mizu.Ctx) error {
	id := c.Param("id")
	cfg, err := h.store.GetConfig(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Config not found"})
	}

	var input UpdateHyperdriveConfigInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name != nil {
		cfg.Name = *input.Name
	}
	if input.Origin != nil {
		cfg.Origin = *input.Origin
	}
	if input.Caching != nil {
		cfg.Caching = *input.Caching
	}

	if err := h.store.UpdateConfig(c.Request().Context(), cfg); err != nil {
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
	if err := h.store.DeleteConfig(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// GetStats retrieves stats for a Hyperdrive config.
func (h *Hyperdrive) GetStats(c *mizu.Ctx) error {
	id := c.Param("id")
	stats, err := h.store.GetStats(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  stats,
	})
}
