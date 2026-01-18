package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// ProgressStore handles progress operations
type ProgressStore struct {
	db *sql.DB
}

// EnrollCourse enrolls a user in a course
func (s *ProgressStore) EnrollCourse(ctx context.Context, userID, courseID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_courses (user_id, course_id, started_at)
		VALUES (?, ?, ?)
	`, userID.String(), courseID.String(), time.Now())
	return err
}

// GetUserCourses returns all courses a user is enrolled in
func (s *ProgressStore) GetUserCourses(ctx context.Context, userID uuid.UUID) ([]store.UserCourse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, course_id, current_unit_id, current_lesson_id, xp_earned, crowns_earned, started_at, last_practiced_at
		FROM user_courses WHERE user_id = ?
	`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []store.UserCourse
	for rows.Next() {
		var uc store.UserCourse
		var uID, cID string
		var currentUnitID, currentLessonID sql.NullString

		if err := rows.Scan(&uID, &cID, &currentUnitID, &currentLessonID, &uc.XPEarned,
			&uc.CrownsEarned, &uc.StartedAt, &uc.LastPracticedAt); err != nil {
			return nil, err
		}

		uc.UserID, _ = uuid.Parse(uID)
		uc.CourseID, _ = uuid.Parse(cID)
		if currentUnitID.Valid {
			id, _ := uuid.Parse(currentUnitID.String)
			uc.CurrentUnitID = &id
		}
		if currentLessonID.Valid {
			id, _ := uuid.Parse(currentLessonID.String)
			uc.CurrentLessonID = &id
		}

		courses = append(courses, uc)
	}

	return courses, rows.Err()
}

// GetUserCourse returns a specific user course enrollment
func (s *ProgressStore) GetUserCourse(ctx context.Context, userID, courseID uuid.UUID) (*store.UserCourse, error) {
	var uc store.UserCourse
	var uID, cID string
	var currentUnitID, currentLessonID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, course_id, current_unit_id, current_lesson_id, xp_earned, crowns_earned, started_at, last_practiced_at
		FROM user_courses WHERE user_id = ? AND course_id = ?
	`, userID.String(), courseID.String()).Scan(&uID, &cID, &currentUnitID, &currentLessonID,
		&uc.XPEarned, &uc.CrownsEarned, &uc.StartedAt, &uc.LastPracticedAt)
	if err != nil {
		return nil, err
	}

	uc.UserID, _ = uuid.Parse(uID)
	uc.CourseID, _ = uuid.Parse(cID)
	if currentUnitID.Valid {
		id, _ := uuid.Parse(currentUnitID.String)
		uc.CurrentUnitID = &id
	}
	if currentLessonID.Valid {
		id, _ := uuid.Parse(currentLessonID.String)
		uc.CurrentLessonID = &id
	}

	return &uc, nil
}

// UpdateUserCourse updates a user's course progress
func (s *ProgressStore) UpdateUserCourse(ctx context.Context, uc *store.UserCourse) error {
	var currentUnitID, currentLessonID *string
	if uc.CurrentUnitID != nil {
		s := uc.CurrentUnitID.String()
		currentUnitID = &s
	}
	if uc.CurrentLessonID != nil {
		s := uc.CurrentLessonID.String()
		currentLessonID = &s
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE user_courses SET
			current_unit_id = ?, current_lesson_id = ?, xp_earned = ?, crowns_earned = ?, last_practiced_at = ?
		WHERE user_id = ? AND course_id = ?
	`, currentUnitID, currentLessonID, uc.XPEarned, uc.CrownsEarned, uc.LastPracticedAt,
		uc.UserID.String(), uc.CourseID.String())
	return err
}

