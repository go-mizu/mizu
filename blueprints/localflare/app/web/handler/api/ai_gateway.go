package api

import (
	"strconv"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// AIGateway handles AI Gateway requests.
type AIGateway struct {
	store store.AIGatewayStore
}

// NewAIGateway creates a new AIGateway handler.
func NewAIGateway(store store.AIGatewayStore) *AIGateway {
	return &AIGateway{store: store}
}

// ListGateways lists all gateways.
func (h *AIGateway) ListGateways(c *mizu.Ctx) error {
	gateways, err := h.store.ListGateways(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  gateways,
	})
}

// CreateGatewayInput is the input for creating a gateway.
type CreateGatewayInput struct {
	Name             string `json:"name"`
	CollectLogs      bool   `json:"collect_logs"`
	CacheEnabled     bool   `json:"cache_enabled"`
	CacheTTL         int    `json:"cache_ttl"`
	RateLimitEnabled bool   `json:"rate_limit_enabled"`
	RateLimitCount   int    `json:"rate_limit_count"`
	RateLimitPeriod  int    `json:"rate_limit_period"`
}

// CreateGateway creates a new gateway.
func (h *AIGateway) CreateGateway(c *mizu.Ctx) error {
	var input CreateGatewayInput
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

	gw := &store.AIGateway{
		ID:               ulid.Make().String(),
		Name:             input.Name,
		CollectLogs:      input.CollectLogs,
		CacheEnabled:     input.CacheEnabled,
		CacheTTL:         input.CacheTTL,
		RateLimitEnabled: input.RateLimitEnabled,
		RateLimitCount:   input.RateLimitCount,
		RateLimitPeriod:  input.RateLimitPeriod,
		CreatedAt:        time.Now(),
	}

	if err := h.store.CreateGateway(c.Request().Context(), gw); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  gw,
	})
}

// GetGateway retrieves a gateway.
func (h *AIGateway) GetGateway(c *mizu.Ctx) error {
	id := c.Param("id")
	gw, err := h.store.GetGateway(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Gateway not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  gw,
	})
}

// UpdateGatewayInput is the input for updating a gateway.
type UpdateGatewayInput struct {
	CollectLogs      *bool `json:"collect_logs"`
	CacheEnabled     *bool `json:"cache_enabled"`
	CacheTTL         *int  `json:"cache_ttl"`
	RateLimitEnabled *bool `json:"rate_limit_enabled"`
	RateLimitCount   *int  `json:"rate_limit_count"`
	RateLimitPeriod  *int  `json:"rate_limit_period"`
}

// UpdateGateway updates a gateway.
func (h *AIGateway) UpdateGateway(c *mizu.Ctx) error {
	id := c.Param("id")
	gw, err := h.store.GetGateway(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Gateway not found"})
	}

	var input UpdateGatewayInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.CollectLogs != nil {
		gw.CollectLogs = *input.CollectLogs
	}
	if input.CacheEnabled != nil {
		gw.CacheEnabled = *input.CacheEnabled
	}
	if input.CacheTTL != nil {
		gw.CacheTTL = *input.CacheTTL
	}
	if input.RateLimitEnabled != nil {
		gw.RateLimitEnabled = *input.RateLimitEnabled
	}
	if input.RateLimitCount != nil {
		gw.RateLimitCount = *input.RateLimitCount
	}
	if input.RateLimitPeriod != nil {
		gw.RateLimitPeriod = *input.RateLimitPeriod
	}

	if err := h.store.UpdateGateway(c.Request().Context(), gw); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  gw,
	})
}

// DeleteGateway deletes a gateway.
func (h *AIGateway) DeleteGateway(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteGateway(c.Request().Context(), id); err != nil {
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

	logs, err := h.store.GetLogs(c.Request().Context(), id, limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  logs,
	})
}
