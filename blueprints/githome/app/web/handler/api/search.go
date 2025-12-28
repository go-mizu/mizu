package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/search"
	"github.com/go-mizu/mizu"
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
func (h *SearchHandler) SearchRepositories(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	pagination := GetPagination(c)
	opts := &search.SearchReposOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
	}

	results, err := h.search.Repositories(c.Context(), q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// SearchCode handles GET /search/code
func (h *SearchHandler) SearchCode(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	pagination := GetPagination(c)
	opts := &search.SearchCodeOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
	}

	results, err := h.search.Code(c.Context(), q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// SearchCommits handles GET /search/commits
func (h *SearchHandler) SearchCommits(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	pagination := GetPagination(c)
	opts := &search.SearchOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
	}

	results, err := h.search.Commits(c.Context(), q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// SearchIssues handles GET /search/issues
func (h *SearchHandler) SearchIssues(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	pagination := GetPagination(c)
	opts := &search.SearchIssuesOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
	}

	results, err := h.search.IssuesAndPullRequests(c.Context(), q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// SearchUsers handles GET /search/users
func (h *SearchHandler) SearchUsers(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	pagination := GetPagination(c)
	opts := &search.SearchUsersOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
	}

	results, err := h.search.Users(c.Context(), q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// SearchLabels handles GET /search/labels
func (h *SearchHandler) SearchLabels(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	repositoryID := c.Query("repository_id")
	if repositoryID == "" {
		return BadRequest(c, "Query parameter 'repository_id' is required")
	}

	repoID, err := strconv.ParseInt(repositoryID, 10, 64)
	if err != nil {
		return BadRequest(c, "Invalid repository_id")
	}

	pagination := GetPagination(c)
	opts := &search.SearchOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
	}

	results, err := h.search.Labels(c.Context(), repoID, q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// SearchTopics handles GET /search/topics
func (h *SearchHandler) SearchTopics(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return BadRequest(c, "Query parameter 'q' is required")
	}

	pagination := GetPagination(c)
	opts := &search.SearchOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	results, err := h.search.Topics(c.Context(), q, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}