// GetUserSkill returns a user's skill progress
func (s *ProgressStore) GetUserSkill(ctx context.Context, userID, skillID uuid.UUID) (*store.UserSkill, error) {
	var us store.UserSkill
	var uID, sID string
	var isLegendary int

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, skill_id, crown_level, is_legendary, strength, last_practiced_at, next_review_at
		FROM user_skills WHERE user_id = ? AND skill_id = ?
	`, userID.String(), skillID.String()).Scan(&uID, &sID, &us.CrownLevel, &isLegendary,
		&us.Strength, &us.LastPracticedAt, &us.NextReviewAt)
	if err != nil {
		return nil, err
	}

	us.UserID, _ = uuid.Parse(uID)
	us.SkillID, _ = uuid.Parse(sID)
	us.IsLegendary = isLegendary == 1

	return &us, nil
}

// UpdateUserSkill updates a user's skill progress
func (s *ProgressStore) UpdateUserSkill(ctx context.Context, us *store.UserSkill) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_skills (user_id, skill_id, crown_level, is_legendary, strength, last_practiced_at, next_review_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, skill_id) DO UPDATE SET
			crown_level = excluded.crown_level,
			is_legendary = excluded.is_legendary,
			strength = excluded.strength,
			last_practiced_at = excluded.last_practiced_at,
			next_review_at = excluded.next_review_at
	`, us.UserID.String(), us.SkillID.String(), us.CrownLevel, boolToInt(us.IsLegendary),
		us.Strength, us.LastPracticedAt, us.NextReviewAt)
	return err
}

// GetUserLexemes returns lexemes due for review
func (s *ProgressStore) GetUserLexemes(ctx context.Context, userID uuid.UUID, limit int) ([]store.UserLexeme, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, lexeme_id, strength, correct_count, incorrect_count, last_practiced_at, next_review_at, interval_days, ease_factor
		FROM user_lexemes
		WHERE user_id = ? AND (next_review_at IS NULL OR next_review_at <= ?)
		ORDER BY next_review_at LIMIT ?
	`, userID.String(), time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lexemes []store.UserLexeme
	for rows.Next() {
		var ul store.UserLexeme
		var uID, lID string

		if err := rows.Scan(&uID, &lID, &ul.Strength, &ul.CorrectCount, &ul.IncorrectCount,
			&ul.LastPracticedAt, &ul.NextReviewAt, &ul.IntervalDays, &ul.EaseFactor); err != nil {
			return nil, err
		}

		ul.UserID, _ = uuid.Parse(uID)
		ul.LexemeID, _ = uuid.Parse(lID)
		lexemes = append(lexemes, ul)
	}

	return lexemes, rows.Err()
}

// UpdateUserLexeme updates a user's lexeme progress
func (s *ProgressStore) UpdateUserLexeme(ctx context.Context, ul *store.UserLexeme) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_lexemes (user_id, lexeme_id, strength, correct_count, incorrect_count, last_practiced_at, next_review_at, interval_days, ease_factor)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, lexeme_id) DO UPDATE SET
			strength = excluded.strength,
			correct_count = excluded.correct_count,
			incorrect_count = excluded.incorrect_count,
			last_practiced_at = excluded.last_practiced_at,
			next_review_at = excluded.next_review_at,
			interval_days = excluded.interval_days,
			ease_factor = excluded.ease_factor
	`, ul.UserID.String(), ul.LexemeID.String(), ul.Strength, ul.CorrectCount, ul.IncorrectCount,
		ul.LastPracticedAt, ul.NextReviewAt, ul.IntervalDays, ul.EaseFactor)
	return err
}

// RecordMistake records a user mistake
func (s *ProgressStore) RecordMistake(ctx context.Context, mistake *store.UserMistake) error {
	var lexemeID *string
	if mistake.LexemeID != uuid.Nil {
		s := mistake.LexemeID.String()
		lexemeID = &s
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_mistakes (id, user_id, exercise_id, lexeme_id, user_answer, correct_answer, mistake_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, mistake.ID.String(), mistake.UserID.String(), mistake.ExerciseID.String(), lexemeID,
		mistake.UserAnswer, mistake.CorrectAnswer, mistake.MistakeType, mistake.CreatedAt)
	return err
}

// GetUserMistakes returns recent mistakes for a user
func (s *ProgressStore) GetUserMistakes(ctx context.Context, userID uuid.UUID, limit int) ([]store.UserMistake, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, exercise_id, lexeme_id, user_answer, correct_answer, mistake_type, created_at
		FROM user_mistakes WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
	`, userID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mistakes []store.UserMistake
	for rows.Next() {
		var m store.UserMistake
		var id, uID, eID string
		var lexemeID sql.NullString

		if err := rows.Scan(&id, &uID, &eID, &lexemeID, &m.UserAnswer, &m.CorrectAnswer, &m.MistakeType, &m.CreatedAt); err != nil {
			return nil, err
		}

		m.ID, _ = uuid.Parse(id)
		m.UserID, _ = uuid.Parse(uID)
		m.ExerciseID, _ = uuid.Parse(eID)
		if lexemeID.Valid {
			m.LexemeID, _ = uuid.Parse(lexemeID.String)
		}

		mistakes = append(mistakes, m)
	}

	return mistakes, rows.Err()
}

// StartLessonSession starts a new lesson session
func (s *ProgressStore) StartLessonSession(ctx context.Context, session *store.LessonSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO lesson_sessions (id, user_id, lesson_id, started_at)
		VALUES (?, ?, ?, ?)
	`, session.ID.String(), session.UserID.String(), session.LessonID.String(), session.StartedAt)
	return err
}

