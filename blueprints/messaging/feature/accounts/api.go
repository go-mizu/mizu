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
	ErrPhoneTaken         = errors.New("phone number already registered")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidSession     = errors.New("invalid or expired session")
)

// User represents a user account.
type User struct {
	ID                   string    `json:"id"`
	Phone                string    `json:"phone,omitempty"`
	Email                string    `json:"email,omitempty"`
	Username             string    `json:"username"`
	DisplayName          string    `json:"display_name"`
	Bio                  string    `json:"bio,omitempty"`
	AvatarURL            string    `json:"avatar_url,omitempty"`
	Status               string    `json:"status,omitempty"`
	LastSeenAt           time.Time `json:"last_seen_at,omitempty"`
	IsOnline             bool      `json:"is_online"`
	PrivacyLastSeen      string    `json:"privacy_last_seen,omitempty"`
	PrivacyProfilePhoto  string    `json:"privacy_profile_photo,omitempty"`
	PrivacyAbout         string    `json:"privacy_about,omitempty"`
	PrivacyGroups        string    `json:"privacy_groups,omitempty"`
	PrivacyReadReceipts  bool      `json:"privacy_read_receipts"`
	TwoFAEnabled         bool      `json:"two_fa_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CreateIn contains input for creating an account.
type CreateIn struct {
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
}

// UpdateIn contains input for updating an account.
type UpdateIn struct {
	DisplayName          *string `json:"display_name,omitempty"`
	Bio                  *string `json:"bio,omitempty"`
	AvatarURL            *string `json:"avatar_url,omitempty"`
	Status               *string `json:"status,omitempty"`
	PrivacyLastSeen      *string `json:"privacy_last_seen,omitempty"`
	PrivacyProfilePhoto  *string `json:"privacy_profile_photo,omitempty"`
	PrivacyAbout         *string `json:"privacy_about,omitempty"`
	PrivacyGroups        *string `json:"privacy_groups,omitempty"`
	PrivacyReadReceipts  *bool   `json:"privacy_read_receipts,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Login    string `json:"login"` // Phone, email, or username
	Password string `json:"password"`
}

// Session represents an auth session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"token"`
	DeviceName   string    `json:"device_name,omitempty"`
	DeviceType   string    `json:"device_type,omitempty"`
	PushToken    string    `json:"push_token,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ChangePasswordIn contains input for changing password.
type ChangePasswordIn struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// API defines the accounts service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*User, error)
	Delete(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, userID string, in *ChangePasswordIn) error
	Login(ctx context.Context, in *LoginIn) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteAllSessions(ctx context.Context, userID string) error
	Search(ctx context.Context, query string, limit int) ([]*User, error)
	UpdateOnlineStatus(ctx context.Context, userID string, online bool) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, u *User, passwordHash string) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByIDs(ctx context.Context, ids []string) ([]*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	Delete(ctx context.Context, id string) error
	ExistsUsername(ctx context.Context, username string) (bool, error)
	ExistsPhone(ctx context.Context, phone string) (bool, error)
	ExistsEmail(ctx context.Context, email string) (bool, error)
	GetPasswordHash(ctx context.Context, login string) (id, hash string, err error)
	GetPasswordHashByID(ctx context.Context, userID string) (hash string, err error)
	Search(ctx context.Context, query string, limit int) ([]*User, error)
	UpdateOnlineStatus(ctx context.Context, userID string, online bool) error
	UpdateLastSeen(ctx context.Context, userID string) error
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteAllSessions(ctx context.Context, userID string) error
	DeleteExpiredSessions(ctx context.Context) error
}
