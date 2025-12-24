package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/chat/app/web/ws"
	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/messages"
	"github.com/go-mizu/blueprints/chat/feature/presence"
)

// Message handles message endpoints.
type Message struct {
	messages  messages.API
	channels  channels.API
	presence  presence.API
	hub       *ws.Hub
	getUserID func(*mizu.Ctx) string
}

// NewMessage creates a new Message handler.
func NewMessage(
	messages messages.API,
	channels channels.API,
	presence presence.API,
	hub *ws.Hub,
	getUserID func(*mizu.Ctx) string,
) *Message {
	return &Message{
		messages:  messages,
		channels:  channels,
		presence:  presence,
		hub:       hub,
		getUserID: getUserID,
	}
}

// List lists messages in a channel.
func (h *Message) List(c *mizu.Ctx) error {
	channelID := c.Param("id")

	opts := messages.ListOpts{
		Limit:  50,
		Before: c.Query("before"),
		After:  c.Query("after"),
		Around: c.Query("around"),
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	msgs, err := h.messages.List(c.Request().Context(), channelID, opts)
	if err != nil {
		return InternalError(c, "Failed to list messages")
	}

	return Success(c, msgs)
}

// Create creates a new message.
func (h *Message) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")

	var in messages.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.ChannelID = channelID

	if in.Content == "" {
		return BadRequest(c, "Message content is required")
	}

	ctx := c.Request().Context()

	msg, err := h.messages.Create(ctx, userID, &in)
	if err != nil {
		return InternalError(c, "Failed to create message")
	}

	// Update channel last message
	h.channels.UpdateLastMessage(ctx, channelID, msg.ID, msg.CreatedAt)

	// Broadcast message
	ch, _ := h.channels.GetByID(ctx, channelID)
	if ch != nil {
		if ch.ServerID != "" {
			h.hub.BroadcastToServer(ch.ServerID, ws.EventMessageCreate, msg, "")
		} else {
			// DM - broadcast to channel recipients
			h.hub.BroadcastToChannel(channelID, ws.EventMessageCreate, msg, "")
		}
	}

	return Created(c, msg)
}

// Get returns a message.
func (h *Message) Get(c *mizu.Ctx) error {
	messageID := c.Param("msg_id")

	msg, err := h.messages.GetByID(c.Request().Context(), messageID)
	if err != nil {
		return NotFound(c, "Message not found")
	}

	return Success(c, msg)
}

// Update updates a message.
func (h *Message) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	messageID := c.Param("msg_id")

	// Check ownership
	msg, err := h.messages.GetByID(c.Request().Context(), messageID)
	if err != nil {
		return NotFound(c, "Message not found")
	}
	if msg.AuthorID != userID {
		return Forbidden(c, "You can only edit your own messages")
	}

	var in messages.UpdateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	ctx := c.Request().Context()

	msg, err = h.messages.Update(ctx, messageID, &in)
	if err != nil {
		return InternalError(c, "Failed to update message")
	}

	// Broadcast update
	ch, _ := h.channels.GetByID(ctx, msg.ChannelID)
	if ch != nil {
		if ch.ServerID != "" {
			h.hub.BroadcastToServer(ch.ServerID, ws.EventMessageUpdate, msg, "")
		} else {
			h.hub.BroadcastToChannel(msg.ChannelID, ws.EventMessageUpdate, msg, "")
		}
	}

	return Success(c, msg)
}

// Delete deletes a message.
func (h *Message) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")
	messageID := c.Param("msg_id")

	// Check ownership
	msg, err := h.messages.GetByID(c.Request().Context(), messageID)
	if err != nil {
		return NotFound(c, "Message not found")
	}
	if msg.AuthorID != userID {
		// TODO: Check for manage messages permission
		return Forbidden(c, "You can only delete your own messages")
	}

	if err := h.messages.Delete(c.Request().Context(), messageID); err != nil {
		return InternalError(c, "Failed to delete message")
	}

	// Broadcast deletion
	ch, _ := h.channels.GetByID(c.Request().Context(), channelID)
	deleteEvent := map[string]string{
		"id":         messageID,
		"channel_id": channelID,
	}
	if ch != nil {
		if ch.ServerID != "" {
			h.hub.BroadcastToServer(ch.ServerID, ws.EventMessageDelete, deleteEvent, "")
		} else {
			h.hub.BroadcastToChannel(channelID, ws.EventMessageDelete, deleteEvent, "")
		}
	}

	return NoContent(c)
}

