package gamification

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrLeagueNotFound    = errors.New("league not found")
	ErrAlreadyInLeague   = errors.New("already in league this week")
	ErrQuestNotFound     = errors.New("quest not found")
	ErrQuestNotCompleted = errors.New("quest not completed")
	ErrQuestClaimed      = errors.New("quest already claimed")
)

// Service handles gamification business logic
type Service struct {
	store        store.Store
	gamification store.GamificationStore
	users        store.UserStore
}

// NewService creates a new gamification service
func NewService(st store.Store) *Service {
	return &Service{
		store:        st,
		gamification: st.Gamification(),
		users:        st.Users(),
	}
}

// LeagueInfo represents league information for a user
type LeagueInfo struct {
	CurrentLeague  *store.League       `json:"current_league"`
	CurrentSeason  *store.LeagueSeason `json:"current_season"`
	UserStats      *store.UserLeague   `json:"user_stats"`
	Rank           int                 `json:"rank"`
	XPThisWeek     int                 `json:"xp_this_week"`
	PromotionZone  bool                `json:"promotion_zone"`
	DemotionZone   bool                `json:"demotion_zone"`
	TimeRemaining  time.Duration       `json:"time_remaining"`
}

// LeaderboardEntry represents a leaderboard entry
type LeaderboardEntry struct {
	Rank        int         `json:"rank"`
	User        *store.User `json:"user"`
	XP          int         `json:"xp"`
	IsCurrentUser bool      `json:"is_current_user"`
}

// DailyQuest represents a daily quest
type DailyQuest struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Target      int    `json:"target"`
	Progress    int    `json:"progress"`
	Completed   bool   `json:"completed"`
	Claimed     bool   `json:"claimed"`
	Reward      int    `json:"reward"`
}

// GetLeagues returns all leagues
func (s *Service) GetLeagues(ctx context.Context) ([]store.League, error) {
	return s.gamification.GetLeagues(ctx)
}

// GetCurrentLeague returns the user's current league info
func (s *Service) GetCurrentLeague(ctx context.Context, userID uuid.UUID) (*LeagueInfo, error) {
	userLeague, err := s.gamification.GetUserLeague(ctx, userID)
	if err != nil {
		// User not in a league, return empty info
		return &LeagueInfo{}, nil
	}

	leagues, _ := s.gamification.GetLeagues(ctx)
	var currentLeague *store.League
	for _, league := range leagues {
		if userLeague.SeasonID != uuid.Nil {
			season, _ := s.gamification.GetCurrentSeason(ctx, league.ID)
			if season != nil && season.ID == userLeague.SeasonID {
				currentLeague = &league
				break
			}
		}
	}

	// Calculate time remaining
	var timeRemaining time.Duration
	if currentLeague != nil {
		season, _ := s.gamification.GetCurrentSeason(ctx, currentLeague.ID)
		if season != nil {
			timeRemaining = time.Until(season.WeekEnd)
		}
	}

	// Determine promotion/demotion zone
	promotionZone := userLeague.Rank <= 10
	demotionZone := userLeague.Rank > 25 // Bottom 5 of 30

	return &LeagueInfo{
		CurrentLeague:  currentLeague,
		UserStats:      userLeague,
		Rank:           userLeague.Rank,
		XPThisWeek:     userLeague.XPEarned,
		PromotionZone:  promotionZone,
		DemotionZone:   demotionZone,
		TimeRemaining:  timeRemaining,
	}, nil
}

// GetLeaderboard returns the leaderboard for a user's current league
func (s *Service) GetLeaderboard(ctx context.Context, userID uuid.UUID, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 30
	}

	userLeague, err := s.gamification.GetUserLeague(ctx, userID)
	if err != nil {
		return nil, ErrLeagueNotFound
	}

	leaderboard, err := s.gamification.GetLeaderboard(ctx, userLeague.SeasonID, limit)
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, len(leaderboard))
	for i, ul := range leaderboard {
		entries[i] = LeaderboardEntry{
			Rank:          ul.Rank,
			User:          ul.User,
			XP:            ul.XPEarned,
			IsCurrentUser: ul.UserID == userID,
		}
	}

	return entries, nil
}

// JoinLeague joins the user to the current week's league
func (s *Service) JoinLeague(ctx context.Context, userID uuid.UUID, leagueID int) error {
	// Check if already in a league this week
	existing, _ := s.gamification.GetUserLeague(ctx, userID)
	if existing != nil {
		return ErrAlreadyInLeague
	}

	// Get or create current season
	season, err := s.gamification.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return ErrLeagueNotFound
	}

	return s.gamification.JoinLeague(ctx, userID, season.ID)
}

// GetDailyQuests returns the user's daily quests
func (s *Service) GetDailyQuests(ctx context.Context, userID uuid.UUID) ([]DailyQuest, error) {
	// This would need to be stored in database
	// For now, return sample quests
	return []DailyQuest{
		{
			ID:          "complete_lessons",
			Type:        "lessons",
			Description: "Complete 3 lessons",
			Target:      3,
			Progress:    0,
			Completed:   false,
			Claimed:     false,
			Reward:      20,
		},
		{
			ID:          "earn_xp",
			Type:        "xp",
			Description: "Earn 50 XP",
			Target:      50,
			Progress:    0,
			Completed:   false,
			Claimed:     false,
			Reward:      15,
		},
		{
			ID:          "perfect_lesson",
			Type:        "perfect",
			Description: "Complete a perfect lesson",
			Target:      1,
			Progress:    0,
			Completed:   false,
			Claimed:     false,
			Reward:      25,
		},
	}, nil
}

// ClaimQuestReward claims a quest reward
func (s *Service) ClaimQuestReward(ctx context.Context, userID uuid.UUID, questID string) (int, error) {
	// Get quests and find the one to claim
	quests, _ := s.GetDailyQuests(ctx, userID)

	for _, quest := range quests {
		if quest.ID == questID {
			if !quest.Completed {
				return 0, ErrQuestNotCompleted
			}
			if quest.Claimed {
				return 0, ErrQuestClaimed
			}

			// Award gems
			user, err := s.users.GetByID(ctx, userID)
			if err != nil {
				return 0, err
			}

			_ = s.users.UpdateGems(ctx, userID, user.Gems+quest.Reward)
			return quest.Reward, nil
		}
	}

	return 0, ErrQuestNotFound
}

// ProcessWeeklyLeagues processes weekly league transitions
func (s *Service) ProcessWeeklyLeagues(ctx context.Context) error {
	return s.gamification.ProcessWeeklyLeagues(ctx)
}
