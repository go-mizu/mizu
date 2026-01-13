package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Cron handles Cron Trigger requests.
type Cron struct {
	store store.CronStore
}

// NewCron creates a new Cron handler.
func NewCron(store store.CronStore) *Cron {
	return &Cron{store: store}
}

// ListTriggers lists all cron triggers.
func (h *Cron) ListTriggers(c *mizu.Ctx) error {
	triggers, err := h.store.ListTriggers(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"schedules": triggers,
		},
	})
}

// CreateTriggerInput is the input for creating a cron trigger.
type CreateTriggerInput struct {
	Cron       string `json:"cron"`
	ScriptName string `json:"script_name"`
}

// CreateTrigger creates a new cron trigger.
func (h *Cron) CreateTrigger(c *mizu.Ctx) error {
	var input CreateTriggerInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Cron == "" || input.ScriptName == "" {
		return c.JSON(400, map[string]string{"error": "cron and script_name are required"})
	}

	now := time.Now()
	trigger := &store.CronTrigger{
		ID:         ulid.Make().String(),
		ScriptName: input.ScriptName,
		Cron:       input.Cron,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.store.CreateTrigger(c.Request().Context(), trigger); err != nil {
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
	trigger, err := h.store.GetTrigger(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Trigger not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  trigger,
	})
}

// UpdateTriggerInput is the input for updating a cron trigger.
type UpdateTriggerInput struct {
	Cron    *string `json:"cron"`
	Enabled *bool   `json:"enabled"`
}

// UpdateTrigger updates a cron trigger.
func (h *Cron) UpdateTrigger(c *mizu.Ctx) error {
	id := c.Param("id")
	trigger, err := h.store.GetTrigger(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Trigger not found"})
	}

	var input UpdateTriggerInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Cron != nil {
		trigger.Cron = *input.Cron
	}
	if input.Enabled != nil {
		trigger.Enabled = *input.Enabled
	}

	if err := h.store.UpdateTrigger(c.Request().Context(), trigger); err != nil {
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
	if err := h.store.DeleteTrigger(c.Request().Context(), id); err != nil {
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
	executions, err := h.store.GetRecentExecutions(c.Request().Context(), id, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  executions,
	})
}

// ListTriggersByScript lists triggers for a specific script.
func (h *Cron) ListTriggersByScript(c *mizu.Ctx) error {
	scriptName := c.Param("script")
	triggers, err := h.store.ListTriggersByScript(c.Request().Context(), scriptName)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"schedules": triggers,
		},
	})
}
