package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// SocialHandler handles social endpoints
type SocialHandler struct {
	store store.Store
}

// NewSocialHandler creates a new social handler
func NewSocialHandler(st store.Store) *SocialHandler {
	return &SocialHandler{store: st}
}

// GetFriends returns the user's friends (following)
func (h *SocialHandler) GetFriends(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	following, err := h.store.Social().GetFollowing(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch friends"})
	}

	followers, _ := h.store.Social().GetFollowers(c.Context(), uid)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"following": following,
		"followers": followers,
	})
}

// Follow follows a user
func (h *SocialHandler) Follow(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	targetIDParam := c.Param("id")
	targetID, err := uuid.Parse(targetIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid target user ID"})
	}

	if uid == targetID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot follow yourself"})
	}

	if err := h.store.Social().Follow(c.Context(), uid, targetID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to follow user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "followed successfully"})
}

// Unfollow unfollows a user
func (h *SocialHandler) Unfollow(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	targetIDParam := c.Param("id")
	targetID, err := uuid.Parse(targetIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid target user ID"})
	}

	if err := h.store.Social().Unfollow(c.Context(), uid, targetID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unfollow user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "unfollowed successfully"})
}

// GetFriendLeaderboard returns the friend leaderboard
func (h *SocialHandler) GetFriendLeaderboard(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	leaderboard, err := h.store.Social().GetFriendLeaderboard(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch leaderboard"})
	}

	return c.JSON(http.StatusOK, leaderboard)
}

// GetFriendQuests returns active friend quests
func (h *SocialHandler) GetFriendQuests(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	quests, err := h.store.Social().GetFriendQuests(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch quests"})
	}

	return c.JSON(http.StatusOK, quests)
}

// GetFriendStreaks returns shared streaks with friends
func (h *SocialHandler) GetFriendStreaks(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	streaks, err := h.store.Social().GetFriendStreaks(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch streaks"})
	}

	return c.JSON(http.StatusOK, streaks)
}

// GetNotifications returns user notifications
func (h *SocialHandler) GetNotifications(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	unreadOnly := c.Query("unread") == "true"
	notifications, err := h.store.Social().GetNotifications(c.Context(), uid, unreadOnly)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch notifications"})
	}

	return c.JSON(http.StatusOK, notifications)
}

// MarkNotificationRead marks a notification as read
func (h *SocialHandler) MarkNotificationRead(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid notification ID"})
	}

	if err := h.store.Social().MarkNotificationRead(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to mark notification read"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "notification marked as read"})
}
