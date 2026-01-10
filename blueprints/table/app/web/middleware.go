package web

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/users"
)

// requestContext stores per-request data.
type requestContext struct {
	userID    string
	user      *users.User
	sessionID string
}

// contextStore stores request contexts by request pointer.
var contextStore sync.Map

// authRequired is middleware that requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		sessionID := s.getSessionID(c)
		if sessionID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "unauthorized",
			})
		}

		user, err := s.users.GetBySession(c.Context(), sessionID)
		if err != nil || user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "unauthorized",
			})
		}

		// Store context data
		contextStore.Store(c.Request(), &requestContext{
			userID:    user.ID,
			user:      user,
			sessionID: sessionID,
		})
		defer contextStore.Delete(c.Request())

		return next(c)
	}
}

// getUserID returns the current user ID from context.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// Check context store first (set by authRequired middleware)
	if ctx, ok := contextStore.Load(c.Request()); ok {
		return ctx.(*requestContext).userID
	}

	// Fall back to checking session directly
	sessionID := s.getSessionID(c)
	if sessionID == "" {
		return ""
	}

	user, err := s.users.GetBySession(c.Context(), sessionID)
	if err != nil || user == nil {
		return ""
	}

	return user.ID
}

// getUser returns the current user from context.
func (s *Server) getUser(c *mizu.Ctx) *users.User {
	if ctx, ok := contextStore.Load(c.Request()); ok {
		return ctx.(*requestContext).user
	}
	return nil
}

// getSessionID extracts session ID from cookie or header.
func (s *Server) getSessionID(c *mizu.Ctx) string {
	// Try cookie first
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try Authorization header
	auth := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}
