package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// Auth contains authentication-related handlers.
type Auth struct {
	accounts accounts.API
}

// NewAuth creates new auth handlers.
func NewAuth(accounts accounts.API) *Auth {
	return &Auth{accounts: accounts}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	account, err := h.accounts.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(400, ErrorResponse("REGISTRATION_FAILED", err.Error()))
	}

	session, err := h.accounts.CreateSession(c.Request().Context(), account.ID)
	if err != nil {
		return c.JSON(500, ErrorResponse("SESSION_FAILED", "Failed to create session"))
	}

	// Set session cookie for auto-login
	c.SetCookie(&http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(200, map[string]any{
		"data": map[string]any{
			"account": account,
			"token":   session.Token,
		},
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	session, err := h.accounts.Login(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(401, ErrorResponse("LOGIN_FAILED", err.Error()))
	}

	account, _ := h.accounts.GetByID(c.Request().Context(), session.AccountID)

	// Set session cookie for persistent login
	c.SetCookie(&http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(200, map[string]any{
		"data": map[string]any{
			"account": account,
			"token":   session.Token,
		},
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	token := c.Request().Header.Get("Authorization")
	if len(token) > 7 {
		token = token[7:] // Remove "Bearer "
		_ = h.accounts.DeleteSession(c.Request().Context(), token)
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

	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}
