package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// AchievementHandler handles achievement endpoints
type AchievementHandler struct {
	store store.Store
}

// NewAchievementHandler creates a new achievement handler
func NewAchievementHandler(st store.Store) *AchievementHandler {
	return &AchievementHandler{store: st}
}

// GetAchievements returns all achievements
func (h *AchievementHandler) GetAchievements(c *mizu.Ctx) error {
	achievements, err := h.store.Achievements().GetAchievements(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch achievements"})
	}
	return c.JSON(http.StatusOK, achievements)
}

// GetMyAchievements returns the user's achievements
func (h *AchievementHandler) GetMyAchievements(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	// Get all achievements
	allAchievements, err := h.store.Achievements().GetAchievements(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch achievements"})
	}

	// Get user's progress
	userAchievements, err := h.store.Achievements().GetUserAchievements(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch user achievements"})
	}

	// Create a map for quick lookup
	progressMap := make(map[string]store.UserAchievement)
	for _, ua := range userAchievements {
		progressMap[ua.AchievementID] = ua
	}

	// Combine achievements with progress
	type AchievementWithProgress struct {
		store.Achievement
		Level      int   `json:"level"`
		Progress   int   `json:"progress"`
		Unlocked   bool  `json:"unlocked"`
		MaxedOut   bool  `json:"maxed_out"`
	}

	result := make([]AchievementWithProgress, 0, len(allAchievements))
	for _, a := range allAchievements {
		awp := AchievementWithProgress{
			Achievement: a,
		}
		if ua, ok := progressMap[a.ID]; ok {
			awp.Level = ua.Level
			awp.Progress = ua.Progress
			awp.Unlocked = ua.Level > 0
			awp.MaxedOut = ua.Level >= a.MaxLevel
		}
		result = append(result, awp)
	}

	return c.JSON(http.StatusOK, result)
}
