package handler

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// Auth handles authentication endpoints.
type Auth struct {
	users     *users.Service
	getUserID func(*mizu.Ctx) string
}

// NewAuth creates a new auth handler.
func NewAuth(users *users.Service, getUserID func(*mizu.Ctx) string) *Auth {
	return &Auth{
		users:     users,
		getUserID: getUserID,
	}
}

// RegisterInput is the input for registration.
type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginInput is the input for login.
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in RegisterInput
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid input")
	}

	user, err := h.users.Create(c.Request().Context(), users.CreateIn{
		Username: in.Username,
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		switch err {
		case users.ErrUsernameTaken:
			return Conflict(c, "username already taken")
		case users.ErrEmailTaken:
			return Conflict(c, "email already taken")
		case users.ErrInvalidUsername:
			return BadRequest(c, "invalid username")
		case users.ErrInvalidEmail:
			return BadRequest(c, "invalid email")
		case users.ErrInvalidPassword:
			return BadRequest(c, "password must be at least 8 characters")
		default:
			return InternalError(c, err)
		}
	}

	// Create session
	session, err := h.users.CreateSession(c.Request().Context(), user.ID)
	if err != nil {
		return InternalError(c, err)
	}

	// Set cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return Created(c, map[string]any{
		"user":  user,
		"token": session.Token,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in LoginInput
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid input")
	}

	user, err := h.users.Login(c.Request().Context(), users.LoginIn{
		Username: in.Username,
		Password: in.Password,
	})
	if err != nil {
		return BadRequest(c, "invalid username or password")
	}

	// Create session
	session, err := h.users.CreateSession(c.Request().Context(), user.ID)
	if err != nil {
		return InternalError(c, err)
	}

	// Set cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return Success(c, map[string]any{
		"user":  user,
		"token": session.Token,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie("session")
	if err == nil && cookie != nil {
		_ = h.users.DeleteSession(c.Request().Context(), cookie.Value)
	}

	// Clear cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return NoContent(c)
}
