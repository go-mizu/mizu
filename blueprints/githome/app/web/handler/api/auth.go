package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/go-mizu/mizu"
)

// Auth handles authentication endpoints
type Auth struct {
	users  users.API
	actors *duckdb.ActorsStore
}

// NewAuth creates a new auth handler
func NewAuth(users users.API, actors *duckdb.ActorsStore) *Auth {
	return &Auth{users: users, actors: actors}
}

// Register handles user registration
func (h *Auth) Register(c *mizu.Ctx) error {
	var in users.RegisterIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	user, session, err := h.users.Register(c.Context(), &in)
	if err != nil {
		switch err {
		case users.ErrUserExists:
			return Conflict(c, "user already exists")
		case users.ErrMissingUsername:
			return BadRequest(c, "username is required")
		case users.ErrMissingEmail:
			return BadRequest(c, "email is required")
		case users.ErrMissingPassword:
			return BadRequest(c, "password is required")
		case users.ErrPasswordTooShort:
			return BadRequest(c, "password must be at least 8 characters")
		default:
			return InternalError(c, "failed to register user")
		}
	}

	// Create actor for the user
	_, err = h.actors.GetOrCreateForUser(c.Context(), user.ID)
	if err != nil {
		// Log but don't fail registration
		// The actor will be created on first repo creation if needed
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

	return Created(c, map[string]any{
		"user":    user,
		"session": session,
	})
}

// Login handles user login
func (h *Auth) Login(c *mizu.Ctx) error {
	var in users.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	user, session, err := h.users.Login(c.Context(), &in)
	if err != nil {
		switch err {
		case users.ErrNotFound, users.ErrInvalidPassword:
			return Unauthorized(c, "invalid credentials")
		case users.ErrInvalidInput:
			return BadRequest(c, "login and password are required")
		default:
			return InternalError(c, "failed to login")
		}
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

	return OK(c, map[string]any{
		"user":    user,
		"session": session,
	})
}

// Logout handles user logout
func (h *Auth) Logout(c *mizu.Ctx) error {
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		h.users.Logout(c.Context(), cookie.Value)
	}

	// Clear session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return OK(c, map[string]string{
		"message": "logged out",
	})
}

// Me returns the current authenticated user
func (h *Auth) Me(c *mizu.Ctx) error {
	cookie, err := c.Cookie("session")
	if err != nil || cookie.Value == "" {
		return Unauthorized(c, "not authenticated")
	}

	user, err := h.users.ValidateSession(c.Context(), cookie.Value)
	if err != nil {
		return Unauthorized(c, "session expired")
	}

	return OK(c, user)
}
