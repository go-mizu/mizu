package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/stars"
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
func (h *StarHandler) ListStargazers(w http.ResponseWriter, r *http.Request) {
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
	opts := &stars.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	// Check if timestamps are requested via Accept header
	accept := r.Header.Get("Accept")
	if accept == "application/vnd.github.v3.star+json" {
		stargazers, err := h.stars.ListStargazersWithTimestamps(r.Context(), owner, repoName, opts)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, stargazers)
		return
	}

	stargazers, err := h.stars.ListStargazers(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, stargazers)
}

// ListStarredRepos handles GET /users/{username}/starred
func (h *StarHandler) ListStarredRepos(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	pagination := GetPaginationParams(r)
	opts := &stars.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	// Check if timestamps are requested via Accept header
	accept := r.Header.Get("Accept")
	if accept == "application/vnd.github.v3.star+json" {
		repoList, err := h.stars.ListForUserWithTimestamps(r.Context(), username, opts)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, repoList)
		return
	}

	repoList, err := h.stars.ListForUser(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}

// ListAuthenticatedUserStarredRepos handles GET /user/starred
func (h *StarHandler) ListAuthenticatedUserStarredRepos(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &stars.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	// Check if timestamps are requested via Accept header
	accept := r.Header.Get("Accept")
	if accept == "application/vnd.github.v3.star+json" {
		repoList, err := h.stars.ListForAuthenticatedUserWithTimestamps(r.Context(), user.ID, opts)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, repoList)
		return
	}

	repoList, err := h.stars.ListForAuthenticatedUser(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}

// CheckRepoStarred handles GET /user/starred/{owner}/{repo}
func (h *StarHandler) CheckRepoStarred(w http.ResponseWriter, r *http.Request) {
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

	isStarred, err := h.stars.IsStarred(r.Context(), user.ID, owner, repoName)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isStarred {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Star")
	}
}

// StarRepo handles PUT /user/starred/{owner}/{repo}
func (h *StarHandler) StarRepo(w http.ResponseWriter, r *http.Request) {
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

	if err := h.stars.Star(r.Context(), user.ID, owner, repoName); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// UnstarRepo handles DELETE /user/starred/{owner}/{repo}
func (h *StarHandler) UnstarRepo(w http.ResponseWriter, r *http.Request) {
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

	if err := h.stars.Unstar(r.Context(), user.ID, owner, repoName); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
