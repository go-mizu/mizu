package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/chat/app/web/ws"
	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/members"
)

// Channel handles channel endpoints.
type Channel struct {
	channels  channels.API
	members   members.API
	hub       *ws.Hub
	getUserID func(*mizu.Ctx) string
}

// NewChannel creates a new Channel handler.
func NewChannel(
	channels channels.API,
	members members.API,
	hub *ws.Hub,
	getUserID func(*mizu.Ctx) string,
) *Channel {
	return &Channel{
		channels:  channels,
		members:   members,
		hub:       hub,
		getUserID: getUserID,
	}
}

// Create creates a new channel.
func (h *Channel) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	serverID := c.Param("id")

	var in channels.CreateIn
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.ServerID = serverID
	if in.Type == "" {
		in.Type = channels.TypeText
	}
	if in.Name == "" {
		return BadRequest(c, "Channel name is required")
	}

	ch, err := h.channels.Create(c.Request().Context(), &in)
	if err != nil {
		return InternalError(c, "Failed to create channel")
	}

	// Broadcast to server
	h.hub.BroadcastToServer(serverID, ws.EventChannelCreate, ch, "")

	return Created(c, ch)
}

// Get returns a channel.
func (h *Channel) Get(c *mizu.Ctx) error {
	channelID := c.Param("id")

	ch, err := h.channels.GetByID(c.Request().Context(), channelID)
	if err != nil {
		return NotFound(c, "Channel not found")
	}

	return Success(c, ch)
}

// Update updates a channel.
func (h *Channel) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")

	var in channels.UpdateIn
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	ch, err := h.channels.Update(c.Request().Context(), channelID, &in)
	if err != nil {
		return InternalError(c, "Failed to update channel")
	}

	// Broadcast to server
	if ch.ServerID != "" {
		h.hub.BroadcastToServer(ch.ServerID, ws.EventChannelUpdate, ch, "")
	}

	return Success(c, ch)
}

// Delete deletes a channel.
func (h *Channel) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")

	ch, err := h.channels.GetByID(c.Request().Context(), channelID)
	if err != nil {
		return NotFound(c, "Channel not found")
	}

	if err := h.channels.Delete(c.Request().Context(), channelID); err != nil {
		return InternalError(c, "Failed to delete channel")
	}

	// Broadcast to server
	if ch.ServerID != "" {
		h.hub.BroadcastToServer(ch.ServerID, ws.EventChannelDelete, map[string]string{
			"id":        channelID,
			"server_id": ch.ServerID,
		}, "")
	}

	return NoContent(c)
}

// ListDMs lists DM channels for the current user.
func (h *Channel) ListDMs(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chs, err := h.channels.ListDMsByUser(c.Request().Context(), userID)
	if err != nil {
		return InternalError(c, "Failed to list DMs")
	}

	return Success(c, chs)
}

// CreateDM creates or gets a DM channel.
func (h *Channel) CreateDM(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	var in struct {
		RecipientID string `json:"recipient_id"`
	}
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.RecipientID == "" {
		return BadRequest(c, "Recipient ID is required")
	}

	ch, err := h.channels.GetOrCreateDM(c.Request().Context(), userID, in.RecipientID)
	if err != nil {
		return InternalError(c, "Failed to create DM")
	}

	return Success(c, ch)
}

// CreateGroupDM creates a group DM.
func (h *Channel) CreateGroupDM(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	var in struct {
		RecipientIDs []string `json:"recipient_ids"`
		Name         string   `json:"name"`
	}
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if len(in.RecipientIDs) == 0 {
		return BadRequest(c, "At least one recipient is required")
	}

	ch, err := h.channels.CreateGroupDM(c.Request().Context(), userID, in.RecipientIDs, in.Name)
	if err != nil {
		return InternalError(c, "Failed to create group DM")
	}

	return Created(c, ch)
}

// CreateCategory creates a new category.
func (h *Channel) CreateCategory(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	serverID := c.Param("id")

	var in struct {
		Name     string `json:"name"`
		Position int    `json:"position"`
	}
	if err := c.Bind(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Name == "" {
		return BadRequest(c, "Category name is required")
	}

	cat, err := h.channels.CreateCategory(c.Request().Context(), serverID, in.Name, in.Position)
	if err != nil {
		return InternalError(c, "Failed to create category")
	}

	return Created(c, cat)
}

// ListCategories lists categories in a server.
func (h *Channel) ListCategories(c *mizu.Ctx) error {
	serverID := c.Param("id")

	cats, err := h.channels.ListCategories(c.Request().Context(), serverID)
	if err != nil {
		return InternalError(c, "Failed to list categories")
	}

	return Success(c, cats)
}
