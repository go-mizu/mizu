package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/watches"
	"github.com/go-mizu/mizu"
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
func (h *WatchHandler) ListWatchers(c *mizu.Ctx) error {
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
	opts := &watches.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	watchers, err := h.watches.ListWatchers(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, watchers)
}

// GetSubscription handles GET /repos/{owner}/{repo}/subscription
func (h *WatchHandler) GetSubscription(c *mizu.Ctx) error {
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

	subscription, err := h.watches.GetSubscription(c.Context(), user.ID, owner, repoName)
	if err != nil {
		if err == watches.ErrNotFound {
			return NotFound(c, "Subscription")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, subscription)
}

// SetSubscription handles PUT /repos/{owner}/{repo}/subscription
func (h *WatchHandler) SetSubscription(c *mizu.Ctx) error {
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
		Subscribed bool `json:"subscribed"`
		Ignored    bool `json:"ignored"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	subscription, err := h.watches.SetSubscription(c.Context(), user.ID, owner, repoName, in.Subscribed, in.Ignored)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, subscription)
}

// DeleteSubscription handles DELETE /repos/{owner}/{repo}/subscription
func (h *WatchHandler) DeleteSubscription(c *mizu.Ctx) error {
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

	if err := h.watches.DeleteSubscription(c.Context(), user.ID, owner, repoName); err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListWatchedRepos handles GET /users/{username}/subscriptions
func (h *WatchHandler) ListWatchedRepos(c *mizu.Ctx) error {
	username := c.Param("username")
	pagination := GetPagination(c)
	opts := &watches.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	repoList, err := h.watches.ListForUser(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}

// ListAuthenticatedUserWatchedRepos handles GET /user/subscriptions
func (h *WatchHandler) ListAuthenticatedUserWatchedRepos(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &watches.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	repoList, err := h.watches.ListForAuthenticatedUser(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}
