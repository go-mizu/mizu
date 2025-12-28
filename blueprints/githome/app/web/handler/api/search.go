package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/search"
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
	opts := &search.RepoSearchOpts{
		Query:     q,
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Order:     QueryParam(r, "order"),
	}

	results, err := h.search.SearchRepositories(r.Context(), opts)
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
	opts := &search.CodeSearchOpts{
		Query:   q,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.SearchCode(r.Context(), opts)
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
	opts := &search.CommitSearchOpts{
		Query:   q,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.SearchCommits(r.Context(), opts)
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
	opts := &search.IssueSearchOpts{
		Query:   q,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.SearchIssues(r.Context(), opts)
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
	opts := &search.UserSearchOpts{
		Query:   q,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    QueryParam(r, "sort"),
		Order:   QueryParam(r, "order"),
	}

	results, err := h.search.SearchUsers(r.Context(), opts)
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

	repoID, err := PathParamInt64(r, "repository_id")
	if err != nil {
		WriteBadRequest(w, "Invalid repository_id")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &search.LabelSearchOpts{
		Query:        q,
		RepositoryID: repoID,
		Page:         pagination.Page,
		PerPage:      pagination.PerPage,
		Sort:         QueryParam(r, "sort"),
		Order:        QueryParam(r, "order"),
	}

	results, err := h.search.SearchLabels(r.Context(), opts)
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
	opts := &search.TopicSearchOpts{
		Query:   q,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	results, err := h.search.SearchTopics(r.Context(), opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, results)
}
