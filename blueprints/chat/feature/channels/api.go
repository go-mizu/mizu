// Package channels provides channel management.
package channels

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("channel not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// ChannelType represents the type of channel.
type ChannelType string

const (
	TypeText    ChannelType = "text"
	TypeVoice   ChannelType = "voice"
	TypeDM      ChannelType = "dm"
	TypeGroupDM ChannelType = "group_dm"
	TypeThread  ChannelType = "thread"
)

// Channel represents a communication channel.
type Channel struct {
	ID            string      `json:"id"`
	ServerID      string      `json:"server_id,omitempty"`
	CategoryID    string      `json:"category_id,omitempty"`
	Type          ChannelType `json:"type"`
	Name          string      `json:"name,omitempty"`
	Topic         string      `json:"topic,omitempty"`
	Position      int         `json:"position"`
	IsPrivate     bool        `json:"is_private"`
	IsNSFW        bool        `json:"is_nsfw"`
	SlowModeDelay int         `json:"slow_mode_delay"`
	Bitrate       int         `json:"bitrate,omitempty"`
	UserLimit     int         `json:"user_limit,omitempty"`
	LastMessageID string      `json:"last_message_id,omitempty"`
	LastMessageAt *time.Time  `json:"last_message_at,omitempty"`
	MessageCount  int64       `json:"message_count"`
	IconURL       string      `json:"icon_url,omitempty"`
	OwnerID       string      `json:"owner_id,omitempty"`
	Recipients    []string    `json:"recipients,omitempty"` // For DMs
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// Category represents a channel category.
type Category struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateIn contains input for creating a channel.
type CreateIn struct {
	ServerID      string      `json:"server_id,omitempty"`
	CategoryID    string      `json:"category_id,omitempty"`
	Type          ChannelType `json:"type"`
	Name          string      `json:"name"`
	Topic         string      `json:"topic,omitempty"`
	IsPrivate     bool        `json:"is_private,omitempty"`
	IsNSFW        bool        `json:"is_nsfw,omitempty"`
	SlowModeDelay int         `json:"slow_mode_delay,omitempty"`
	Bitrate       int         `json:"bitrate,omitempty"`
	UserLimit     int         `json:"user_limit,omitempty"`
}

// UpdateIn contains input for updating a channel.
type UpdateIn struct {
	Name          *string `json:"name,omitempty"`
	Topic         *string `json:"topic,omitempty"`
	Position      *int    `json:"position,omitempty"`
	IsPrivate     *bool   `json:"is_private,omitempty"`
	IsNSFW        *bool   `json:"is_nsfw,omitempty"`
	SlowModeDelay *int    `json:"slow_mode_delay,omitempty"`
	CategoryID    *string `json:"category_id,omitempty"`
}

// API defines the channels service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Channel, error)
	GetByID(ctx context.Context, id string) (*Channel, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Channel, error)
	Delete(ctx context.Context, id string) error
	ListByServer(ctx context.Context, serverID string) ([]*Channel, error)
	ListDMsByUser(ctx context.Context, userID string) ([]*Channel, error)
	GetOrCreateDM(ctx context.Context, userID1, userID2 string) (*Channel, error)
	CreateGroupDM(ctx context.Context, ownerID string, recipientIDs []string, name string) (*Channel, error)
	AddRecipient(ctx context.Context, channelID, userID string) error
	RemoveRecipient(ctx context.Context, channelID, userID string) error
	GetRecipients(ctx context.Context, channelID string) ([]string, error)
	UpdateLastMessage(ctx context.Context, channelID, messageID string, at time.Time) error
	CreateCategory(ctx context.Context, serverID, name string, position int) (*Category, error)
	GetCategory(ctx context.Context, id string) (*Category, error)
	ListCategories(ctx context.Context, serverID string) ([]*Category, error)
	DeleteCategory(ctx context.Context, id string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, c *Channel) error
	GetByID(ctx context.Context, id string) (*Channel, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByServer(ctx context.Context, serverID string) ([]*Channel, error)
	ListDMsByUser(ctx context.Context, userID string) ([]*Channel, error)
	GetDMChannel(ctx context.Context, userID1, userID2 string) (*Channel, error)
	AddRecipient(ctx context.Context, channelID, userID string) error
	RemoveRecipient(ctx context.Context, channelID, userID string) error
	GetRecipients(ctx context.Context, channelID string) ([]string, error)
	UpdateLastMessage(ctx context.Context, channelID, messageID string, at time.Time) error
	InsertCategory(ctx context.Context, c *Category) error
	GetCategory(ctx context.Context, id string) (*Category, error)
	ListCategories(ctx context.Context, serverID string) ([]*Category, error)
	DeleteCategory(ctx context.Context, id string) error
}
