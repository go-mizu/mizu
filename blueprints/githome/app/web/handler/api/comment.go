package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// CommentHandler handles comment endpoints
type CommentHandler struct {
	comments comments.API
	repos    repos.API
}

// NewCommentHandler creates a new comment handler
func NewCommentHandler(comments comments.API, repos repos.API) *CommentHandler {
	return &CommentHandler{comments: comments, repos: repos}
}

// ListIssueComments handles GET /repos/{owner}/{repo}/issues/{issue_number}/comments
func (h *CommentHandler) ListIssueComments(c *mizu.Ctx) error {
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
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListForIssue(c.Context(), owner, repoName, issueNumber, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commentList)
}

// GetIssueComment handles GET /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) GetIssueComment(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	commentID, err := ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	comment, err := h.comments.GetIssueComment(c.Context(), owner, repoName, commentID)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, comment)
}

// CreateIssueComment handles POST /repos/{owner}/{repo}/issues/{issue_number}/comments
func (h *CommentHandler) CreateIssueComment(c *mizu.Ctx) error {
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
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	comment, err := h.comments.CreateIssueComment(c.Context(), owner, repoName, issueNumber, user.ID, in.Body)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, comment)
}

// UpdateIssueComment handles PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) UpdateIssueComment(c *mizu.Ctx) error {
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

	commentID, err := ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	comment, err := h.comments.UpdateIssueComment(c.Context(), owner, repoName, commentID, in.Body)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, comment)
}

// DeleteIssueComment handles DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) DeleteIssueComment(c *mizu.Ctx) error {
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

	commentID, err := ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	if err := h.comments.DeleteIssueComment(c.Context(), owner, repoName, commentID); err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListRepoComments handles GET /repos/{owner}/{repo}/issues/comments
func (h *CommentHandler) ListRepoComments(c *mizu.Ctx) error {
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
	opts := &comments.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	commentList, err := h.comments.ListForRepo(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commentList)
}

// ListCommitComments handles GET /repos/{owner}/{repo}/commits/{commit_sha}/comments
func (h *CommentHandler) ListCommitComments(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	commitSHA := c.Param("commit_sha")

	pagination := GetPagination(c)
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListForCommit(c.Context(), owner, repoName, commitSHA, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commentList)
}

// CreateCommitComment handles POST /repos/{owner}/{repo}/commits/{commit_sha}/comments
func (h *CommentHandler) CreateCommitComment(c *mizu.Ctx) error {
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

	commitSHA := c.Param("commit_sha")

	var in comments.CreateCommitCommentIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	comment, err := h.comments.CreateCommitComment(c.Context(), owner, repoName, commitSHA, user.ID, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, comment)
}

// GetCommitComment handles GET /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) GetCommitComment(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	commentID, err := ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	comment, err := h.comments.GetCommitComment(c.Context(), owner, repoName, commentID)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, comment)
}

// UpdateCommitComment handles PATCH /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) UpdateCommitComment(c *mizu.Ctx) error {
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

	commentID, err := ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	comment, err := h.comments.UpdateCommitComment(c.Context(), owner, repoName, commentID, in.Body)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, comment)
}

// DeleteCommitComment handles DELETE /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) DeleteCommitComment(c *mizu.Ctx) error {
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

	commentID, err := ParamInt64(c, "comment_id")
	if err != nil {
		return BadRequest(c, "Invalid comment ID")
	}

	if err := h.comments.DeleteCommitComment(c.Context(), owner, repoName, commentID); err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListRepoCommitComments handles GET /repos/{owner}/{repo}/comments
func (h *CommentHandler) ListRepoCommitComments(c *mizu.Ctx) error {
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
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListCommitCommentsForRepo(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commentList)
}
