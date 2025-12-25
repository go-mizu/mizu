package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/feature/chats"
)

// Chat handles chat endpoints.
type Chat struct {
	chats     chats.API
	getUserID func(*mizu.Ctx) string
}

// NewChat creates a new Chat handler.
func NewChat(chats chats.API, getUserID func(*mizu.Ctx) string) *Chat {
	return &Chat{chats: chats, getUserID: getUserID}
}

// List lists chats for the current user.
func (h *Chat) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	includeArchived := c.Query("archived") == "true"

	opts := chats.ListOpts{
		Limit:           limit,
		Offset:          offset,
		IncludeArchived: includeArchived,
	}

	chatList, err := h.chats.List(c.Request().Context(), userID, opts)
	if err != nil {
		return InternalError(c, "Failed to list chats")
	}

	return Success(c, chatList)
}

// Create creates a new chat.
func (h *Chat) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	var req struct {
		Type           string   `json:"type"`
		RecipientID    string   `json:"recipient_id,omitempty"`
		Name           string   `json:"name,omitempty"`
		Description    string   `json:"description,omitempty"`
		ParticipantIDs []string `json:"participant_ids,omitempty"`
	}

	if err := c.BindJSON(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	var chat *chats.Chat
	var err error

	switch req.Type {
	case "direct", "":
		if req.RecipientID == "" {
			return BadRequest(c, "Recipient ID is required")
		}
		chat, err = h.chats.CreateDirect(c.Request().Context(), userID, &chats.CreateDirectIn{
			RecipientID: req.RecipientID,
		})
	case "group":
		if req.Name == "" {
			return BadRequest(c, "Group name is required")
		}
		chat, err = h.chats.CreateGroup(c.Request().Context(), userID, &chats.CreateGroupIn{
			Name:           req.Name,
			Description:    req.Description,
			ParticipantIDs: req.ParticipantIDs,
		})
	default:
		return BadRequest(c, "Invalid chat type")
	}

	if err != nil {
		return InternalError(c, "Failed to create chat")
	}

	return Created(c, chat)
}

// Get retrieves a chat.
func (h *Chat) Get(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	chat, err := h.chats.GetByIDForUser(c.Request().Context(), chatID, userID)
	if err != nil {
		return NotFound(c, "Chat not found")
	}

	return Success(c, chat)
}

// Update updates a chat.
func (h *Chat) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	var in chats.UpdateIn
	if err := c.BindJSON(&in); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	chat, err := h.chats.Update(c.Request().Context(), chatID, &in)
	if err != nil {
		return InternalError(c, "Failed to update chat")
	}

	return Success(c, chat)
}

// Delete deletes a chat.
func (h *Chat) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Delete(c.Request().Context(), chatID); err != nil {
		return InternalError(c, "Failed to delete chat")
	}

	return Success(c, nil)
}

// Archive archives a chat.
func (h *Chat) Archive(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Archive(c.Request().Context(), chatID, userID); err != nil {
		return InternalError(c, "Failed to archive chat")
	}

	return Success(c, nil)
}

// Unarchive unarchives a chat.
func (h *Chat) Unarchive(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Unarchive(c.Request().Context(), chatID, userID); err != nil {
		return InternalError(c, "Failed to unarchive chat")
	}

	return Success(c, nil)
}

// Pin pins a chat.
func (h *Chat) Pin(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Pin(c.Request().Context(), chatID, userID); err != nil {
		return InternalError(c, "Failed to pin chat")
	}

	return Success(c, nil)
}

// Unpin unpins a chat.
func (h *Chat) Unpin(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Unpin(c.Request().Context(), chatID, userID); err != nil {
		return InternalError(c, "Failed to unpin chat")
	}

	return Success(c, nil)
}

// Mute mutes a chat.
func (h *Chat) Mute(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Mute(c.Request().Context(), chatID, userID, nil); err != nil {
		return InternalError(c, "Failed to mute chat")
	}

	return Success(c, nil)
}

// Unmute unmutes a chat.
func (h *Chat) Unmute(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	if err := h.chats.Unmute(c.Request().Context(), chatID, userID); err != nil {
		return InternalError(c, "Failed to unmute chat")
	}

	return Success(c, nil)
}

// MarkAsRead marks a chat as read.
func (h *Chat) MarkAsRead(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	chatID := c.Param("id")
	var req struct {
		MessageID string `json:"message_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if err := h.chats.MarkAsRead(c.Request().Context(), chatID, userID, req.MessageID); err != nil {
		return InternalError(c, "Failed to mark as read")
	}

	return Success(c, nil)
}
