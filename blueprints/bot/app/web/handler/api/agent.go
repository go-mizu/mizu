package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/feature/agent"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// AgentHandler handles agent API requests.
type AgentHandler struct {
	svc *agent.Service
}

// NewAgentHandler creates an agent handler.
func NewAgentHandler(svc *agent.Service) *AgentHandler {
	return &AgentHandler{svc: svc}
}

func (h *AgentHandler) List(c *mizu.Ctx) error {
	agents, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, agents)
}

func (h *AgentHandler) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	a, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, a)
}

func (h *AgentHandler) Create(c *mizu.Ctx) error {
	var a types.Agent
	if err := c.BindJSON(&a, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}
	if a.ID == "" || a.Name == "" {
		return c.JSON(400, map[string]string{"error": "id and name are required"})
	}
	if err := h.svc.Create(c.Request().Context(), &a); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, a)
}

func (h *AgentHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var a types.Agent
	if err := c.BindJSON(&a, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}
	a.ID = id
	if err := h.svc.Update(c.Request().Context(), &a); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, a)
}

func (h *AgentHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"deleted": id})
}
