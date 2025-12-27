package handler

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/users"
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
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	user, session, err := h.users.Register(c.Context(), &in)
	if err != nil {
		if err == users.ErrUserExists {
			return c.JSON(http.StatusConflict, errResponse("user already exists"))
		}
		if err == users.ErrMissingUsername || err == users.ErrMissingEmailAddr {
			return c.JSON(http.StatusBadRequest, errResponse(err.Error()))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to register"))
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
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	user, session, err := h.users.Login(c.Context(), &in)
	if err != nil {
		if err == users.ErrInvalidEmail || err == users.ErrInvalidPassword {
			return c.JSON(http.StatusUnauthorized, errResponse("invalid email or password"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to login"))
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

	// Redirect for form submissions, JSON for API calls
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" || contentType == "application/x-www-form-urlencoded" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out",
	})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	cookie, err := c.Cookie("session")
	if err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, errResponse("unauthorized"))
	}

	user, err := h.users.GetBySession(c.Context(), cookie.Value)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, errResponse("unauthorized"))
	}

	return c.JSON(http.StatusOK, user)
}
