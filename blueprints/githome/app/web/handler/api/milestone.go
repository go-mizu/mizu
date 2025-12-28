package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// MilestoneHandler handles milestone endpoints
type MilestoneHandler struct {
	milestones milestones.API
	repos      repos.API
}

// NewMilestoneHandler creates a new milestone handler
func NewMilestoneHandler(milestones milestones.API, repos repos.API) *MilestoneHandler {
	return &MilestoneHandler{milestones: milestones, repos: repos}
}

// ListMilestones handles GET /repos/{owner}/{repo}/milestones
func (h *MilestoneHandler) ListMilestones(c *mizu.Ctx) error {
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
	opts := &milestones.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     c.Query("state"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	milestoneList, err := h.milestones.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, milestoneList)
}

// GetMilestone handles GET /repos/{owner}/{repo}/milestones/{milestone_number}
func (h *MilestoneHandler) GetMilestone(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	milestoneNumber, err := ParamInt(c, "milestone_number")
	if err != nil {
		return BadRequest(c, "Invalid milestone number")
	}

	milestone, err := h.milestones.Get(c.Context(), owner, repoName, milestoneNumber)
	if err != nil {
		if err == milestones.ErrNotFound {
			return NotFound(c, "Milestone")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, milestone)
}

// CreateMilestone handles POST /repos/{owner}/{repo}/milestones
func (h *MilestoneHandler) CreateMilestone(c *mizu.Ctx) error {
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

	var in milestones.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	milestone, err := h.milestones.Create(c.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, milestone)
}

// UpdateMilestone handles PATCH /repos/{owner}/{repo}/milestones/{milestone_number}
func (h *MilestoneHandler) UpdateMilestone(c *mizu.Ctx) error {
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

	milestoneNumber, err := ParamInt(c, "milestone_number")
	if err != nil {
		return BadRequest(c, "Invalid milestone number")
	}

	var in milestones.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.milestones.Update(c.Context(), owner, repoName, milestoneNumber, &in)
	if err != nil {
		if err == milestones.ErrNotFound {
			return NotFound(c, "Milestone")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteMilestone handles DELETE /repos/{owner}/{repo}/milestones/{milestone_number}
func (h *MilestoneHandler) DeleteMilestone(c *mizu.Ctx) error {
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

	milestoneNumber, err := ParamInt(c, "milestone_number")
	if err != nil {
		return BadRequest(c, "Invalid milestone number")
	}

	if err := h.milestones.Delete(c.Context(), owner, repoName, milestoneNumber); err != nil {
		if err == milestones.ErrNotFound {
			return NotFound(c, "Milestone")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}
