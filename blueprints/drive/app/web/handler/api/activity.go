package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/activity"
)

// Activity handles activity endpoints.
type Activity struct {
	activity  activity.API
	getUserID func(*mizu.Ctx) string
}

// NewActivity creates a new Activity handler.
func NewActivity(activity activity.API, getUserID func(*mizu.Ctx) string) *Activity {
	return &Activity{
		activity:  activity,
		getUserID: getUserID,
	}
}

// List lists user activity.
func (h *Activity) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	activities, err := h.activity.ListByUser(c.Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, activities)
}

// ListForResource lists activity for a resource.
func (h *Activity) ListForResource(c *mizu.Ctx) error {
	resourceType := c.Param("type")
	resourceID := c.Param("id")

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	activities, err := h.activity.ListForResource(c.Context(), resourceType, resourceID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, activities)
}
