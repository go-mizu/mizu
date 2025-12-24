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
)

// MessageType represents the type of message.
type MessageType string

const (
	TypeDefault       MessageType = "default"
	TypeReply         MessageType = "reply"
	TypeSystemJoin    MessageType = "system_join"
	TypeSystemLeave   MessageType = "system_leave"
	TypeChannelPinned MessageType = "channel_pinned"
)

// Message represents a chat message.
type Message struct {
	ID              string       `json:"id"`
	ChannelID       string       `json:"channel_id"`
	AuthorID        string       `json:"author_id"`
	Content         string       `json:"content"`
	ContentHTML     string       `json:"content_html,omitempty"`
	Type            MessageType  `json:"type"`
	ReplyToID       string       `json:"reply_to_id,omitempty"`
	ThreadID        string       `json:"thread_id,omitempty"`
	Flags           int          `json:"flags"`
	Mentions        []string     `json:"mentions,omitempty"`
	MentionRoles    []string     `json:"mention_roles,omitempty"`
	MentionEveryone bool         `json:"mention_everyone"`
	Attachments     []Attachment `json:"attachments,omitempty"`
	Embeds          []Embed      `json:"embeds,omitempty"`
	Reactions       []Reaction   `json:"reactions,omitempty"`
	IsPinned        bool         `json:"is_pinned"`
	IsEdited        bool         `json:"is_edited"`
	EditedAt        *time.Time   `json:"edited_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`

	// Populated from joins
	Author   any      `json:"author,omitempty"`
	ReplyTo  *Message `json:"reply_to,omitempty"`
}

// Attachment represents a file attachment.
type Attachment struct {
	ID          string    `json:"id"`
	MessageID   string    `json:"message_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type,omitempty"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
	ProxyURL    string    `json:"proxy_url,omitempty"`
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	IsSpoiler   bool      `json:"is_spoiler"`
	CreatedAt   time.Time `json:"created_at"`
}

// Embed represents a rich embed.
type Embed struct {
	ID          string       `json:"id,omitempty"`
	Type        string       `json:"type"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Color       int          `json:"color,omitempty"`
	ImageURL    string       `json:"image_url,omitempty"`
	VideoURL    string       `json:"video_url,omitempty"`
	Thumbnail   string       `json:"thumbnail,omitempty"`
	Footer      string       `json:"footer,omitempty"`
	AuthorName  string       `json:"author_name,omitempty"`
	AuthorURL   string       `json:"author_url,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
}

// EmbedField represents a field in an embed.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Reaction represents a reaction on a message.
type Reaction struct {
	Emoji string   `json:"emoji"`
	Count int      `json:"count"`
	Users []string `json:"users,omitempty"`
	Me    bool     `json:"me,omitempty"`
}

// CreateIn contains input for creating a message.
type CreateIn struct {
	ChannelID       string      `json:"channel_id"`
	Content         string      `json:"content"`
	Type            MessageType `json:"type,omitempty"`
	ReplyToID       string      `json:"reply_to_id,omitempty"`
	MentionEveryone bool        `json:"mention_everyone,omitempty"`
	Mentions        []string    `json:"mentions,omitempty"`
}

// UpdateIn contains input for updating a message.
type UpdateIn struct {
	Content     *string `json:"content,omitempty"`
	ContentHTML *string `json:"content_html,omitempty"`
}

// ListOpts specifies options for listing messages.
type ListOpts struct {
	Limit  int
	Before string
	After  string
	Around string
}

// SearchOpts specifies options for searching messages.
type SearchOpts struct {
	Query     string
	ChannelID string
	AuthorID  string
	Limit     int
}

// API defines the messages service contract.
type API interface {
	Create(ctx context.Context, authorID string, in *CreateIn) (*Message, error)
	GetByID(ctx context.Context, id string) (*Message, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Message, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, channelID string, opts ListOpts) ([]*Message, error)
	Search(ctx context.Context, opts SearchOpts) ([]*Message, error)
	Pin(ctx context.Context, channelID, messageID, userID string) error
	Unpin(ctx context.Context, channelID, messageID string) error
	ListPinned(ctx context.Context, channelID string) ([]*Message, error)
	AddReaction(ctx context.Context, messageID, userID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID, emoji string) error
	GetReactionUsers(ctx context.Context, messageID, emoji string, limit int) ([]string, error)
	CreateAttachment(ctx context.Context, att *Attachment) error
	CreateEmbed(ctx context.Context, messageID string, embed *Embed) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, m *Message) error
	GetByID(ctx context.Context, id string) (*Message, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, channelID string, opts ListOpts) ([]*Message, error)
	Search(ctx context.Context, opts SearchOpts) ([]*Message, error)
	Pin(ctx context.Context, channelID, messageID, userID string) error
	Unpin(ctx context.Context, channelID, messageID string) error
	ListPinned(ctx context.Context, channelID string) ([]*Message, error)
	AddReaction(ctx context.Context, messageID, userID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID, emoji string) error
	GetReactionUsers(ctx context.Context, messageID, emoji string, limit int) ([]string, error)
	InsertAttachment(ctx context.Context, att *Attachment) error
	InsertEmbed(ctx context.Context, messageID string, embed *Embed) error
}
