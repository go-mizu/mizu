// Package friendcode provides QR code friend sharing functionality.
package friendcode

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound    = errors.New("friend code not found")
	ErrExpired     = errors.New("friend code expired")
	ErrSelfAdd     = errors.New("cannot add yourself as a friend")
	ErrAlreadyAdded = errors.New("user is already a contact")
)

// FriendCode represents a shareable friend code.
type FriendCode struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// FriendCodeResponse includes the friend code with QR data.
type FriendCodeResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	QRData    string    `json:"qr_data"`
}

// ResolvedUser contains user info from a friend code.
type ResolvedUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Status      string `json:"status,omitempty"`
}

// Contact represents the result of adding a friend.
type Contact struct {
	UserID        string    `json:"user_id"`
	ContactUserID string    `json:"contact_user_id"`
	DisplayName   string    `json:"display_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// API defines the friend code service contract.
type API interface {
	// Generate creates or returns existing valid friend code for user
	Generate(ctx context.Context, userID string) (*FriendCodeResponse, error)

	// Resolve validates a code and returns the associated user info
	Resolve(ctx context.Context, code string) (*ResolvedUser, error)

	// AddFriend adds a contact using a friend code
	AddFriend(ctx context.Context, userID, code string) (*Contact, error)

	// Revoke invalidates a user's current friend code
	Revoke(ctx context.Context, userID string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, fc *FriendCode) error
	GetByUserID(ctx context.Context, userID string) (*FriendCode, error)
	GetByCode(ctx context.Context, code string) (*FriendCode, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

// UserStore defines the user lookup contract.
type UserStore interface {
	GetByID(ctx context.Context, id string) (*ResolvedUser, error)
}

// ContactStore defines the contact management contract.
type ContactStore interface {
	Exists(ctx context.Context, userID, contactUserID string) (bool, error)
	Insert(ctx context.Context, userID, contactUserID, displayName string) error
}
