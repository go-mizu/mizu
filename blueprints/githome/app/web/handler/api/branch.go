package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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

// ListBranches handles GET /repos/{owner}/{repo}/branches
func (h *BranchHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
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
	var protected *bool
	if r.URL.Query().Has("protected") {
		p := QueryParamBool(r, "protected")
		protected = &p
	}
	opts := &branches.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Protected: protected,
	}

	branchList, err := h.branches.List(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, branchList)
}

// GetBranch handles GET /repos/{owner}/{repo}/branches/{branch}
func (h *BranchHandler) GetBranch(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	branch, err := h.branches.Get(r.Context(), owner, repoName, branchName)
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

	branchName := PathParam(r, "branch")

	var in struct {
		NewName string `json:"new_name"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	branch, err := h.branches.Rename(r.Context(), owner, repoName, branchName, in.NewName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch")
			return
		}
		if err == branches.ErrBranchExists {
			WriteConflict(w, "Branch already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, branch)
}

// GetBranchProtection handles GET /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) GetBranchProtection(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	protection, err := h.branches.GetProtection(r.Context(), owner, repoName, branchName)
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

	branchName := PathParam(r, "branch")

	var in branches.UpdateProtectionIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	protection, err := h.branches.UpdateProtection(r.Context(), owner, repoName, branchName, &in)
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

	branchName := PathParam(r, "branch")

	if err := h.branches.DeleteProtection(r.Context(), owner, repoName, branchName); err != nil {
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

	branchName := PathParam(r, "branch")

	checks, err := h.branches.GetRequiredStatusChecks(r.Context(), owner, repoName, branchName)
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

	branchName := PathParam(r, "branch")

	var in branches.RequiredStatusChecksIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	checks, err := h.branches.UpdateRequiredStatusChecks(r.Context(), owner, repoName, branchName, &in)
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

	branchName := PathParam(r, "branch")

	if err := h.branches.RemoveRequiredStatusChecks(r.Context(), owner, repoName, branchName); err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required status checks")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetRequiredSignatures handles GET /repos/{owner}/{repo}/branches/{branch}/protection/required_signatures
func (h *BranchHandler) GetRequiredSignatures(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	setting, err := h.branches.GetRequiredSignatures(r.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required signatures")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, setting)
}

// CreateRequiredSignatures handles POST /repos/{owner}/{repo}/branches/{branch}/protection/required_signatures
func (h *BranchHandler) CreateRequiredSignatures(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	setting, err := h.branches.CreateRequiredSignatures(r.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Branch protection")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, setting)
}

// DeleteRequiredSignatures handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_signatures
func (h *BranchHandler) DeleteRequiredSignatures(w http.ResponseWriter, r *http.Request) {
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

	branchName := PathParam(r, "branch")

	if err := h.branches.DeleteRequiredSignatures(r.Context(), owner, repoName, branchName); err != nil {
		if err == branches.ErrNotFound {
			WriteNotFound(w, "Required signatures")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
