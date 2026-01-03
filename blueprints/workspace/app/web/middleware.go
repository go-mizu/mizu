package web

import (
	"net/http"
	"os"

	"github.com/go-mizu/mizu"
)

const sessionCookieName = "workspace_session"

// devUserID is used in development mode to bypass authentication
const devUserID = "dev-user-001"

// isDevMode returns true if running in development mode
func isDevMode() bool {
	return os.Getenv("DEV_MODE") == "true" || os.Getenv("GO_ENV") == "development"
}

// getUserID extracts the user ID from the session cookie.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// In dev mode, return a default user ID
	if isDevMode() {
		return devUserID
	}

	cookie, err := c.Request().Cookie(sessionCookieName)
	if err != nil {
		return ""
	}

	user, err := s.users.GetBySession(c.Request().Context(), cookie.Value)
	if err != nil {
		return ""
	}

	return user.ID
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
