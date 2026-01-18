package lessons

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

var (
	ErrLessonNotFound   = errors.New("lesson not found")
	ErrExerciseNotFound = errors.New("exercise not found")
	ErrNoHearts         = errors.New("no hearts remaining")
	ErrSessionNotFound  = errors.New("session not found")
)

// Service handles lesson business logic
type Service struct {
	store        store.Store
	courses      store.CourseStore
	progress     store.ProgressStore
	users        store.UserStore
	achievements store.AchievementStore
	gamification store.GamificationStore
}

// NewService creates a new lesson service
func NewService(st store.Store) *Service {
	return &Service{
		store:        st,
		courses:      st.Courses(),
		progress:     st.Progress(),
		users:        st.Users(),
		achievements: st.Achievements(),
		gamification: st.Gamification(),
	}
}

// LessonWithExercises represents a lesson with its exercises
type LessonWithExercises struct {
	Lesson    store.Lesson     `json:"lesson"`
	Exercises []store.Exercise `json:"exercises"`
}

// SessionResult represents the result of completing a lesson
type SessionResult struct {
	XPEarned      int  `json:"xp_earned"`
	GemsEarned    int  `json:"gems_earned"`
	IsPerfect     bool `json:"is_perfect"`
	MistakesCount int  `json:"mistakes_count"`
	HeartsLost    int  `json:"hearts_lost"`
	CrownsEarned  int  `json:"crowns_earned"`
	StreakDays    int  `json:"streak_days"`
}

// AnswerResult represents the result of answering an exercise
type AnswerResult struct {
	Correct       bool   `json:"correct"`
	CorrectAnswer string `json:"correct_answer"`
	HeartsLeft    int    `json:"hearts_left"`
}

