package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/feature/gateway"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// GatewayHandler handles gateway status and routing API requests.
type GatewayHandler struct {
	svc   *gateway.Service
	store store.Store
	port  int
}

// NewGatewayHandler creates a gateway handler.
func NewGatewayHandler(svc *gateway.Service, s store.Store) *GatewayHandler {
	return &GatewayHandler{svc: svc, store: s, port: 18789}
}

func (h *GatewayHandler) Status(c *mizu.Ctx) error {
	status, err := h.svc.Status(c.Request().Context(), h.port)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, status)
}

func (h *GatewayHandler) Health(c *mizu.Ctx) error {
	return c.JSON(200, map[string]string{"status": "ok"})
}

func (h *GatewayHandler) Commands(c *mizu.Ctx) error {
	return c.JSON(200, h.svc.Commands())
}

// Send processes a message through the gateway (API-driven).
func (h *GatewayHandler) Send(c *mizu.Ctx) error {
	var msg types.InboundMessage
	if err := c.BindJSON(&msg, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}

	if msg.Content == "" {
		return c.JSON(400, map[string]string{"error": "content is required"})
	}
	if msg.ChannelType == "" {
		msg.ChannelType = types.ChannelWebhook
	}
	if msg.PeerID == "" {
		msg.PeerID = "api-client"
	}
	if msg.Origin == "" {
		msg.Origin = "dm"
	}

	response, err := h.svc.ProcessMessage(c.Request().Context(), &msg)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"response": response})
}

// ListBindings returns all routing bindings.
func (h *GatewayHandler) ListBindings(c *mizu.Ctx) error {
	bindings, err := h.store.ListBindings(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, bindings)
}

// CreateBinding creates a new routing binding.
func (h *GatewayHandler) CreateBinding(c *mizu.Ctx) error {
	var b types.Binding
	if err := c.BindJSON(&b, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}
	if b.AgentID == "" {
		return c.JSON(400, map[string]string{"error": "agentId is required"})
	}
	if err := h.store.CreateBinding(c.Request().Context(), &b); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, b)
}

// DeleteBinding removes a routing binding.
func (h *GatewayHandler) DeleteBinding(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteBinding(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"deleted": id})
}
