package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/branches"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
)

// BranchHandler handles branch endpoints
type BranchHandler struct {
	branches branches.API
	repos    repos.API
}

// NewBranchHandler creates a new branch handler
func NewBranchHandler(branches branches.API, repos repos.API) *BranchHandler {
	return &BranchHandler{branches: branches, repos: repos}
}

// getRepoFromPath gets repository from path parameters
func (h *BranchHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListBranches handles GET /repos/{owner}/{repo}/branches
func (h *BranchHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
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
	opts := &branches.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Protected: QueryParamBool(r, "protected"),
	}

	branchList, err := h.branches.ListForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, branchList)
}

// GetBranch handles GET /repos/{owner}/{repo}/branches/{branch}
func (h *BranchHandler) GetBranch(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	branchName := PathParam(r, "branch")

	branch, err := h.branches.GetByName(r.Context(), repo.ID, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, branch)
}

// RenameBranch handles POST /repos/{owner}/{repo}/branches/{branch}/rename
func (h *BranchHandler) RenameBranch(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	var in struct {
		NewName string `json:"new_name"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	branch, err := h.branches.Rename(r.Context(), repo.ID, branchName, in.NewName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch")
			return
		}
		if err == branches.ErrExists {
			WriteConflict(w, "Branch already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, branch)
}

// SyncFork handles POST /repos/{owner}/{repo}/merge-upstream
func (h *BranchHandler) SyncFork(w http.ResponseWriter, r *http.Request) {
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

	var in struct {
		Branch string `json:"branch"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	result, err := h.branches.SyncFork(r.Context(), repo.ID, in.Branch)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetBranchProtection handles GET /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) GetBranchProtection(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	protection, err := h.branches.GetProtection(r.Context(), repo.ID, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch protection")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, protection)
}

// UpdateBranchProtection handles PUT /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) UpdateBranchProtection(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	var in branches.UpdateProtectionIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	protection, err := h.branches.UpdateProtection(r.Context(), repo.ID, branchName, &in)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, protection)
}

// DeleteBranchProtection handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) DeleteBranchProtection(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	if err := h.branches.DeleteProtection(r.Context(), repo.ID, branchName); err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch protection")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetRequiredStatusChecks handles GET /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks
func (h *BranchHandler) GetRequiredStatusChecks(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	checks, err := h.branches.GetRequiredStatusChecks(r.Context(), repo.ID, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required status checks")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, checks)
}

// UpdateRequiredStatusChecks handles PATCH /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks
func (h *BranchHandler) UpdateRequiredStatusChecks(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	var in branches.RequiredStatusChecksIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	checks, err := h.branches.UpdateRequiredStatusChecks(r.Context(), repo.ID, branchName, &in)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch protection")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, checks)
}

// DeleteRequiredStatusChecks handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks
func (h *BranchHandler) DeleteRequiredStatusChecks(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	if err := h.branches.DeleteRequiredStatusChecks(r.Context(), repo.ID, branchName); err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required status checks")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetRequiredPullRequestReviews handles GET /repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews
func (h *BranchHandler) GetRequiredPullRequestReviews(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	reviews, err := h.branches.GetRequiredPullRequestReviews(r.Context(), repo.ID, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required pull request reviews")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reviews)
}

// UpdateRequiredPullRequestReviews handles PATCH /repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews
func (h *BranchHandler) UpdateRequiredPullRequestReviews(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	var in branches.RequiredPullRequestReviewsIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reviews, err := h.branches.UpdateRequiredPullRequestReviews(r.Context(), repo.ID, branchName, &in)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch protection")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reviews)
}

// DeleteRequiredPullRequestReviews handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews
func (h *BranchHandler) DeleteRequiredPullRequestReviews(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	if err := h.branches.DeleteRequiredPullRequestReviews(r.Context(), repo.ID, branchName); err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required pull request reviews")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetAdminEnforcement handles GET /repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins
func (h *BranchHandler) GetAdminEnforcement(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	enforcement, err := h.branches.GetAdminEnforcement(r.Context(), repo.ID, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Admin enforcement")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, enforcement)
}

// SetAdminEnforcement handles POST /repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins
func (h *BranchHandler) SetAdminEnforcement(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	enforcement, err := h.branches.SetAdminEnforcement(r.Context(), repo.ID, branchName, true)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch protection")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, enforcement)
}

// DeleteAdminEnforcement handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins
func (h *BranchHandler) DeleteAdminEnforcement(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	if err := h.branches.DeleteAdminEnforcement(r.Context(), repo.ID, branchName); err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Admin enforcement")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
