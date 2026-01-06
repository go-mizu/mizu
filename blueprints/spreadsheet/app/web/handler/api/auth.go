package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
)

// Auth handles authentication endpoints.
type Auth struct {
	users users.API
}

// NewAuth creates a new Auth handler.
func NewAuth(users users.API) *Auth {
	return &Auth{users: users}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in users.RegisterIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, token, err := h.users.Register(c.Request().Context(), &in)
	if err != nil {
		if err == users.ErrEmailExists {
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"user":  user,
		"token": token,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in users.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, token, err := h.users.Login(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user":  user,
		"token": token,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	// JWT tokens are stateless, so logout is just a client-side operation
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	// Get user ID from context (set by middleware)
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	user, err := h.users.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	return c.JSON(http.StatusOK, user)
}
