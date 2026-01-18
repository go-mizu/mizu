package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// UserHandler handles user endpoints
type UserHandler struct {
	store store.Store
}

// NewUserHandler creates a new user handler
func NewUserHandler(st store.Store) *UserHandler {
	return &UserHandler{store: st}
}

// GetMe returns the current user
func (h *UserHandler) GetMe(c *mizu.Ctx) error {
	// In production, get user ID from JWT token
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateMe updates the current user
func (h *UserHandler) UpdateMe(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	var req struct {
		DisplayName      string `json:"display_name"`
		Bio              string `json:"bio"`
		DailyGoalMinutes int    `json:"daily_goal_minutes"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}
	if req.DailyGoalMinutes > 0 {
		user.DailyGoalMinutes = req.DailyGoalMinutes
	}

	if err := h.store.Users().Update(c.Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update user"})
	}

	return c.JSON(http.StatusOK, user)
}

// GetByUsername returns a user by username
func (h *UserHandler) GetByUsername(c *mizu.Ctx) error {
	username := c.Param("username")
	user, err := h.store.Users().GetByUsername(c.Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Don't expose sensitive data for other users
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
		"xp_total":     user.XPTotal,
		"streak_days":  user.StreakDays,
		"created_at":   user.CreatedAt,
	})
}

// GetStats returns user statistics
func (h *UserHandler) GetStats(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Get additional stats
	courses, _ := h.store.Progress().GetUserCourses(c.Context(), id)
	achievements, _ := h.store.Achievements().GetUserAchievements(c.Context(), id)
	streakHistory, _ := h.store.Progress().GetStreakHistory(c.Context(), id, 30)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":           user,
		"courses":        len(courses),
		"achievements":   len(achievements),
		"streak_history": streakHistory,
	})
}

// UpdateSettings updates user settings
func (h *UserHandler) UpdateSettings(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	var req struct {
		DailyGoalMinutes int `json:"daily_goal_minutes"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.DailyGoalMinutes > 0 {
		user.DailyGoalMinutes = req.DailyGoalMinutes
	}

	if err := h.store.Users().Update(c.Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update settings"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "settings updated"})
}
