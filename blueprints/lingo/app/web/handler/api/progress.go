package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// ProgressHandler handles progress endpoints
type ProgressHandler struct {
	store store.Store
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(st store.Store) *ProgressHandler {
	return &ProgressHandler{store: st}
}

// GetProgress returns the user's overall progress
func (h *ProgressHandler) GetProgress(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	courses, err := h.store.Progress().GetUserCourses(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch progress"})
	}

	user, _ := h.store.Users().GetByID(c.Context(), uid)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"courses":     courses,
		"total_xp":    user.XPTotal,
		"streak_days": user.StreakDays,
		"gems":        user.Gems,
		"hearts":      user.Hearts,
	})
}

// GetXPHistory returns XP history
func (h *ProgressHandler) GetXPHistory(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	history, err := h.store.Progress().GetXPHistory(c.Context(), uid, 30)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch XP history"})
	}

	return c.JSON(http.StatusOK, history)
}

// GetStreaks returns streak information
func (h *ProgressHandler) GetStreaks(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	history, _ := h.store.Progress().GetStreakHistory(c.Context(), uid, 30)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_streak":      user.StreakDays,
		"streak_freeze_count": user.StreakFreezeCount,
		"history":             history,
	})
}

// UseStreakFreeze uses a streak freeze
func (h *ProgressHandler) UseStreakFreeze(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	if user.StreakFreezeCount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no streak freezes available"})
	}

	// Use streak freeze (in production, this would be more complex)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":             "streak freeze used",
		"streak_freeze_count": user.StreakFreezeCount - 1,
	})
}

// GetHearts returns heart information
func (h *ProgressHandler) GetHearts(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"hearts":           user.Hearts,
		"max_hearts":       5,
		"is_premium":       user.IsPremium,
		"next_refill_at":   user.HeartsUpdatedAt,
		"refill_time_mins": 300, // 5 hours per heart
	})
}

// RefillHearts refills hearts using gems
func (h *ProgressHandler) RefillHearts(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	user, err := h.store.Users().GetByID(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	gemCost := 350
	if user.Gems < gemCost {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "not enough gems"})
	}

	// Deduct gems and refill hearts
	h.store.Users().UpdateGems(c.Context(), uid, user.Gems-gemCost)
	h.store.Users().UpdateHearts(c.Context(), uid, 5)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"hearts": 5,
		"gems":   user.Gems - gemCost,
	})
}

// GetMistakes returns recent mistakes for practice
func (h *ProgressHandler) GetMistakes(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	mistakes, err := h.store.Progress().GetUserMistakes(c.Context(), uid, 50)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch mistakes"})
	}

	return c.JSON(http.StatusOK, mistakes)
}
