package handler

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/accounts"
)

// Auth handles authentication endpoints.
type Auth struct {
	accounts accounts.API
}

// NewAuth creates a new auth handler.
func NewAuth(accountsSvc accounts.API) *Auth {
	return &Auth{accounts: accountsSvc}
}

// Register handles POST /api/v1/auth/register
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if in.Username == "" {
		return UnprocessableEntity(c, "username is required")
	}
	if in.Email == "" {
		return UnprocessableEntity(c, "email is required")
	}
	if in.Password == "" || len(in.Password) < 8 {
		return UnprocessableEntity(c, "password must be at least 8 characters")
	}

	account, err := h.accounts.Create(c.Request().Context(), &in)
	if err != nil {
		switch err {
		case accounts.ErrUsernameTaken:
			return Conflict(c, "username already taken")
		case accounts.ErrEmailTaken:
			return Conflict(c, "email already taken")
		default:
			return InternalError(c, err)
		}
	}

	// Create session
	session, err := h.accounts.CreateSession(c.Request().Context(), account.ID, c.Request().UserAgent(), c.Request().RemoteAddr)
	if err != nil {
		return InternalError(c, err)
	}

	// Set cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return Created(c, map[string]interface{}{
		"account": account,
		"token":   session.Token,
	})
}

// Login handles POST /api/v1/auth/login
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if in.Username == "" {
		return UnprocessableEntity(c, "username or email is required")
	}
	if in.Password == "" {
		return UnprocessableEntity(c, "password is required")
	}

	session, err := h.accounts.Login(c.Request().Context(), &in)
	if err != nil {
		switch err {
		case accounts.ErrInvalidCredentials:
			return Unauthorized(c)
		case accounts.ErrAccountSuspended:
			return Forbidden(c)
		default:
			return InternalError(c, err)
		}
	}

	account, err := h.accounts.GetByID(c.Request().Context(), session.AccountID)
	if err != nil {
		return InternalError(c, err)
	}

	// Set cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return Success(c, map[string]interface{}{
		"account": account,
		"token":   session.Token,
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *Auth) Logout(c *mizu.Ctx) error {
	token := ""
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
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return NoContent(c)
}
