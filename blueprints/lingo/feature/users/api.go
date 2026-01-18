package users

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for users
type Handler struct {
	svc *Service
}

// NewHandler creates a new user handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers user routes
func (h *Handler) RegisterRoutes(r *mizu.Router) {
	r.Get("/users/me", h.GetMe)
	r.Put("/users/me", h.UpdateMe)
	r.Put("/users/me/course", h.SetActiveCourse)
	r.Get("/users/{username}", h.GetByUsername)
	r.Get("/users/{id}/stats", h.GetStats)
	r.Put("/users/me/settings", h.UpdateSettings)
}

// GetMe handles GET /users/me
func (h *Handler) GetMe(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	user, err := h.svc.GetByID(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateMe handles PUT /users/me
func (h *Handler) UpdateMe(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var input UpdateProfileInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user, err := h.svc.UpdateProfile(c.Context(), userID, input)
	if err != nil {
		switch err {
		case ErrUserNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		case ErrUsernameExists:
			return c.JSON(http.StatusConflict, map[string]string{"error": "username already exists"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update user"})
		}
	}

	return c.JSON(http.StatusOK, user)
}

// GetByUsername handles GET /users/{username}
func (h *Handler) GetByUsername(c *mizu.Ctx) error {
	username := c.Param("username")

	user, err := h.svc.GetByUsername(c.Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Return public profile only
	return c.JSON(http.StatusOK, map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
		"bio":          user.Bio,
		"xp_total":     user.XPTotal,
		"streak_days":  user.StreakDays,
		"created_at":   user.CreatedAt,
	})
}

// GetStats handles GET /users/{id}/stats
func (h *Handler) GetStats(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	stats, err := h.svc.GetStats(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, stats)
}

// UpdateSettings handles PUT /users/me/settings
func (h *Handler) UpdateSettings(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var input UpdateSettingsInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user, err := h.svc.UpdateSettings(c.Context(), userID, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update settings"})
	}

	return c.JSON(http.StatusOK, user)
}

// SetActiveCourse handles PUT /users/me/course
func (h *Handler) SetActiveCourse(c *mizu.Ctx) error {
	userID := getUserID(c)
	if userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var input struct {
		CourseID string `json:"course_id"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	courseID, err := uuid.Parse(input.CourseID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid course id"})
	}

	user, err := h.svc.SetActiveCourse(c.Context(), userID, courseID)
	if err != nil {
		switch err {
		case ErrUserNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to set active course"})
		}
	}

	return c.JSON(http.StatusOK, user)
}

// getUserID extracts the user ID from the request context
func getUserID(c *mizu.Ctx) uuid.UUID {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}
