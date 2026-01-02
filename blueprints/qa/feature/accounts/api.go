package accounts

import (
	"context"
	"errors"
	"regexp"
	"time"
)

var (
	ErrNotFound         = errors.New("account not found")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrEmailTaken       = errors.New("email already taken")
	ErrInvalidUsername  = errors.New("invalid username format")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrAccountSuspended = errors.New("account is suspended")
	ErrSessionExpired   = errors.New("session expired")
)

const (
	UsernameMinLen = 3
	UsernameMaxLen = 20
	PasswordMinLen = 8
	BioMaxLen      = 500
	SessionTTL     = 30 * 24 * time.Hour
)

var (
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	EmailRegex    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// Account represents a user account.
type Account struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Bio          string    `json:"bio"`
	AvatarURL    string    `json:"avatar_url"`
	Location     string    `json:"location"`
	WebsiteURL   string    `json:"website_url"`
	Reputation   int64     `json:"reputation"`
	IsModerator  bool      `json:"is_moderator"`
	IsAdmin      bool      `json:"is_admin"`
	IsSuspended  bool      `json:"is_suspended"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	QuestionCount int64 `json:"question_count,omitempty"`
	AnswerCount   int64 `json:"answer_count,omitempty"`
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
	Location    *string `json:"location,omitempty"`
	WebsiteURL  *string `json:"website_url,omitempty"`
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
	Create(ctx context.Context, in CreateIn) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Account, error)
	UpdatePassword(ctx context.Context, id string, currentPassword, newPassword string) error
	Delete(ctx context.Context, id string) error

	Login(ctx context.Context, in LoginIn) (*Account, error)
	CreateSession(ctx context.Context, accountID, userAgent, ip string) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteAllSessions(ctx context.Context, accountID string) error

	UpdateReputation(ctx context.Context, id string, delta int64) error
	SetModerator(ctx context.Context, id string, isModerator bool) error
	SetAdmin(ctx context.Context, id string, isAdmin bool) error

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

	CreateSession(ctx context.Context, session *Session) error
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteSessionsByAccount(ctx context.Context, accountID string) error
	CleanExpiredSessions(ctx context.Context) error

	List(ctx context.Context, opts ListOpts) ([]*Account, error)
	Search(ctx context.Context, query string, limit int) ([]*Account, error)
}
