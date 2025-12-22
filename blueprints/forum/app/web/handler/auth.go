package handler

import (
	"net/http"

	"github.com/go-mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu"
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
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	account, err := h.accounts.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	// Login to create session
	session, err := h.accounts.Login(c.Request().Context(), &accounts.LoginIn{
		UsernameOrEmail: in.Email,
		Password:        in.Password,
	})
	if err != nil {
		// If login fails, still return account but without session
		return c.JSON(200, DataResponse(map[string]any{
			"account": account,
			"session": map[string]any{
				"token":      "",
				"account_id": account.ID,
			},
		}))
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(200, DataResponse(map[string]any{
		"session": session,
		"account": account,
	}))
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	session, err := h.accounts.Login(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(401, ErrorResponse(err.Error()))
	}

	account, _ := h.accounts.GetByID(c.Request().Context(), session.AccountID)

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(200, DataResponse(map[string]any{
		"session": session,
		"account": account,
	}))
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	// Try to get token from header or cookie
	token := c.Request().Header.Get("Authorization")
	if len(token) > 7 {
		token = token[7:] // Remove "Bearer "
	} else {
		cookie, err := c.Cookie("session_token")
		if err == nil {
			token = cookie.Value
		}
	}

	if token != "" {
		_ = h.accounts.DeleteSession(c.Request().Context(), token)
	}

	// Clear session cookie
	c.SetCookie(&http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Logged out successfully",
	}))
}

// VerifyCredentials returns the current user's account.
func (h *Auth) VerifyCredentials(c *mizu.Ctx, getAccountID func(*mizu.Ctx) string) error {
	accountID := getAccountID(c)
	account, err := h.accounts.GetByID(c.Request().Context(), accountID)
	if err != nil {
		return c.JSON(404, ErrorResponse("Account not found"))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"account": account,
	}))
}
