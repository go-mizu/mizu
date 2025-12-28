package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

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
func (h *CommitHandler) ListCommits(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	pagination := GetPaginationParams(r)
	opts := &commits.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		SHA:     QueryParam(r, "sha"),
		Path:    QueryParam(r, "path"),
		Author:  QueryParam(r, "author"),
	}

	commitList, err := h.commits.List(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commitList)
}

// GetCommit handles GET /repos/{owner}/{repo}/commits/{ref}
func (h *CommitHandler) GetCommit(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	commit, err := h.commits.Get(r.Context(), owner, repoName, ref)
	if err != nil {
		if err == commits.ErrNotFound {
			WriteNotFound(w, "Commit")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commit)
}

// CompareCommits handles GET /repos/{owner}/{repo}/compare/{basehead}
func (h *CommitHandler) CompareCommits(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	basehead := PathParam(r, "basehead")

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
		WriteBadRequest(w, "Invalid basehead format. Expected: base...head")
		return
	}

	comparison, err := h.commits.Compare(r.Context(), owner, repoName, base, head)
	if err != nil {
		if err == commits.ErrNotFound {
			WriteNotFound(w, "Commit")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comparison)
}

// ListBranchesForHead handles GET /repos/{owner}/{repo}/commits/{commit_sha}/branches-where-head
func (h *CommitHandler) ListBranchesForHead(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	commitSHA := PathParam(r, "commit_sha")

	branches, err := h.commits.ListBranchesForHead(r.Context(), owner, repoName, commitSHA)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, branches)
}

// ListPullsForCommit handles GET /repos/{owner}/{repo}/commits/{commit_sha}/pulls
func (h *CommitHandler) ListPullsForCommit(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	commitSHA := PathParam(r, "commit_sha")

	pagination := GetPaginationParams(r)
	opts := &commits.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	pulls, err := h.commits.ListPullsForCommit(r.Context(), owner, repoName, commitSHA, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, pulls)
}

// GetCombinedStatus handles GET /repos/{owner}/{repo}/commits/{ref}/status
func (h *CommitHandler) GetCombinedStatus(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	status, err := h.commits.GetCombinedStatus(r.Context(), owner, repoName, ref)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, status)
}

// ListStatuses handles GET /repos/{owner}/{repo}/commits/{ref}/statuses
func (h *CommitHandler) ListStatuses(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	pagination := GetPaginationParams(r)
	opts := &commits.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	statuses, err := h.commits.ListStatuses(r.Context(), owner, repoName, ref, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, statuses)
}

// CreateStatus handles POST /repos/{owner}/{repo}/statuses/{sha}
func (h *CommitHandler) CreateStatus(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	sha := PathParam(r, "sha")

	var in commits.CreateStatusIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	status, err := h.commits.CreateStatus(r.Context(), owner, repoName, sha, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, status)
}
