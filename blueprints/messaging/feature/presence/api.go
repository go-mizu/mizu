// Package presence provides online status management.
package presence

import (
	"context"
	"time"
)

// Status represents presence status.
type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusTyping  Status = "typing"
)

// Presence represents a user's presence state.
type Presence struct {
	UserID       string    `json:"user_id"`
	Status       Status    `json:"status"`
	CustomStatus string    `json:"custom_status,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

// TypingState represents typing indicator state.
type TypingState struct {
	UserID   string    `json:"user_id"`
	ChatID   string    `json:"chat_id"`
	StartedAt time.Time `json:"started_at"`
}

// API defines the presence service contract.
type API interface {
	Get(ctx context.Context, userID string) (*Presence, error)
	GetMany(ctx context.Context, userIDs []string) ([]*Presence, error)
	SetOnline(ctx context.Context, userID string) error
	SetOffline(ctx context.Context, userID string) error
	SetCustomStatus(ctx context.Context, userID, status string) error
	StartTyping(ctx context.Context, userID, chatID string) error
	StopTyping(ctx context.Context, userID, chatID string) error
	GetTyping(ctx context.Context, chatID string) ([]*TypingState, error)
}

// Store defines the data access contract.
type Store interface {
	Upsert(ctx context.Context, p *Presence) error
	Get(ctx context.Context, userID string) (*Presence, error)
	GetMany(ctx context.Context, userIDs []string) ([]*Presence, error)
	UpdateStatus(ctx context.Context, userID string, status Status) error
	UpdateLastSeen(ctx context.Context, userID string) error
	UpdateCustomStatus(ctx context.Context, userID, status string) error
}
