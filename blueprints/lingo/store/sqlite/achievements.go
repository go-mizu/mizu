package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// AchievementStore handles achievement operations
type AchievementStore struct {
	db *sql.DB
}

// GetAchievements returns all achievement definitions
func (s *AchievementStore) GetAchievements(ctx context.Context) ([]store.Achievement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, category, icon_url, max_level, thresholds
		FROM achievements ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []store.Achievement
	for rows.Next() {
		var a store.Achievement
		var iconURL sql.NullString
		var thresholdsJSON string

		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Category, &iconURL, &a.MaxLevel, &thresholdsJSON); err != nil {
			return nil, err
		}

		if iconURL.Valid {
			a.IconURL = iconURL.String
		}

		_ = json.Unmarshal([]byte(thresholdsJSON), &a.Thresholds)

		achievements = append(achievements, a)
	}

	return achievements, rows.Err()
}

// GetUserAchievements returns a user's achievement progress
func (s *AchievementStore) GetUserAchievements(ctx context.Context, userID uuid.UUID) ([]store.UserAchievement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, achievement_id, level, progress, unlocked_at
		FROM user_achievements WHERE user_id = ?
	`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []store.UserAchievement
	for rows.Next() {
		var ua store.UserAchievement
		var uID string

		if err := rows.Scan(&uID, &ua.AchievementID, &ua.Level, &ua.Progress, &ua.UnlockedAt); err != nil {
			return nil, err
		}

		ua.UserID, _ = uuid.Parse(uID)
		achievements = append(achievements, ua)
	}

	return achievements, rows.Err()
}

// UpdateUserAchievement updates a user's achievement progress
func (s *AchievementStore) UpdateUserAchievement(ctx context.Context, ua *store.UserAchievement) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_achievements (user_id, achievement_id, level, progress, unlocked_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, achievement_id) DO UPDATE SET
			level = excluded.level,
			progress = excluded.progress,
			unlocked_at = COALESCE(user_achievements.unlocked_at, excluded.unlocked_at)
	`, ua.UserID.String(), ua.AchievementID, ua.Level, ua.Progress, ua.UnlockedAt)
	return err
}

// CheckAndUnlock checks if progress unlocks a new achievement level
func (s *AchievementStore) CheckAndUnlock(ctx context.Context, userID uuid.UUID, achievementID string, progressToAdd int) (*store.UserAchievement, error) {
	// Get achievement definition
	var ach store.Achievement
	var thresholdsJSON string
	var iconURL sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, category, icon_url, max_level, thresholds
		FROM achievements WHERE id = ?
	`, achievementID).Scan(&ach.ID, &ach.Name, &ach.Description, &ach.Category, &iconURL, &ach.MaxLevel, &thresholdsJSON)
	if err != nil {
		return nil, err
	}

	if iconURL.Valid {
		ach.IconURL = iconURL.String
	}
	_ = json.Unmarshal([]byte(thresholdsJSON), &ach.Thresholds)

	// Get or create user achievement
	var ua store.UserAchievement
	var uID string

	err = s.db.QueryRowContext(ctx, `
		SELECT user_id, achievement_id, level, progress, unlocked_at
		FROM user_achievements WHERE user_id = ? AND achievement_id = ?
	`, userID.String(), achievementID).Scan(&uID, &ua.AchievementID, &ua.Level, &ua.Progress, &ua.UnlockedAt)

	if err == sql.ErrNoRows {
		ua = store.UserAchievement{
			UserID:        userID,
			AchievementID: achievementID,
			Level:         0,
			Progress:      0,
		}
	} else if err != nil {
		return nil, err
	} else {
		ua.UserID, _ = uuid.Parse(uID)
	}

	// Add progress
	ua.Progress += progressToAdd

	// Check if new level unlocked
	unlocked := false
	for ua.Level < len(ach.Thresholds) && ua.Progress >= ach.Thresholds[ua.Level] {
		ua.Level++
		unlocked = true
	}

	if unlocked && ua.UnlockedAt == nil {
		now := time.Now()
		ua.UnlockedAt = &now
	}

	// Save
	if err := s.UpdateUserAchievement(ctx, &ua); err != nil {
		return nil, err
	}

	if unlocked {
		return &ua, nil
	}

	return nil, nil
}
