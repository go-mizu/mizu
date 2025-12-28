package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// IssueHandler handles issue endpoints
type IssueHandler struct {
	issues issues.API
	repos  repos.API
}

// NewIssueHandler creates a new issue handler
func NewIssueHandler(issues issues.API, repos repos.API) *IssueHandler {
	return &IssueHandler{issues: issues, repos: repos}
}

// ListRepoIssues handles GET /repos/{owner}/{repo}/issues
func (h *IssueHandler) ListRepoIssues(c *mizu.Ctx) error {
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
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     c.Query("state"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
		Labels:    c.Query("labels"),
		Milestone: c.Query("milestone"),
		Assignee:  c.Query("assignee"),
		Creator:   c.Query("creator"),
		Mentioned: c.Query("mentioned"),
	}

	issueList, err := h.issues.ListForRepo(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, issueList)
}

// GetIssue handles GET /repos/{owner}/{repo}/issues/{issue_number}
func (h *IssueHandler) GetIssue(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	issue, err := h.issues.Get(c.Context(), owner, repoName, issueNumber)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, issue)
}

// CreateIssue handles POST /repos/{owner}/{repo}/issues
func (h *IssueHandler) CreateIssue(c *mizu.Ctx) error {
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

	var in issues.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	issue, err := h.issues.Create(c.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, issue)
}

// UpdateIssue handles PATCH /repos/{owner}/{repo}/issues/{issue_number}
func (h *IssueHandler) UpdateIssue(c *mizu.Ctx) error {
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

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	var in issues.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.issues.Update(c.Context(), owner, repoName, issueNumber, &in)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// LockIssue handles PUT /repos/{owner}/{repo}/issues/{issue_number}/lock
func (h *IssueHandler) LockIssue(c *mizu.Ctx) error {
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

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	var in struct {
		LockReason string `json:"lock_reason,omitempty"`
	}
	c.BindJSON(&in, 1<<20) // optional body

	if err := h.issues.Lock(c.Context(), owner, repoName, issueNumber, in.LockReason); err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// UnlockIssue handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/lock
func (h *IssueHandler) UnlockIssue(c *mizu.Ctx) error {
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

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	if err := h.issues.Unlock(c.Context(), owner, repoName, issueNumber); err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListIssueAssignees handles GET /repos/{owner}/{repo}/assignees
func (h *IssueHandler) ListIssueAssignees(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	assignees, err := h.issues.ListAssignees(c.Context(), owner, repoName)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, assignees)
}

// CheckAssignee handles GET /repos/{owner}/{repo}/assignees/{assignee}
func (h *IssueHandler) CheckAssignee(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	assignee := c.Param("assignee")

	isAssignable, err := h.issues.CheckAssignee(c.Context(), owner, repoName, assignee)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isAssignable {
		return NoContent(c)
	}
	return NotFound(c, "Assignee")
}

// AddAssignees handles POST /repos/{owner}/{repo}/issues/{issue_number}/assignees
func (h *IssueHandler) AddAssignees(c *mizu.Ctx) error {
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

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	var in struct {
		Assignees []string `json:"assignees"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.issues.AddAssignees(c.Context(), owner, repoName, issueNumber, in.Assignees)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, updated)
}

// RemoveAssignees handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/assignees
func (h *IssueHandler) RemoveAssignees(c *mizu.Ctx) error {
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

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	var in struct {
		Assignees []string `json:"assignees"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.issues.RemoveAssignees(c.Context(), owner, repoName, issueNumber, in.Assignees)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// ListAuthenticatedUserIssues handles GET /user/issues
func (h *IssueHandler) ListAuthenticatedUserIssues(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     c.Query("state"),
		Labels:    c.Query("labels"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	issueList, err := h.issues.ListForUser(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, issueList)
}

// ListIssues handles GET /issues (global issues for authenticated user)
func (h *IssueHandler) ListIssues(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     c.Query("state"),
		Labels:    c.Query("labels"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	issueList, err := h.issues.ListForUser(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, issueList)
}

// ListOrgIssues handles GET /orgs/{org}/issues
func (h *IssueHandler) ListOrgIssues(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	pagination := GetPagination(c)
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     c.Query("state"),
		Labels:    c.Query("labels"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	issueList, err := h.issues.ListForOrg(c.Context(), org, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, issueList)
}

// ListIssueEvents handles GET /repos/{owner}/{repo}/issues/{issue_number}/events
func (h *IssueHandler) ListIssueEvents(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	issueNumber, err := ParamInt(c, "issue_number")
	if err != nil {
		return BadRequest(c, "Invalid issue number")
	}

	pagination := GetPagination(c)
	opts := &issues.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.issues.ListEvents(c.Context(), owner, repoName, issueNumber, opts)
	if err != nil {
		if err == issues.ErrNotFound {
			return NotFound(c, "Issue")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

// GetIssueEvent handles GET /repos/{owner}/{repo}/issues/events/{event_id}
func (h *IssueHandler) GetIssueEvent(c *mizu.Ctx) error {
	// TODO: Implement when service method is available
	return NotFound(c, "Issue event")
}

// ListRepoIssueEvents handles GET /repos/{owner}/{repo}/issues/events
func (h *IssueHandler) ListRepoIssueEvents(c *mizu.Ctx) error {
	// TODO: Implement when service method is available
	return c.JSON(http.StatusOK, []any{})
}

// ListIssueTimeline handles GET /repos/{owner}/{repo}/issues/{issue_number}/timeline
func (h *IssueHandler) ListIssueTimeline(c *mizu.Ctx) error {
	// TODO: Implement when service method is available
	return c.JSON(http.StatusOK, []any{})
}
