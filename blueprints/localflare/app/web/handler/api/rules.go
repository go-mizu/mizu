package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Rules handles rules-related requests.
type Rules struct {
	store store.RulesStore
}

// NewRules creates a new Rules handler.
func NewRules(store store.RulesStore) *Rules {
	return &Rules{store: store}
}

// ListPageRules lists all page rules for a zone.
func (h *Rules) ListPageRules(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	rules, err := h.store.ListPageRules(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rules,
	})
}

// CreatePageRuleInput is the input for creating a page rule.
type CreatePageRuleInput struct {
	Targets  []string               `json:"targets"`
	Actions  map[string]interface{} `json:"actions"`
	Priority int                    `json:"priority"`
	Status   string                 `json:"status"`
}

// CreatePageRule creates a new page rule.
func (h *Rules) CreatePageRule(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreatePageRuleInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if len(input.Targets) == 0 {
		return c.JSON(400, map[string]string{"error": "Targets are required"})
	}

	rule := &store.PageRule{
		ID:        ulid.Make().String(),
		ZoneID:    zoneID,
		Targets:   input.Targets,
		Actions:   input.Actions,
		Priority:  input.Priority,
		Status:    input.Status,
		CreatedAt: time.Now(),
	}

	if rule.Status == "" {
		rule.Status = "active"
	}

	if err := h.store.CreatePageRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}

// DeletePageRule deletes a page rule.
func (h *Rules) DeletePageRule(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeletePageRule(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListTransformRules lists all transform rules for a zone.
func (h *Rules) ListTransformRules(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	rules, err := h.store.ListTransformRules(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rules,
	})
}

// CreateTransformRuleInput is the input for creating a transform rule.
type CreateTransformRuleInput struct {
	Type        string `json:"type"`
	Expression  string `json:"expression"`
	Action      string `json:"action"`
	ActionValue string `json:"action_value"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// CreateTransformRule creates a new transform rule.
func (h *Rules) CreateTransformRule(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateTransformRuleInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Type == "" || input.Expression == "" {
		return c.JSON(400, map[string]string{"error": "Type and expression are required"})
	}

	rule := &store.TransformRule{
		ID:          ulid.Make().String(),
		ZoneID:      zoneID,
		Type:        input.Type,
		Expression:  input.Expression,
		Action:      input.Action,
		ActionValue: input.ActionValue,
		Priority:    input.Priority,
		Enabled:     input.Enabled,
		CreatedAt:   time.Now(),
	}

	if err := h.store.CreateTransformRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}

// DeleteTransformRule deletes a transform rule.
func (h *Rules) DeleteTransformRule(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteTransformRule(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}
