package web

import (
	"strings"

	"github.com/go-mizu/mizu"
)

// authRequired is a middleware that requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		accountID := s.getAccountID(c)
		if accountID == "" {
			return c.JSON(401, map[string]any{
				"error": map[string]any{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
		}
		return next(c)
	}
}

// getAccountID extracts the account ID from the request.
func (s *Server) getAccountID(c *mizu.Ctx) string {
	// Try Authorization header first
	auth := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.accounts.GetSession(c.Request().Context(), token)
		if err == nil {
			return session.AccountID
		}
	}

	// Try cookie
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil {
			return session.AccountID
		}
	}

	return ""
}

// optionalAuth extracts account ID if present but doesn't require it.
func (s *Server) optionalAuth(c *mizu.Ctx) string {
	return s.getAccountID(c)
}
