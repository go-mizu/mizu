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
	Name         string    `json:"name"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	PasswordHash string    `json:"-"`
	Settings     Settings  `json:"settings"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Settings holds user preferences.
type Settings struct {
	Theme         string `json:"theme"`          // light, dark, system
	Timezone      string `json:"timezone"`
	DateFormat    string `json:"date_format"`
	StartOfWeek   int    `json:"start_of_week"` // 0=Sunday, 1=Monday
	EmailDigest   bool   `json:"email_digest"`
	DesktopNotify bool   `json:"desktop_notify"`
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

// API defines the users service contract.
type API interface {
	Register(ctx context.Context, in *RegisterIn) (*User, *Session, error)
	Login(ctx context.Context, in *LoginIn) (*User, *Session, error)
	Logout(ctx context.Context, sessionID string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetBySession(ctx context.Context, sessionID string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
	UpdateSettings(ctx context.Context, id string, settings Settings) error
	UpdatePassword(ctx context.Context, id string, oldPass, newPass string) error
}

// Store defines the data access contract for users.
type Store interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdateSettings(ctx context.Context, id string, settings Settings) error
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
	CreateSession(ctx context.Context, sess *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	GetUserBySession(ctx context.Context, sessionID string) (*User, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}
