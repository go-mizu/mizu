package courses

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for courses
type Handler struct {
	svc *Service
}

// NewHandler creates a new course handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers course routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/languages", h.ListLanguages)
	r.Get("/courses", h.ListCourses)
	r.Get("/courses/{id}", h.GetCourse)
	r.Post("/courses/{id}/enroll", h.Enroll)
	r.Get("/courses/{id}/path", h.GetPath)
	r.Get("/courses/{id}/vocabulary", h.GetVocabulary)
	r.Get("/units/{id}", h.GetUnit)
	r.Get("/skills/{id}", h.GetSkill)
	r.Get("/stories", h.GetStories)
	r.Get("/stories/{id}", h.GetStory)
	r.Post("/stories/{id}/complete", h.CompleteStory)
}

// ListLanguages handles GET /languages
func (h *Handler) ListLanguages(c *mizu.Ctx) error {
	languages, err := h.svc.ListLanguages(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list languages"})
	}

	return c.JSON(http.StatusOK, languages)
}

// ListCourses handles GET /courses
func (h *Handler) ListCourses(c *mizu.Ctx) error {
	fromLang := c.Query("from")
	if fromLang == "" {
		fromLang = "en" // Default to English
	}

	courses, err := h.svc.ListCourses(c.Context(), fromLang)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list courses"})
	}

	return c.JSON(http.StatusOK, courses)
}

// GetCourse handles GET /courses/{id}
func (h *Handler) GetCourse(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course id"})
	}

	userID := getUserID(c)
	if userID != uuid.Nil {
		// Return course with progress
		result, err := h.svc.GetCourseWithProgress(c.Context(), userID, id)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
		}
		return c.JSON(http.StatusOK, result)
	}

	course, err := h.svc.GetCourse(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
	}

	return c.JSON(http.StatusOK, course)
}

// Enroll handles POST /courses/{id}/enroll
func (h *Handler) Enroll(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	courseID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course id"})
	}

	if err := h.svc.EnrollInCourse(c.Context(), userID, courseID); err != nil {
		switch err {
		case ErrCourseNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
		case ErrAlreadyEnrolled:
			return c.JSON(http.StatusConflict, map[string]string{"error": "already enrolled"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to enroll"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "enrolled successfully"})
}

// GetPath handles GET /courses/{id}/path
func (h *Handler) GetPath(c *mizu.Ctx) error {
	idStr := c.Param("id")
	courseID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course id"})
	}

	path, err := h.svc.GetCoursePath(c.Context(), courseID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get course path"})
	}

	return c.JSON(http.StatusOK, path)
}

// GetVocabulary handles GET /courses/{id}/vocabulary
func (h *Handler) GetVocabulary(c *mizu.Ctx) error {
	idStr := c.Param("id")
	courseID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course id"})
	}

	lexemes, err := h.svc.GetLexemesByCourse(c.Context(), courseID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get vocabulary"})
	}

	return c.JSON(http.StatusOK, lexemes)
}

// GetUnit handles GET /units/{id}
func (h *Handler) GetUnit(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid unit id"})
	}

	unit, err := h.svc.GetUnit(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "unit not found"})
	}

	return c.JSON(http.StatusOK, unit)
}

// GetSkill handles GET /skills/{id}
func (h *Handler) GetSkill(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid skill id"})
	}

	skill, err := h.svc.GetSkill(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "skill not found"})
	}

	return c.JSON(http.StatusOK, skill)
}

// GetStories handles GET /stories
func (h *Handler) GetStories(c *mizu.Ctx) error {
	courseIDStr := c.Query("course_id")
	if courseIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "course_id is required"})
	}

	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course_id"})
	}

	stories, err := h.svc.GetStories(c.Context(), courseID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get stories"})
	}

	return c.JSON(http.StatusOK, stories)
}

// GetStory handles GET /stories/{id}
func (h *Handler) GetStory(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid story id"})
	}

	story, err := h.svc.GetStory(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "story not found"})
	}

	return c.JSON(http.StatusOK, story)
}

// CompleteStory handles POST /stories/{id}/complete
func (h *Handler) CompleteStory(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid story id"})
	}

	story, err := h.svc.GetStory(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "story not found"})
	}

	// Return XP reward
	return c.JSON(http.StatusOK, map[string]any{
		"xp_earned": story.XPReward,
		"message":   "story completed",
	})
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	if userIDStr := c.Request().Header.Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
