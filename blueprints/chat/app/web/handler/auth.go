package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
)

// Auth handles authentication endpoints.
type Auth struct {
	accounts accounts.API
}

// NewAuth creates a new Auth handler.
func NewAuth(accounts accounts.API) *Auth {
	return &Auth{accounts: accounts}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Username == "" {
		return BadRequest(c, "Username is required")
	}
	if in.Email == "" {
		return BadRequest(c, "Email is required")
	}
	if in.Password == "" {
		return BadRequest(c, "Password is required")
	}
	if len(in.Password) < 8 {
		return BadRequest(c, "Password must be at least 8 characters")
	}

	user, err := h.accounts.Create(c.Request().Context(), &in)
	if err != nil {
		switch err {
		case accounts.ErrUsernameTaken:
			return Conflict(c, "Username is taken")
		case accounts.ErrEmailTaken:
			return Conflict(c, "Email is already registered")
		default:
			return InternalError(c, "Failed to create account")
		}
	}

	return Created(c, user)
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 0); err != nil {
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

	// Set cookie (HttpOnly false to allow WebSocket JS access)
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	// Get user
	user, _ := h.accounts.GetByID(c.Request().Context(), session.UserID)

	return Success(c, map[string]any{
		"token": session.Token,
		"user":  user,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx, userID string) error {
	// Get token from Authorization header first
	token := ""
	if auth := c.Request().Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}

	// Fall back to cookie
	if token == "" {
		if cookie, err := c.Cookie("session"); err == nil {
			token = cookie.Value
		}
	}

	if token != "" {
		h.accounts.DeleteSession(c.Request().Context(), token)
	}

	// Clear cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	return NoContent(c)
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
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	user, err := h.accounts.Update(c.Request().Context(), userID, &in)
	if err != nil {
		return InternalError(c, "Failed to update user")
	}

	return Success(c, user)
}
