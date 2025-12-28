package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// LabelHandler handles label endpoints
type LabelHandler struct {
	labels labels.API
	repos  repos.API
}

// NewLabelHandler creates a new label handler
func NewLabelHandler(labels labels.API, repos repos.API) *LabelHandler {
	return &LabelHandler{labels: labels, repos: repos}
}

// ListRepoLabels handles GET /repos/{owner}/{repo}/labels
func (h *LabelHandler) ListRepoLabels(c *mizu.Ctx) error {
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
	opts := &labels.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	labelList, err := h.labels.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, labelList)
}

// GetLabel handles GET /repos/{owner}/{repo}/labels/{name}
func (h *LabelHandler) GetLabel(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	name := c.Param("name")

	label, err := h.labels.Get(c.Context(), owner, repoName, name)
	if err != nil {
		if err == labels.ErrNotFound {
			return NotFound(c, "Label")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, label)
}

// CreateLabel handles POST /repos/{owner}/{repo}/labels
func (h *LabelHandler) CreateLabel(c *mizu.Ctx) error {
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

	var in labels.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	label, err := h.labels.Create(c.Context(), owner, repoName, &in)
	if err != nil {
		if err == labels.ErrLabelExists {
			return Conflict(c, "Label already exists")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, label)
}

// UpdateLabel handles PATCH /repos/{owner}/{repo}/labels/{name}
func (h *LabelHandler) UpdateLabel(c *mizu.Ctx) error {
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

	name := c.Param("name")

	var in labels.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.labels.Update(c.Context(), owner, repoName, name, &in)
	if err != nil {
		if err == labels.ErrNotFound {
			return NotFound(c, "Label")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteLabel handles DELETE /repos/{owner}/{repo}/labels/{name}
func (h *LabelHandler) DeleteLabel(c *mizu.Ctx) error {
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

	name := c.Param("name")

	if err := h.labels.Delete(c.Context(), owner, repoName, name); err != nil {
		if err == labels.ErrNotFound {
			return NotFound(c, "Label")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListIssueLabels handles GET /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) ListIssueLabels(c *mizu.Ctx) error {
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
	opts := &labels.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	labelList, err := h.labels.ListForIssue(c.Context(), owner, repoName, issueNumber, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, labelList)
}

// AddIssueLabels handles POST /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) AddIssueLabels(c *mizu.Ctx) error {
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
		Labels []string `json:"labels"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	labelList, err := h.labels.AddToIssue(c.Context(), owner, repoName, issueNumber, in.Labels)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, labelList)
}

// SetIssueLabels handles PUT /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) SetIssueLabels(c *mizu.Ctx) error {
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
		Labels []string `json:"labels"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	labelList, err := h.labels.SetForIssue(c.Context(), owner, repoName, issueNumber, in.Labels)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, labelList)
}

// RemoveAllIssueLabels handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) RemoveAllIssueLabels(c *mizu.Ctx) error {
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

	if err := h.labels.RemoveAllFromIssue(c.Context(), owner, repoName, issueNumber); err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// RemoveIssueLabel handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels/{name}
func (h *LabelHandler) RemoveIssueLabel(c *mizu.Ctx) error {
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

	name := c.Param("name")

	if err := h.labels.RemoveFromIssue(c.Context(), owner, repoName, issueNumber, name); err != nil {
		if err == labels.ErrNotFound {
			return NotFound(c, "Label")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListLabelsForMilestone handles GET /repos/{owner}/{repo}/milestones/{milestone_number}/labels
func (h *LabelHandler) ListLabelsForMilestone(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &labels.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	labelList, err := h.labels.ListForMilestone(c.Context(), owner, repoName, milestoneNumber, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, labelList)
}
