package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/users"
)

// Auth handles authentication endpoints.
type Auth struct {
	users     *users.Service
	getUserID func(*mizu.Ctx) string
}

// NewAuth creates a new auth handler.
func NewAuth(users *users.Service, getUserID func(*mizu.Ctx) string) *Auth {
	return &Auth{users: users, getUserID: getUserID}
}

// RegisterRequest is the request body for registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginRequest is the request body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var req RegisterRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return BadRequest(c, "email and password are required")
	}

	user, session, err := h.users.Register(c.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		if err == users.ErrEmailTaken {
			return BadRequest(c, "email already taken")
		}
		return InternalError(c, "failed to register")
	}

	return Created(c, map[string]any{
		"token": session.Token,
		"user":  user,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var req LoginRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return BadRequest(c, "email and password are required")
	}

	user, session, err := h.users.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		if err == users.ErrInvalidAuth {
			return Unauthorized(c, "invalid email or password")
		}
		return InternalError(c, "failed to login")
	}

	return OK(c, map[string]any{
		"token": session.Token,
		"user":  user,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	// Session deletion is handled by middleware
	return OK(c, map[string]any{"message": "logged out"})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return NotFound(c, "user not found")
	}

	return OK(c, map[string]any{"user": user})
}
