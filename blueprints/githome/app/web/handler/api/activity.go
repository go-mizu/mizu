package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/activities"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
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

// ListPublicEvents handles GET /events
func (h *ActivityHandler) ListPublicEvents(c *mizu.Ctx) error {
	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListPublic(c.Context(), opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListRepoEvents handles GET /repos/{owner}/{repo}/events
func (h *ActivityHandler) ListRepoEvents(c *mizu.Ctx) error {
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
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListForRepo(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListRepoNetworkEvents handles GET /networks/{owner}/{repo}/events
func (h *ActivityHandler) ListRepoNetworkEvents(c *mizu.Ctx) error {
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
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListNetworkEvents(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListOrgEvents handles GET /orgs/{org}/events
func (h *ActivityHandler) ListOrgEvents(c *mizu.Ctx) error {
	org := c.Param("org")

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListForOrg(c.Context(), org, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListUserReceivedEvents handles GET /users/{username}/received_events
func (h *ActivityHandler) ListUserReceivedEvents(c *mizu.Ctx) error {
	username := c.Param("username")

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListReceivedEvents(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListUserReceivedPublicEvents handles GET /users/{username}/received_events/public
func (h *ActivityHandler) ListUserReceivedPublicEvents(c *mizu.Ctx) error {
	username := c.Param("username")

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListPublicReceivedEvents(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListUserEvents handles GET /users/{username}/events
func (h *ActivityHandler) ListUserEvents(c *mizu.Ctx) error {
	username := c.Param("username")

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListForUser(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListUserPublicEvents handles GET /users/{username}/events/public
func (h *ActivityHandler) ListUserPublicEvents(c *mizu.Ctx) error {
	username := c.Param("username")

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListPublicForUser(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListUserOrgEvents handles GET /users/{username}/events/orgs/{org}
func (h *ActivityHandler) ListUserOrgEvents(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	username := c.Param("username")
	org := c.Param("org")

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListOrgEventsForUser(c.Context(), username, org, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListAuthenticatedUserEvents handles GET /users/{username}/events (authenticated)
func (h *ActivityHandler) ListAuthenticatedUserEvents(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &activities.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.activities.ListForUser(c.Context(), user.Login, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// ListFeeds handles GET /feeds
func (h *ActivityHandler) ListFeeds(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)

	var userID int64
	if user != nil {
		userID = user.ID
	}

	feeds, err := h.activities.GetFeeds(c.Context(), userID)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, feeds)
}
