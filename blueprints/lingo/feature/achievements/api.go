package achievements

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for achievements
type Handler struct {
	svc *Service
}

// NewHandler creates a new achievement handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers achievement routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/achievements", h.GetAchievements)
	r.Get("/achievements/me", h.GetMyAchievements)
	r.Get("/achievements/categories", h.GetCategories)
	r.Get("/achievements/category/{category}", h.GetByCategory)
}

// GetAchievements handles GET /achievements
func (h *Handler) GetAchievements(c *mizu.Ctx) error {
	achievements, err := h.svc.GetAllAchievements(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get achievements"})
	}

	return c.JSON(http.StatusOK, achievements)
}

// GetMyAchievements handles GET /achievements/me
func (h *Handler) GetMyAchievements(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	achievements, err := h.svc.GetUserAchievements(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get achievements"})
	}

	// Get counts
	unlocked, total, _ := h.svc.GetUnlockedCount(c.Context(), userID)

	return c.JSON(http.StatusOK, map[string]any{
		"achievements": achievements,
		"unlocked":     unlocked,
		"total":        total,
	})
}

// GetCategories handles GET /achievements/categories
func (h *Handler) GetCategories(c *mizu.Ctx) error {
	return c.JSON(http.StatusOK, h.svc.Categories())
}

// GetByCategory handles GET /achievements/category/{category}
func (h *Handler) GetByCategory(c *mizu.Ctx) error {
	category := c.Param("category")

	achievements, err := h.svc.GetAchievementsByCategory(c.Context(), category)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get achievements"})
	}

	return c.JSON(http.StatusOK, achievements)
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
