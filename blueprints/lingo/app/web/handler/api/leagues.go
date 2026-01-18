package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// LeagueHandler handles league endpoints
type LeagueHandler struct {
	store store.Store
}

// NewLeagueHandler creates a new league handler
func NewLeagueHandler(st store.Store) *LeagueHandler {
	return &LeagueHandler{store: st}
}

// GetLeagues returns all leagues
func (h *LeagueHandler) GetLeagues(c *mizu.Ctx) error {
	leagues, err := h.store.Gamification().GetLeagues(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch leagues"})
	}
	return c.JSON(http.StatusOK, leagues)
}

// GetCurrentLeague returns the user's current league
func (h *LeagueHandler) GetCurrentLeague(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	userLeague, err := h.store.Gamification().GetUserLeague(c.Context(), uid)
	if err != nil {
		// User not in a league yet, assign to Bronze
		season, err := h.store.Gamification().GetCurrentSeason(c.Context(), 1) // Bronze
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get league season"})
		}

		if err := h.store.Gamification().JoinLeague(c.Context(), uid, season.ID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to join league"})
		}

		userLeague, _ = h.store.Gamification().GetUserLeague(c.Context(), uid)
	}

	leagues, _ := h.store.Gamification().GetLeagues(c.Context())

	// Find current league info
	var currentLeague *store.League
	for _, l := range leagues {
		if l.ID == 1 { // Default to Bronze
			currentLeague = &l
			break
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_league":    userLeague,
		"current_league": currentLeague,
		"all_leagues":    leagues,
	})
}

// GetLeaderboard returns the leaderboard
func (h *LeagueHandler) GetLeaderboard(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	userLeague, err := h.store.Gamification().GetUserLeague(c.Context(), uid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not in a league"})
	}

	leaderboard, err := h.store.Gamification().GetLeaderboard(c.Context(), userLeague.SeasonID, 30)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch leaderboard"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"leaderboard": leaderboard,
		"user_rank":   userLeague.Rank,
		"user_xp":     userLeague.XPEarned,
	})
}
