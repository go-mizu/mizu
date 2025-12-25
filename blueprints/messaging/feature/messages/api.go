// Package messages provides message management.
package messages

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("message not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrExpired      = errors.New("message expired")
)

// MessageType represents the type of message.
type MessageType string

const (
	TypeText     MessageType = "text"
	TypeImage    MessageType = "image"
	TypeVideo    MessageType = "video"
	TypeAudio    MessageType = "audio"
	TypeVoice    MessageType = "voice"
	TypeDocument MessageType = "document"
	TypeSticker  MessageType = "sticker"
	TypeLocation MessageType = "location"
	TypeContact  MessageType = "contact"
	TypeSystem   MessageType = "system"
)

// MessageStatus represents delivery status.
type MessageStatus string

const (
	StatusPending   MessageStatus = "pending"
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
)

// Message represents a chat message.
type Message struct {
	ID                    string        `json:"id"`
	ChatID                string        `json:"chat_id"`
	SenderID              string        `json:"sender_id"`
	Type                  MessageType   `json:"type"`
	Content               string        `json:"content,omitempty"`
	ContentHTML           string        `json:"content_html,omitempty"`
	ReplyToID             string        `json:"reply_to_id,omitempty"`
	ForwardFromID         string        `json:"forward_from_id,omitempty"`
	ForwardFromChatID     string        `json:"forward_from_chat_id,omitempty"`
	ForwardFromSenderName string        `json:"forward_from_sender_name,omitempty"`
	IsForwarded           bool          `json:"is_forwarded"`
	IsEdited              bool          `json:"is_edited"`
	EditedAt              *time.Time    `json:"edited_at,omitempty"`
	IsDeleted             bool          `json:"is_deleted"`
	DeletedAt             *time.Time    `json:"deleted_at,omitempty"`
	DeletedForEveryone    bool          `json:"deleted_for_everyone"`
	ExpiresAt             *time.Time    `json:"expires_at,omitempty"`
	MentionEveryone       bool          `json:"mention_everyone"`
	Status                MessageStatus `json:"status,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`

	// Populated from joins
	Sender    any        `json:"sender,omitempty"`
	ReplyTo   *Message   `json:"reply_to,omitempty"`
	Media     []*Media   `json:"media,omitempty"`
	Reactions []Reaction `json:"reactions,omitempty"`
	Mentions  []string   `json:"mentions,omitempty"`
}

// Media represents a message attachment.
type Media struct {
	ID           string    `json:"id"`
	MessageID    string    `json:"message_id"`
	Type         string    `json:"type"`
	Filename     string    `json:"filename,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	Size         int64     `json:"size"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Duration     int       `json:"duration,omitempty"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Waveform     string    `json:"waveform,omitempty"`
	IsVoiceNote  bool      `json:"is_voice_note"`
	IsViewOnce   bool      `json:"is_view_once"`
	ViewCount    int       `json:"view_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// Reaction represents a reaction on a message.
type Reaction struct {
	Emoji string   `json:"emoji"`
	Count int      `json:"count"`
	Users []string `json:"users,omitempty"`
	Me    bool     `json:"me,omitempty"`
}

// Recipient represents message delivery status for a recipient.
type Recipient struct {
	MessageID   string     `json:"message_id"`
	UserID      string     `json:"user_id"`
	Status      MessageStatus `json:"status"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}

// CreateIn contains input for creating a message.
type CreateIn struct {
	ChatID          string      `json:"chat_id"`
	Type            MessageType `json:"type"`
	Content         string      `json:"content,omitempty"`
	ReplyToID       string      `json:"reply_to_id,omitempty"`
	MentionEveryone bool        `json:"mention_everyone,omitempty"`
	Mentions        []string    `json:"mentions,omitempty"`
	ExpiresIn       int         `json:"expires_in,omitempty"` // seconds
}

// UpdateIn contains input for updating a message.
type UpdateIn struct {
	Content     *string `json:"content,omitempty"`
	ContentHTML *string `json:"content_html,omitempty"`
}

// ForwardIn contains input for forwarding a message.
type ForwardIn struct {
	ToChatIDs []string `json:"to_chat_ids"`
}

// ListOpts specifies options for listing messages.
type ListOpts struct {
	Limit  int
	Before string
	After  string
}

// SearchOpts specifies options for searching messages.
type SearchOpts struct {
	Query    string
	ChatID   string
	SenderID string
	Type     MessageType
	Limit    int
}

// API defines the messages service contract.
type API interface {
	Create(ctx context.Context, senderID string, in *CreateIn) (*Message, error)
	GetByID(ctx context.Context, id string) (*Message, error)
	Update(ctx context.Context, id, userID string, in *UpdateIn) (*Message, error)
	Delete(ctx context.Context, id, userID string, forEveryone bool) error
	Forward(ctx context.Context, id, userID string, in *ForwardIn) ([]*Message, error)
	List(ctx context.Context, chatID string, opts ListOpts) ([]*Message, error)
	Search(ctx context.Context, opts SearchOpts) ([]*Message, error)

	// Reactions
	AddReaction(ctx context.Context, messageID, userID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID string) error
	GetReactions(ctx context.Context, messageID string) ([]Reaction, error)

	// Starring
	Star(ctx context.Context, messageID, userID string) error
	Unstar(ctx context.Context, messageID, userID string) error
	ListStarred(ctx context.Context, userID string, limit int) ([]*Message, error)

	// Media
	AddMedia(ctx context.Context, messageID string, media *Media) error
	GetMedia(ctx context.Context, messageID string) ([]*Media, error)
	ViewMedia(ctx context.Context, mediaID, userID string) error

	// Delivery status
	MarkDelivered(ctx context.Context, messageID, userID string) error
	MarkRead(ctx context.Context, messageID, userID string) error
	GetDeliveryStatus(ctx context.Context, messageID string) ([]*Recipient, error)

	// Pinning
	Pin(ctx context.Context, chatID, messageID, userID string) error
	Unpin(ctx context.Context, chatID, messageID string) error
	ListPinned(ctx context.Context, chatID string) ([]*Message, error)
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, m *Message) error
	GetByID(ctx context.Context, id string) (*Message, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string, forEveryone bool) error
	List(ctx context.Context, chatID string, opts ListOpts) ([]*Message, error)
	Search(ctx context.Context, opts SearchOpts) ([]*Message, error)

	// Reactions
	AddReaction(ctx context.Context, messageID, userID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID string) error
	GetReactions(ctx context.Context, messageID string) ([]Reaction, error)

	// Starring
	Star(ctx context.Context, messageID, userID string) error
	Unstar(ctx context.Context, messageID, userID string) error
	ListStarred(ctx context.Context, userID string, limit int) ([]*Message, error)

	// Media
	InsertMedia(ctx context.Context, media *Media) error
	GetMedia(ctx context.Context, messageID string) ([]*Media, error)
	IncrementViewCount(ctx context.Context, mediaID string) error

	// Mentions
	InsertMention(ctx context.Context, messageID, userID string) error
	GetMentions(ctx context.Context, messageID string) ([]string, error)

	// Recipients/Delivery
	InsertRecipient(ctx context.Context, r *Recipient) error
	UpdateRecipientStatus(ctx context.Context, messageID, userID string, status MessageStatus) error
	GetRecipients(ctx context.Context, messageID string) ([]*Recipient, error)

	// Pinning
	Pin(ctx context.Context, chatID, messageID, userID string) error
	Unpin(ctx context.Context, chatID, messageID string) error
	ListPinned(ctx context.Context, chatID string) ([]*Message, error)
}
