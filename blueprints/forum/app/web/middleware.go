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
				"error": "Authentication required",
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
		if err == nil && session != nil {
			return session.AccountID
		}
	}

	// Try cookie
	cookie, err := c.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil && session != nil {
			return session.AccountID
		}
	}

	return ""
}

// optionalAuth extracts account ID if present but doesn't require it.
func (s *Server) optionalAuth(c *mizu.Ctx) string {
	return s.getAccountID(c)
}

// moderatorRequired checks if the user is a moderator of the forum.
func (s *Server) moderatorRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		accountID := s.getAccountID(c)
		if accountID == "" {
			return c.JSON(401, map[string]any{
				"error": "Authentication required",
			})
		}

		forumID := c.Param("id")
		if forumID == "" {
			forumID = c.Param("forum_id")
		}

		if forumID != "" {
			isMod, _, err := s.forums.IsModerator(c.Request().Context(), forumID, accountID)
			if err != nil || !isMod {
				return c.JSON(403, map[string]any{
					"error": "Moderator access required",
				})
			}
		}

		return next(c)
	}
}

// adminRequired checks if the user is an admin.
func (s *Server) adminRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		accountID := s.getAccountID(c)
		if accountID == "" {
			return c.JSON(401, map[string]any{
				"error": "Authentication required",
			})
		}

		account, err := s.accounts.GetByID(c.Request().Context(), accountID)
		if err != nil || !account.Admin {
			return c.JSON(403, map[string]any{
				"error": "Admin access required",
			})
		}

		return next(c)
	}
}
