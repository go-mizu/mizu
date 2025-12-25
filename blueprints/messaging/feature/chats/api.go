// Package chats provides chat/conversation management.
package chats

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("chat not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// ChatType represents the type of chat.
type ChatType string

const (
	TypeDirect    ChatType = "direct"
	TypeGroup     ChatType = "group"
	TypeBroadcast ChatType = "broadcast"
)

// Chat represents a conversation.
type Chat struct {
	ID            string    `json:"id"`
	Type          ChatType  `json:"type"`
	Name          string    `json:"name,omitempty"`
	Description   string    `json:"description,omitempty"`
	IconURL       string    `json:"icon_url,omitempty"`
	OwnerID       string    `json:"owner_id,omitempty"`
	LastMessageID string    `json:"last_message_id,omitempty"`
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
	MessageCount  int64     `json:"message_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// For current user context
	UnreadCount       int    `json:"unread_count,omitempty"`
	IsMuted           bool   `json:"is_muted,omitempty"`
	IsPinned          bool   `json:"is_pinned,omitempty"`
	IsArchived        bool   `json:"is_archived,omitempty"`
	LastReadMessageID string `json:"last_read_message_id,omitempty"`

	// Populated from joins
	Participants     []*Participant `json:"participants,omitempty"`
	LastMessage      any            `json:"last_message,omitempty"`
	OtherUser        any            `json:"other_user,omitempty"`        // For direct chats: the other participant
	ParticipantCount int            `json:"participant_count,omitempty"` // For group chats
}

// Participant represents a chat participant.
type Participant struct {
	ChatID             string    `json:"chat_id"`
	UserID             string    `json:"user_id"`
	Role               string    `json:"role"`
	JoinedAt           time.Time `json:"joined_at"`
	IsMuted            bool      `json:"is_muted"`
	MuteUntil          time.Time `json:"mute_until,omitempty"`
	UnreadCount        int       `json:"unread_count"`
	LastReadMessageID  string    `json:"last_read_message_id,omitempty"`
	LastReadAt         time.Time `json:"last_read_at,omitempty"`
	NotificationLevel  string    `json:"notification_level"`

	// Joined user info
	User any `json:"user,omitempty"`
}

// CreateDirectIn contains input for creating a direct chat.
type CreateDirectIn struct {
	RecipientID string `json:"recipient_id"`
}

// CreateGroupIn contains input for creating a group chat.
type CreateGroupIn struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	IconURL      string   `json:"icon_url,omitempty"`
	ParticipantIDs []string `json:"participant_ids"`
}

// UpdateIn contains input for updating a chat.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IconURL     *string `json:"icon_url,omitempty"`
}

// ListOpts specifies options for listing chats.
type ListOpts struct {
	Limit        int
	Offset       int
	IncludeArchived bool
}

// API defines the chats service contract.
type API interface {
	CreateDirect(ctx context.Context, userID string, in *CreateDirectIn) (*Chat, error)
	CreateGroup(ctx context.Context, userID string, in *CreateGroupIn) (*Chat, error)
	GetByID(ctx context.Context, id string) (*Chat, error)
	GetByIDForUser(ctx context.Context, id, userID string) (*Chat, error)
	GetDirectChat(ctx context.Context, userID1, userID2 string) (*Chat, error)
	List(ctx context.Context, userID string, opts ListOpts) ([]*Chat, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Chat, error)
	Delete(ctx context.Context, id string) error

	// Participant management
	AddParticipant(ctx context.Context, chatID, userID string, role string) error
	RemoveParticipant(ctx context.Context, chatID, userID string) error
	UpdateParticipantRole(ctx context.Context, chatID, userID, role string) error
	GetParticipants(ctx context.Context, chatID string) ([]*Participant, error)
	IsParticipant(ctx context.Context, chatID, userID string) (bool, error)

	// User-specific actions
	Mute(ctx context.Context, chatID, userID string, until *time.Time) error
	Unmute(ctx context.Context, chatID, userID string) error
	Archive(ctx context.Context, chatID, userID string) error
	Unarchive(ctx context.Context, chatID, userID string) error
	Pin(ctx context.Context, chatID, userID string) error
	Unpin(ctx context.Context, chatID, userID string) error
	MarkAsRead(ctx context.Context, chatID, userID, messageID string) error

	// Message count
	IncrementMessageCount(ctx context.Context, chatID string) error
	UpdateLastMessage(ctx context.Context, chatID, messageID string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, c *Chat) error
	GetByID(ctx context.Context, id string) (*Chat, error)
	GetByIDForUser(ctx context.Context, id, userID string) (*Chat, error)
	GetDirectChat(ctx context.Context, userID1, userID2 string) (*Chat, error)
	List(ctx context.Context, userID string, opts ListOpts) ([]*Chat, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error

	// Participants
	InsertParticipant(ctx context.Context, p *Participant) error
	DeleteParticipant(ctx context.Context, chatID, userID string) error
	UpdateParticipantRole(ctx context.Context, chatID, userID, role string) error
	GetParticipants(ctx context.Context, chatID string) ([]*Participant, error)
	GetParticipant(ctx context.Context, chatID, userID string) (*Participant, error)
	IsParticipant(ctx context.Context, chatID, userID string) (bool, error)

	// User-specific
	Mute(ctx context.Context, chatID, userID string, until *time.Time) error
	Unmute(ctx context.Context, chatID, userID string) error
	Archive(ctx context.Context, chatID, userID string) error
	Unarchive(ctx context.Context, chatID, userID string) error
	Pin(ctx context.Context, chatID, userID string) error
	Unpin(ctx context.Context, chatID, userID string) error
	MarkAsRead(ctx context.Context, chatID, userID, messageID string) error
	IncrementUnreadCount(ctx context.Context, chatID string, excludeUserID string) error
	ResetUnreadCount(ctx context.Context, chatID, userID string) error

	// Message tracking
	IncrementMessageCount(ctx context.Context, chatID string) error
	UpdateLastMessage(ctx context.Context, chatID, messageID string) error
}
