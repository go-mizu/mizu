package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProgressStore implements store.ProgressStore
type ProgressStore struct {
	pool *pgxpool.Pool
}

// EnrollCourse enrolls a user in a course
func (s *ProgressStore) EnrollCourse(ctx context.Context, userID, courseID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_courses (user_id, course_id, started_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, course_id) DO NOTHING
	`, userID, courseID)
	if err != nil {
		return fmt.Errorf("enroll course: %w", err)
	}
	return nil
}

// GetUserCourses gets all courses for a user
func (s *ProgressStore) GetUserCourses(ctx context.Context, userID uuid.UUID) ([]store.UserCourse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, course_id, current_unit_id, current_lesson_id, xp_earned, crowns_earned, started_at, last_practiced_at
		FROM user_courses WHERE user_id = $1 ORDER BY last_practiced_at DESC NULLS LAST
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user courses: %w", err)
	}
	defer rows.Close()

	var courses []store.UserCourse
	for rows.Next() {
		var uc store.UserCourse
		if err := rows.Scan(&uc.UserID, &uc.CourseID, &uc.CurrentUnitID, &uc.CurrentLessonID, &uc.XPEarned, &uc.CrownsEarned, &uc.StartedAt, &uc.LastPracticedAt); err != nil {
			return nil, fmt.Errorf("scan user course: %w", err)
		}
		courses = append(courses, uc)
	}
	return courses, nil
}

// GetUserCourse gets a user's progress in a specific course
func (s *ProgressStore) GetUserCourse(ctx context.Context, userID, courseID uuid.UUID) (*store.UserCourse, error) {
	uc := &store.UserCourse{}
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, course_id, current_unit_id, current_lesson_id, xp_earned, crowns_earned, started_at, last_practiced_at
		FROM user_courses WHERE user_id = $1 AND course_id = $2
	`, userID, courseID).Scan(&uc.UserID, &uc.CourseID, &uc.CurrentUnitID, &uc.CurrentLessonID, &uc.XPEarned, &uc.CrownsEarned, &uc.StartedAt, &uc.LastPracticedAt)
	if err != nil {
		return nil, fmt.Errorf("query user course: %w", err)
	}
	return uc, nil
}

// UpdateUserCourse updates a user's course progress
func (s *ProgressStore) UpdateUserCourse(ctx context.Context, uc *store.UserCourse) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_courses SET current_unit_id = $3, current_lesson_id = $4, xp_earned = $5, crowns_earned = $6, last_practiced_at = NOW()
		WHERE user_id = $1 AND course_id = $2
	`, uc.UserID, uc.CourseID, uc.CurrentUnitID, uc.CurrentLessonID, uc.XPEarned, uc.CrownsEarned)
	if err != nil {
		return fmt.Errorf("update user course: %w", err)
	}
	return nil
}

// GetUserSkill gets a user's progress on a skill
func (s *ProgressStore) GetUserSkill(ctx context.Context, userID, skillID uuid.UUID) (*store.UserSkill, error) {
	us := &store.UserSkill{}
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, skill_id, crown_level, is_legendary, strength, last_practiced_at, next_review_at
		FROM user_skills WHERE user_id = $1 AND skill_id = $2
	`, userID, skillID).Scan(&us.UserID, &us.SkillID, &us.CrownLevel, &us.IsLegendary, &us.Strength, &us.LastPracticedAt, &us.NextReviewAt)
	if err != nil {
		return nil, fmt.Errorf("query user skill: %w", err)
	}
	return us, nil
}

// UpdateUserSkill updates a user's skill progress
func (s *ProgressStore) UpdateUserSkill(ctx context.Context, us *store.UserSkill) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_skills (user_id, skill_id, crown_level, is_legendary, strength, last_practiced_at, next_review_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, skill_id) DO UPDATE SET crown_level = $3, is_legendary = $4, strength = $5, last_practiced_at = $6, next_review_at = $7
	`, us.UserID, us.SkillID, us.CrownLevel, us.IsLegendary, us.Strength, us.LastPracticedAt, us.NextReviewAt)
	if err != nil {
		return fmt.Errorf("update user skill: %w", err)
	}
	return nil
}

// GetUserLexemes gets a user's vocabulary progress
func (s *ProgressStore) GetUserLexemes(ctx context.Context, userID uuid.UUID, limit int) ([]store.UserLexeme, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, lexeme_id, strength, correct_count, incorrect_count, last_practiced_at, next_review_at, interval_days, ease_factor
		FROM user_lexemes WHERE user_id = $1 ORDER BY next_review_at LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query user lexemes: %w", err)
	}
	defer rows.Close()

	var lexemes []store.UserLexeme
	for rows.Next() {
		var ul store.UserLexeme
		if err := rows.Scan(&ul.UserID, &ul.LexemeID, &ul.Strength, &ul.CorrectCount, &ul.IncorrectCount, &ul.LastPracticedAt, &ul.NextReviewAt, &ul.IntervalDays, &ul.EaseFactor); err != nil {
			return nil, fmt.Errorf("scan user lexeme: %w", err)
		}
		lexemes = append(lexemes, ul)
	}
	return lexemes, nil
}

// UpdateUserLexeme updates a user's lexeme progress (spaced repetition)
func (s *ProgressStore) UpdateUserLexeme(ctx context.Context, ul *store.UserLexeme) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_lexemes (user_id, lexeme_id, strength, correct_count, incorrect_count, last_practiced_at, next_review_at, interval_days, ease_factor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, lexeme_id) DO UPDATE SET strength = $3, correct_count = $4, incorrect_count = $5, last_practiced_at = $6, next_review_at = $7, interval_days = $8, ease_factor = $9
	`, ul.UserID, ul.LexemeID, ul.Strength, ul.CorrectCount, ul.IncorrectCount, ul.LastPracticedAt, ul.NextReviewAt, ul.IntervalDays, ul.EaseFactor)
	if err != nil {
		return fmt.Errorf("update user lexeme: %w", err)
	}
	return nil
}

