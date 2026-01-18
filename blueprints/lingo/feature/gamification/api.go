package gamification

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for gamification
type Handler struct {
	svc *Service
}

// NewHandler creates a new gamification handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers gamification routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/leagues", h.GetLeagues)
	r.Get("/leagues/current", h.GetCurrentLeague)
	r.Get("/leagues/leaderboard", h.GetLeaderboard)
	r.Post("/leagues/{id}/join", h.JoinLeague)
	r.Get("/quests/daily", h.GetDailyQuests)
	r.Post("/quests/{id}/claim", h.ClaimQuestReward)
}

// GetLeagues handles GET /leagues
func (h *Handler) GetLeagues(c *mizu.Ctx) error {
	leagues, err := h.svc.GetLeagues(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get leagues"})
	}

	return c.JSON(http.StatusOK, leagues)
}

// GetCurrentLeague handles GET /leagues/current
func (h *Handler) GetCurrentLeague(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	info, err := h.svc.GetCurrentLeague(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not in a league"})
	}

	return c.JSON(http.StatusOK, info)
}

// GetLeaderboard handles GET /leagues/leaderboard
func (h *Handler) GetLeaderboard(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	limit := 30
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	leaderboard, err := h.svc.GetLeaderboard(c.Context(), userID, limit)
	if err != nil {
		switch err {
		case ErrLeagueNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not in a league"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get leaderboard"})
		}
	}

	return c.JSON(http.StatusOK, leaderboard)
}

// JoinLeague handles POST /leagues/{id}/join
func (h *Handler) JoinLeague(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	leagueID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid league id"})
	}

	if err := h.svc.JoinLeague(c.Context(), userID, leagueID); err != nil {
		switch err {
		case ErrAlreadyInLeague:
			return c.JSON(http.StatusConflict, map[string]string{"error": "already in a league this week"})
		case ErrLeagueNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "league not found"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to join league"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "joined league"})
}

// GetDailyQuests handles GET /quests/daily
func (h *Handler) GetDailyQuests(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	quests, err := h.svc.GetDailyQuests(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get quests"})
	}

	return c.JSON(http.StatusOK, quests)
}

// ClaimQuestReward handles POST /quests/{id}/claim
func (h *Handler) ClaimQuestReward(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	questID := c.Param("id")

	reward, err := h.svc.ClaimQuestReward(c.Context(), userID, questID)
	if err != nil {
		switch err {
		case ErrQuestNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "quest not found"})
		case ErrQuestNotCompleted:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "quest not completed"})
		case ErrQuestClaimed:
			return c.JSON(http.StatusConflict, map[string]string{"error": "quest already claimed"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to claim reward"})
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":     "reward claimed",
		"gems_earned": reward,
	})
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	if userIDStr := c.Request().Header.Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
