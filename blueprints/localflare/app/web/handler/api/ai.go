package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/ai"
)

// AI handles Workers AI requests.
type AI struct {
	svc ai.API
}

// NewAI creates a new AI handler.
func NewAI(svc ai.API) *AI {
	return &AI{svc: svc}
}

// RunModel runs inference on a model.
func (h *AI) RunModel(c *mizu.Ctx) error {
	model := c.Param("model")

	var input ai.RunModelIn
	if err := c.BindJSON(&input, 10<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	result, err := h.svc.RunModel(c.Request().Context(), model, &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  result,
	})
}

// ListModels lists available models.
func (h *AI) ListModels(c *mizu.Ctx) error {
	task := c.Query("task")
	models, err := h.svc.ListModels(c.Request().Context(), task)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  models,
	})
}

// GetModel retrieves a model by name.
func (h *AI) GetModel(c *mizu.Ctx) error {
	name := c.Param("model")
	model, err := h.svc.GetModel(c.Request().Context(), name)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Model not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  model,
	})
}
