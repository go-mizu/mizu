package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/search"
)

// SearchHandler handles search endpoints
type SearchHandler struct {
	search search.API
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(search search.API) *SearchHandler {
	return &SearchHandler{search: search}
}

// SearchRepositories handles GET /search/repositories
func (h *SearchHandler) SearchRepositories(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchReposOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.Repositories(r.Context(), q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// SearchCode handles GET /search/code
func (h *SearchHandler) SearchCode(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchCodeOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.Code(r.Context(), q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// SearchCommits handles GET /search/commits
func (h *SearchHandler) SearchCommits(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.Commits(r.Context(), q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// SearchIssues handles GET /search/issues
func (h *SearchHandler) SearchIssues(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchIssuesOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.IssuesAndPullRequests(r.Context(), q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// SearchUsers handles GET /search/users
func (h *SearchHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchUsersOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.Users(r.Context(), q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// SearchLabels handles GET /search/labels
func (h *SearchHandler) SearchLabels(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	repositoryID := QueryParam(r, "repository_id")
	if repositoryID == "" {
		WriteBadRequest(w, "Query parameter 'repository_id' is required")
		return
	}

	repoID, err := strconv.ParseInt(repositoryID, 10, 64)
	if err != nil {
		WriteBadRequest(w, "Invalid repository_id")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.Labels(r.Context(), repoID, q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// SearchTopics handles GET /search/topics
func (h *SearchHandler) SearchTopics(w http.ResponseWriter, r *http.Request) {
	q := QueryParam(r, "q")
	if q == "" {
		WriteBadRequest(w, "Query parameter 'q' is required")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.SearchOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	results, err := h.search.Topics(r.Context(), q, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}
