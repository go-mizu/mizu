// Package accounts provides user account management.
package accounts

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound           = errors.New("user not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidSession     = errors.New("invalid or expired session")
)

// Status represents user online status.
type Status string

const (
	StatusOnline    Status = "online"
	StatusIdle      Status = "idle"
	StatusDND       Status = "dnd"
	StatusInvisible Status = "invisible"
	StatusOffline   Status = "offline"
)

// User represents a user account.
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	DisplayName   string    `json:"display_name,omitempty"`
	Email         string    `json:"-"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	BannerURL     string    `json:"banner_url,omitempty"`
	Bio           string    `json:"bio,omitempty"`
	Status        Status    `json:"status"`
	CustomStatus  string    `json:"custom_status,omitempty"`
	IsBot         bool      `json:"is_bot"`
	IsVerified    bool      `json:"is_verified"`
	IsAdmin       bool      `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Tag returns the full user tag (username#discriminator).
func (u *User) Tag() string {
	return u.Username + "#" + u.Discriminator
}

// CreateIn contains input for creating an account.
type CreateIn struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
}

// UpdateIn contains input for updating an account.
type UpdateIn struct {
	DisplayName  *string `json:"display_name,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	BannerURL    *string `json:"banner_url,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	Status       *Status `json:"status,omitempty"`
	CustomStatus *string `json:"custom_status,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Login    string `json:"login"` // Username or email
	Password string `json:"password"`
}

// Session represents an auth session.
type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Token      string    `json:"token"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	DeviceType string    `json:"device_type,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// API defines the accounts service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
	Login(ctx context.Context, in *LoginIn) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	Search(ctx context.Context, query string, limit int) ([]*User, error)
	UpdateStatus(ctx context.Context, userID string, status Status) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, u *User, passwordHash string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	ExistsUsername(ctx context.Context, username string) (bool, error)
	ExistsEmail(ctx context.Context, email string) (bool, error)
	GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, err error)
	GetNextDiscriminator(ctx context.Context, username string) (string, error)
	Search(ctx context.Context, query string, limit int) ([]*User, error)
	UpdateStatus(ctx context.Context, userID string, status string) error
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteExpiredSessions(ctx context.Context) error
}
