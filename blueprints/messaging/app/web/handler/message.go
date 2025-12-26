package handler

import (
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/app/web/ws"
	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
	"github.com/go-mizu/blueprints/messaging/pkg/sanitize"
)

// Message handles message endpoints.
type Message struct {
	messages  messages.API
	chats     chats.API
	accounts  accounts.API
	hub       *ws.Hub
	getUserID func(*mizu.Ctx) string
}

// NewMessage creates a new Message handler.
func NewMessage(msgs messages.API, chats chats.API, accounts accounts.API, hub *ws.Hub, getUserID func(*mizu.Ctx) string) *Message {
	return &Message{
		messages:  msgs,
		chats:     chats,
		accounts:  accounts,
		hub:       hub,
		getUserID: getUserID,
	}
}

// List lists messages in a chat.
func (h *Message) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	before := c.Query("before")
	after := c.Query("after")

	opts := messages.ListOpts{
		Limit:  limit,
		Before: before,
		After:  after,
	}

	msgs, err := h.messages.List(c.Request().Context(), chatID, opts)
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

	chatID := c.Param("id")
	var in messages.CreateIn
	if err := c.BindJSON(&in, 10<<20); err != nil { // 10MB for messages with media
		return BadRequest(c, "Invalid request body")
	}
	in.ChatID = chatID

	// Determine if this is a media/sticker message
	isMediaMessage := in.MediaID != "" || in.MediaURL != ""
	isStickerMessage := in.StickerPackID != "" && in.StickerID != ""
	isTextMessage := !isMediaMessage && !isStickerMessage

	// Validate content - only required for text messages
	if isTextMessage && strings.TrimSpace(in.Content) == "" {
		return BadRequest(c, "Message content cannot be empty")
	}

	// Sanitize content if present
	if in.Content != "" {
		content, err := sanitize.MessageContent(in.Content)
		if err != nil {
			return BadRequest(c, err.Error())
		}
		in.Content = content
	}

	// Set default type based on content
	if in.Type == "" {
		if isStickerMessage {
			in.Type = messages.TypeSticker
		} else if isMediaMessage {
			// Determine type from media_type
			switch in.MediaType {
			case "image":
				in.Type = messages.TypeImage
			case "video":
				in.Type = messages.TypeVideo
			case "audio":
				in.Type = messages.TypeAudio
			case "voice":
				in.Type = messages.TypeVoice
			case "document":
				in.Type = messages.TypeDocument
			default:
				in.Type = messages.TypeImage // default for media
			}
		} else {
			in.Type = messages.TypeText
		}
	}

	msg, err := h.messages.Create(c.Request().Context(), userID, &in)
	if err != nil {
		return InternalError(c, "Failed to create message")
	}

	// Get sender info
	sender, _ := h.accounts.GetByID(c.Request().Context(), userID)
	msg.Sender = sender

	// Update chat last message
	h.chats.UpdateLastMessage(c.Request().Context(), chatID, msg.ID)
	h.chats.IncrementMessageCount(c.Request().Context(), chatID)

	// Broadcast to chat
	h.hub.BroadcastToChat(chatID, ws.EventMessageCreate, msg, userID)

	return Created(c, msg)
}

// Get retrieves a message.
func (h *Message) Get(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	msgID := c.Param("msg_id")
	msg, err := h.messages.GetByID(c.Request().Context(), msgID)
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

	chatID := c.Param("id")
	msgID := c.Param("msg_id")
	var in messages.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	// Sanitize content if being updated
	if in.Content != nil && *in.Content != "" {
		content, err := sanitize.MessageContent(*in.Content)
		if err != nil {
			return BadRequest(c, err.Error())
		}
		in.Content = &content
	}

	msg, err := h.messages.Update(c.Request().Context(), msgID, userID, &in)
	if err != nil {
		if err == messages.ErrForbidden {
			return Forbidden(c, "Cannot edit this message")
		}
		return InternalError(c, "Failed to update message")
	}

	// Broadcast update
	h.hub.BroadcastToChat(chatID, ws.EventMessageUpdate, msg, "")

	return Success(c, msg)
}

