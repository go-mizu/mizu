package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// LessonHandler handles lesson endpoints
type LessonHandler struct {
	store store.Store
}

// NewLessonHandler creates a new lesson handler
func NewLessonHandler(st store.Store) *LessonHandler {
	return &LessonHandler{store: st}
}

// GetLesson returns a lesson with exercises
func (h *LessonHandler) GetLesson(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid lesson ID"})
	}

	lesson, err := h.store.Courses().GetLesson(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
	}

	exercises, err := h.store.Courses().GetExercises(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch exercises"})
	}

	lesson.Exercises = exercises
	return c.JSON(http.StatusOK, lesson)
}

// StartLesson starts a new lesson session
func (h *LessonHandler) StartLesson(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	lessonIDParam := c.Param("id")
	lessonID, err := uuid.Parse(lessonIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid lesson ID"})
	}

	// Get lesson with exercises
	lesson, err := h.store.Courses().GetLesson(c.Context(), lessonID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
	}

	exercises, err := h.store.Courses().GetExercises(c.Context(), lessonID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch exercises"})
	}

	// Create session
	session := &store.LessonSession{
		ID:       uuid.New(),
		UserID:   uid,
		LessonID: lessonID,
	}

	if err := h.store.Progress().StartLessonSession(c.Context(), session); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start session"})
	}

	// Get user hearts
	user, _ := h.store.Users().GetByID(c.Context(), uid)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session_id": session.ID,
		"lesson":     lesson,
		"exercises":  exercises,
		"hearts":     user.Hearts,
	})
}

// CompleteLessonRequest represents a lesson completion request
type CompleteLessonRequest struct {
	SessionID     uuid.UUID `json:"session_id"`
	MistakesCount int       `json:"mistakes_count"`
	HeartsLost    int       `json:"hearts_lost"`
	TimeSpent     int       `json:"time_spent_seconds"`
}

// CompleteLesson completes a lesson session
func (h *LessonHandler) CompleteLesson(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	var req CompleteLessonRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Calculate XP (base 10, bonus for perfect)
	xpEarned := 10
	isPerfect := req.MistakesCount == 0
	if isPerfect {
		xpEarned = 15
	}

	// Complete session
	session := &store.LessonSession{
		ID:            req.SessionID,
		XPEarned:      xpEarned,
		MistakesCount: req.MistakesCount,
		HeartsLost:    req.HeartsLost,
		IsPerfect:     isPerfect,
	}

	if err := h.store.Progress().CompleteLessonSession(c.Context(), session); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to complete session"})
	}

	// Update user XP
	if err := h.store.Users().UpdateXP(c.Context(), uid, xpEarned); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update XP"})
	}

	// Update streak
	if err := h.store.Users().UpdateStreak(c.Context(), uid); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update streak"})
	}

	// Record XP event
	xpEvent := &store.XPEvent{
		ID:       uuid.New(),
		UserID:   uid,
		Amount:   xpEarned,
		Source:   "lesson",
		SourceID: req.SessionID,
	}
	h.store.Progress().RecordXPEvent(c.Context(), xpEvent)

	// Record streak day
	h.store.Progress().RecordStreakDay(c.Context(), uid, xpEarned, 1, req.TimeSpent)

	// Update hearts if lost
	if req.HeartsLost > 0 {
		user, _ := h.store.Users().GetByID(c.Context(), uid)
		newHearts := user.Hearts - req.HeartsLost
		if newHearts < 0 {
			newHearts = 0
		}
		h.store.Users().UpdateHearts(c.Context(), uid, newHearts)
	}

	// Get updated user
	user, _ := h.store.Users().GetByID(c.Context(), uid)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"xp_earned":  xpEarned,
		"is_perfect": isPerfect,
		"total_xp":   user.XPTotal,
		"streak":     user.StreakDays,
		"hearts":     user.Hearts,
		"gems":       user.Gems,
	})
}

// AnswerExerciseRequest represents an exercise answer request
type AnswerExerciseRequest struct {
	Answer string `json:"answer"`
}

// AnswerExercise checks an exercise answer
func (h *LessonHandler) AnswerExercise(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	exerciseIDParam := c.Param("id")
	exerciseID, err := uuid.Parse(exerciseIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid exercise ID"})
	}

	var req AnswerExerciseRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// In a real implementation, fetch the exercise and check the answer
	// For now, we'll simulate a check
	isCorrect := true // Simplified

	if !isCorrect {
		// Record mistake
		mistake := &store.UserMistake{
			ID:            uuid.New(),
			UserID:        uid,
			ExerciseID:    exerciseID,
			UserAnswer:    req.Answer,
			CorrectAnswer: "correct_answer",
			MistakeType:   "wrong_answer",
		}
		h.store.Progress().RecordMistake(c.Context(), mistake)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"correct":        isCorrect,
		"correct_answer": "the correct answer",
	})
}
