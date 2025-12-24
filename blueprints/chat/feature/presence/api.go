// Package presence provides user presence tracking.
package presence

import (
	"context"
	"time"
)

// Status represents user online status.
type Status string

const (
	StatusOnline    Status = "online"
	StatusIdle      Status = "idle"
	StatusDND       Status = "dnd"
	StatusInvisible Status = "invisible"
	StatusOffline   Status = "offline"
)

// Presence represents a user's presence state.
type Presence struct {
	UserID       string       `json:"user_id"`
	Status       Status       `json:"status"`
	CustomStatus string       `json:"custom_status,omitempty"`
	Activities   []Activity   `json:"activities,omitempty"`
	ClientStatus ClientStatus `json:"client_status,omitempty"`
	LastSeenAt   time.Time    `json:"last_seen_at"`
}

// Activity represents what a user is doing.
type Activity struct {
	Type    string `json:"type"` // playing, streaming, listening, watching, custom
	Name    string `json:"name"`
	Details string `json:"details,omitempty"`
	State   string `json:"state,omitempty"`
	URL     string `json:"url,omitempty"`
}

// ClientStatus represents status per client platform.
type ClientStatus struct {
	Desktop string `json:"desktop,omitempty"`
	Mobile  string `json:"mobile,omitempty"`
	Web     string `json:"web,omitempty"`
}

// TypingIndicator represents a typing event.
type TypingIndicator struct {
	UserID    string    `json:"user_id"`
	ChannelID string    `json:"channel_id"`
	Timestamp time.Time `json:"timestamp"`
}

// UpdateIn contains input for updating presence.
type UpdateIn struct {
	Status       *Status     `json:"status,omitempty"`
	CustomStatus *string     `json:"custom_status,omitempty"`
	Activities   *[]Activity `json:"activities,omitempty"`
}

// API defines the presence service contract.
type API interface {
	Update(ctx context.Context, userID string, in *UpdateIn) (*Presence, error)
	Get(ctx context.Context, userID string) (*Presence, error)
	GetBulk(ctx context.Context, userIDs []string) ([]*Presence, error)
	SetOnline(ctx context.Context, userID string) error
	SetOffline(ctx context.Context, userID string) error
	SetIdle(ctx context.Context, userID string) error
	Heartbeat(ctx context.Context, userID string) error
	StartTyping(ctx context.Context, userID, channelID string) error
	CleanupStale(ctx context.Context) error
}

// Store defines the data access contract.
type Store interface {
	Upsert(ctx context.Context, p *Presence) error
	Get(ctx context.Context, userID string) (*Presence, error)
	GetBulk(ctx context.Context, userIDs []string) ([]*Presence, error)
	UpdateStatus(ctx context.Context, userID string, status Status) error
	SetOffline(ctx context.Context, userID string) error
	Delete(ctx context.Context, userID string) error
	CleanupStale(ctx context.Context, before time.Time) error
}
