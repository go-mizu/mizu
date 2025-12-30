// Package accounts provides user account management.
package accounts

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// Errors
var (
	ErrNotFound        = errors.New("user not found")
	ErrLoginTaken      = errors.New("username already taken")
	ErrEmailTaken      = errors.New("email already taken")
	ErrInvalidLogin    = errors.New("invalid username format")
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionExpired  = errors.New("session expired")
	ErrForbidden       = errors.New("forbidden")
)

// Validation
var (
	LoginRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	EmailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

const (
	LoginMinLen    = 3
	LoginMaxLen    = 60
	PasswordMinLen = 8
	SessionTTL     = 30 * 24 * time.Hour // 30 days
)

// User represents a WordPress-compatible user.
type User struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`           // user_login
	Email             string    `json:"email,omitempty"`    // user_email
	Nicename          string    `json:"slug"`               // user_nicename (URL slug)
	URL               string    `json:"url,omitempty"`      // user_url
	DisplayName       string    `json:"name"`               // display_name
	Registered        time.Time `json:"registered_date"`    // user_registered
	Status            int       `json:"status,omitempty"`   // user_status
	Roles             []string  `json:"roles,omitempty"`    // From usermeta
	Capabilities      map[string]bool `json:"capabilities,omitempty"` // From usermeta
	Description       string    `json:"description,omitempty"` // From usermeta
	AvatarURLs        map[string]string `json:"avatar_urls,omitempty"` // Gravatar URLs
	Meta              map[string]interface{} `json:"meta,omitempty"` // User meta
	Link              string    `json:"link,omitempty"`     // Author archive URL
}

// Session represents a user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	UserAgent string    `json:"user_agent,omitempty"`
	IP        string    `json:"ip,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CreateIn contains input for creating a user.
type CreateIn struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"name,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

// Validate validates the create input.
func (in *CreateIn) Validate() error {
	if len(in.Username) < LoginMinLen || len(in.Username) > LoginMaxLen {
		return ErrInvalidLogin
	}
	if !LoginRegex.MatchString(in.Username) {
		return ErrInvalidLogin
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
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"name,omitempty"`
	URL         *string `json:"url,omitempty"`
	Description *string `json:"description,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Password    *string `json:"password,omitempty"`
	Nicename    *string `json:"slug,omitempty"`
}

// LoginIn contains input for login.
type LoginIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ListOpts contains options for listing users.
type ListOpts struct {
	Page        int      `json:"page"`
	PerPage     int      `json:"per_page"`
	Search      string   `json:"search"`
	Include     []string `json:"include"`
	Exclude     []string `json:"exclude"`
	OrderBy     string   `json:"orderby"`
	Order       string   `json:"order"`
	Roles       []string `json:"roles"`
	Capabilities []string `json:"capabilities"`
	Who         string   `json:"who"` // "authors" for users who have published posts
	HasPublishedPosts bool `json:"has_published_posts"`
}

// API defines the accounts service interface.
type API interface {
	// User management
	Create(ctx context.Context, in CreateIn) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetBySlug(ctx context.Context, slug string) (*User, error)
	Update(ctx context.Context, id string, in UpdateIn) (*User, error)
	Delete(ctx context.Context, id string, reassign string) error

	// Authentication
	Login(ctx context.Context, in LoginIn) (*User, error)
	CreateSession(ctx context.Context, userID, userAgent, ip string) (*Session, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteAllSessions(ctx context.Context, userID string) error

	// User meta
	GetMeta(ctx context.Context, userID, key string) (string, error)
	SetMeta(ctx context.Context, userID, key, value string) error
	DeleteMeta(ctx context.Context, userID, key string) error

	// Roles and capabilities
	GetRoles(ctx context.Context, userID string) ([]string, error)
	SetRoles(ctx context.Context, userID string, roles []string) error
	HasCapability(ctx context.Context, userID, capability string) (bool, error)

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*User, int, error)
	Count(ctx context.Context) (int, error)

	// Current user
	GetCurrent(ctx context.Context, token string) (*User, error)
}
