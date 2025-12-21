package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/notifications"
)

// Notification contains notification-related handlers.
type Notification struct {
	notifications notifications.API
	getAccountID  func(*mizu.Ctx) string
}

// NewNotification creates new notification handlers.
func NewNotification(
	notifications notifications.API,
	getAccountID func(*mizu.Ctx) string,
) *Notification {
	return &Notification{
		notifications: notifications,
		getAccountID:  getAccountID,
	}
}

// List returns the user's notifications.
func (h *Notification) List(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	limit := IntQuery(c, "limit", 30)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	notifs, err := h.notifications.List(c.Request().Context(), accountID, nil, limit, maxID, sinceID, nil)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": notifs})
}

// Clear marks all notifications as read.
func (h *Notification) Clear(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if err := h.notifications.MarkAllAsRead(c.Request().Context(), accountID); err != nil {
		return c.JSON(500, ErrorResponse("CLEAR_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}

// Dismiss dismisses a specific notification.
func (h *Notification) Dismiss(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	id := c.Param("id")
	if err := h.notifications.Dismiss(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(500, ErrorResponse("DISMISS_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": map[string]any{"success": true}})
}
