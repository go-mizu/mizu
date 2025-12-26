package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/pkg/password"
)

// SetupNewUserFunc is a callback to setup default chats for a new user.
type SetupNewUserFunc func(ctx context.Context, userID string)

// Auth handles authentication endpoints.
type Auth struct {
	accounts     accounts.API
	setupNewUser SetupNewUserFunc
}

// NewAuth creates a new Auth handler.
func NewAuth(accounts accounts.API, setupNewUser SetupNewUserFunc) *Auth {
	return &Auth{
		accounts:     accounts,
		setupNewUser: setupNewUser,
	}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Username == "" {
		return BadRequest(c, "Username is required")
	}
	if in.Password == "" {
		return BadRequest(c, "Password is required")
	}

	// Validate password strength
	if err := password.Validate(in.Password); err != nil {
		return BadRequest(c, err.Error())
	}

	ctx := c.Request().Context()

	user, err := h.accounts.Create(ctx, &in)
	if err != nil {
		switch err {
		case accounts.ErrUsernameTaken:
			return BadRequest(c, "Username already taken")
		case accounts.ErrEmailTaken:
			return BadRequest(c, "Email already registered")
		case accounts.ErrPhoneTaken:
			return BadRequest(c, "Phone number already registered")
		default:
			return InternalError(c, "Failed to create account")
		}
	}

	// Setup default chats for the new user (Saved Messages + Agent chat)
	if h.setupNewUser != nil {
		h.setupNewUser(ctx, user.ID)
	}

	// Auto-login after registration
	session, err := h.accounts.Login(ctx, &accounts.LoginIn{
		Login:    in.Username,
		Password: in.Password,
	})
	if err != nil {
		return Created(c, user)
	}

	// Set session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	return Created(c, map[string]any{
		"user":    user,
		"session": session,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Login == "" {
		return BadRequest(c, "Username or email is required")
	}
	if in.Password == "" {
		return BadRequest(c, "Password is required")
	}

	session, err := h.accounts.Login(c.Request().Context(), &in)
	if err != nil {
		return Unauthorized(c, "Invalid credentials")
	}

	user, _ := h.accounts.GetByID(c.Request().Context(), session.UserID)

	// Set session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	return Success(c, map[string]any{
		"user":    user,
		"session": session,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx, userID string) error {
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		h.accounts.DeleteSession(c.Request().Context(), cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})

	return Success(c, nil)
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx, userID string) error {
	user, err := h.accounts.GetByID(c.Request().Context(), userID)
	if err != nil {
		return NotFound(c, "User not found")
	}
	return Success(c, user)
}

// UpdateMe updates the current user.
func (h *Auth) UpdateMe(c *mizu.Ctx, userID string) error {
	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	user, err := h.accounts.Update(c.Request().Context(), userID, &in)
	if err != nil {
		return InternalError(c, "Failed to update profile")
	}

	return Success(c, user)
}

// ChangePassword changes the current user's password.
func (h *Auth) ChangePassword(c *mizu.Ctx, userID string) error {
	var in accounts.ChangePasswordIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.CurrentPassword == "" {
		return BadRequest(c, "Current password is required")
	}
	if in.NewPassword == "" {
		return BadRequest(c, "New password is required")
	}

	err := h.accounts.ChangePassword(c.Request().Context(), userID, &in)
	if err != nil {
		switch err {
		case accounts.ErrInvalidCredentials:
			return Unauthorized(c, "Current password is incorrect")
		default:
			// Check if it's a password validation error
			if err.Error() != "" {
				return BadRequest(c, err.Error())
			}
			return InternalError(c, "Failed to change password")
		}
	}

	return Success(c, map[string]any{
		"message": "Password changed successfully",
	})
}

// DeleteMe deletes the current user's account.
func (h *Auth) DeleteMe(c *mizu.Ctx, userID string) error {
	// Delete all sessions first
	h.accounts.DeleteAllSessions(c.Request().Context(), userID)

	// Delete the account
	err := h.accounts.Delete(c.Request().Context(), userID)
	if err != nil {
		return InternalError(c, "Failed to delete account")
	}

	// Clear session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})

	return Success(c, map[string]any{
		"message": "Account deleted successfully",
	})
}
