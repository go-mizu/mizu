package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// UserStore handles user operations
type UserStore struct {
	db *sql.DB
}

// Create creates a new user
func (s *UserStore) Create(ctx context.Context, user *store.User) error {
	var activeCourseID sql.NullString
	if user.ActiveCourseID != nil {
		activeCourseID = sql.NullString{String: user.ActiveCourseID.String(), Valid: true}
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, username, display_name, avatar_url, bio, encrypted_password,
			xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at,
			streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes,
			active_course_id, native_language_id, created_at, last_active_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID.String(), user.Email, user.Username, user.DisplayName, user.AvatarURL, user.Bio,
		user.EncryptedPassword, user.XPTotal, user.Gems, user.Hearts, user.HeartsUpdatedAt,
		user.StreakDays, user.StreakUpdatedAt, user.StreakFreezeCount, boolToInt(user.IsPremium),
		user.PremiumExpiresAt, user.DailyGoalMinutes, activeCourseID, nullString(user.NativeLanguageID),
		user.CreatedAt, user.LastActiveAt)
	return err
}

// GetByID retrieves a user by ID
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*store.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, avatar_url, bio, encrypted_password,
			xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at,
			streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes,
			active_course_id, native_language_id, created_at, last_active_at
		FROM users WHERE id = ?
	`, id.String()))
}

// GetByEmail retrieves a user by email
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, avatar_url, bio, encrypted_password,
			xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at,
			streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes,
			active_course_id, native_language_id, created_at, last_active_at
		FROM users WHERE email = ?
	`, email))
}

// GetByUsername retrieves a user by username
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, avatar_url, bio, encrypted_password,
			xp_total, gems, hearts, hearts_updated_at, streak_days, streak_updated_at,
			streak_freeze_count, is_premium, premium_expires_at, daily_goal_minutes,
			active_course_id, native_language_id, created_at, last_active_at
		FROM users WHERE username = ?
	`, username))
}

// Update updates a user
func (s *UserStore) Update(ctx context.Context, user *store.User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			email = ?, username = ?, display_name = ?, avatar_url = ?, bio = ?,
			xp_total = ?, gems = ?, hearts = ?, hearts_updated_at = ?,
			streak_days = ?, streak_updated_at = ?, streak_freeze_count = ?,
			is_premium = ?, premium_expires_at = ?, daily_goal_minutes = ?, last_active_at = ?
		WHERE id = ?
	`, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.Bio,
		user.XPTotal, user.Gems, user.Hearts, user.HeartsUpdatedAt,
		user.StreakDays, user.StreakUpdatedAt, user.StreakFreezeCount,
		boolToInt(user.IsPremium), user.PremiumExpiresAt, user.DailyGoalMinutes, user.LastActiveAt,
		user.ID.String())
	return err
}

// UpdateXP updates a user's XP
func (s *UserStore) UpdateXP(ctx context.Context, userID uuid.UUID, amount int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET xp_total = xp_total + ? WHERE id = ?
	`, amount, userID.String())
	return err
}

// UpdateStreak updates a user's streak
func (s *UserStore) UpdateStreak(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	// Get current streak info
	var streakDays int
	var streakUpdatedAt *time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT streak_days, streak_updated_at FROM users WHERE id = ?
	`, userID.String()).Scan(&streakDays, &streakUpdatedAt)
	if err != nil {
		return err
	}

	if streakUpdatedAt != nil {
		lastUpdate := streakUpdatedAt.Truncate(24 * time.Hour)
		if lastUpdate.Equal(today) {
			// Already updated today
			return nil
		}
		yesterday := today.AddDate(0, 0, -1)
		if lastUpdate.Equal(yesterday) {
			// Streak continues
			streakDays++
		} else {
			// Streak broken
			streakDays = 1
		}
	} else {
		streakDays = 1
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users SET streak_days = ?, streak_updated_at = ? WHERE id = ?
	`, streakDays, today, userID.String())
	return err
}

// UpdateHearts updates a user's hearts
func (s *UserStore) UpdateHearts(ctx context.Context, userID uuid.UUID, hearts int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET hearts = ?, hearts_updated_at = ? WHERE id = ?
	`, hearts, time.Now(), userID.String())
	return err
}

// UpdateGems updates a user's gems
func (s *UserStore) UpdateGems(ctx context.Context, userID uuid.UUID, gems int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET gems = ? WHERE id = ?
	`, gems, userID.String())
	return err
}

// SetActiveCourse sets the user's active course
func (s *UserStore) SetActiveCourse(ctx context.Context, userID, courseID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET active_course_id = ? WHERE id = ?
	`, courseID.String(), userID.String())
	return err
}

func (s *UserStore) scanUser(row *sql.Row) (*store.User, error) {
	var user store.User
	var id string
	var isPremium int
	var activeCourseID, nativeLanguageID sql.NullString

	err := row.Scan(&id, &user.Email, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio,
		&user.EncryptedPassword, &user.XPTotal, &user.Gems, &user.Hearts, &user.HeartsUpdatedAt,
		&user.StreakDays, &user.StreakUpdatedAt, &user.StreakFreezeCount, &isPremium,
		&user.PremiumExpiresAt, &user.DailyGoalMinutes, &activeCourseID, &nativeLanguageID,
		&user.CreatedAt, &user.LastActiveAt)
	if err != nil {
		return nil, err
	}

	user.ID, _ = uuid.Parse(id)
	user.IsPremium = isPremium == 1
	if activeCourseID.Valid {
		courseID, _ := uuid.Parse(activeCourseID.String)
		user.ActiveCourseID = &courseID
	}
	if nativeLanguageID.Valid {
		user.NativeLanguageID = nativeLanguageID.String
	}

	return &user, nil
}
