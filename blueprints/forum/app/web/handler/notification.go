package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/notifications"
)

// Notification handles notification endpoints.
type Notification struct {
	notifications notifications.API
	getAccountID  func(*mizu.Ctx) string
}

// NewNotification creates a new notification handler.
func NewNotification(notifications notifications.API, getAccountID func(*mizu.Ctx) string) *Notification {
	return &Notification{
		notifications: notifications,
		getAccountID:  getAccountID,
	}
}

// List lists notifications.
func (h *Notification) List(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	opts := notifications.ListOpts{
		Limit:  50,
		Unread: c.Query("unread") == "true",
	}

	notificationList, err := h.notifications.List(c.Request().Context(), accountID, opts)
	if err != nil {
		return InternalError(c)
	}

	// Get unread count
	unreadCount, _ := h.notifications.GetUnreadCount(c.Request().Context(), accountID)

	return Success(c, map[string]any{
		"notifications": notificationList,
		"unread_count":  unreadCount,
	})
}

// MarkRead marks notifications as read.
func (h *Notification) MarkRead(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in struct {
		IDs []string `json:"ids"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if err := h.notifications.MarkRead(c.Request().Context(), accountID, in.IDs); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Marked as read"})
}

// MarkAllRead marks all notifications as read.
func (h *Notification) MarkAllRead(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.notifications.MarkAllRead(c.Request().Context(), accountID); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "All marked as read"})
}
