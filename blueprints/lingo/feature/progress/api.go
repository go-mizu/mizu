package progress

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for progress
type Handler struct {
	svc *Service
}

// NewHandler creates a new progress handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers progress routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/progress", h.GetProgress)
	r.Get("/xp/history", h.GetXPHistory)
	r.Get("/streaks", h.GetStreaks)
	r.Post("/streaks/freeze", h.UseStreakFreeze)
	r.Get("/hearts", h.GetHearts)
	r.Post("/hearts/refill", h.RefillHearts)
	r.Get("/practice/mistakes", h.GetMistakes)
	r.Get("/practice/review", h.GetLexemesForReview)
}

// GetProgress handles GET /progress
func (h *Handler) GetProgress(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	progress, err := h.svc.GetProgress(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, progress)
}

// GetXPHistory handles GET /xp/history
func (h *Handler) GetXPHistory(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	history, err := h.svc.GetXPHistory(c.Context(), userID, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get history"})
	}

	return c.JSON(http.StatusOK, history)
}

// GetStreaks handles GET /streaks
func (h *Handler) GetStreaks(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	info, err := h.svc.GetStreakInfo(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, info)
}

// UseStreakFreeze handles POST /streaks/freeze
func (h *Handler) UseStreakFreeze(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	if err := h.svc.UseStreakFreeze(c.Context(), userID); err != nil {
		switch err {
		case ErrNoStreakFreeze:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no streak freeze available"})
		case ErrStreakNotAtRisk:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "streak not at risk"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to use freeze"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "streak freeze used"})
}

// GetHearts handles GET /hearts
func (h *Handler) GetHearts(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	info, err := h.svc.GetHeartsInfo(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, info)
}

// RefillHearts handles POST /hearts/refill
func (h *Handler) RefillHearts(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	if err := h.svc.RefillHearts(c.Context(), userID); err != nil {
		switch err {
		case ErrInsufficientGems:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "insufficient gems"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to refill hearts"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "hearts refilled"})
}

// GetMistakes handles GET /practice/mistakes
func (h *Handler) GetMistakes(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	mistakes, err := h.svc.GetMistakes(c.Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get mistakes"})
	}

	return c.JSON(http.StatusOK, mistakes)
}

// GetLexemesForReview handles GET /practice/review
func (h *Handler) GetLexemesForReview(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	lexemes, err := h.svc.GetLexemesForReview(c.Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get lexemes"})
	}

	return c.JSON(http.StatusOK, lexemes)
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
