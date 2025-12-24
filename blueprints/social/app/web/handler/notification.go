package handler

import (
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/notifications"
)

// Notification handles notification endpoints.
type Notification struct {
	notifications notifications.API
	getAccountID  func(*mizu.Ctx) string
}

// NewNotification creates a new notification handler.
func NewNotification(notificationsSvc notifications.API, getAccountID func(*mizu.Ctx) string) *Notification {
	return &Notification{
		notifications: notificationsSvc,
		getAccountID:  getAccountID,
	}
}

// List handles GET /api/v1/notifications
func (h *Notification) List(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	var types []string
	if t := c.Query("types[]"); t != "" {
		types = strings.Split(t, ",")
	}

	var excludeTypes []string
	if t := c.Query("exclude_types[]"); t != "" {
		excludeTypes = strings.Split(t, ",")
	}

	opts := notifications.ListOpts{
		Limit:        limit,
		MaxID:        maxID,
		SinceID:      sinceID,
		Types:        types,
		ExcludeTypes: excludeTypes,
	}

	ns, err := h.notifications.List(c.Request().Context(), accountID, opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, ns)
}

// UnreadCount handles GET /api/v1/notifications/unread_count
func (h *Notification) UnreadCount(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	count, err := h.notifications.UnreadCount(c.Request().Context(), accountID)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, map[string]int{"count": count})
}

// Clear handles POST /api/v1/notifications/clear
func (h *Notification) Clear(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	if err := h.notifications.Clear(c.Request().Context(), accountID); err != nil {
		return InternalError(c, err)
	}

	return NoContent(c)
}

// Dismiss handles POST /api/v1/notifications/:id/dismiss
func (h *Notification) Dismiss(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	id := c.Param("id")

	if err := h.notifications.Dismiss(c.Request().Context(), accountID, id); err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "notification")
		}
		return InternalError(c, err)
	}

	return NoContent(c)
}
