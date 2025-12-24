package users

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound        = errors.New("user not found")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailTaken      = errors.New("email already taken")
	ErrInvalidUsername = errors.New("invalid username format")
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPassword = errors.New("invalid password")
	ErrSessionExpired  = errors.New("session expired")
)

// User represents a user account.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"-"`
	PasswordHash string    `json:"-"`
	About        string    `json:"about,omitempty"`
	Karma        int64     `json:"karma"`
	IsAdmin      bool      `json:"is_admin,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	// Computed fields
	StoryCount   int64 `json:"story_count,omitempty"`
	CommentCount int64 `json:"comment_count,omitempty"`
}

// Session represents an authentication session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// API defines the users service interface.
type API interface {
	// User management
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)

	// Authentication
	GetSession(ctx context.Context, token string) (*Session, error)

	// Lists
	List(ctx context.Context, limit, offset int) ([]*User, error)
}

// Store defines the data storage interface for users.
type Store interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Create
	Create(ctx context.Context, user *User) error

	// Sessions
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	CreateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, token string) error

	// Lists
	List(ctx context.Context, limit, offset int) ([]*User, error)
}
