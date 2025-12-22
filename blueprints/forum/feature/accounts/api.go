// Package accounts provides user account management functionality.
package accounts

import (
	"context"
	"errors"
	"regexp"
	"time"
)

var (
	// ErrNotFound is returned when an account is not found.
	ErrNotFound = errors.New("account not found")

	// ErrInvalidUsername is returned for invalid usernames.
	ErrInvalidUsername = errors.New("invalid username")

	// ErrUsernameTaken is returned when username is already registered.
	ErrUsernameTaken = errors.New("username already taken")

	// ErrEmailTaken is returned when email is already registered.
	ErrEmailTaken = errors.New("email already taken")

	// ErrInvalidCredentials is returned for failed login.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrSuspended is returned when account is suspended.
	ErrSuspended = errors.New("account suspended")

	// Username validation
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
)

// Account represents a user account.
type Account struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	HeaderURL    string    `json:"header_url,omitempty"`
	Signature    string    `json:"signature,omitempty"`
	PostKarma    int       `json:"post_karma"`
	CommentKarma int       `json:"comment_karma"`
	TotalKarma   int       `json:"total_karma"`
	TrustLevel   int       `json:"trust_level"`
	Verified     bool      `json:"verified"`
	Admin        bool      `json:"admin,omitempty"`
	Suspended    bool      `json:"suspended,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an authentication session.
type Session struct {
	Token     string    `json:"token"`
	AccountID string    `json:"account_id"`
	Account   *Account  `json:"account,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateIn contains input for creating an account.
type CreateIn struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

// UpdateIn contains input for updating an account.
type UpdateIn struct {
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	HeaderURL   *string `json:"header_url,omitempty"`
	Signature   *string `json:"signature,omitempty"`
}

// LoginIn contains input for logging in.
type LoginIn struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
}

// AccountList is a paginated list of accounts.
type AccountList struct {
	Accounts []*Account `json:"accounts"`
	Total    int        `json:"total"`
}

// API defines the accounts service contract.
type API interface {
	// Account operations
	Create(ctx context.Context, in *CreateIn) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Account, error)
	List(ctx context.Context, limit, offset int) (*AccountList, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)

	// Authentication
	Login(ctx context.Context, in *LoginIn) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error

	// Karma management
	AddKarma(ctx context.Context, accountID string, postKarma, commentKarma int) error
	UpdateTrustLevel(ctx context.Context, accountID string) error

	// Admin operations
	SetVerified(ctx context.Context, id string, verified bool) error
	SetSuspended(ctx context.Context, id string, suspended bool) error
	SetAdmin(ctx context.Context, id string, admin bool) error
}

// Store defines the data access contract for accounts.
type Store interface {
	// Account operations
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

	// Session operations
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error

	// Karma operations
	AddKarma(ctx context.Context, accountID string, postKarma, commentKarma int) error

	// Admin operations
	SetVerified(ctx context.Context, id string, verified bool) error
	SetSuspended(ctx context.Context, id string, suspended bool) error
	SetAdmin(ctx context.Context, id string, admin bool) error
}
