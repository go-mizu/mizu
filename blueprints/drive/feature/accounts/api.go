// Package accounts provides user account management.
package accounts

import (
	"context"
	"time"
)

// User represents a user account.
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	PasswordHash  string    `json:"-"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	StorageQuota  int64     `json:"storage_quota"`
	StorageUsed   int64     `json:"storage_used"`
	IsAdmin       bool      `json:"is_admin"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Session represents an authenticated session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	TokenHash    string    `json:"-"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	LastActiveAt time.Time `json:"last_active_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// RegisterIn contains input for registering a user.
type RegisterIn struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginIn contains input for logging in.
type LoginIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateIn contains input for updating a user.
type UpdateIn struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// ChangePasswordIn contains input for changing password.
type ChangePasswordIn struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// API defines the accounts service contract.
type API interface {
	Register(ctx context.Context, in *RegisterIn) (*User, *Session, error)
	Login(ctx context.Context, in *LoginIn) (*User, *Session, error)
	Logout(ctx context.Context, sessionID string) error
	LogoutAll(ctx context.Context, userID string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetBySession(ctx context.Context, token string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
	ChangePassword(ctx context.Context, id string, in *ChangePasswordIn) error
	Delete(ctx context.Context, id string) error
	ListSessions(ctx context.Context, userID string) ([]*Session, error)
	DeleteSession(ctx context.Context, userID, sessionID string) error
}
