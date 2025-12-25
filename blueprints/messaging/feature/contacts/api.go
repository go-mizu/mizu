// Package contacts provides contact management.
package contacts

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("contact not found")
	ErrAlreadyAdded = errors.New("contact already added")
	ErrSelfContact  = errors.New("cannot add yourself as contact")
)

// Contact represents a user contact.
type Contact struct {
	UserID        string    `json:"user_id"`
	ContactUserID string    `json:"contact_user_id"`
	DisplayName   string    `json:"display_name,omitempty"`
	IsBlocked     bool      `json:"is_blocked"`
	IsFavorite    bool      `json:"is_favorite"`
	BlockedAt     time.Time `json:"blocked_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`

	// Joined user info
	User *ContactUser `json:"user,omitempty"`
}

// ContactUser represents basic user info for a contact.
type ContactUser struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Status      string    `json:"status,omitempty"`
	IsOnline    bool      `json:"is_online"`
	LastSeenAt  time.Time `json:"last_seen_at,omitempty"`
}

// AddIn contains input for adding a contact.
type AddIn struct {
	ContactUserID string `json:"contact_user_id"`
	DisplayName   string `json:"display_name,omitempty"`
}

// UpdateIn contains input for updating a contact.
type UpdateIn struct {
	DisplayName *string `json:"display_name,omitempty"`
	IsFavorite  *bool   `json:"is_favorite,omitempty"`
}

// API defines the contacts service contract.
type API interface {
	Add(ctx context.Context, userID string, in *AddIn) (*Contact, error)
	Get(ctx context.Context, userID, contactUserID string) (*Contact, error)
	List(ctx context.Context, userID string) ([]*Contact, error)
	ListFavorites(ctx context.Context, userID string) ([]*Contact, error)
	ListBlocked(ctx context.Context, userID string) ([]*Contact, error)
	Update(ctx context.Context, userID, contactUserID string, in *UpdateIn) (*Contact, error)
	Remove(ctx context.Context, userID, contactUserID string) error
	Block(ctx context.Context, userID, contactUserID string) error
	Unblock(ctx context.Context, userID, contactUserID string) error
	IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error)
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, c *Contact) error
	Get(ctx context.Context, userID, contactUserID string) (*Contact, error)
	List(ctx context.Context, userID string) ([]*Contact, error)
	ListFavorites(ctx context.Context, userID string) ([]*Contact, error)
	ListBlocked(ctx context.Context, userID string) ([]*Contact, error)
	Update(ctx context.Context, userID, contactUserID string, in *UpdateIn) error
	Delete(ctx context.Context, userID, contactUserID string) error
	Block(ctx context.Context, userID, contactUserID string) error
	Unblock(ctx context.Context, userID, contactUserID string) error
	Exists(ctx context.Context, userID, contactUserID string) (bool, error)
	IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error)
}
