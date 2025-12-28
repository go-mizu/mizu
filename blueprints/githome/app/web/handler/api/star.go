package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/stars"
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

// getRepoFromPath gets repository from path parameters
func (h *StarHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListStargazers handles GET /repos/{owner}/{repo}/stargazers
func (h *StarHandler) ListStargazers(w http.ResponseWriter, r *http.Request) {
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
	opts := &stars.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	stargazers, err := h.stars.ListStargazers(r.Context(), repo.ID, opts)
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

	repoList, err := h.stars.ListStarredByUser(r.Context(), username, opts)
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

	repoList, err := h.stars.ListStarredByUser(r.Context(), user.Login, opts)
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

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	isStarred, err := h.stars.IsStarred(r.Context(), user.ID, repo.ID)
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

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.stars.Star(r.Context(), user.ID, repo.ID); err != nil {
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

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.stars.Unstar(r.Context(), user.ID, repo.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
