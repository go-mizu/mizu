package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/activities"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
)

// ActivityHandler handles activity/event endpoints
type ActivityHandler struct {
	activities activities.API
	repos      repos.API
}

// NewActivityHandler creates a new activity handler
func NewActivityHandler(activities activities.API, repos repos.API) *ActivityHandler {
	return &ActivityHandler{activities: activities, repos: repos}
}

// getRepoFromPath gets repository from path parameters
func (h *ActivityHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListPublicEvents handles GET /events
func (h *ActivityHandler) ListPublicEvents(w http.ResponseWriter, r *http.Request) {
	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListPublicEvents(r.Context(), opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListRepoEvents handles GET /repos/{owner}/{repo}/events
func (h *ActivityHandler) ListRepoEvents(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListRepoEvents(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListRepoNetworkEvents handles GET /networks/{owner}/{repo}/events
func (h *ActivityHandler) ListRepoNetworkEvents(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListNetworkEvents(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListOrgEvents handles GET /orgs/{org}/events
func (h *ActivityHandler) ListOrgEvents(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListOrgEvents(r.Context(), org, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListUserReceivedEvents handles GET /users/{username}/received_events
func (h *ActivityHandler) ListUserReceivedEvents(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListReceivedEvents(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListUserReceivedPublicEvents handles GET /users/{username}/received_events/public
func (h *ActivityHandler) ListUserReceivedPublicEvents(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListReceivedPublicEvents(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListUserEvents handles GET /users/{username}/events
func (h *ActivityHandler) ListUserEvents(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListUserEvents(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListUserPublicEvents handles GET /users/{username}/events/public
func (h *ActivityHandler) ListUserPublicEvents(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListUserPublicEvents(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListUserOrgEvents handles GET /users/{username}/events/orgs/{org}
func (h *ActivityHandler) ListUserOrgEvents(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	username := PathParam(r, "username")
	org := PathParam(r, "org")

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListUserOrgEvents(r.Context(), username, org, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListAuthenticatedUserEvents handles GET /users/{username}/events (authenticated)
func (h *ActivityHandler) ListAuthenticatedUserEvents(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListUserEvents(r.Context(), user.Login, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// ListFeeds handles GET /feeds
func (h *ActivityHandler) ListFeeds(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())

	var userID int64
	if user != nil {
		userID = user.ID
	}

	feeds, err := h.activities.ListFeeds(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, feeds)
}
