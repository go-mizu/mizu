package accounts

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// Errors
var (
	ErrNotFound        = errors.New("account not found")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailTaken      = errors.New("email already taken")
	ErrInvalidUsername = errors.New("invalid username format")
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPassword = errors.New("invalid password")
	ErrAccountSuspended = errors.New("account is suspended")
	ErrSessionExpired  = errors.New("session expired")
)

// Validation constants
const (
	UsernameMinLen = 3
	UsernameMaxLen = 20
	PasswordMinLen = 8
	BioMaxLen      = 500
	SessionTTL     = 30 * 24 * time.Hour // 30 days
)

var (
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	EmailRegex    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// Account represents a user account.
type Account struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email,omitempty"`
	PasswordHash  string     `json:"-"`
	DisplayName   string     `json:"display_name"`
	Bio           string     `json:"bio"`
	AvatarURL     string     `json:"avatar_url"`
	BannerURL     string     `json:"banner_url"`
	Karma         int64      `json:"karma"`
	PostKarma     int64      `json:"post_karma"`
	CommentKarma  int64      `json:"comment_karma"`
	IsAdmin       bool       `json:"is_admin"`
	IsSuspended   bool       `json:"is_suspended"`
	SuspendReason string     `json:"suspend_reason,omitempty"`
	SuspendUntil  *time.Time `json:"suspend_until,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Computed fields
	ThreadCount  int64 `json:"thread_count,omitempty"`
	CommentCount int64 `json:"comment_count,omitempty"`
	CakeDay      bool  `json:"cake_day,omitempty"`
}

// Session represents an authentication session.
type Session struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Token     string    `json:"token"`
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CreateIn contains input for creating an account.
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

// UpdateIn contains input for updating an account.
type UpdateIn struct {
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	BannerURL   *string `json:"banner_url,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ListOpts contains options for listing accounts.
type ListOpts struct {
	Limit   int
	Cursor  string
	OrderBy string
}

// API defines the accounts service interface.
type API interface {
	// Account management
	Create(ctx context.Context, in CreateIn) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Account, error)
	UpdatePassword(ctx context.Context, id string, currentPassword, newPassword string) error
	Delete(ctx context.Context, id string) error

	// Authentication
	Login(ctx context.Context, in LoginIn) (*Account, error)
	CreateSession(ctx context.Context, accountID, userAgent, ip string) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteAllSessions(ctx context.Context, accountID string) error

	// Karma
	UpdateKarma(ctx context.Context, id string, postDelta, commentDelta int64) error

	// Admin
	Suspend(ctx context.Context, id string, reason string, until *time.Time) error
	Unsuspend(ctx context.Context, id string) error
	SetAdmin(ctx context.Context, id string, isAdmin bool) error

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*Account, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)
}

// Store defines the data storage interface for accounts.
type Store interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, id string) error

	// Sessions
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteSessionsByAccount(ctx context.Context, accountID string) error
	CleanExpiredSessions(ctx context.Context) error

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*Account, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)
}
