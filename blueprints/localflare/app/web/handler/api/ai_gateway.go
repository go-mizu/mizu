package api

import (
	"strconv"

	"github.com/go-mizu/mizu"

	gw "github.com/go-mizu/blueprints/localflare/feature/ai_gateway"
)

// AIGateway handles AI Gateway requests.
type AIGateway struct {
	svc gw.API
}

// NewAIGateway creates a new AIGateway handler.
func NewAIGateway(svc gw.API) *AIGateway {
	return &AIGateway{svc: svc}
}

// ListGateways lists all gateways.
func (h *AIGateway) ListGateways(c *mizu.Ctx) error {
	gateways, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"gateways": gateways,
		},
	})
}

// CreateGateway creates a new gateway.
func (h *AIGateway) CreateGateway(c *mizu.Ctx) error {
	var input gw.CreateGatewayIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	// Set defaults
	if input.CacheTTL == 0 {
		input.CacheTTL = 300
	}

	gateway, err := h.svc.Create(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  gateway,
	})
}

// GetGateway retrieves a gateway.
func (h *AIGateway) GetGateway(c *mizu.Ctx) error {
	id := c.Param("id")
	gateway, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Gateway not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  gateway,
	})
}

// UpdateGateway updates a gateway.
func (h *AIGateway) UpdateGateway(c *mizu.Ctx) error {
	id := c.Param("id")

	var input gw.UpdateGatewayIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	gateway, err := h.svc.Update(c.Request().Context(), id, &input)
	if err != nil {
		if err == gw.ErrNotFound {
			return c.JSON(404, map[string]string{"error": "Gateway not found"})
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  gateway,
	})
}

// DeleteGateway deletes a gateway.
func (h *AIGateway) DeleteGateway(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// GetLogs retrieves logs for a gateway.
func (h *AIGateway) GetLogs(c *mizu.Ctx) error {
	id := c.Param("id")

	// Parse query params
	limit := 100
	offset := 0

	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	logs, err := h.svc.GetLogs(c.Request().Context(), id, limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"logs":  logs,
			"total": len(logs),
		},
	})
}
