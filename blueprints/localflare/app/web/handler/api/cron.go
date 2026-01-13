package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/cron"
)

// Cron handles Cron Trigger requests.
type Cron struct {
	svc cron.API
}

// NewCron creates a new Cron handler.
func NewCron(svc cron.API) *Cron {
	return &Cron{svc: svc}
}

// ListTriggers lists all cron triggers.
func (h *Cron) ListTriggers(c *mizu.Ctx) error {
	triggers, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"triggers": triggers,
		},
	})
}

// CreateTrigger creates a new cron trigger.
func (h *Cron) CreateTrigger(c *mizu.Ctx) error {
	var input cron.CreateTriggerIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Cron == "" || input.ScriptName == "" {
		return c.JSON(400, map[string]string{"error": "cron and script_name are required"})
	}

	trigger, err := h.svc.Create(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  trigger,
	})
}

// GetTrigger retrieves a cron trigger by ID.
func (h *Cron) GetTrigger(c *mizu.Ctx) error {
	id := c.Param("id")
	trigger, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Trigger not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  trigger,
	})
}

// UpdateTrigger updates a cron trigger.
func (h *Cron) UpdateTrigger(c *mizu.Ctx) error {
	id := c.Param("id")

	var input cron.UpdateTriggerIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	trigger, err := h.svc.Update(c.Request().Context(), id, &input)
	if err != nil {
		if err == cron.ErrNotFound {
			return c.JSON(404, map[string]string{"error": "Trigger not found"})
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  trigger,
	})
}

// DeleteTrigger deletes a cron trigger.
func (h *Cron) DeleteTrigger(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// GetExecutions retrieves recent executions for a trigger.
func (h *Cron) GetExecutions(c *mizu.Ctx) error {
	id := c.Param("id")

	limit := 50
	executions, err := h.svc.GetExecutions(c.Request().Context(), id, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"executions": executions,
		},
	})
}

// ListTriggersByScript lists triggers for a specific script.
func (h *Cron) ListTriggersByScript(c *mizu.Ctx) error {
	scriptName := c.Param("script")
	triggers, err := h.svc.ListByScript(c.Request().Context(), scriptName)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"triggers": triggers,
		},
	})
}
