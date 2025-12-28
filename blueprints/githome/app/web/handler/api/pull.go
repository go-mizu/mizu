package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// PullHandler handles pull request endpoints
type PullHandler struct {
	pulls pulls.API
	repos repos.API
}

// NewPullHandler creates a new pull handler
func NewPullHandler(pulls pulls.API, repos repos.API) *PullHandler {
	return &PullHandler{pulls: pulls, repos: repos}
}

// ListPulls handles GET /repos/{owner}/{repo}/pulls
func (h *PullHandler) ListPulls(c *mizu.Ctx) error {
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
	opts := &pulls.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     c.Query("state"),
		Head:      c.Query("head"),
		Base:      c.Query("base"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	pullList, err := h.pulls.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, pullList)
}

// GetPull handles GET /repos/{owner}/{repo}/pulls/{pull_number}
func (h *PullHandler) GetPull(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	pull, err := h.pulls.Get(c.Context(), owner, repoName, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, pull)
}

// CreatePull handles POST /repos/{owner}/{repo}/pulls
func (h *PullHandler) CreatePull(c *mizu.Ctx) error {
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

	var in pulls.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	pull, err := h.pulls.Create(c.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, pull)
}

// UpdatePull handles PATCH /repos/{owner}/{repo}/pulls/{pull_number}
func (h *PullHandler) UpdatePull(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	var in pulls.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.pulls.Update(c.Context(), owner, repoName, pullNumber, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// ListPullCommits handles GET /repos/{owner}/{repo}/pulls/{pull_number}/commits
func (h *PullHandler) ListPullCommits(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	pagination := GetPagination(c)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commits, err := h.pulls.ListCommits(c.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commits)
}

// ListPullFiles handles GET /repos/{owner}/{repo}/pulls/{pull_number}/files
func (h *PullHandler) ListPullFiles(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	pagination := GetPagination(c)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	files, err := h.pulls.ListFiles(c.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, files)
}

// CheckPullMerged handles GET /repos/{owner}/{repo}/pulls/{pull_number}/merge
func (h *PullHandler) CheckPullMerged(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	merged, err := h.pulls.IsMerged(c.Context(), owner, repoName, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if merged {
		return NoContent(c)
	}
	return NotFound(c, "Pull Request not merged")
}

// MergePull handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge
func (h *PullHandler) MergePull(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	var in pulls.MergeIn
	c.BindJSON(&in, 1<<20) // optional body

	result, err := h.pulls.Merge(c.Context(), owner, repoName, pullNumber, &in)
	if err != nil {
		if err == pulls.ErrNotMergeable {
			return WriteError(c, http.StatusMethodNotAllowed, "Pull Request is not mergeable")
		}
		if err == pulls.ErrAlreadyMerged {
			return WriteError(c, http.StatusMethodNotAllowed, "Pull Request already merged")
		}
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// UpdatePullBranch handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/update-branch
func (h *PullHandler) UpdatePullBranch(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	if err := h.pulls.UpdateBranch(c.Context(), owner, repoName, pullNumber); err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, map[string]string{"message": "Updating branch"})
}

// ListPullReviews handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews
func (h *PullHandler) ListPullReviews(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	pagination := GetPagination(c)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	reviews, err := h.pulls.ListReviews(c.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reviews)
}

// GetPullReview handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) GetPullReview(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	reviewID, err := ParamInt64(c, "review_id")
	if err != nil {
		return BadRequest(c, "Invalid review ID")
	}

	review, err := h.pulls.GetReview(c.Context(), owner, repoName, pullNumber, reviewID)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Review")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, review)
}

// CreatePullReview handles POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews
func (h *PullHandler) CreatePullReview(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	var in pulls.CreateReviewIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	review, err := h.pulls.CreateReview(c.Context(), owner, repoName, pullNumber, user.ID, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, review)
}

// UpdatePullReview handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) UpdatePullReview(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	reviewID, err := ParamInt64(c, "review_id")
	if err != nil {
		return BadRequest(c, "Invalid review ID")
	}

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	review, err := h.pulls.UpdateReview(c.Context(), owner, repoName, pullNumber, reviewID, in.Body)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Review")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, review)
}

// SubmitPullReview handles POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events
func (h *PullHandler) SubmitPullReview(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	reviewID, err := ParamInt64(c, "review_id")
	if err != nil {
		return BadRequest(c, "Invalid review ID")
	}

	var in pulls.SubmitReviewIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	review, err := h.pulls.SubmitReview(c.Context(), owner, repoName, pullNumber, reviewID, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Review")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, review)
}

// DismissPullReview handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/dismissals
func (h *PullHandler) DismissPullReview(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	reviewID, err := ParamInt64(c, "review_id")
	if err != nil {
		return BadRequest(c, "Invalid review ID")
	}

	var in struct {
		Message string `json:"message"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	review, err := h.pulls.DismissReview(c.Context(), owner, repoName, pullNumber, reviewID, in.Message)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Review")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, review)
}

// ListPullReviewComments handles GET /repos/{owner}/{repo}/pulls/{pull_number}/comments
func (h *PullHandler) ListPullReviewComments(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	pagination := GetPagination(c)
	opts := &pulls.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	comments, err := h.pulls.ListReviewComments(c.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, comments)
}

// CreatePullReviewComment handles POST /repos/{owner}/{repo}/pulls/{pull_number}/comments
func (h *PullHandler) CreatePullReviewComment(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	var in pulls.CreateReviewCommentIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	comment, err := h.pulls.CreateReviewComment(c.Context(), owner, repoName, pullNumber, user.ID, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, comment)
}

// RequestReviewers handles POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) RequestReviewers(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	var in struct {
		Reviewers     []string `json:"reviewers,omitempty"`
		TeamReviewers []string `json:"team_reviewers,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.pulls.RequestReviewers(c.Context(), owner, repoName, pullNumber, in.Reviewers, in.TeamReviewers)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, updated)
}

// RemoveRequestedReviewers handles DELETE /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) RemoveRequestedReviewers(c *mizu.Ctx) error {
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

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	var in struct {
		Reviewers     []string `json:"reviewers,omitempty"`
		TeamReviewers []string `json:"team_reviewers,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.pulls.RemoveReviewers(c.Context(), owner, repoName, pullNumber, in.Reviewers, in.TeamReviewers)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// ListRequestedReviewers handles GET /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) ListRequestedReviewers(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pullNumber, err := ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	// Get PR to access requested reviewers from the model
	pr, err := h.pulls.Get(c.Context(), owner, repoName, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			return NotFound(c, "Pull Request")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"users": pr.RequestedReviewers,
		"teams": pr.RequestedTeams,
	})
}

// DeletePullReview handles DELETE /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) DeletePullReview(c *mizu.Ctx) error {
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

	_, err = ParamInt(c, "pull_number")
	if err != nil {
		return BadRequest(c, "Invalid pull number")
	}

	_, err = ParamInt64(c, "review_id")
	if err != nil {
		return BadRequest(c, "Invalid review ID")
	}

	// TODO: Implement when pulls.API.DeleteReview is available
	return WriteError(c, http.StatusNotImplemented, "Delete review not implemented")
}

// ListReviewComments handles GET /repos/{owner}/{repo}/pulls/comments
func (h *PullHandler) ListReviewComments(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	// TODO: Implement when pulls.API.ListAllReviewComments is available
	// This lists all review comments for a repo (not a specific PR)
	return c.JSON(http.StatusOK, []*pulls.ReviewComment{})
}

// GetPullReviewComment handles GET /repos/{owner}/{repo}/pulls/comments/{comment_id}
func (h *PullHandler) GetPullReviewComment(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	_, err = ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	// TODO: Implement when pulls.API.GetReviewComment is available
	return NotFound(c, "Review Comment")
}

// UpdatePullReviewComment handles PATCH /repos/{owner}/{repo}/pulls/comments/{comment_id}
func (h *PullHandler) UpdatePullReviewComment(c *mizu.Ctx) error {
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

	_, err = ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	// TODO: Implement when pulls.API.UpdateReviewComment is available
	return NotFound(c, "Review Comment")
}

// DeletePullReviewComment handles DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}
func (h *PullHandler) DeletePullReviewComment(c *mizu.Ctx) error {
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

	_, err = ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	// TODO: Implement when pulls.API.DeleteReviewComment is available
	return NotFound(c, "Review Comment")
}
