// Package users provides user management functionality.
package users

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already taken")
	ErrInvalidEmail = errors.New("invalid email")
	ErrInvalidAuth  = errors.New("invalid email or password")
)

// User represents a user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateIn contains input for creating a user.
type CreateIn struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// UpdateIn contains input for updating a user.
type UpdateIn struct {
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// API defines the users service interface.
type API interface {
	Create(ctx context.Context, in CreateIn) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, in UpdateIn) (*User, error)
	Delete(ctx context.Context, id string) error
	Authenticate(ctx context.Context, email, password string) (*User, error)
	ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error
}

// Store defines the users data access interface.
type Store interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}
