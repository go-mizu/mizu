// Package auth provides authentication functionality.
package auth

import (
	"context"
	"time"
)

// User represents an authenticated user.
type User struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	FirstName string         `json:"firstName,omitempty"`
	LastName  string         `json:"lastName,omitempty"`
	Roles     []string       `json:"roles,omitempty"`
	Data      map[string]any `json:"-"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// LoginInput holds login credentials.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResult holds login response data.
type LoginResult struct {
	User         *User  `json:"user"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken,omitempty"`
	Exp          int64  `json:"exp"`
}

// RegisterInput holds registration data.
type RegisterInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

// RefreshResult holds token refresh response data.
type RefreshResult struct {
	User         *User  `json:"user"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken,omitempty"`
	Exp          int64  `json:"exp"`
}

// ForgotPasswordInput holds forgot password data.
type ForgotPasswordInput struct {
	Email string `json:"email"`
}

// ResetPasswordInput holds reset password data.
type ResetPasswordInput struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// API defines the auth service interface.
type API interface {
	// Login authenticates a user and returns tokens.
	Login(ctx context.Context, collection string, input *LoginInput) (*LoginResult, error)

	// Logout invalidates the current session.
	Logout(ctx context.Context, collection, token string) error

	// Me returns the current authenticated user.
	Me(ctx context.Context, collection, token string) (*User, error)

	// RefreshToken refreshes an expired token.
	RefreshToken(ctx context.Context, collection, refreshToken string) (*RefreshResult, error)

	// Register creates a new user account.
	Register(ctx context.Context, collection string, input *RegisterInput) (*LoginResult, error)

	// ForgotPassword initiates password reset flow.
	ForgotPassword(ctx context.Context, collection string, input *ForgotPasswordInput) error

	// ResetPassword resets a user's password.
	ResetPassword(ctx context.Context, collection string, input *ResetPasswordInput) error

	// VerifyEmail verifies a user's email address.
	VerifyEmail(ctx context.Context, collection, token string) error

	// Unlock removes account lockout.
	Unlock(ctx context.Context, collection string, email string) error

	// ValidateToken validates a token and returns the user ID.
	ValidateToken(token string) (userID string, collection string, err error)
}
