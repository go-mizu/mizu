package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// NotificationHandler handles notification endpoints
type NotificationHandler struct {
	notifications notifications.API
	repos         repos.API
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notifications notifications.API, repos repos.API) *NotificationHandler {
	return &NotificationHandler{notifications: notifications, repos: repos}
}

// ListNotifications handles GET /notifications
func (h *NotificationHandler) ListNotifications(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &notifications.ListOpts{
		Page:          pagination.Page,
		PerPage:       pagination.PerPage,
		All:           QueryBool(c, "all"),
		Participating: QueryBool(c, "participating"),
	}

	if since := c.Query("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = t
		}
	}

	if before := c.Query("before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			opts.Before = t
		}
	}

	notificationList, err := h.notifications.List(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, notificationList)
}

// MarkAllAsRead handles PUT /notifications
func (h *NotificationHandler) MarkAllAsRead(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	var in struct {
		LastReadAt time.Time `json:"last_read_at,omitempty"`
		Read       bool      `json:"read,omitempty"`
	}
	c.BindJSON(&in, 1<<20) // optional

	if in.LastReadAt.IsZero() {
		in.LastReadAt = time.Now()
	}

	if err := h.notifications.MarkAsRead(c.Context(), user.ID, in.LastReadAt); err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, map[string]string{"message": "Notifications marked as read"})
}

// GetThread handles GET /notifications/threads/{thread_id}
func (h *NotificationHandler) GetThread(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	threadID := c.Param("thread_id")

	thread, err := h.notifications.GetThread(c.Context(), user.ID, threadID)
	if err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, thread)
}

// MarkThreadAsRead handles PATCH /notifications/threads/{thread_id}
func (h *NotificationHandler) MarkThreadAsRead(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	threadID := c.Param("thread_id")

	if err := h.notifications.MarkThreadAsRead(c.Context(), user.ID, threadID); err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// MarkThreadAsDone handles DELETE /notifications/threads/{thread_id}
func (h *NotificationHandler) MarkThreadAsDone(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	threadID := c.Param("thread_id")

	if err := h.notifications.MarkThreadAsDone(c.Context(), user.ID, threadID); err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetThreadSubscription handles GET /notifications/threads/{thread_id}/subscription
func (h *NotificationHandler) GetThreadSubscription(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	threadID := c.Param("thread_id")

	subscription, err := h.notifications.GetThreadSubscription(c.Context(), user.ID, threadID)
	if err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "Subscription")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, subscription)
}

// SetThreadSubscription handles PUT /notifications/threads/{thread_id}/subscription
func (h *NotificationHandler) SetThreadSubscription(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	threadID := c.Param("thread_id")

	var in struct {
		Ignored bool `json:"ignored"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	subscription, err := h.notifications.SetThreadSubscription(c.Context(), user.ID, threadID, in.Ignored)
	if err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, subscription)
}

// DeleteThreadSubscription handles DELETE /notifications/threads/{thread_id}/subscription
func (h *NotificationHandler) DeleteThreadSubscription(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	threadID := c.Param("thread_id")

	if err := h.notifications.DeleteThreadSubscription(c.Context(), user.ID, threadID); err != nil {
		if err == notifications.ErrNotFound {
			return NotFound(c, "Subscription")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListRepoNotifications handles GET /repos/{owner}/{repo}/notifications
func (h *NotificationHandler) ListRepoNotifications(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pagination := GetPagination(c)
	opts := &notifications.ListOpts{
		Page:          pagination.Page,
		PerPage:       pagination.PerPage,
		All:           QueryBool(c, "all"),
		Participating: QueryBool(c, "participating"),
	}

	notificationList, err := h.notifications.ListForRepo(c.Context(), user.ID, owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, notificationList)
}

// MarkRepoNotificationsAsRead handles PUT /repos/{owner}/{repo}/notifications
func (h *NotificationHandler) MarkRepoNotificationsAsRead(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	var in struct {
		LastReadAt time.Time `json:"last_read_at,omitempty"`
	}
	c.BindJSON(&in, 1<<20) // optional

	if in.LastReadAt.IsZero() {
		in.LastReadAt = time.Now()
	}

	if err := h.notifications.MarkRepoAsRead(c.Context(), user.ID, owner, repoName, in.LastReadAt); err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, map[string]string{"message": "Notifications marked as read"})
}
