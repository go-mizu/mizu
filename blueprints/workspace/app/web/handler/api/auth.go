package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/users"
)

const sessionCookieName = "workspace_session"

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

	user, session, err := h.users.Register(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	setSessionCookie(c, session)
	return c.JSON(http.StatusCreated, user)
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in users.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, session, err := h.users.Login(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	setSessionCookie(c, session)
	return c.JSON(http.StatusOK, user)
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie(sessionCookieName)
	if err == nil {
		h.users.Logout(c.Request().Context(), cookie.Value)
	}

	clearSessionCookie(c)
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie(sessionCookieName)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	user, err := h.users.GetBySession(c.Request().Context(), cookie.Value)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	return c.JSON(http.StatusOK, user)
}

func setSessionCookie(c *mizu.Ctx, session *users.Session) {
	// Determine if we should set Secure flag based on request
	secure := c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(c *mizu.Ctx) {
	secure := c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
