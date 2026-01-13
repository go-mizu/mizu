package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Firewall handles firewall-related requests.
type Firewall struct {
	store store.FirewallStore
}

// NewFirewall creates a new Firewall handler.
func NewFirewall(store store.FirewallStore) *Firewall {
	return &Firewall{store: store}
}

// ListRules lists all firewall rules for a zone.
func (h *Firewall) ListRules(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	rules, err := h.store.ListRules(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rules,
	})
}

// CreateFirewallRuleInput is the input for creating a firewall rule.
type CreateFirewallRuleInput struct {
	Description string `json:"description"`
	Expression  string `json:"expression"`
	Action      string `json:"action"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// CreateRule creates a new firewall rule.
func (h *Firewall) CreateRule(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateFirewallRuleInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Expression == "" || input.Action == "" {
		return c.JSON(400, map[string]string{"error": "Expression and action are required"})
	}

	now := time.Now()
	rule := &store.FirewallRule{
		ID:          ulid.Make().String(),
		ZoneID:      zoneID,
		Description: input.Description,
		Expression:  input.Expression,
		Action:      input.Action,
		Priority:    input.Priority,
		Enabled:     input.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}

// UpdateRule updates a firewall rule.
func (h *Firewall) UpdateRule(c *mizu.Ctx) error {
	id := c.Param("id")
	rule, err := h.store.GetRule(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Rule not found"})
	}

	var input CreateFirewallRuleInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Description != "" {
		rule.Description = input.Description
	}
	if input.Expression != "" {
		rule.Expression = input.Expression
	}
	if input.Action != "" {
		rule.Action = input.Action
	}
	rule.Priority = input.Priority
	rule.Enabled = input.Enabled
	rule.UpdatedAt = time.Now()

	if err := h.store.UpdateRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}

// DeleteRule deletes a firewall rule.
func (h *Firewall) DeleteRule(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteRule(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListIPAccessRules lists all IP access rules for a zone.
func (h *Firewall) ListIPAccessRules(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	rules, err := h.store.ListIPAccessRules(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rules,
	})
}

// CreateIPAccessRuleInput is the input for creating an IP access rule.
type CreateIPAccessRuleInput struct {
	Mode   string `json:"mode"`   // block, challenge, whitelist
	Target string `json:"target"` // ip, ip_range, asn, country
	Value  string `json:"value"`
	Notes  string `json:"notes"`
}

// CreateIPAccessRule creates a new IP access rule.
func (h *Firewall) CreateIPAccessRule(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateIPAccessRuleInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Mode == "" || input.Value == "" {
		return c.JSON(400, map[string]string{"error": "Mode and value are required"})
	}

	if input.Target == "" {
		input.Target = "ip"
	}

	rule := &store.IPAccessRule{
		ID:        ulid.Make().String(),
		ZoneID:    zoneID,
		Mode:      input.Mode,
		Target:    input.Target,
		Value:     input.Value,
		Notes:     input.Notes,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateIPAccessRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}

// DeleteIPAccessRule deletes an IP access rule.
func (h *Firewall) DeleteIPAccessRule(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteIPAccessRule(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListRateLimits lists all rate limit rules for a zone.
func (h *Firewall) ListRateLimits(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	rules, err := h.store.ListRateLimitRules(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rules,
	})
}

// CreateRateLimitInput is the input for creating a rate limit rule.
type CreateRateLimitInput struct {
	Description   string `json:"description"`
	Expression    string `json:"expression"`
	Threshold     int    `json:"threshold"`
	Period        int    `json:"period"` // seconds
	Action        string `json:"action"`
	ActionTimeout int    `json:"action_timeout"`
	Enabled       bool   `json:"enabled"`
}

// CreateRateLimit creates a new rate limit rule.
func (h *Firewall) CreateRateLimit(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateRateLimitInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Threshold == 0 || input.Period == 0 {
		return c.JSON(400, map[string]string{"error": "Threshold and period are required"})
	}

	if input.Action == "" {
		input.Action = "block"
	}

	if input.ActionTimeout == 0 {
		input.ActionTimeout = 60
	}

	rule := &store.RateLimitRule{
		ID:            ulid.Make().String(),
		ZoneID:        zoneID,
		Description:   input.Description,
		Expression:    input.Expression,
		Threshold:     input.Threshold,
		Period:        input.Period,
		Action:        input.Action,
		ActionTimeout: input.ActionTimeout,
		Enabled:       input.Enabled,
		CreatedAt:     time.Now(),
	}

	if err := h.store.CreateRateLimitRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}