// AddReaction adds a reaction to a message.
func (h *Message) AddReaction(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")
	messageID := c.Param("msg_id")
	emoji := c.Param("emoji")

	if err := h.messages.AddReaction(c.Request().Context(), messageID, userID, emoji); err != nil {
		return InternalError(c, "Failed to add reaction")
	}

	// Broadcast
	ch, _ := h.channels.GetByID(c.Request().Context(), channelID)
	reactionEvent := map[string]string{
		"message_id": messageID,
		"channel_id": channelID,
		"user_id":    userID,
		"emoji":      emoji,
	}
	if ch != nil && ch.ServerID != "" {
		h.hub.BroadcastToServer(ch.ServerID, ws.EventMessageReactionAdd, reactionEvent, "")
	}

	return NoContent(c)
}

// RemoveReaction removes a reaction from a message.
func (h *Message) RemoveReaction(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")
	messageID := c.Param("msg_id")
	emoji := c.Param("emoji")

	if err := h.messages.RemoveReaction(c.Request().Context(), messageID, userID, emoji); err != nil {
		return InternalError(c, "Failed to remove reaction")
	}

	// Broadcast
	ch, _ := h.channels.GetByID(c.Request().Context(), channelID)
	reactionEvent := map[string]string{
		"message_id": messageID,
		"channel_id": channelID,
		"user_id":    userID,
		"emoji":      emoji,
	}
	if ch != nil && ch.ServerID != "" {
		h.hub.BroadcastToServer(ch.ServerID, ws.EventMessageReactionRemove, reactionEvent, "")
	}

	return NoContent(c)
}

// ListPinned lists pinned messages.
func (h *Message) ListPinned(c *mizu.Ctx) error {
	channelID := c.Param("id")

	msgs, err := h.messages.ListPinned(c.Request().Context(), channelID)
	if err != nil {
		return InternalError(c, "Failed to list pinned messages")
	}

	return Success(c, msgs)
}

// Pin pins a message.
func (h *Message) Pin(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")
	messageID := c.Param("msg_id")

	if err := h.messages.Pin(c.Request().Context(), channelID, messageID, userID); err != nil {
		return InternalError(c, "Failed to pin message")
	}

	return NoContent(c)
}

// Unpin unpins a message.
func (h *Message) Unpin(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")
	messageID := c.Param("msg_id")

	if err := h.messages.Unpin(c.Request().Context(), channelID, messageID); err != nil {
		return InternalError(c, "Failed to unpin message")
	}

	return NoContent(c)
}

// Typing triggers a typing indicator.
func (h *Message) Typing(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	channelID := c.Param("id")
	ctx := c.Request().Context()

	h.presence.StartTyping(ctx, userID, channelID)

	// Broadcast typing
	ch, _ := h.channels.GetByID(ctx, channelID)
	typingEvent := map[string]string{
		"channel_id": channelID,
		"user_id":    userID,
	}
	if ch != nil {
		if ch.ServerID != "" {
			h.hub.BroadcastToServer(ch.ServerID, ws.EventTypingStart, typingEvent, userID)
		} else {
			h.hub.BroadcastToChannel(channelID, ws.EventTypingStart, typingEvent, userID)
		}
	}

	return NoContent(c)
}

// Search searches messages.
func (h *Message) Search(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	opts := messages.SearchOpts{
		Query:     c.Query("q"),
		ChannelID: c.Query("channel_id"),
		AuthorID:  c.Query("author_id"),
		Limit:     25,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	msgs, err := h.messages.Search(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Failed to search messages")
	}

	return Success(c, msgs)
}