// RecordMistake records a user mistake
func (s *ProgressStore) RecordMistake(ctx context.Context, mistake *store.UserMistake) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_mistakes (id, user_id, exercise_id, lexeme_id, user_answer, correct_answer, mistake_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, mistake.ID, mistake.UserID, mistake.ExerciseID, mistake.LexemeID, mistake.UserAnswer, mistake.CorrectAnswer, mistake.MistakeType)
	if err != nil {
		return fmt.Errorf("insert mistake: %w", err)
	}
	return nil
}

// GetUserMistakes gets a user's recent mistakes
func (s *ProgressStore) GetUserMistakes(ctx context.Context, userID uuid.UUID, limit int) ([]store.UserMistake, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, exercise_id, lexeme_id, user_answer, correct_answer, mistake_type, created_at
		FROM user_mistakes WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query mistakes: %w", err)
	}
	defer rows.Close()

	var mistakes []store.UserMistake
	for rows.Next() {
		var m store.UserMistake
		if err := rows.Scan(&m.ID, &m.UserID, &m.ExerciseID, &m.LexemeID, &m.UserAnswer, &m.CorrectAnswer, &m.MistakeType, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mistake: %w", err)
		}
		mistakes = append(mistakes, m)
	}
	return mistakes, nil
}

// StartLessonSession starts a new lesson session
func (s *ProgressStore) StartLessonSession(ctx context.Context, session *store.LessonSession) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO lesson_sessions (id, user_id, lesson_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, session.ID, session.UserID, session.LessonID)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	return nil
}

// CompleteLessonSession completes a lesson session
func (s *ProgressStore) CompleteLessonSession(ctx context.Context, session *store.LessonSession) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE lesson_sessions SET completed_at = NOW(), xp_earned = $2, mistakes_count = $3, hearts_lost = $4, is_perfect = $5
		WHERE id = $1
	`, session.ID, session.XPEarned, session.MistakesCount, session.HeartsLost, session.IsPerfect)
	if err != nil {
		return fmt.Errorf("complete session: %w", err)
	}
	return nil
}

// RecordXPEvent records an XP earning event
func (s *ProgressStore) RecordXPEvent(ctx context.Context, event *store.XPEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO xp_events (id, user_id, amount, source, source_id, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, event.ID, event.UserID, event.Amount, event.Source, event.SourceID)
	if err != nil {
		return fmt.Errorf("record xp event: %w", err)
	}
	return nil
}

// GetXPHistory gets a user's XP history
func (s *ProgressStore) GetXPHistory(ctx context.Context, userID uuid.UUID, days int) ([]store.XPEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, amount, source, source_id, created_at
		FROM xp_events WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day' * $2
		ORDER BY created_at DESC
	`, userID, days)
	if err != nil {
		return nil, fmt.Errorf("query xp history: %w", err)
	}
	defer rows.Close()

	var events []store.XPEvent
	for rows.Next() {
		var e store.XPEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.Amount, &e.Source, &e.SourceID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan xp event: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}

// RecordStreakDay records a day in the user's streak
func (s *ProgressStore) RecordStreakDay(ctx context.Context, userID uuid.UUID, xp, lessons, seconds int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO streak_history (id, user_id, date, xp_earned, lessons_completed, time_spent_seconds)
		VALUES ($1, $2, CURRENT_DATE, $3, $4, $5)
		ON CONFLICT (user_id, date) DO UPDATE SET xp_earned = streak_history.xp_earned + $3, lessons_completed = streak_history.lessons_completed + $4, time_spent_seconds = streak_history.time_spent_seconds + $5
	`, uuid.New(), userID, xp, lessons, seconds)
	if err != nil {
		return fmt.Errorf("record streak day: %w", err)
	}
	return nil
}

// GetStreakHistory gets a user's streak history
func (s *ProgressStore) GetStreakHistory(ctx context.Context, userID uuid.UUID, days int) ([]store.StreakDay, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, date, xp_earned, lessons_completed, time_spent_seconds, freeze_used
		FROM streak_history WHERE user_id = $1 ORDER BY date DESC LIMIT $2
	`, userID, days)
	if err != nil {
		return nil, fmt.Errorf("query streak history: %w", err)
	}
	defer rows.Close()

	var history []store.StreakDay
	for rows.Next() {
		var sd store.StreakDay
		var date time.Time
		if err := rows.Scan(&sd.ID, &sd.UserID, &date, &sd.XPEarned, &sd.LessonsCompleted, &sd.TimeSpentSeconds, &sd.FreezeUsed); err != nil {
			return nil, fmt.Errorf("scan streak day: %w", err)
		}
		sd.Date = date
		history = append(history, sd)
	}
	return history, nil
}
