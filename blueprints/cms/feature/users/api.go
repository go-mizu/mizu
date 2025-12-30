// Package users provides user management functionality.
package users

import (
	"context"
	"time"
)

// User represents a user account.
type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	Bio          string     `json:"bio,omitempty"`
	AvatarURL    string     `json:"avatar_url,omitempty"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	Meta         string     `json:"meta,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Session represents an authenticated session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	RefreshToken string    `json:"-"`
	UserAgent    string    `json:"user_agent,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// RegisterIn contains input for registering a user.
type RegisterIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginIn contains input for logging in.
type LoginIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateIn contains input for updating a user.
type UpdateIn struct {
	Name      *string `json:"name,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Role      *string `json:"role,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// ListIn contains input for listing users.
type ListIn struct {
	Role   string
	Status string
	Search string
	Limit  int
	Offset int
}

// API defines the users service contract.
type API interface {
	Register(ctx context.Context, in *RegisterIn) (*User, *Session, error)
	Login(ctx context.Context, in *LoginIn) (*User, *Session, error)
	Logout(ctx context.Context, sessionID string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetBySlug(ctx context.Context, slug string) (*User, error)
	GetBySession(ctx context.Context, sessionID string) (*User, error)
	List(ctx context.Context, in *ListIn) ([]*User, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
	Delete(ctx context.Context, id string) error
}

// Store defines the data access contract for users.
type Store interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetBySlug(ctx context.Context, slug string) (*User, error)
	List(ctx context.Context, in *ListIn) ([]*User, int, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
	UpdateLastLogin(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	CreateSession(ctx context.Context, sess *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	GetUserBySession(ctx context.Context, sessionID string) (*User, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}
