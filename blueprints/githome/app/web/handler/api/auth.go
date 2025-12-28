package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	users users.API
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(users users.API) *AuthHandler {
	return &AuthHandler{users: users}
}

// Login handles POST /login
func (h *AuthHandler) Login(c *mizu.Ctx) error {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	user, err := h.users.Authenticate(c.Context(), in.Login, in.Password)
	if err != nil {
		return Unauthorized(c)
	}

	return c.JSON(http.StatusOK, user)
}

// Register handles POST /register
func (h *AuthHandler) Register(c *mizu.Ctx) error {
	var in users.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	user, err := h.users.Create(c.Context(), &in)
	if err != nil {
		switch err {
		case users.ErrUserExists:
			return Conflict(c, "Login already exists")
		case users.ErrEmailExists:
			return Conflict(c, "Email already exists")
		default:
			return WriteError(c, http.StatusInternalServerError, err.Error())
		}
	}

	return Created(c, user)
}
