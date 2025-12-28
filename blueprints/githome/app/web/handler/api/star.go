package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/stars"
	"github.com/go-mizu/mizu"
)

// StarHandler handles star endpoints
type StarHandler struct {
	stars stars.API
	repos repos.API
}

// NewStarHandler creates a new star handler
func NewStarHandler(stars stars.API, repos repos.API) *StarHandler {
	return &StarHandler{stars: stars, repos: repos}
}

// ListStargazers handles GET /repos/{owner}/{repo}/stargazers
func (h *StarHandler) ListStargazers(c *mizu.Ctx) error {
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
	opts := &stars.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	// Check if timestamps are requested via Accept header
	req := c.Request()
	accept := req.Header.Get("Accept")
	if accept == "application/vnd.github.v3.star+json" {
		stargazers, err := h.stars.ListStargazersWithTimestamps(c.Context(), owner, repoName, opts)
		if err != nil {
			return WriteError(c, http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, stargazers)
	}

	stargazers, err := h.stars.ListStargazers(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, stargazers)
}

// ListStarredRepos handles GET /users/{username}/starred
func (h *StarHandler) ListStarredRepos(c *mizu.Ctx) error {
	username := c.Param("username")
	pagination := GetPagination(c)
	opts := &stars.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	// Check if timestamps are requested via Accept header
	req := c.Request()
	accept := req.Header.Get("Accept")
	if accept == "application/vnd.github.v3.star+json" {
		repoList, err := h.stars.ListForUserWithTimestamps(c.Context(), username, opts)
		if err != nil {
			return WriteError(c, http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, repoList)
	}

	repoList, err := h.stars.ListForUser(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}

// ListAuthenticatedUserStarredRepos handles GET /user/starred
func (h *StarHandler) ListAuthenticatedUserStarredRepos(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &stars.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	// Check if timestamps are requested via Accept header
	req := c.Request()
	accept := req.Header.Get("Accept")
	if accept == "application/vnd.github.v3.star+json" {
		repoList, err := h.stars.ListForAuthenticatedUserWithTimestamps(c.Context(), user.ID, opts)
		if err != nil {
			return WriteError(c, http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, repoList)
	}

	repoList, err := h.stars.ListForAuthenticatedUser(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}

// CheckRepoStarred handles GET /user/starred/{owner}/{repo}
func (h *StarHandler) CheckRepoStarred(c *mizu.Ctx) error {
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

	isStarred, err := h.stars.IsStarred(c.Context(), user.ID, owner, repoName)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isStarred {
		return NoContent(c)
	}
	return NotFound(c, "Star")
}

// StarRepo handles PUT /user/starred/{owner}/{repo}
func (h *StarHandler) StarRepo(c *mizu.Ctx) error {
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

	if err := h.stars.Star(c.Context(), user.ID, owner, repoName); err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// UnstarRepo handles DELETE /user/starred/{owner}/{repo}
func (h *StarHandler) UnstarRepo(c *mizu.Ctx) error {
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

	if err := h.stars.Unstar(c.Context(), user.ID, owner, repoName); err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}
