// Package auth provides authentication management.
package auth

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session expired")
	ErrEmailRequired      = errors.New("email is required")
	ErrPasswordRequired   = errors.New("password is required")
)

// User represents a user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResult represents login result.
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *UserInfo `json:"user"`
}

// UserInfo represents public user info.
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterIn contains input for registration.
type RegisterIn struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// API defines the Auth service contract.
type API interface {
	Login(ctx context.Context, in *LoginIn) (*LoginResult, error)
	Register(ctx context.Context, in *RegisterIn) (*LoginResult, error)
	Logout(ctx context.Context, token string) error
	GetCurrentUser(ctx context.Context, token string) (*UserInfo, error)
}

// Store defines the data access contract.
type Store interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
}
