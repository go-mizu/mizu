package web

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

const authHeaderName = "Authorization"

// devUserID is used in development mode to bypass authentication
const devUserID = "dev-user-001"

// getUserID extracts the user ID from the JWT token in the Authorization header.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// In dev mode, return a default user ID
	if s.cfg.Dev {
		return devUserID
	}

	// Get token from Authorization header
	auth := c.Request().Header.Get(authHeaderName)
	if auth == "" {
		slog.Debug("no authorization header found", "path", c.Request().URL.Path)
		return ""
	}

	// Parse Bearer token
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		slog.Debug("invalid authorization header format")
		return ""
	}

	// Verify token
	userID, err := s.users.VerifyToken(c.Request().Context(), token)
	if err != nil {
		slog.Debug("token verification failed", "error", err)
		return ""
	}

	return userID
}

// authRequired is middleware that requires authentication.
func (s *Server) authRequired(h mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		userID := s.getUserID(c)
		if userID == "" {
			// Check if this is an API request
			if len(c.Request().URL.Path) > 4 && c.Request().URL.Path[:5] == "/api/" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
			}
			// Redirect to login for UI requests
			http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
			return nil
		}
		return h(c)
	}
}
