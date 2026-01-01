// Package accounts provides user account management.
package accounts

import (
	"context"
	"time"
)

// Account represents a user account.
type Account struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name,omitempty"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	StorageQuota int64     `json:"storage_quota"`
	StorageUsed  int64     `json:"storage_used"`
	IsAdmin      bool      `json:"is_admin,omitempty"`
	IsSuspended  bool      `json:"is_suspended,omitempty"`
	Preferences  string    `json:"preferences,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an authenticated session.
type Session struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Token     string    `json:"token,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	LastUsed  time.Time `json:"last_used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterIn contains registration input.
type RegisterIn struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
}

// LoginIn contains login input.
type LoginIn struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

// UpdateIn contains profile update input.
type UpdateIn struct {
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// ChangePasswordIn contains password change input.
type ChangePasswordIn struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// StorageUsage represents storage usage breakdown.
type StorageUsage struct {
	Quota     int64            `json:"quota"`
	Used      int64            `json:"used"`
	Available int64            `json:"available"`
	Percent   float64          `json:"percent_used"`
	Breakdown map[string]int64 `json:"breakdown"`
}

// API defines the accounts service contract.
type API interface {
	Register(ctx context.Context, in *RegisterIn) (*Account, error)
	Login(ctx context.Context, in *LoginIn) (*Session, *Account, error)
	Logout(ctx context.Context, token string) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByToken(ctx context.Context, token string) (*Account, *Session, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Account, error)
	ChangePassword(ctx context.Context, id string, in *ChangePasswordIn) error
	GetStorageUsage(ctx context.Context, id string) (*StorageUsage, error)
	UpdateStorageUsed(ctx context.Context, id string, delta int64) error
	ListSessions(ctx context.Context, accountID string) ([]*Session, error)
	RevokeSession(ctx context.Context, accountID, sessionID string) error
}

// Store defines the data access contract.
type Store interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	UpdateStorageUsed(ctx context.Context, id string, delta int64) error

	CreateSession(ctx context.Context, s *Session) error
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	UpdateSessionLastUsed(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, id string) error
	DeleteSessionsByAccount(ctx context.Context, accountID string) error
	ListSessionsByAccount(ctx context.Context, accountID string) ([]*Session, error)

	GetStorageByCategory(ctx context.Context, accountID string) (map[string]int64, error)
}
