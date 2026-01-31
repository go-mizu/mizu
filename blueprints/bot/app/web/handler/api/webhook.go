package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/feature/gateway"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// WebhookHandler handles inbound webhook messages from external services.
type WebhookHandler struct {
	gateway *gateway.Service
}

// NewWebhookHandler creates a webhook handler.
func NewWebhookHandler(gw *gateway.Service) *WebhookHandler {
	return &WebhookHandler{gateway: gw}
}

// Receive processes an inbound webhook message.
func (h *WebhookHandler) Receive(c *mizu.Ctx) error {
	channelID := c.Param("channelId")

	var body struct {
		PeerID   string `json:"peerId"`
		PeerName string `json:"peerName"`
		Content  string `json:"content"`
		Origin   string `json:"origin"`
	}
	if err := c.BindJSON(&body, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}

	if body.Content == "" {
		return c.JSON(400, map[string]string{"error": "content is required"})
	}
	if body.PeerID == "" {
		body.PeerID = "webhook-client"
	}
	if body.Origin == "" {
		body.Origin = "webhook"
	}

	msg := &types.InboundMessage{
		ChannelType: types.ChannelWebhook,
		ChannelID:   channelID,
		PeerID:      body.PeerID,
		PeerName:    body.PeerName,
		Content:     body.Content,
		Origin:      body.Origin,
	}

	response, err := h.gateway.ProcessMessage(c.Request().Context(), msg)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"response": response})
}
