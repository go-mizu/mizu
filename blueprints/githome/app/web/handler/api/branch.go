package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
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
func (h *BranchHandler) ListBranches(c *mizu.Ctx) error {
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
	var protected *bool
	if c.Query("protected") != "" {
		p := QueryBool(c, "protected")
		protected = &p
	}
	opts := &branches.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Protected: protected,
	}

	branchList, err := h.branches.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, branchList)
}

// GetBranch handles GET /repos/{owner}/{repo}/branches/{branch}
func (h *BranchHandler) GetBranch(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	branchName := c.Param("branch")

	branch, err := h.branches.Get(c.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, branch)
}

// RenameBranch handles POST /repos/{owner}/{repo}/branches/{branch}/rename
func (h *BranchHandler) RenameBranch(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	var in struct {
		NewName string `json:"new_name"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	branch, err := h.branches.Rename(c.Context(), owner, repoName, branchName, in.NewName)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch")
		}
		if err == branches.ErrBranchExists {
			return Conflict(c, "Branch already exists")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, branch)
}

// GetBranchProtection handles GET /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) GetBranchProtection(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	protection, err := h.branches.GetProtection(c.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch protection")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, protection)
}

// UpdateBranchProtection handles PUT /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) UpdateBranchProtection(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	var in branches.UpdateProtectionIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	protection, err := h.branches.UpdateProtection(c.Context(), owner, repoName, branchName, &in)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, protection)
}

// DeleteBranchProtection handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection
func (h *BranchHandler) DeleteBranchProtection(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	if err := h.branches.DeleteProtection(c.Context(), owner, repoName, branchName); err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch protection")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetRequiredStatusChecks handles GET /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks
func (h *BranchHandler) GetRequiredStatusChecks(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	checks, err := h.branches.GetRequiredStatusChecks(c.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Required status checks")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, checks)
}

// UpdateRequiredStatusChecks handles PATCH /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks
func (h *BranchHandler) UpdateRequiredStatusChecks(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	var in branches.RequiredStatusChecksIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	checks, err := h.branches.UpdateRequiredStatusChecks(c.Context(), owner, repoName, branchName, &in)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch protection")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, checks)
}

// DeleteRequiredStatusChecks handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks
func (h *BranchHandler) DeleteRequiredStatusChecks(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	if err := h.branches.RemoveRequiredStatusChecks(c.Context(), owner, repoName, branchName); err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Required status checks")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetRequiredSignatures handles GET /repos/{owner}/{repo}/branches/{branch}/protection/required_signatures
func (h *BranchHandler) GetRequiredSignatures(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	setting, err := h.branches.GetRequiredSignatures(c.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Required signatures")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, setting)
}

// CreateRequiredSignatures handles POST /repos/{owner}/{repo}/branches/{branch}/protection/required_signatures
func (h *BranchHandler) CreateRequiredSignatures(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	setting, err := h.branches.CreateRequiredSignatures(c.Context(), owner, repoName, branchName)
	if err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Branch protection")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, setting)
}

// DeleteRequiredSignatures handles DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_signatures
func (h *BranchHandler) DeleteRequiredSignatures(c *mizu.Ctx) error {
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

	branchName := c.Param("branch")

	if err := h.branches.DeleteRequiredSignatures(c.Context(), owner, repoName, branchName); err != nil {
		if err == branches.ErrNotFound {
			return NotFound(c, "Required signatures")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}
