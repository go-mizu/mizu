package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AchievementStore implements store.AchievementStore
type AchievementStore struct {
	pool *pgxpool.Pool
}

// GetAchievements gets all achievements
func (s *AchievementStore) GetAchievements(ctx context.Context) ([]store.Achievement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, category, icon_url, max_level, thresholds
		FROM achievements ORDER BY category, name
	`)
	if err != nil {
		return nil, fmt.Errorf("query achievements: %w", err)
	}
	defer rows.Close()

	var achievements []store.Achievement
	for rows.Next() {
		var a store.Achievement
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Category, &a.IconURL, &a.MaxLevel, &a.Thresholds); err != nil {
			return nil, fmt.Errorf("scan achievement: %w", err)
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

// GetUserAchievements gets a user's achievements
func (s *AchievementStore) GetUserAchievements(ctx context.Context, userID uuid.UUID) ([]store.UserAchievement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, achievement_id, level, progress, unlocked_at
		FROM user_achievements WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user achievements: %w", err)
	}
	defer rows.Close()

	var achievements []store.UserAchievement
	for rows.Next() {
		var ua store.UserAchievement
		if err := rows.Scan(&ua.UserID, &ua.AchievementID, &ua.Level, &ua.Progress, &ua.UnlockedAt); err != nil {
			return nil, fmt.Errorf("scan user achievement: %w", err)
		}
		achievements = append(achievements, ua)
	}
	return achievements, nil
}

// UpdateUserAchievement updates a user's achievement progress
func (s *AchievementStore) UpdateUserAchievement(ctx context.Context, ua *store.UserAchievement) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_achievements (user_id, achievement_id, level, progress, unlocked_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, achievement_id) DO UPDATE SET level = $3, progress = $4, unlocked_at = COALESCE(user_achievements.unlocked_at, $5)
	`, ua.UserID, ua.AchievementID, ua.Level, ua.Progress, ua.UnlockedAt)
	if err != nil {
		return fmt.Errorf("update user achievement: %w", err)
	}
	return nil
}

// CheckAndUnlock checks if an achievement should be unlocked and updates it
func (s *AchievementStore) CheckAndUnlock(ctx context.Context, userID uuid.UUID, achievementID string, progress int) (*store.UserAchievement, error) {
	// Get achievement definition
	var a store.Achievement
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, description, category, icon_url, max_level, thresholds
		FROM achievements WHERE id = $1
	`, achievementID).Scan(&a.ID, &a.Name, &a.Description, &a.Category, &a.IconURL, &a.MaxLevel, &a.Thresholds)
	if err != nil {
		return nil, fmt.Errorf("get achievement: %w", err)
	}

	// Get current user progress
	ua := &store.UserAchievement{
		UserID:        userID,
		AchievementID: achievementID,
		Progress:      progress,
	}

	_ = s.pool.QueryRow(ctx, `
		SELECT level, progress, unlocked_at FROM user_achievements WHERE user_id = $1 AND achievement_id = $2
	`, userID, achievementID).Scan(&ua.Level, &ua.Progress, &ua.UnlockedAt)

	// Update progress
	ua.Progress = progress

	// Check if we should level up
	newLevel := ua.Level
	for i, threshold := range a.Thresholds {
		if progress >= threshold && i+1 > newLevel {
			newLevel = i + 1
		}
	}

	if newLevel > ua.Level {
		ua.Level = newLevel
		now := time.Now()
		ua.UnlockedAt = &now
	}

	// Save
	if err := s.UpdateUserAchievement(ctx, ua); err != nil {
		return nil, err
	}

	return ua, nil
}
