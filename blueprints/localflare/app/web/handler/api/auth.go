package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/auth"
)

// Auth handles authentication requests.
type Auth struct {
	svc auth.API
}

// NewAuth creates a new Auth handler.
func NewAuth(svc auth.API) *Auth {
	return &Auth{svc: svc}
}

// Login authenticates a user.
func (h *Auth) Login(c *mizu.Ctx) error {
	var input auth.LoginIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	result, err := h.svc.Login(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "Invalid credentials"})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// Register creates a new user.
func (h *Auth) Register(c *mizu.Ctx) error {
	var input auth.RegisterIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	result, err := h.svc.Register(c.Request().Context(), &input)
	if err != nil {
		if err == auth.ErrUserExists {
			return c.JSON(409, map[string]string{"error": "User already exists"})
		}
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// Logout invalidates a session.
func (h *Auth) Logout(c *mizu.Ctx) error {
	token := extractToken(c)
	if token == "" {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	if err := h.svc.Logout(c.Request().Context(), token); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	token := extractToken(c)
	if token == "" {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	user, err := h.svc.GetCurrentUser(c.Request().Context(), token)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  user,
	})
}

func extractToken(c *mizu.Ctx) string {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return ""
	}
	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:]
	}
	return token
}
