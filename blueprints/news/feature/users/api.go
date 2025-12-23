package users

import (
	"context"
	"errors"
	"regexp"
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

// Validation constants
const (
	UsernameMinLen = 2
	UsernameMaxLen = 15
	PasswordMinLen = 8
	SessionTTL     = 30 * 24 * time.Hour // 30 days
)

var (
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	EmailRegex    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
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

// CreateIn contains input for creating a user.
type CreateIn struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the create input.
func (in *CreateIn) Validate() error {
	if len(in.Username) < UsernameMinLen || len(in.Username) > UsernameMaxLen {
		return ErrInvalidUsername
	}
	if !UsernameRegex.MatchString(in.Username) {
		return ErrInvalidUsername
	}
	if !EmailRegex.MatchString(in.Email) {
		return ErrInvalidEmail
	}
	if len(in.Password) < PasswordMinLen {
		return ErrInvalidPassword
	}
	return nil
}

// UpdateIn contains input for updating a user.
type UpdateIn struct {
	About *string `json:"about,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// API defines the users service interface.
type API interface {
	// User management
	Create(ctx context.Context, in CreateIn) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, id string, in UpdateIn) (*User, error)

	// Authentication
	Login(ctx context.Context, in LoginIn) (*User, error)
	CreateSession(ctx context.Context, userID string) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error

	// Karma
	UpdateKarma(ctx context.Context, id string, delta int64) error

	// Admin
	SetAdmin(ctx context.Context, id string, isAdmin bool) error

	// Lists
	List(ctx context.Context, limit, offset int) ([]*User, error)
}

// Store defines the data storage interface for users.
type Store interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error

	// Sessions
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	CleanExpiredSessions(ctx context.Context) error

	// Lists
	List(ctx context.Context, limit, offset int) ([]*User, error)
}
