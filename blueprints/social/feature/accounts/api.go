// Package accounts provides account management functionality.
package accounts

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound          = errors.New("account not found")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrEmailTaken        = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountSuspended  = errors.New("account is suspended")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidSession    = errors.New("invalid or expired session")
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
	Location    string    `json:"location,omitempty"`
	Website     string    `json:"website,omitempty"`
	Fields      []Field   `json:"fields,omitempty"`
	Verified    bool      `json:"verified"`
	Admin       bool      `json:"-"` // Don't expose admin status
	Suspended   bool      `json:"-"` // Don't expose suspension
	Private     bool      `json:"private"`
	Discoverable bool     `json:"discoverable"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Computed fields (from joins/aggregations)
	FollowersCount int  `json:"followers_count"`
	FollowingCount int  `json:"following_count"`
	PostsCount     int  `json:"posts_count"`
	Following      bool `json:"following,omitempty"`   // Current user follows this
	FollowedBy     bool `json:"followed_by,omitempty"` // This user follows current
	Requested      bool `json:"requested,omitempty"`   // Follow request pending
	Blocking       bool `json:"blocking,omitempty"`    // Current user blocks this
	Muting         bool `json:"muting,omitempty"`      // Current user mutes this
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
	DisplayName  *string  `json:"display_name,omitempty"`
	Bio          *string  `json:"bio,omitempty"`
	AvatarURL    *string  `json:"avatar_url,omitempty"`
	HeaderURL    *string  `json:"header_url,omitempty"`
	Location     *string  `json:"location,omitempty"`
	Website      *string  `json:"website,omitempty"`
	Fields       *[]Field `json:"fields,omitempty"`
	Private      *bool    `json:"private,omitempty"`
	Discoverable *bool    `json:"discoverable,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Username string `json:"username"` // Can be username or email
	Password string `json:"password"`
}

// Session represents an auth session.
type Session struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Token     string    `json:"token"`
	UserAgent string    `json:"user_agent,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AccountList is a paginated list of accounts.
type AccountList struct {
	Accounts []*Account `json:"accounts"`
	Total    int        `json:"total"`
}

// ListOpts specifies options for listing accounts.
type ListOpts struct {
	Limit  int
	Offset int
}

// API defines the accounts service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Account, error)
	Login(ctx context.Context, in *LoginIn) (*Session, error)
	CreateSession(ctx context.Context, accountID, userAgent, ipAddress string) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	Verify(ctx context.Context, id string, verified bool) error
	Suspend(ctx context.Context, id string, suspended bool) error
	SetAdmin(ctx context.Context, id string, admin bool) error
	List(ctx context.Context, opts ListOpts) (*AccountList, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)

	// Enrichment
	PopulateStats(ctx context.Context, a *Account) error
	PopulateRelationship(ctx context.Context, a *Account, viewerID string) error
}

// Store defines the data access contract for accounts.
type Store interface {
	// Account CRUD
	Insert(ctx context.Context, a *Account, passwordHash string) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Account, error)
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

	// Stats
	GetFollowersCount(ctx context.Context, id string) (int, error)
	GetFollowingCount(ctx context.Context, id string) (int, error)
	GetPostsCount(ctx context.Context, id string) (int, error)

	// Session operations
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteExpiredSessions(ctx context.Context) error
}
