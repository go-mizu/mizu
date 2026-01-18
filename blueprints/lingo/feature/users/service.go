package users

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUsernameExists   = errors.New("username already exists")
	ErrInsufficientGems = errors.New("insufficient gems")
)

// Service handles user business logic
type Service struct {
	store    store.Store
	users    store.UserStore
	progress store.ProgressStore
	social   store.SocialStore
}

// NewService creates a new user service
func NewService(st store.Store) *Service {
	return &Service{
		store:    st,
		users:    st.Users(),
		progress: st.Progress(),
		social:   st.Social(),
	}
}

// UserStats represents aggregated user statistics
type UserStats struct {
	TotalXP           int64   `json:"total_xp"`
	StreakDays        int     `json:"streak_days"`
	Gems              int     `json:"gems"`
	Hearts            int     `json:"hearts"`
	LessonsCompleted  int     `json:"lessons_completed"`
	WordsLearned      int     `json:"words_learned"`
	CoursesEnrolled   int     `json:"courses_enrolled"`
	AchievementsCount int     `json:"achievements_count"`
	LeagueRank        int     `json:"league_rank"`
	LeagueName        string  `json:"league_name"`
	DaysActive        int     `json:"days_active"`
	AvgXPPerDay       float64 `json:"avg_xp_per_day"`
}

// UpdateProfileInput represents profile update data
type UpdateProfileInput struct {
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	Username    *string `json:"username,omitempty"`
}

// UpdateSettingsInput represents settings update data
type UpdateSettingsInput struct {
	DailyGoalMinutes *int  `json:"daily_goal_minutes,omitempty"`
	IsPremium        *bool `json:"is_premium,omitempty"`
}

// GetByID gets a user by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*store.User, error) {
	return s.users.GetByID(ctx, id)
}

// GetByUsername gets a user by username
func (s *Service) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	return s.users.GetByUsername(ctx, username)
}

// UpdateProfile updates a user's profile
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*store.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check username uniqueness if changing
	if input.Username != nil && *input.Username != user.Username {
		existing, _ := s.users.GetByUsername(ctx, *input.Username)
		if existing != nil {
			return nil, ErrUsernameExists
		}
		user.Username = *input.Username
	}

	if input.DisplayName != nil {
		user.DisplayName = *input.DisplayName
	}
	if input.AvatarURL != nil {
		user.AvatarURL = *input.AvatarURL
	}
	if input.Bio != nil {
		user.Bio = *input.Bio
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateSettings updates a user's settings
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, input UpdateSettingsInput) (*store.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if input.DailyGoalMinutes != nil {
		user.DailyGoalMinutes = *input.DailyGoalMinutes
	}
	if input.IsPremium != nil {
		user.IsPremium = *input.IsPremium
		if user.IsPremium {
			expires := time.Now().Add(30 * 24 * time.Hour)
			user.PremiumExpiresAt = &expires
		}
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetStats gets aggregated statistics for a user
func (s *Service) GetStats(ctx context.Context, userID uuid.UUID) (*UserStats, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Get enrolled courses
	courses, _ := s.progress.GetUserCourses(ctx, userID)

	// Get streak history for days active
	streakHistory, _ := s.progress.GetStreakHistory(ctx, userID, 365)

	// Calculate average XP per day
	var avgXP float64
	if len(streakHistory) > 0 {
		var totalXP int64
		for _, day := range streakHistory {
			totalXP += int64(day.XPEarned)
		}
		avgXP = float64(totalXP) / float64(len(streakHistory))
	}

	// Get user's current league
	userLeague, _ := s.store.Gamification().GetUserLeague(ctx, userID)
	var leagueRank int
	var leagueName string
	if userLeague != nil {
		leagueRank = userLeague.Rank
		// Would need to join with leagues table to get name
		leagueName = "Bronze" // Placeholder
	}

	return &UserStats{
		TotalXP:          user.XPTotal,
		StreakDays:       user.StreakDays,
		Gems:             user.Gems,
		Hearts:           user.Hearts,
		CoursesEnrolled:  len(courses),
		DaysActive:       len(streakHistory),
		AvgXPPerDay:      avgXP,
		LeagueRank:       leagueRank,
		LeagueName:       leagueName,
		LessonsCompleted: 0, // Would need to query lesson_sessions
		WordsLearned:     0, // Would need to query user_lexemes
	}, nil
}

// SpendGems deducts gems from user account
func (s *Service) SpendGems(ctx context.Context, userID uuid.UUID, amount int) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.Gems < amount {
		return ErrInsufficientGems
	}

	return s.users.UpdateGems(ctx, userID, user.Gems-amount)
}

// AddGems adds gems to user account
func (s *Service) AddGems(ctx context.Context, userID uuid.UUID, amount int) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	return s.users.UpdateGems(ctx, userID, user.Gems+amount)
}

// SetActiveCourse sets the user's active course
func (s *Service) SetActiveCourse(ctx context.Context, userID, courseID uuid.UUID) (*store.User, error) {
	// Verify user exists
	_, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Set active course
	if err := s.users.SetActiveCourse(ctx, userID, courseID); err != nil {
		return nil, err
	}

	// Return updated user
	return s.users.GetByID(ctx, userID)
}
