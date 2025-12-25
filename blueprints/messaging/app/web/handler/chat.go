package handler

import (
	"context"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
)

// Chat handles chat endpoints.
type Chat struct {
	chats     chats.API
	accounts  accounts.API
	messages  messages.API
	getUserID func(*mizu.Ctx) string
}

// NewChat creates a new Chat handler.
func NewChat(chats chats.API, accounts accounts.API, messages messages.API, getUserID func(*mizu.Ctx) string) *Chat {
	return &Chat{chats: chats, accounts: accounts, messages: messages, getUserID: getUserID}
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

	ctx := c.Request().Context()
	chatList, err := h.chats.List(ctx, userID, opts)
	if err != nil {
		return InternalError(c, "Failed to list chats")
	}

	// Enrich chats with additional data
	h.enrichChats(ctx, chatList, userID)

	return Success(c, chatList)
}

// enrichChats adds OtherUser, LastMessage, and ParticipantCount to chats.
func (h *Chat) enrichChats(ctx context.Context, chatList []*chats.Chat, userID string) {
	// Collect all user IDs needed for direct chats
	userIDs := make([]string, 0)
	messageIDs := make([]string, 0)

	for _, chat := range chatList {
		if chat.Type == chats.TypeDirect {
			// Get participants to find the other user
			participants, _ := h.chats.GetParticipants(ctx, chat.ID)
			for _, p := range participants {
				if p.UserID != userID {
					userIDs = append(userIDs, p.UserID)
				}
			}
			// For self-chat, include the user themselves
			if len(participants) == 1 && participants[0].UserID == userID {
				userIDs = append(userIDs, userID)
			}
			chat.ParticipantCount = len(participants)
		} else {
			// For groups, count participants
			participants, _ := h.chats.GetParticipants(ctx, chat.ID)
			chat.ParticipantCount = len(participants)
		}

		if chat.LastMessageID != "" {
			messageIDs = append(messageIDs, chat.LastMessageID)
		}
	}

	// Fetch all users at once
	users := make(map[string]*accounts.User)
	if len(userIDs) > 0 {
		userList, _ := h.accounts.GetByIDs(ctx, userIDs)
		for _, u := range userList {
			users[u.ID] = u
		}
	}

	// Fetch last messages
	lastMessages := make(map[string]*messages.Message)
	for _, msgID := range messageIDs {
		if msg, err := h.messages.GetByID(ctx, msgID); err == nil {
			lastMessages[msgID] = msg
		}
	}

	// Assign OtherUser and LastMessage to each chat
	for _, chat := range chatList {
		if chat.Type == chats.TypeDirect {
			participants, _ := h.chats.GetParticipants(ctx, chat.ID)
			for _, p := range participants {
				if p.UserID != userID {
					if u, ok := users[p.UserID]; ok {
						chat.OtherUser = u
					}
					break
				}
			}
			// For self-chat, set OtherUser to current user
			if chat.OtherUser == nil && len(participants) == 1 {
				if u, ok := users[userID]; ok {
					chat.OtherUser = u
				}
			}
		}

		if chat.LastMessageID != "" {
			if msg, ok := lastMessages[chat.LastMessageID]; ok {
				chat.LastMessage = msg
			}
		}
	}
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

	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	ctx := c.Request().Context()
	var chat *chats.Chat
	var err error

	switch req.Type {
	case "direct", "":
		if req.RecipientID == "" {
			return BadRequest(c, "Recipient ID is required")
		}
		chat, err = h.chats.CreateDirect(ctx, userID, &chats.CreateDirectIn{
			RecipientID: req.RecipientID,
		})
	case "group":
		if req.Name == "" {
			return BadRequest(c, "Group name is required")
		}
		chat, err = h.chats.CreateGroup(ctx, userID, &chats.CreateGroupIn{
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

	// Enrich the chat with OtherUser, ParticipantCount, etc.
	h.enrichChats(ctx, []*chats.Chat{chat}, userID)

	return Created(c, chat)
}

// Get retrieves a chat.
func (h *Chat) Get(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	ctx := c.Request().Context()
	chatID := c.Param("id")
	chat, err := h.chats.GetByIDForUser(ctx, chatID, userID)
	if err != nil {
		return NotFound(c, "Chat not found")
	}

	// Enrich the chat with OtherUser, ParticipantCount, etc.
	h.enrichChats(ctx, []*chats.Chat{chat}, userID)

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
	if err := c.BindJSON(&in, 1<<20); err != nil {
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
	// Ignore body parse errors - message_id is optional
	_ = c.BindJSON(&req, 1<<20)

	if err := h.chats.MarkAsRead(c.Request().Context(), chatID, userID, req.MessageID); err != nil {
		return InternalError(c, "Failed to mark as read")
	}

	return Success(c, nil)
}
