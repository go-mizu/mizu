// Package accounts provides account management functionality.
package accounts

import (
	"context"
	"time"
)

// Account represents a user account.
type Account struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name,omitempty"`
	Email       string    `json:"-"` // Never expose email in JSON
	Bio         string    `json:"bio,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	HeaderURL   string    `json:"header_url,omitempty"`
	Fields      []Field   `json:"fields,omitempty"`
	Verified    bool      `json:"verified"`
	Admin       bool      `json:"-"` // Don't expose admin status
	Suspended   bool      `json:"-"` // Don't expose suspension
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Computed fields (from joins)
	FollowersCount int  `json:"followers_count,omitempty"`
	FollowingCount int  `json:"following_count,omitempty"`
	PostsCount     int  `json:"posts_count,omitempty"`
	Following      bool `json:"following,omitempty"`   // Current user follows this
	FollowedBy     bool `json:"followed_by,omitempty"` // This user follows current
}

// Field is a custom profile field (key-value pair).
type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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
	DisplayName *string  `json:"display_name,omitempty"`
	Bio         *string  `json:"bio,omitempty"`
	AvatarURL   *string  `json:"avatar_url,omitempty"`
	HeaderURL   *string  `json:"header_url,omitempty"`
	Fields      *[]Field `json:"fields,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Session represents an auth session.
type Session struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AccountList is a paginated list of accounts.
type AccountList struct {
	Accounts []*Account `json:"accounts"`
	Total    int        `json:"total"`
}

// API defines the accounts service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Account, error)
	Login(ctx context.Context, in *LoginIn) (*Session, error)
	CreateSession(ctx context.Context, accountID string) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	Verify(ctx context.Context, id string, verified bool) error
	Suspend(ctx context.Context, id string, suspended bool) error
	SetAdmin(ctx context.Context, id string, admin bool) error
	List(ctx context.Context, limit, offset int) (*AccountList, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)
}

// Store defines the data access contract for accounts.
type Store interface {
	// Account CRUD
	Insert(ctx context.Context, a *Account, passwordHash string) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	ExistsUsername(ctx context.Context, username string) (bool, error)
	ExistsEmail(ctx context.Context, email string) (bool, error)
	GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, suspended bool, err error)
	List(ctx context.Context, limit, offset int) ([]*Account, int, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)
	SetVerified(ctx context.Context, id string, verified bool) error
	SetSuspended(ctx context.Context, id string, suspended bool) error
	SetAdmin(ctx context.Context, id string, admin bool) error

	// Session operations
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
}
