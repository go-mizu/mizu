package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/notifications"
)

// Notification handles notification endpoints.
type Notification struct {
	notifications notifications.API
	getUserID     func(*mizu.Ctx) string
}

// NewNotification creates a new notification handler.
func NewNotification(notifications notifications.API, getUserID func(*mizu.Ctx) string) *Notification {
	return &Notification{notifications: notifications, getUserID: getUserID}
}

// List returns all notifications for the current user.
func (h *Notification) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	list, err := h.notifications.ListByUser(c.Context(), userID, 50)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list notifications"))
	}

	return c.JSON(http.StatusOK, list)
}

// MarkRead marks a notification as read.
func (h *Notification) MarkRead(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.notifications.MarkRead(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to mark notification as read"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "notification marked as read"})
}

// MarkAllRead marks all notifications as read.
func (h *Notification) MarkAllRead(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	if err := h.notifications.MarkAllRead(c.Context(), userID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to mark all notifications as read"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "all notifications marked as read"})
}
