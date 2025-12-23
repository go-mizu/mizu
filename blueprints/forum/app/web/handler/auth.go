package handler

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
)

// Auth handles authentication endpoints.
type Auth struct {
	accounts     accounts.API
	getAccountID func(*mizu.Ctx) string
}

// NewAuth creates a new auth handler.
func NewAuth(accounts accounts.API, getAccountID func(*mizu.Ctx) string) *Auth {
	return &Auth{accounts: accounts, getAccountID: getAccountID}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	account, err := h.accounts.Create(c.Request().Context(), in)
	if err != nil {
		switch err {
		case accounts.ErrUsernameTaken:
			return Conflict(c, "Username already taken")
		case accounts.ErrEmailTaken:
			return Conflict(c, "Email already taken")
		case accounts.ErrInvalidUsername:
			return BadRequest(c, "Invalid username format")
		case accounts.ErrInvalidEmail:
			return BadRequest(c, "Invalid email format")
		case accounts.ErrInvalidPassword:
			return BadRequest(c, "Password must be at least 8 characters")
		default:
			return InternalError(c)
		}
	}

	// Create session
	session, err := h.accounts.CreateSession(
		c.Request().Context(),
		account.ID,
		c.Request().UserAgent(),
		c.Request().RemoteAddr,
	)
	if err != nil {
		return InternalError(c)
	}

	// Set cookie
	h.setSessionCookie(c, session.Token, session.ExpiresAt)

	return Created(c, map[string]any{
		"account": account,
		"token":   session.Token,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	account, err := h.accounts.Login(c.Request().Context(), in)
	if err != nil {
		switch err {
		case accounts.ErrInvalidPassword:
			return ErrorResponse(c, 401, "INVALID_CREDENTIALS", "Invalid username or password")
		case accounts.ErrAccountSuspended:
			return ErrorResponse(c, 403, "ACCOUNT_SUSPENDED", "Your account has been suspended")
		default:
			return ErrorResponse(c, 401, "INVALID_CREDENTIALS", "Invalid username or password")
		}
	}

	// Create session
	session, err := h.accounts.CreateSession(
		c.Request().Context(),
		account.ID,
		c.Request().UserAgent(),
		c.Request().RemoteAddr,
	)
	if err != nil {
		return InternalError(c)
	}

	// Set cookie
	h.setSessionCookie(c, session.Token, session.ExpiresAt)

	return Success(c, map[string]any{
		"account": account,
		"token":   session.Token,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	// Get token from cookie or header
	var token string
	if cookie, err := c.Request().Cookie("session"); err == nil {
		token = cookie.Value
	}

	if token != "" {
		_ = h.accounts.DeleteSession(c.Request().Context(), token)
	}

	// Clear cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return Success(c, map[string]any{"message": "Logged out"})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	account, err := h.accounts.GetByID(c.Request().Context(), accountID)
	if err != nil {
		return InternalError(c)
	}

	return Success(c, account)
}

func (h *Auth) setSessionCookie(c *mizu.Ctx, token string, expires time.Time) {
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Request().TLS != nil,
	})
}
