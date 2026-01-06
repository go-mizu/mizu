// Package users provides user management functionality.
package users

import (
	"context"
	"time"
)

// User represents a user account.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Password  string    `json:"-"` // Never expose password
	Avatar    string    `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegisterIn contains registration input.
type RegisterIn struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginIn contains login input.
type LoginIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateIn contains user update input.
type UpdateIn struct {
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

// API defines the users service interface.
type API interface {
	// Register creates a new user account.
	Register(ctx context.Context, in *RegisterIn) (*User, string, error)

	// Login authenticates a user and returns a token.
	Login(ctx context.Context, in *LoginIn) (*User, string, error)

	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id string) (*User, error)

	// GetByEmail retrieves a user by email.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update updates a user's profile.
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)

	// VerifyToken verifies a JWT token and returns the user ID.
	VerifyToken(ctx context.Context, token string) (string, error)
}

// Store defines the users data access interface.
type Store interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}