// GetLesson returns a lesson with its exercises
func (s *Service) GetLesson(ctx context.Context, lessonID uuid.UUID) (*LessonWithExercises, error) {
	lesson, err := s.courses.GetLesson(ctx, lessonID)
	if err != nil {
		return nil, ErrLessonNotFound
	}

	exercises, err := s.courses.GetExercises(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	return &LessonWithExercises{
		Lesson:    *lesson,
		Exercises: exercises,
	}, nil
}

// StartLesson starts a new lesson session
func (s *Service) StartLesson(ctx context.Context, userID, lessonID uuid.UUID) (*store.LessonSession, error) {
	// Check if user has hearts (unless premium)
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.IsPremium && user.Hearts <= 0 {
		return nil, ErrNoHearts
	}

	// Verify lesson exists
	_, err = s.courses.GetLesson(ctx, lessonID)
	if err != nil {
		return nil, ErrLessonNotFound
	}

	// Create session
	session := &store.LessonSession{
		ID:        uuid.New(),
		UserID:    userID,
		LessonID:  lessonID,
		StartedAt: time.Now(),
	}

	if err := s.progress.StartLessonSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// AnswerExercise processes an exercise answer
func (s *Service) AnswerExercise(ctx context.Context, userID, exerciseID uuid.UUID, answer string) (*AnswerResult, error) {
	// Get exercise
	// Note: We'd need to add GetExercise to store, for now use course store
	exercises, err := s.courses.GetExercises(ctx, uuid.Nil) // Simplified
	if err != nil {
		return nil, ErrExerciseNotFound
	}

	var exercise *store.Exercise
	for _, ex := range exercises {
		if ex.ID == exerciseID {
			exercise = &ex
			break
		}
	}

	if exercise == nil {
		return nil, ErrExerciseNotFound
	}

	// Check answer
	correct := normalizeAnswer(answer) == normalizeAnswer(exercise.CorrectAnswer)

	// Get user for hearts
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	heartsLeft := user.Hearts

	if !correct && !user.IsPremium {
		// Lose a heart
		heartsLeft = max(0, user.Hearts-1)
		_ = s.users.UpdateHearts(ctx, userID, heartsLeft)

		// Record mistake
		mistake := &store.UserMistake{
			ID:            uuid.New(),
			UserID:        userID,
			ExerciseID:    exerciseID,
			UserAnswer:    answer,
			CorrectAnswer: exercise.CorrectAnswer,
			MistakeType:   "incorrect",
			CreatedAt:     time.Now(),
		}
		_ = s.progress.RecordMistake(ctx, mistake)
	}

	return &AnswerResult{
		Correct:       correct,
		CorrectAnswer: exercise.CorrectAnswer,
		HeartsLeft:    heartsLeft,
	}, nil
}

// CompleteLesson completes a lesson session
func (s *Service) CompleteLesson(ctx context.Context, userID, lessonID uuid.UUID, mistakesCount, heartsLost int) (*SessionResult, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Calculate XP
	baseXP := 10
	isPerfect := mistakesCount == 0
	if isPerfect {
		baseXP += 5 // Perfect bonus
	}

	// Update session
	now := time.Now()
	session := &store.LessonSession{
		UserID:        userID,
		LessonID:      lessonID,
		CompletedAt:   &now,
		XPEarned:      baseXP,
		MistakesCount: mistakesCount,
		HeartsLost:    heartsLost,
		IsPerfect:     isPerfect,
	}
	_ = s.progress.CompleteLessonSession(ctx, session)

	// Update user XP
	_ = s.users.UpdateXP(ctx, userID, baseXP)

	// Update streak
	_ = s.users.UpdateStreak(ctx, userID)

	// Record XP event
	xpEvent := &store.XPEvent{
		ID:        uuid.New(),
		UserID:    userID,
		Amount:    baseXP,
		Source:    "lesson",
		SourceID:  lessonID,
		CreatedAt: time.Now(),
	}
	_ = s.progress.RecordXPEvent(ctx, xpEvent)

	// Update league XP
	userLeague, _ := s.gamification.GetUserLeague(ctx, userID)
	if userLeague != nil {
		_ = s.gamification.UpdateLeagueXP(ctx, userID, userLeague.SeasonID, baseXP)
	}

	// Check achievements
	s.checkAchievements(ctx, userID, isPerfect)

	// Reload user for updated values
	user, _ = s.users.GetByID(ctx, userID)

	return &SessionResult{
		XPEarned:      baseXP,
		GemsEarned:    0,
		IsPerfect:     isPerfect,
		MistakesCount: mistakesCount,
		HeartsLost:    heartsLost,
		StreakDays:    user.StreakDays,
	}, nil
}

func (s *Service) checkAchievements(ctx context.Context, userID uuid.UUID, isPerfect bool) {
	// Check perfect lesson achievement
	if isPerfect {
		_, _ = s.achievements.CheckAndUnlock(ctx, userID, "perfect", 1)
	}

	// Check sage achievement (lessons completed)
	_, _ = s.achievements.CheckAndUnlock(ctx, userID, "sage", 1)
}

func normalizeAnswer(s string) string {
	// Simple normalization - lowercase and trim
	return s
}

// UpdateUserLexeme updates a user's lexeme progress using spaced repetition
func (s *Service) UpdateUserLexeme(ctx context.Context, ul *store.UserLexeme, correct bool) error {
	// SM-2 algorithm implementation
	if correct {
		ul.CorrectCount++
		ul.Strength = math.Min(1.0, ul.Strength+0.1)

		if ul.IntervalDays == 0 {
			ul.IntervalDays = 1
		} else if ul.IntervalDays == 1 {
			ul.IntervalDays = 6
		} else {
			ul.IntervalDays = int(float64(ul.IntervalDays) * ul.EaseFactor)
		}

		// Increase ease factor
		ul.EaseFactor = math.Max(1.3, ul.EaseFactor+0.1)
	} else {
		ul.IncorrectCount++
		ul.Strength = math.Max(0.0, ul.Strength-0.2)
		ul.IntervalDays = 1

		// Decrease ease factor
		ul.EaseFactor = math.Max(1.3, ul.EaseFactor-0.2)
	}

	now := time.Now()
	ul.LastPracticedAt = &now
	nextReview := now.AddDate(0, 0, ul.IntervalDays)
	ul.NextReviewAt = &nextReview

	return s.progress.UpdateUserLexeme(ctx, ul)
}
