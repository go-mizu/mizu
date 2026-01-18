package api

import (
	"net/http"
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/google/uuid"
)

// CourseHandler handles course endpoints
type CourseHandler struct {
	store store.Store
}

// NewCourseHandler creates a new course handler
func NewCourseHandler(st store.Store) *CourseHandler {
	return &CourseHandler{store: st}
}

// ListLanguages lists all available languages
func (h *CourseHandler) ListLanguages(c *mizu.Ctx) error {
	languages, err := h.store.Courses().ListLanguages(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch languages"})
	}
	return c.JSON(http.StatusOK, languages)
}

// ListCourses lists courses available from a language
func (h *CourseHandler) ListCourses(c *mizu.Ctx) error {
	fromLang := c.Query("from")
	if fromLang == "" {
		fromLang = "en"
	}
	courses, err := h.store.Courses().ListCourses(c.Context(), fromLang)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch courses"})
	}
	return c.JSON(http.StatusOK, courses)
}

// GetCourse returns a course by ID
func (h *CourseHandler) GetCourse(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course ID"})
	}

	course, err := h.store.Courses().GetCourse(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
	}
	return c.JSON(http.StatusOK, course)
}

// Enroll enrolls the current user in a course
func (h *CourseHandler) Enroll(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
	}

	courseIDParam := c.Param("id")
	courseID, err := uuid.Parse(courseIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course ID"})
	}

	if err := h.store.Progress().EnrollCourse(c.Context(), uid, courseID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to enroll"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "enrolled successfully"})
}

// GetPath returns the learning path for a course
func (h *CourseHandler) GetPath(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course ID"})
	}

	units, err := h.store.Courses().GetCoursePath(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch path"})
	}
	return c.JSON(http.StatusOK, units)
}

// GetUnit returns a unit by ID
func (h *CourseHandler) GetUnit(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid unit ID"})
	}

	unit, err := h.store.Courses().GetUnit(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "unit not found"})
	}
	return c.JSON(http.StatusOK, unit)
}

// GetSkill returns a skill by ID
func (h *CourseHandler) GetSkill(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
	}

	skill, err := h.store.Courses().GetSkill(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "skill not found"})
	}
	return c.JSON(http.StatusOK, skill)
}

// GetStories returns stories for a course
func (h *CourseHandler) GetStories(c *mizu.Ctx) error {
	courseID := c.Query("course_id")
	if courseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "course_id is required"})
	}

	id, err := uuid.Parse(courseID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course ID"})
	}

	stories, err := h.store.Courses().GetStories(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch stories"})
	}
	return c.JSON(http.StatusOK, stories)
}

// GetStory returns a story by ID
func (h *CourseHandler) GetStory(c *mizu.Ctx) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid story ID"})
	}

	story, err := h.store.Courses().GetStory(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "story not found"})
	}
	return c.JSON(http.StatusOK, story)
}

// CompleteStory marks a story as completed
func (h *CourseHandler) CompleteStory(c *mizu.Ctx) error {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// In a real implementation, record story completion and award XP
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "story completed",
		"xp_earned": 14,
	})
}
