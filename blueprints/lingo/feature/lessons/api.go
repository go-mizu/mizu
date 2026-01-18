package lessons

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for lessons
type Handler struct {
	svc *Service
}

// NewHandler creates a new lesson handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers lesson routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/lessons/{id}", h.GetLesson)
	r.Get("/skills/{id}/lesson", h.GetLessonBySkill)
	r.Post("/lessons/{id}/start", h.StartLesson)
	r.Post("/lessons/{id}/complete", h.CompleteLesson)
	r.Post("/exercises/{id}/answer", h.AnswerExercise)
}

// GetLesson handles GET /lessons/{id}
func (h *Handler) GetLesson(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid lesson id"})
	}

	lesson, err := h.svc.GetLesson(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
	}

	return c.JSON(http.StatusOK, lesson)
}

// GetLessonBySkill handles GET /skills/{id}/lesson
// Returns the first lesson for a skill (used when starting from skill node)
func (h *Handler) GetLessonBySkill(c *mizu.Ctx) error {
	idStr := c.Param("id")
	skillID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid skill id"})
	}

	lesson, err := h.svc.GetLessonBySkill(c.Context(), skillID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no lesson found for this skill"})
	}

	return c.JSON(http.StatusOK, lesson)
}

// StartLesson handles POST /lessons/{id}/start
func (h *Handler) StartLesson(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	lessonID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid lesson id"})
	}

	session, err := h.svc.StartLesson(c.Context(), userID, lessonID)
	if err != nil {
		switch err {
		case ErrNoHearts:
			return c.JSON(http.StatusForbidden, map[string]string{"error": "no hearts remaining"})
		case ErrLessonNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start lesson"})
		}
	}

	return c.JSON(http.StatusOK, session)
}

// CompleteLessonRequest represents a lesson completion request
type CompleteLessonRequest struct {
	MistakesCount int `json:"mistakes_count"`
	HeartsLost    int `json:"hearts_lost"`
}

// CompleteLesson handles POST /lessons/{id}/complete
func (h *Handler) CompleteLesson(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	lessonID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid lesson id"})
	}

	var req CompleteLessonRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	result, err := h.svc.CompleteLesson(c.Context(), userID, lessonID, req.MistakesCount, req.HeartsLost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to complete lesson"})
	}

	return c.JSON(http.StatusOK, result)
}

// AnswerExerciseRequest represents an exercise answer request
type AnswerExerciseRequest struct {
	Answer string `json:"answer"`
}

// AnswerExercise handles POST /exercises/{id}/answer
func (h *Handler) AnswerExercise(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	exerciseID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid exercise id"})
	}

	var req AnswerExerciseRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	result, err := h.svc.AnswerExercise(c.Context(), userID, exerciseID, req.Answer)
	if err != nil {
		switch err {
		case ErrExerciseNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "exercise not found"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process answer"})
		}
	}

	return c.JSON(http.StatusOK, result)
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	if userIDStr := c.Header().Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
