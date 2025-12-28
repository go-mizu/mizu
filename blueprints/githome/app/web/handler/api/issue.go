package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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
func (h *IssueHandler) ListRepoIssues(w http.ResponseWriter, r *http.Request) {
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
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
		Labels:    QueryParam(r, "labels"),
		Milestone: QueryParam(r, "milestone"),
		Assignee:  QueryParam(r, "assignee"),
		Creator:   QueryParam(r, "creator"),
		Mentioned: QueryParam(r, "mentioned"),
	}

	issueList, err := h.issues.ListForRepo(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, issueList)
}

// GetIssue handles GET /repos/{owner}/{repo}/issues/{issue_number}
func (h *IssueHandler) GetIssue(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	issue, err := h.issues.Get(r.Context(), owner, repoName, issueNumber)
	if err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, issue)
}

// CreateIssue handles POST /repos/{owner}/{repo}/issues
func (h *IssueHandler) CreateIssue(w http.ResponseWriter, r *http.Request) {
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

	var in issues.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	issue, err := h.issues.Create(r.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, issue)
}

// UpdateIssue handles PATCH /repos/{owner}/{repo}/issues/{issue_number}
func (h *IssueHandler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	var in issues.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.issues.Update(r.Context(), owner, repoName, issueNumber, &in)
	if err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// LockIssue handles PUT /repos/{owner}/{repo}/issues/{issue_number}/lock
func (h *IssueHandler) LockIssue(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	var in struct {
		LockReason string `json:"lock_reason,omitempty"`
	}
	DecodeJSON(r, &in) // optional body

	if err := h.issues.Lock(r.Context(), owner, repoName, issueNumber, in.LockReason); err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// UnlockIssue handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/lock
func (h *IssueHandler) UnlockIssue(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	if err := h.issues.Unlock(r.Context(), owner, repoName, issueNumber); err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListIssueAssignees handles GET /repos/{owner}/{repo}/assignees
func (h *IssueHandler) ListIssueAssignees(w http.ResponseWriter, r *http.Request) {
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

	assignees, err := h.issues.ListAssignees(r.Context(), owner, repoName)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, assignees)
}

// CheckAssignee handles GET /repos/{owner}/{repo}/assignees/{assignee}
func (h *IssueHandler) CheckAssignee(w http.ResponseWriter, r *http.Request) {
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

	assignee := PathParam(r, "assignee")

	isAssignable, err := h.issues.CheckAssignee(r.Context(), owner, repoName, assignee)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isAssignable {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Assignee")
	}
}

// AddAssignees handles POST /repos/{owner}/{repo}/issues/{issue_number}/assignees
func (h *IssueHandler) AddAssignees(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	var in struct {
		Assignees []string `json:"assignees"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.issues.AddAssignees(r.Context(), owner, repoName, issueNumber, in.Assignees)
	if err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, updated)
}

// RemoveAssignees handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/assignees
func (h *IssueHandler) RemoveAssignees(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	var in struct {
		Assignees []string `json:"assignees"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.issues.RemoveAssignees(r.Context(), owner, repoName, issueNumber, in.Assignees)
	if err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// ListAuthenticatedUserIssues handles GET /user/issues
func (h *IssueHandler) ListAuthenticatedUserIssues(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Labels:    QueryParam(r, "labels"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	issueList, err := h.issues.ListForUser(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, issueList)
}

// ListIssues handles GET /issues (global issues for authenticated user)
func (h *IssueHandler) ListIssues(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Labels:    QueryParam(r, "labels"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	issueList, err := h.issues.ListForUser(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, issueList)
}

// ListOrgIssues handles GET /orgs/{org}/issues
func (h *IssueHandler) ListOrgIssues(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	pagination := GetPaginationParams(r)
	opts := &issues.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Labels:    QueryParam(r, "labels"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	issueList, err := h.issues.ListForOrg(r.Context(), org, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, issueList)
}

// ListIssueEvents handles GET /repos/{owner}/{repo}/issues/{issue_number}/events
func (h *IssueHandler) ListIssueEvents(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &issues.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	events, err := h.issues.ListEvents(r.Context(), owner, repoName, issueNumber, opts)
	if err != nil {
		if err == issues.ErrNotFound {
			WriteNotFound(w, "Issue")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}