// Delete deletes a message.
func (h *Message) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	msgID := c.Param("msg_id")
	forEveryone := c.Query("for_everyone") == "true"

	if err := h.messages.Delete(c.Request().Context(), msgID, userID, forEveryone); err != nil {
		if err == messages.ErrForbidden {
			return Forbidden(c, "Cannot delete this message")
		}
		return InternalError(c, "Failed to delete message")
	}

	// Broadcast deletion
	if forEveryone {
		h.hub.BroadcastToChat(chatID, ws.EventMessageDelete, map[string]any{
			"id":      msgID,
			"chat_id": chatID,
		}, "")
	}

	return Success(c, nil)
}

// AddReaction adds a reaction to a message.
func (h *Message) AddReaction(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	msgID := c.Param("msg_id")
	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if err := h.messages.AddReaction(c.Request().Context(), msgID, userID, req.Emoji); err != nil {
		return InternalError(c, "Failed to add reaction")
	}

	return Success(c, nil)
}

// RemoveReaction removes a reaction from a message.
func (h *Message) RemoveReaction(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	msgID := c.Param("msg_id")
	if err := h.messages.RemoveReaction(c.Request().Context(), msgID, userID); err != nil {
		return InternalError(c, "Failed to remove reaction")
	}

	return Success(c, nil)
}

// Forward forwards a message.
func (h *Message) Forward(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	msgID := c.Param("msg_id")
	var in messages.ForwardIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	forwarded, err := h.messages.Forward(c.Request().Context(), msgID, userID, &in)
	if err != nil {
		return InternalError(c, "Failed to forward message")
	}

	// Broadcast to destination chats
	for _, msg := range forwarded {
		h.hub.BroadcastToChat(msg.ChatID, ws.EventMessageCreate, msg, userID)
	}

	return Success(c, forwarded)
}

// Star stars a message.
func (h *Message) Star(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	msgID := c.Param("msg_id")
	if err := h.messages.Star(c.Request().Context(), msgID, userID); err != nil {
		return InternalError(c, "Failed to star message")
	}

	return Success(c, nil)
}

// Unstar unstars a message.
func (h *Message) Unstar(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	msgID := c.Param("msg_id")
	if err := h.messages.Unstar(c.Request().Context(), msgID, userID); err != nil {
		return InternalError(c, "Failed to unstar message")
	}

	return Success(c, nil)
}

// ListStarred lists starred messages.
func (h *Message) ListStarred(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	msgs, err := h.messages.ListStarred(c.Request().Context(), userID, limit)
	if err != nil {
		return InternalError(c, "Failed to list starred messages")
	}

	return Success(c, msgs)
}

// ListPinned lists pinned messages.
func (h *Message) ListPinned(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	msgs, err := h.messages.ListPinned(c.Request().Context(), chatID)
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

	chatID := c.Param("id")
	msgID := c.Param("msg_id")
	if err := h.messages.Pin(c.Request().Context(), chatID, msgID, userID); err != nil {
		return InternalError(c, "Failed to pin message")
	}

	return Success(c, nil)
}

// Unpin unpins a message.
func (h *Message) Unpin(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	msgID := c.Param("msg_id")
	if err := h.messages.Unpin(c.Request().Context(), chatID, msgID); err != nil {
		return InternalError(c, "Failed to unpin message")
	}

	return Success(c, nil)
}

// Search searches messages.
func (h *Message) Search(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	query := sanitize.SearchQuery(c.Query("q"))
	if query == "" {
		return BadRequest(c, "Search query is required")
	}

	chatID := c.Query("chat_id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	opts := messages.SearchOpts{
		Query:  query,
		ChatID: chatID,
		Limit:  limit,
	}

	msgs, err := h.messages.Search(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Failed to search messages")
	}

	return Success(c, msgs)
}

// Typing sends a typing indicator.
func (h *Message) Typing(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	h.hub.BroadcastToChat(chatID, ws.EventTypingStart, map[string]any{
		"user_id": userID,
		"chat_id": chatID,
	}, userID)

	return Success(c, nil)
}
