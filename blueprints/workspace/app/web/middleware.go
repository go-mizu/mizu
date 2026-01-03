package web

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

const sessionCookieName = "workspace_session"

// getUserID extracts the user ID from the session cookie.
func (s *Server) getUserID(c *mizu.Ctx) string {
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
