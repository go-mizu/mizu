// Package users provides user management functionality.
package users

import (
	"context"
	"time"
)

// User represents a user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	PasswordHash string    `json:"-"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an authenticated session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterIn contains input for registering a user.
type RegisterIn struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// LoginIn contains input for logging in.
type LoginIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateIn contains input for updating a user.
type UpdateIn struct {
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// API defines the users service contract.
type API interface {
	Register(ctx context.Context, in *RegisterIn) (*User, *Session, error)
	Login(ctx context.Context, in *LoginIn) (*User, *Session, error)
	Logout(ctx context.Context, sessionID string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetBySession(ctx context.Context, sessionID string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
}

// Store defines the data access contract for users.
type Store interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	CreateSession(ctx context.Context, sess *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}
