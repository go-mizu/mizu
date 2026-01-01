package handler

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
)

// Auth handles authentication endpoints.
type Auth struct {
	accounts accounts.API
}

// NewAuth creates a new auth handler.
func NewAuth(accounts accounts.API) *Auth {
	return &Auth{accounts: accounts}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.RegisterIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	account, err := h.accounts.Register(c.Request().Context(), &in)
	if err != nil {
		switch err {
		case accounts.ErrUsernameTaken:
			return Conflict(c, "Username already taken")
		case accounts.ErrEmailTaken:
			return Conflict(c, "Email already registered")
		case accounts.ErrWeakPassword:
			return BadRequest(c, "Password must be at least 8 characters")
		default:
			return InternalError(c, "Failed to create account")
		}
	}

	return Created(c, account)
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	in.UserAgent = c.Request().UserAgent()
	in.IPAddress = c.Request().RemoteAddr

	session, account, err := h.accounts.Login(c.Request().Context(), &in)
	if err != nil {
		switch err {
		case accounts.ErrInvalidCredentials:
			return Unauthorized(c, "Invalid username or password")
		case accounts.ErrAccountSuspended:
			return Forbidden(c, "Account suspended")
		default:
			return InternalError(c, "Failed to login")
		}
	}

	// Set session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})

	return OK(c, map[string]any{
		"session": session,
		"user":    account,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie("session")
	if err == nil {
		h.accounts.Logout(c.Request().Context(), cookie.Value)
	}

	// Clear cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	return NoContent(c)
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return Unauthorized(c, "Not authenticated")
	}

	account, _, err := h.accounts.GetByToken(c.Request().Context(), cookie.Value)
	if err != nil {
		return Unauthorized(c, "Invalid session")
	}

	return OK(c, account)
}

// ChangePassword handles password change.
func (h *Auth) ChangePassword(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return Unauthorized(c, "Not authenticated")
	}

	account, _, err := h.accounts.GetByToken(c.Request().Context(), cookie.Value)
	if err != nil {
		return Unauthorized(c, "Invalid session")
	}

	var in accounts.ChangePasswordIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if err := h.accounts.ChangePassword(c.Request().Context(), account.ID, &in); err != nil {
		switch err {
		case accounts.ErrInvalidPassword:
			return Unauthorized(c, "Current password is incorrect")
		case accounts.ErrWeakPassword:
			return BadRequest(c, "New password must be at least 8 characters")
		default:
			return InternalError(c, "Failed to change password")
		}
	}

	return OK(c, map[string]string{"message": "Password changed successfully"})
}

// StorageUsage returns storage usage.
func (h *Auth) StorageUsage(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return Unauthorized(c, "Not authenticated")
	}

	account, _, err := h.accounts.GetByToken(c.Request().Context(), cookie.Value)
	if err != nil {
		return Unauthorized(c, "Invalid session")
	}

	usage, err := h.accounts.GetStorageUsage(c.Request().Context(), account.ID)
	if err != nil {
		return InternalError(c, "Failed to get storage usage")
	}

	return OK(c, usage)
}

func getAccountFromCookie(c *mizu.Ctx, accountsSvc accounts.API) (*accounts.Account, error) {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return nil, err
	}

	account, _, err := accountsSvc.GetByToken(c.Request().Context(), cookie.Value)
	return account, err
}

func getAccountIDFromCookie(c *mizu.Ctx, accountsSvc accounts.API) string {
	account, err := getAccountFromCookie(c, accountsSvc)
	if err != nil {
		return ""
	}
	return account.ID
}

// SessionDuration is the default session duration.
const SessionDuration = 30 * 24 * time.Hour
