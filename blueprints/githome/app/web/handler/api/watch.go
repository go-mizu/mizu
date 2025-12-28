package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/watches"
)

// WatchHandler handles watch/subscription endpoints
type WatchHandler struct {
	watches watches.API
	repos   repos.API
}

// NewWatchHandler creates a new watch handler
func NewWatchHandler(watches watches.API, repos repos.API) *WatchHandler {
	return &WatchHandler{watches: watches, repos: repos}
}

// ListWatchers handles GET /repos/{owner}/{repo}/subscribers
func (h *WatchHandler) ListWatchers(w http.ResponseWriter, r *http.Request) {
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
	opts := &watches.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	watchers, err := h.watches.ListWatchers(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, watchers)
}

// GetSubscription handles GET /repos/{owner}/{repo}/subscription
func (h *WatchHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
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

	subscription, err := h.watches.GetSubscription(r.Context(), user.ID, owner, repoName)
	if err != nil {
		if err == watches.ErrNotFound {
			WriteNotFound(w, "Subscription")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, subscription)
}

// SetSubscription handles PUT /repos/{owner}/{repo}/subscription
func (h *WatchHandler) SetSubscription(w http.ResponseWriter, r *http.Request) {
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
		Subscribed bool `json:"subscribed"`
		Ignored    bool `json:"ignored"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	subscription, err := h.watches.SetSubscription(r.Context(), user.ID, owner, repoName, in.Subscribed, in.Ignored)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, subscription)
}

// DeleteSubscription handles DELETE /repos/{owner}/{repo}/subscription
func (h *WatchHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
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

	if err := h.watches.DeleteSubscription(r.Context(), user.ID, owner, repoName); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListWatchedRepos handles GET /users/{username}/subscriptions
func (h *WatchHandler) ListWatchedRepos(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	pagination := GetPaginationParams(r)
	opts := &watches.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	repoList, err := h.watches.ListForUser(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}

// ListAuthenticatedUserWatchedRepos handles GET /user/subscriptions
func (h *WatchHandler) ListAuthenticatedUserWatchedRepos(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &watches.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	repoList, err := h.watches.ListForAuthenticatedUser(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}
