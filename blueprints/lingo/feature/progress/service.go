package progress

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInsufficientGems   = errors.New("insufficient gems")
	ErrNoStreakFreeze     = errors.New("no streak freeze available")
	ErrStreakNotAtRisk    = errors.New("streak not at risk")
	ErrInsufficientHearts = errors.New("insufficient hearts")
)

// Service handles progress business logic
type Service struct {
	store    store.Store
	progress store.ProgressStore
	users    store.UserStore
}

// NewService creates a new progress service
func NewService(st store.Store) *Service {
	return &Service{
		store:    st,
		progress: st.Progress(),
		users:    st.Users(),
	}
}

// ProgressOverview represents a user's overall progress
type ProgressOverview struct {
	TotalXP       int64            `json:"total_xp"`
	StreakDays    int              `json:"streak_days"`
	Hearts        int              `json:"hearts"`
	Gems          int              `json:"gems"`
	IsPremium     bool             `json:"is_premium"`
	Courses       []store.UserCourse `json:"courses"`
	TodayXP       int              `json:"today_xp"`
	WeekXP        int              `json:"week_xp"`
	DailyGoal     int              `json:"daily_goal"`
	GoalCompleted bool             `json:"goal_completed"`
}

// StreakInfo represents streak information
type StreakInfo struct {
	CurrentStreak   int               `json:"current_streak"`
	LongestStreak   int               `json:"longest_streak"`
	StreakFreezes   int               `json:"streak_freezes"`
	TodayCompleted  bool              `json:"today_completed"`
	StreakAtRisk    bool              `json:"streak_at_risk"`
	History         []store.StreakDay `json:"history"`
}

// HeartsInfo represents hearts information
type HeartsInfo struct {
	Hearts          int        `json:"hearts"`
	MaxHearts       int        `json:"max_hearts"`
	IsPremium       bool       `json:"is_premium"`
	NextRefillAt    *time.Time `json:"next_refill_at,omitempty"`
	RefillCostGems  int        `json:"refill_cost_gems"`
}

// GetProgress returns a user's overall progress
func (s *Service) GetProgress(ctx context.Context, userID uuid.UUID) (*ProgressOverview, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	courses, _ := s.progress.GetUserCourses(ctx, userID)

	// Get today's XP
	todayHistory, _ := s.progress.GetXPHistory(ctx, userID, 1)
	var todayXP int
	if len(todayHistory) > 0 {
		todayXP = todayHistory[0].Amount
	}

	// Get week XP
	weekHistory, _ := s.progress.GetXPHistory(ctx, userID, 7)
	var weekXP int
	for _, event := range weekHistory {
		weekXP += event.Amount
	}

	// Check if daily goal completed
	goalCompleted := todayXP >= user.DailyGoalMinutes*10 // Approximate XP per minute

	return &ProgressOverview{
		TotalXP:       user.XPTotal,
		StreakDays:    user.StreakDays,
		Hearts:        user.Hearts,
		Gems:          user.Gems,
		IsPremium:     user.IsPremium,
		Courses:       courses,
		TodayXP:       todayXP,
		WeekXP:        weekXP,
		DailyGoal:     user.DailyGoalMinutes * 10,
		GoalCompleted: goalCompleted,
	}, nil
}

// GetXPHistory returns XP history for the last N days
func (s *Service) GetXPHistory(ctx context.Context, userID uuid.UUID, days int) ([]store.XPEvent, error) {
	return s.progress.GetXPHistory(ctx, userID, days)
}

// GetStreakInfo returns streak information for a user
func (s *Service) GetStreakInfo(ctx context.Context, userID uuid.UUID) (*StreakInfo, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	history, _ := s.progress.GetStreakHistory(ctx, userID, 30)

	// Check if practiced today
	today := time.Now().Truncate(24 * time.Hour)
	todayCompleted := false
	if user.StreakUpdatedAt != nil {
		lastPractice := user.StreakUpdatedAt.Truncate(24 * time.Hour)
		todayCompleted = lastPractice.Equal(today)
	}

	// Check if streak is at risk (not practiced today and has active streak)
	streakAtRisk := !todayCompleted && user.StreakDays > 0

	return &StreakInfo{
		CurrentStreak:  user.StreakDays,
		LongestStreak:  user.StreakDays, // Would need separate tracking
		StreakFreezes:  user.StreakFreezeCount,
		TodayCompleted: todayCompleted,
		StreakAtRisk:   streakAtRisk,
		History:        history,
	}, nil
}

// UseStreakFreeze uses a streak freeze
func (s *Service) UseStreakFreeze(ctx context.Context, userID uuid.UUID) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.StreakFreezeCount <= 0 {
		return ErrNoStreakFreeze
	}

	// Check if streak is at risk
	today := time.Now().Truncate(24 * time.Hour)
	if user.StreakUpdatedAt != nil {
		lastPractice := user.StreakUpdatedAt.Truncate(24 * time.Hour)
		if lastPractice.Equal(today) {
			return ErrStreakNotAtRisk
		}
	}

	// Use freeze
	user.StreakFreezeCount--
	now := time.Now()
	user.StreakUpdatedAt = &now

	// Record freeze usage
	_ = s.progress.RecordStreakDay(ctx, userID, 0, 0, 0)

	return s.users.Update(ctx, user)
}

// GetHeartsInfo returns hearts information for a user
func (s *Service) GetHeartsInfo(ctx context.Context, userID uuid.UUID) (*HeartsInfo, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	info := &HeartsInfo{
		Hearts:         user.Hearts,
		MaxHearts:      5,
		IsPremium:      user.IsPremium,
		RefillCostGems: 350,
	}

	// Calculate next refill time if not full
	if user.Hearts < 5 && !user.IsPremium && user.HeartsUpdatedAt != nil {
		// Hearts refill 1 per 4 hours
		heartsMissing := 5 - user.Hearts
		refillDuration := time.Duration(heartsMissing) * 4 * time.Hour
		nextRefill := user.HeartsUpdatedAt.Add(refillDuration)
		info.NextRefillAt = &nextRefill
	}

	return info, nil
}

// RefillHearts refills hearts using gems
func (s *Service) RefillHearts(ctx context.Context, userID uuid.UUID) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.IsPremium {
		// Premium users have unlimited hearts
		return nil
	}

	if user.Hearts >= 5 {
		// Already full
		return nil
	}

	gemCost := 350
	if user.Gems < gemCost {
		return ErrInsufficientGems
	}

	// Deduct gems and refill hearts
	_ = s.users.UpdateGems(ctx, userID, user.Gems-gemCost)
	return s.users.UpdateHearts(ctx, userID, 5)
}

// GetMistakes returns common mistakes for practice
func (s *Service) GetMistakes(ctx context.Context, userID uuid.UUID, limit int) ([]store.UserMistake, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.progress.GetUserMistakes(ctx, userID, limit)
}

// GetLexemesForReview returns lexemes due for review
func (s *Service) GetLexemesForReview(ctx context.Context, userID uuid.UUID, limit int) ([]store.UserLexeme, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.progress.GetUserLexemes(ctx, userID, limit)
}
