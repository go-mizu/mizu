package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Zones handles zone-related requests.
type Zones struct {
	store store.ZoneStore
}

// NewZones creates a new Zones handler.
func NewZones(store store.ZoneStore) *Zones {
	return &Zones{store: store}
}

// List lists all zones.
func (h *Zones) List(c *mizu.Ctx) error {
	zones, err := h.store.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  zones,
	})
}

// Get retrieves a zone by ID.
func (h *Zones) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	zone, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Zone not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  zone,
	})
}

// CreateInput is the input for creating a zone.
type CreateZoneInput struct {
	Name string `json:"name"`
	Plan string `json:"plan"`
}

// Create creates a new zone.
func (h *Zones) Create(c *mizu.Ctx) error {
	var input CreateZoneInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "Name is required"})
	}

	now := time.Now()
	zone := &store.Zone{
		ID:          ulid.Make().String(),
		Name:        input.Name,
		Status:      "active",
		Plan:        input.Plan,
		NameServers: []string{"ns1.localflare.local", "ns2.localflare.local"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if zone.Plan == "" {
		zone.Plan = "free"
	}

	if err := h.store.Create(c.Request().Context(), zone); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  zone,
	})
}

// UpdateZoneInput is the input for updating a zone.
type UpdateZoneInput struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Plan   string `json:"plan"`
}

// Update updates a zone.
func (h *Zones) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	zone, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Zone not found"})
	}

	var input UpdateZoneInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name != "" {
		zone.Name = input.Name
	}
	if input.Status != "" {
		zone.Status = input.Status
	}
	if input.Plan != "" {
		zone.Plan = input.Plan
	}
	zone.UpdatedAt = time.Now()

	if err := h.store.Update(c.Request().Context(), zone); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  zone,
	})
}

// Delete deletes a zone.
func (h *Zones) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}
