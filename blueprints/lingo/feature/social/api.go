package social

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for social features
type Handler struct {
	svc *Service
}

// NewHandler creates a new social handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers social routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/friends", h.GetFriends)
	r.Post("/friends/{id}/follow", h.Follow)
	r.Delete("/friends/{id}/unfollow", h.Unfollow)
	r.Get("/friends/leaderboard", h.GetFriendLeaderboard)
	r.Get("/friends/quests", h.GetFriendQuests)
	r.Get("/friends/streaks", h.GetFriendStreaks)
	r.Get("/notifications", h.GetNotifications)
	r.Put("/notifications/{id}/read", h.MarkNotificationRead)
}

// GetFriends handles GET /friends
func (h *Handler) GetFriends(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	friends, err := h.svc.GetFriends(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get friends"})
	}

	return c.JSON(http.StatusOK, friends)
}

// Follow handles POST /friends/{id}/follow
func (h *Handler) Follow(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	targetIDStr := c.Param("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	if err := h.svc.Follow(c.Context(), userID, targetID); err != nil {
		switch err {
		case ErrCannotFollowSelf:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot follow yourself"})
		case ErrUserNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		case ErrAlreadyFollowing:
			return c.JSON(http.StatusConflict, map[string]string{"error": "already following"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to follow"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "followed"})
}

// Unfollow handles DELETE /friends/{id}/unfollow
func (h *Handler) Unfollow(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	targetIDStr := c.Param("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	if err := h.svc.Unfollow(c.Context(), userID, targetID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unfollow"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "unfollowed"})
}

// GetFriendLeaderboard handles GET /friends/leaderboard
func (h *Handler) GetFriendLeaderboard(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	leaderboard, err := h.svc.GetFriendLeaderboard(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get leaderboard"})
	}

	return c.JSON(http.StatusOK, leaderboard)
}

// GetFriendQuests handles GET /friends/quests
func (h *Handler) GetFriendQuests(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	quests, err := h.svc.GetFriendQuests(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get quests"})
	}

	return c.JSON(http.StatusOK, quests)
}

// GetFriendStreaks handles GET /friends/streaks
func (h *Handler) GetFriendStreaks(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	streaks, err := h.svc.GetFriendStreaks(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get streaks"})
	}

	return c.JSON(http.StatusOK, streaks)
}

// GetNotifications handles GET /notifications
func (h *Handler) GetNotifications(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	unreadOnly := c.Query("unread") == "true"

	notifications, err := h.svc.GetNotifications(c.Context(), userID, unreadOnly)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get notifications"})
	}

	return c.JSON(http.StatusOK, notifications)
}

// MarkNotificationRead handles PUT /notifications/{id}/read
func (h *Handler) MarkNotificationRead(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid notification id"})
	}

	if err := h.svc.MarkNotificationRead(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to mark as read"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "marked as read"})
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	if userIDStr := c.Header().Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
