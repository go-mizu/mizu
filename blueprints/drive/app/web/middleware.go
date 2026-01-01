package web

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
)

// requestContext stores per-request data
type requestContext struct {
	userID    string
	user      *accounts.User
	sessionID string
}

// contextStore stores request contexts by request pointer
var contextStore sync.Map

// authRequired is middleware that requires authentication.
// In LocalMode, authentication is skipped and a default local user is used.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		// Skip auth in local mode
		if s.cfg.LocalMode {
			contextStore.Store(c.Request(), &requestContext{
				userID: "local",
				user:   &accounts.User{ID: "local", Name: "Local User"},
			})
			defer contextStore.Delete(c.Request())
			return next(c)
		}

		token := s.getAuthToken(c)
		if token == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		user, err := s.accounts.GetBySession(c.Context(), token)
		if err != nil || user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		// Store context data
		contextStore.Store(c.Request(), &requestContext{
			userID: user.ID,
			user:   user,
		})
		defer contextStore.Delete(c.Request())

		return next(c)
	}
}

// optionalAuth loads user if authenticated (returns nil if not).
func (s *Server) optionalAuth(c *mizu.Ctx) *accounts.User {
	token := s.getAuthToken(c)
	if token == "" {
		return nil
	}

	user, err := s.accounts.GetBySession(c.Context(), token)
	if err != nil {
		return nil
	}

	return user
}

// getUserID returns the current user ID from context.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// Return local user in local mode
	if s.cfg.LocalMode {
		return "local"
	}

	if ctx, ok := contextStore.Load(c.Request()); ok {
		return ctx.(*requestContext).userID
	}

	token := s.getAuthToken(c)
	if token == "" {
		return ""
	}

	user, err := s.accounts.GetBySession(c.Context(), token)
	if err != nil || user == nil {
		return ""
	}

	return user.ID
}

// getUser returns the current user from context.
func (s *Server) getUser(c *mizu.Ctx) *accounts.User {
	if ctx, ok := contextStore.Load(c.Request()); ok {
		return ctx.(*requestContext).user
	}
	return nil
}

// getAuthToken extracts auth token from cookie or header.
func (s *Server) getAuthToken(c *mizu.Ctx) string {
	// Try cookie first
	cookie, err := c.Cookie("token")
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