// CompleteLessonSession completes a lesson session
func (s *ProgressStore) CompleteLessonSession(ctx context.Context, session *store.LessonSession) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE lesson_sessions SET
			completed_at = ?, xp_earned = ?, mistakes_count = ?, hearts_lost = ?, is_perfect = ?
		WHERE user_id = ? AND lesson_id = ? AND completed_at IS NULL
	`, session.CompletedAt, session.XPEarned, session.MistakesCount, session.HeartsLost,
		boolToInt(session.IsPerfect), session.UserID.String(), session.LessonID.String())
	return err
}

// RecordXPEvent records an XP earning event
func (s *ProgressStore) RecordXPEvent(ctx context.Context, event *store.XPEvent) error {
	var sourceID *string
	if event.SourceID != uuid.Nil {
		s := event.SourceID.String()
		sourceID = &s
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO xp_events (id, user_id, amount, source, source_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID.String(), event.UserID.String(), event.Amount, event.Source, sourceID, event.CreatedAt)
	return err
}

// GetXPHistory returns XP history for a user
func (s *ProgressStore) GetXPHistory(ctx context.Context, userID uuid.UUID, days int) ([]store.XPEvent, error) {
	since := time.Now().AddDate(0, 0, -days)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, amount, source, source_id, created_at
		FROM xp_events WHERE user_id = ? AND created_at >= ? ORDER BY created_at DESC
	`, userID.String(), since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []store.XPEvent
	for rows.Next() {
		var e store.XPEvent
		var id, uID string
		var sourceID sql.NullString

		if err := rows.Scan(&id, &uID, &e.Amount, &e.Source, &sourceID, &e.CreatedAt); err != nil {
			return nil, err
		}

		e.ID, _ = uuid.Parse(id)
		e.UserID, _ = uuid.Parse(uID)
		if sourceID.Valid {
			e.SourceID, _ = uuid.Parse(sourceID.String)
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

// RecordStreakDay records a streak day
func (s *ProgressStore) RecordStreakDay(ctx context.Context, userID uuid.UUID, xp, lessons, seconds int) error {
	today := time.Now().Truncate(24 * time.Hour)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO streak_history (id, user_id, date, xp_earned, lessons_completed, time_spent_seconds)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET
			xp_earned = streak_history.xp_earned + excluded.xp_earned,
			lessons_completed = streak_history.lessons_completed + excluded.lessons_completed,
			time_spent_seconds = streak_history.time_spent_seconds + excluded.time_spent_seconds
	`, uuid.New().String(), userID.String(), today, xp, lessons, seconds)
	return err
}

// GetStreakHistory returns streak history for a user
func (s *ProgressStore) GetStreakHistory(ctx context.Context, userID uuid.UUID, days int) ([]store.StreakDay, error) {
	since := time.Now().AddDate(0, 0, -days)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, date, xp_earned, lessons_completed, time_spent_seconds, freeze_used
		FROM streak_history WHERE user_id = ? AND date >= ? ORDER BY date DESC
	`, userID.String(), since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []store.StreakDay
	for rows.Next() {
		var sd store.StreakDay
		var id, uID string
		var freezeUsed int

		if err := rows.Scan(&id, &uID, &sd.Date, &sd.XPEarned, &sd.LessonsCompleted, &sd.TimeSpentSeconds, &freezeUsed); err != nil {
			return nil, err
		}

		sd.ID, _ = uuid.Parse(id)
		sd.UserID, _ = uuid.Parse(uID)
		sd.FreezeUsed = freezeUsed == 1

		history = append(history, sd)
	}

	return history, rows.Err()
}
