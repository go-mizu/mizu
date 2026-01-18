package achievements

import (
	"context"
	"errors"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrAchievementNotFound = errors.New("achievement not found")
)

// Service handles achievement business logic
type Service struct {
	store        store.Store
	achievements store.AchievementStore
	users        store.UserStore
}

// NewService creates a new achievement service
func NewService(st store.Store) *Service {
	return &Service{
		store:        st,
		achievements: st.Achievements(),
		users:        st.Users(),
	}
}

// AchievementWithProgress represents an achievement with user's progress
type AchievementWithProgress struct {
	Achievement store.Achievement     `json:"achievement"`
	Progress    *store.UserAchievement `json:"progress,omitempty"`
	Percentage  float64               `json:"percentage"`
	NextLevel   int                   `json:"next_level"`
	NextTarget  int                   `json:"next_target"`
}

// GetAllAchievements returns all achievement definitions
func (s *Service) GetAllAchievements(ctx context.Context) ([]store.Achievement, error) {
	return s.achievements.GetAchievements(ctx)
}

// GetUserAchievements returns user's achievements with progress
func (s *Service) GetUserAchievements(ctx context.Context, userID uuid.UUID) ([]AchievementWithProgress, error) {
	achievements, err := s.achievements.GetAchievements(ctx)
	if err != nil {
		return nil, err
	}

	userAchievements, _ := s.achievements.GetUserAchievements(ctx, userID)

	// Create a map for quick lookup
	progressMap := make(map[string]*store.UserAchievement)
	for _, ua := range userAchievements {
		uaCopy := ua
		progressMap[ua.AchievementID] = &uaCopy
	}

	result := make([]AchievementWithProgress, len(achievements))
	for i, ach := range achievements {
		progress := progressMap[ach.ID]

		var percentage float64
		var nextLevel, nextTarget int

		if progress != nil {
			// Calculate progress percentage
			currentLevel := progress.Level
			if currentLevel < len(ach.Thresholds) {
				nextLevel = currentLevel + 1
				nextTarget = ach.Thresholds[currentLevel]
				if currentLevel > 0 {
					prevTarget := ach.Thresholds[currentLevel-1]
					percentage = float64(progress.Progress-prevTarget) / float64(nextTarget-prevTarget) * 100
				} else {
					percentage = float64(progress.Progress) / float64(nextTarget) * 100
				}
			} else {
				// Max level reached
				percentage = 100
				nextLevel = currentLevel
				nextTarget = ach.Thresholds[len(ach.Thresholds)-1]
			}
		} else if len(ach.Thresholds) > 0 {
			nextLevel = 1
			nextTarget = ach.Thresholds[0]
		}

		result[i] = AchievementWithProgress{
			Achievement: ach,
			Progress:    progress,
			Percentage:  percentage,
			NextLevel:   nextLevel,
			NextTarget:  nextTarget,
		}
	}

	return result, nil
}

// CheckAndUnlock checks if user progress unlocks an achievement level
func (s *Service) CheckAndUnlock(ctx context.Context, userID uuid.UUID, achievementID string, progressToAdd int) (*store.UserAchievement, error) {
	return s.achievements.CheckAndUnlock(ctx, userID, achievementID, progressToAdd)
}

// GetAchievementsByCategory returns achievements filtered by category
func (s *Service) GetAchievementsByCategory(ctx context.Context, category string) ([]store.Achievement, error) {
	achievements, err := s.achievements.GetAchievements(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []store.Achievement
	for _, ach := range achievements {
		if ach.Category == category {
			filtered = append(filtered, ach)
		}
	}

	return filtered, nil
}

// GetUnlockedCount returns the count of unlocked achievements for a user
func (s *Service) GetUnlockedCount(ctx context.Context, userID uuid.UUID) (int, int, error) {
	achievements, err := s.achievements.GetAchievements(ctx)
	if err != nil {
		return 0, 0, err
	}

	userAchievements, _ := s.achievements.GetUserAchievements(ctx, userID)

	unlocked := 0
	for _, ua := range userAchievements {
		if ua.Level > 0 {
			unlocked++
		}
	}

	return unlocked, len(achievements), nil
}

// Categories returns all achievement categories
func (s *Service) Categories() []string {
	return []string{
		"streak",
		"xp",
		"learning",
		"social",
		"league",
		"special",
	}
}
