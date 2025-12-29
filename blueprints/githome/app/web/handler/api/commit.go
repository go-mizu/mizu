package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// parseTime parses an ISO 8601 timestamp (GitHub API format)
func parseTime(s string) (time.Time, error) {
	// Try RFC3339 first (2006-01-02T15:04:05Z)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try date-only format
	return time.Parse("2006-01-02", s)
}

// CommitHandler handles commit endpoints
type CommitHandler struct {
	commits commits.API
	repos   repos.API
}

// NewCommitHandler creates a new commit handler
func NewCommitHandler(commits commits.API, repos repos.API) *CommitHandler {
	return &CommitHandler{commits: commits, repos: repos}
}

// ListCommits handles GET /repos/{owner}/{repo}/commits
func (h *CommitHandler) ListCommits(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	pagination := GetPagination(c)
	opts := &commits.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		SHA:       c.Query("sha"),
		Path:      c.Query("path"),
		Author:    c.Query("author"),
		Committer: c.Query("committer"),
	}

	// Parse since/until time filters
	if since := c.Query("since"); since != "" {
		if t, err := parseTime(since); err == nil {
			opts.Since = t
		}
	}
	if until := c.Query("until"); until != "" {
		if t, err := parseTime(until); err == nil {
			opts.Until = t
		}
	}

	commitList, err := h.commits.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commitList)
}

// GetCommit handles GET /repos/{owner}/{repo}/commits/{ref}
func (h *CommitHandler) GetCommit(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	commit, err := h.commits.Get(c.Context(), owner, repoName, ref)
	if err != nil {
		if err == commits.ErrNotFound {
			return NotFound(c, "Commit")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commit)
}

// CompareCommits handles GET /repos/{owner}/{repo}/compare/{basehead}
func (h *CommitHandler) CompareCommits(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	basehead := c.Param("basehead")

	// basehead format: base...head
	var base, head string
	for i := 0; i < len(basehead)-2; i++ {
		if basehead[i:i+3] == "..." {
			base = basehead[:i]
			head = basehead[i+3:]
			break
		}
	}
	if base == "" || head == "" {
		return BadRequest(c, "Invalid basehead format. Expected: base...head")
	}

	comparison, err := h.commits.Compare(c.Context(), owner, repoName, base, head)
	if err != nil {
		if err == commits.ErrNotFound {
			return NotFound(c, "Commit")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, comparison)
}

// ListBranchesForHead handles GET /repos/{owner}/{repo}/commits/{commit_sha}/branches-where-head
func (h *CommitHandler) ListBranchesForHead(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	commitSHA := c.Param("commit_sha")

	branches, err := h.commits.ListBranchesForHead(c.Context(), owner, repoName, commitSHA)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, branches)
}

// ListPullsForCommit handles GET /repos/{owner}/{repo}/commits/{commit_sha}/pulls
func (h *CommitHandler) ListPullsForCommit(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	commitSHA := c.Param("commit_sha")

	pagination := GetPagination(c)
	opts := &commits.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	pulls, err := h.commits.ListPullsForCommit(c.Context(), owner, repoName, commitSHA, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, pulls)
}

// GetCombinedStatus handles GET /repos/{owner}/{repo}/commits/{ref}/status
func (h *CommitHandler) GetCombinedStatus(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	status, err := h.commits.GetCombinedStatus(c.Context(), owner, repoName, ref)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, status)
}

// ListStatuses handles GET /repos/{owner}/{repo}/commits/{ref}/statuses
func (h *CommitHandler) ListStatuses(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	pagination := GetPagination(c)
	opts := &commits.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	statuses, err := h.commits.ListStatuses(c.Context(), owner, repoName, ref, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, statuses)
}

// CreateStatus handles POST /repos/{owner}/{repo}/statuses/{sha}
func (h *CommitHandler) CreateStatus(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	sha := c.Param("sha")

	var in commits.CreateStatusIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	status, err := h.commits.CreateStatus(c.Context(), owner, repoName, sha, user.ID, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, status)
}
