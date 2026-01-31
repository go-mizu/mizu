package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/store"
)

// MessageHandler handles message API requests.
type MessageHandler struct {
	store store.MessageStore
}

// NewMessageHandler creates a message handler.
func NewMessageHandler(s store.MessageStore) *MessageHandler {
	return &MessageHandler{store: s}
}

func (h *MessageHandler) List(c *mizu.Ctx) error {
	sessionID := c.Query("session")
	if sessionID == "" {
		return c.JSON(400, map[string]string{"error": "session query parameter is required"})
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	messages, err := h.store.ListMessages(c.Request().Context(), sessionID, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, messages)
}

// Send handles sending a message through the gateway via API.
// This is used by the dashboard or external clients.
func (h *MessageHandler) Send(c *mizu.Ctx) error {
	// This endpoint delegates to the gateway handler's Send method.
	// The gateway handler is registered separately.
	return c.JSON(501, map[string]string{"error": "use POST /api/gateway/send instead"})
}
