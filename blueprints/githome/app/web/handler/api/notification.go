package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &notifications.ListOpts{
		Page:          pagination.Page,
		PerPage:       pagination.PerPage,
		All:           QueryParamBool(r, "all"),
		Participating: QueryParamBool(r, "participating"),
	}

	if since := QueryParam(r, "since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = t
		}
	}

	if before := QueryParam(r, "before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			opts.Before = t
		}
	}

	notificationList, err := h.notifications.List(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, notificationList)
}

// MarkAllAsRead handles PUT /notifications
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	var in struct {
		LastReadAt time.Time `json:"last_read_at,omitempty"`
		Read       bool      `json:"read,omitempty"`
	}
	DecodeJSON(r, &in) // optional

	if in.LastReadAt.IsZero() {
		in.LastReadAt = time.Now()
	}

	if err := h.notifications.MarkAsRead(r.Context(), user.ID, in.LastReadAt); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, map[string]string{"message": "Notifications marked as read"})
}

// GetThread handles GET /notifications/threads/{thread_id}
func (h *NotificationHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	threadID := PathParam(r, "thread_id")

	thread, err := h.notifications.GetThread(r.Context(), user.ID, threadID)
	if err != nil {
		if err == notifications.ErrNotFound {
			WriteNotFound(w, "Thread")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, thread)
}

// MarkThreadAsRead handles PATCH /notifications/threads/{thread_id}
func (h *NotificationHandler) MarkThreadAsRead(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	threadID := PathParam(r, "thread_id")

	if err := h.notifications.MarkThreadAsRead(r.Context(), user.ID, threadID); err != nil {
		if err == notifications.ErrNotFound {
			WriteNotFound(w, "Thread")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// MarkThreadAsDone handles DELETE /notifications/threads/{thread_id}
func (h *NotificationHandler) MarkThreadAsDone(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	threadID := PathParam(r, "thread_id")

	if err := h.notifications.MarkThreadAsDone(r.Context(), user.ID, threadID); err != nil {
		if err == notifications.ErrNotFound {
			WriteNotFound(w, "Thread")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetThreadSubscription handles GET /notifications/threads/{thread_id}/subscription
func (h *NotificationHandler) GetThreadSubscription(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	threadID := PathParam(r, "thread_id")

	subscription, err := h.notifications.GetThreadSubscription(r.Context(), user.ID, threadID)
	if err != nil {
		if err == notifications.ErrNotFound {
			WriteNotFound(w, "Subscription")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, subscription)
}

// SetThreadSubscription handles PUT /notifications/threads/{thread_id}/subscription
func (h *NotificationHandler) SetThreadSubscription(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	threadID := PathParam(r, "thread_id")

	var in struct {
		Ignored bool `json:"ignored"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	subscription, err := h.notifications.SetThreadSubscription(r.Context(), user.ID, threadID, in.Ignored)
	if err != nil {
		if err == notifications.ErrNotFound {
			WriteNotFound(w, "Thread")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, subscription)
}

// DeleteThreadSubscription handles DELETE /notifications/threads/{thread_id}/subscription
func (h *NotificationHandler) DeleteThreadSubscription(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	threadID := PathParam(r, "thread_id")

	if err := h.notifications.DeleteThreadSubscription(r.Context(), user.ID, threadID); err != nil {
		if err == notifications.ErrNotFound {
			WriteNotFound(w, "Subscription")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListRepoNotifications handles GET /repos/{owner}/{repo}/notifications
func (h *NotificationHandler) ListRepoNotifications(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &notifications.ListOpts{
		Page:          pagination.Page,
		PerPage:       pagination.PerPage,
		All:           QueryParamBool(r, "all"),
		Participating: QueryParamBool(r, "participating"),
	}

	notificationList, err := h.notifications.ListForRepo(r.Context(), user.ID, owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, notificationList)
}

// MarkRepoNotificationsAsRead handles PUT /repos/{owner}/{repo}/notifications
func (h *NotificationHandler) MarkRepoNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in struct {
		LastReadAt time.Time `json:"last_read_at,omitempty"`
	}
	DecodeJSON(r, &in) // optional

	if in.LastReadAt.IsZero() {
		in.LastReadAt = time.Now()
	}

	if err := h.notifications.MarkRepoAsRead(r.Context(), user.ID, owner, repoName, in.LastReadAt); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, map[string]string{"message": "Notifications marked as read"})
}
