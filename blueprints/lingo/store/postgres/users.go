package postgres

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserStore implements store.UserStore
type UserStore struct {
	pool *pgxpool.Pool
}

// Create creates a new user
func (s *UserStore) Create(ctx context.Context, user *store.User) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, username, display_name, avatar_url, bio, encrypted_password, xp_total, gems, hearts, streak_days, daily_goal_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.Bio, user.EncryptedPassword, user.XPTotal, user.Gems, user.Hearts, user.StreakDays, user.DailyGoalMinutes)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByID gets a user by ID
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*store.User, error) {
	user := &store.User{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, COALESCE(display_name, ''), COALESCE(avatar_url, ''), COALESCE(bio, ''), encrypted_password, xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at, streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes, active_course_id, native_language_id, created_at, last_active_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.EncryptedPassword, &user.XPTotal, &user.Gems, &user.Hearts, &user.HeartsUpdatedAt, &user.StreakDays, &user.StreakUpdatedAt, &user.StreakFreezeCount, &user.IsPremium, &user.PremiumExpiresAt, &user.DailyGoalMinutes, &user.ActiveCourseID, &user.NativeLanguageID, &user.CreatedAt, &user.LastActiveAt)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

// GetByEmail gets a user by email
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	user := &store.User{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, COALESCE(display_name, ''), COALESCE(avatar_url, ''), COALESCE(bio, ''), encrypted_password, xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at, streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes, active_course_id, native_language_id, created_at, last_active_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.EncryptedPassword, &user.XPTotal, &user.Gems, &user.Hearts, &user.HeartsUpdatedAt, &user.StreakDays, &user.StreakUpdatedAt, &user.StreakFreezeCount, &user.IsPremium, &user.PremiumExpiresAt, &user.DailyGoalMinutes, &user.ActiveCourseID, &user.NativeLanguageID, &user.CreatedAt, &user.LastActiveAt)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

// GetByUsername gets a user by username
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	user := &store.User{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, COALESCE(display_name, ''), COALESCE(avatar_url, ''), COALESCE(bio, ''), encrypted_password, xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at, streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes, active_course_id, native_language_id, created_at, last_active_at
		FROM users WHERE username = $1
	`, username).Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.EncryptedPassword, &user.XPTotal, &user.Gems, &user.Hearts, &user.HeartsUpdatedAt, &user.StreakDays, &user.StreakUpdatedAt, &user.StreakFreezeCount, &user.IsPremium, &user.PremiumExpiresAt, &user.DailyGoalMinutes, &user.ActiveCourseID, &user.NativeLanguageID, &user.CreatedAt, &user.LastActiveAt)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

// Update updates a user
func (s *UserStore) Update(ctx context.Context, user *store.User) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET display_name = $2, avatar_url = $3, bio = $4, daily_goal_minutes = $5, last_active_at = NOW()
		WHERE id = $1
	`, user.ID, user.DisplayName, user.AvatarURL, user.Bio, user.DailyGoalMinutes)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// UpdateXP updates user XP
func (s *UserStore) UpdateXP(ctx context.Context, userID uuid.UUID, amount int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET xp_total = xp_total + $2, last_active_at = NOW()
		WHERE id = $1
	`, userID, amount)
	if err != nil {
		return fmt.Errorf("update xp: %w", err)
	}
	return nil
}

// UpdateStreak updates user streak
func (s *UserStore) UpdateStreak(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET
			streak_days = CASE
				WHEN streak_updated_at = CURRENT_DATE THEN streak_days
				WHEN streak_updated_at = CURRENT_DATE - 1 THEN streak_days + 1
				ELSE 1
			END,
			streak_updated_at = CURRENT_DATE,
			last_active_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("update streak: %w", err)
	}
	return nil
}

// UpdateHearts updates user hearts
func (s *UserStore) UpdateHearts(ctx context.Context, userID uuid.UUID, hearts int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET hearts = $2, hearts_updated_at = NOW()
		WHERE id = $1
	`, userID, hearts)
	if err != nil {
		return fmt.Errorf("update hearts: %w", err)
	}
	return nil
}

// UpdateGems updates user gems
func (s *UserStore) UpdateGems(ctx context.Context, userID uuid.UUID, gems int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET gems = $2
		WHERE id = $1
	`, userID, gems)
	if err != nil {
		return fmt.Errorf("update gems: %w", err)
	}
	return nil
}

// SetActiveCourse sets the user's active course
func (s *UserStore) SetActiveCourse(ctx context.Context, userID, courseID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET active_course_id = $2
		WHERE id = $1
	`, userID, courseID)
	if err != nil {
		return fmt.Errorf("set active course: %w", err)
	}
	return nil
}
