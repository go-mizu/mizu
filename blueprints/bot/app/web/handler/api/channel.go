package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ChannelHandler handles channel API requests.
type ChannelHandler struct {
	store store.ChannelStore
}

// NewChannelHandler creates a channel handler.
func NewChannelHandler(s store.ChannelStore) *ChannelHandler {
	return &ChannelHandler{store: s}
}

func (h *ChannelHandler) List(c *mizu.Ctx) error {
	channels, err := h.store.ListChannels(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, channels)
}

func (h *ChannelHandler) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	ch, err := h.store.GetChannel(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, ch)
}

func (h *ChannelHandler) Create(c *mizu.Ctx) error {
	var ch types.Channel
	if err := c.BindJSON(&ch, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}
	if ch.ID == "" || ch.Name == "" || ch.Type == "" {
		return c.JSON(400, map[string]string{"error": "id, name, and type are required"})
	}
	if err := h.store.CreateChannel(c.Request().Context(), &ch); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, ch)
}

func (h *ChannelHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var ch types.Channel
	if err := c.BindJSON(&ch, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON: " + err.Error()})
	}
	ch.ID = id
	if err := h.store.UpdateChannel(c.Request().Context(), &ch); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, ch)
}

func (h *ChannelHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteChannel(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"deleted": id})
}
