package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
)

// Auth handles authentication endpoints.
type Auth struct {
	accounts  accounts.API
	getUserID func(*mizu.Ctx) string
}

// NewAuth creates a new Auth handler.
func NewAuth(accounts accounts.API, getUserID func(*mizu.Ctx) string) *Auth {
	return &Auth{accounts: accounts, getUserID: getUserID}
}

// Register handles user registration.
func (h *Auth) Register(c *mizu.Ctx) error {
	var in accounts.RegisterIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	user, session, err := h.accounts.Register(c.Context(), &in)
	if err != nil {
		if err == accounts.ErrUserExists {
			return c.JSON(http.StatusConflict, errResponse("user already exists"))
		}
		if err == accounts.ErrMissingName || err == accounts.ErrMissingEmail {
			return c.JSON(http.StatusBadRequest, errResponse(err.Error()))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to register"))
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"user":    user,
		"session": session,
	})
}

// Login handles user login.
func (h *Auth) Login(c *mizu.Ctx) error {
	var in accounts.LoginIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	user, session, err := h.accounts.Login(c.Context(), &in)
	if err != nil {
		if err == accounts.ErrInvalidEmail || err == accounts.ErrInvalidPassword {
			return c.JSON(http.StatusUnauthorized, errResponse("invalid email or password"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to login"))
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"user":    user,
		"session": session,
	})
}

// Logout handles user logout.
func (h *Auth) Logout(c *mizu.Ctx) error {
	// Get session from cookie
	cookie, err := c.Cookie("token")
	if err == nil && cookie.Value != "" {
		_ = h.accounts.Logout(c.Context(), cookie.Value)
	}

	// Clear session cookie
	c.SetCookie(&http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect for form submissions, JSON for API calls
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" || contentType == "application/x-www-form-urlencoded" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out",
	})
}

// Me returns the current user.
func (h *Auth) Me(c *mizu.Ctx) error {
	cookie, err := c.Cookie("token")
	if err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, errResponse("unauthorized"))
	}

	user, err := h.accounts.GetBySession(c.Context(), cookie.Value)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, errResponse("unauthorized"))
	}

	return c.JSON(http.StatusOK, user)
}

// Update updates the current user.
func (h *Auth) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	user, err := h.accounts.Update(c.Context(), userID, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, user)
}

// ChangePassword changes the user's password.
func (h *Auth) ChangePassword(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in accounts.ChangePasswordIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	if err := h.accounts.ChangePassword(c.Context(), userID, &in); err != nil {
		if err == accounts.ErrInvalidPassword {
			return c.JSON(http.StatusBadRequest, errResponse("current password is incorrect"))
		}
		return c.JSON(http.StatusBadRequest, errResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListSessions lists the user's sessions.
func (h *Auth) ListSessions(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	sessions, err := h.accounts.ListSessions(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, sessions)
}

// DeleteSession deletes a session.
func (h *Auth) DeleteSession(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	sessionID := c.Param("id")

	if err := h.accounts.DeleteSession(c.Context(), userID, sessionID); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// StorageInfo returns storage usage information.
func (h *Auth) StorageInfo(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	user, err := h.accounts.GetByID(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errResponse("user not found"))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"used":       user.StorageUsed,
		"quota":      user.StorageQuota,
		"updated_at": time.Now(),
	})
}
