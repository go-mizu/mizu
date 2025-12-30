package rest

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/users"
)

// Auth handles authentication endpoints.
type Auth struct {
	users users.API
}

// NewAuth creates a new auth handler.
func NewAuth(users users.API) *Auth {
	return &Auth{users: users}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in users.RegisterIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	user, session, err := h.users.Register(c.Context(), &in)
	if err != nil {
		if err == users.ErrUserExists {
			return c.JSON(http.StatusConflict, errResponse("user already exists"))
		}
		if err == users.ErrMissingName || err == users.ErrMissingEmail {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to register")
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"user":    user,
		"session": session,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in users.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	user, session, err := h.users.Login(c.Context(), &in)
	if err != nil {
		if err == users.ErrInvalidEmail || err == users.ErrInvalidPassword {
			return Unauthorized(c, "invalid email or password")
		}
		return InternalError(c, "failed to login")
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"user":    user,
		"session": session,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	// Get session from cookie
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		_ = h.users.Logout(c.Context(), cookie.Value)
	}

	// Clear session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out",
	})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	cookie, err := c.Cookie("session")
	if err != nil || cookie.Value == "" {
		return Unauthorized(c, "unauthorized")
	}

	user, err := h.users.GetBySession(c.Context(), cookie.Value)
	if err != nil || user == nil {
		return Unauthorized(c, "unauthorized")
	}

	return OK(c, user)
}
