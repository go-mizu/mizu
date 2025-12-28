package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/reactions"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// ReactionHandler handles reaction endpoints
type ReactionHandler struct {
	reactions reactions.API
	repos     repos.API
}

// NewReactionHandler creates a new reaction handler
func NewReactionHandler(reactions reactions.API, repos repos.API) *ReactionHandler {
	return &ReactionHandler{reactions: reactions, repos: repos}
}

// ListIssueReactions handles GET /repos/{owner}/{repo}/issues/{issue_number}/reactions
func (h *ReactionHandler) ListIssueReactions(c *mizu.Ctx) error {
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
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: c.Query("content"),
	}

	reactionList, err := h.reactions.ListForIssue(c.Context(), owner, repoName, issueNumber, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reactionList)
}

// CreateIssueReaction handles POST /repos/{owner}/{repo}/issues/{issue_number}/reactions
func (h *ReactionHandler) CreateIssueReaction(c *mizu.Ctx) error {
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
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reaction, err := h.reactions.CreateForIssue(c.Context(), owner, repoName, issueNumber, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			return BadRequest(c, "Invalid reaction content")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, reaction)
}

// DeleteIssueReaction handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteIssueReaction(c *mizu.Ctx) error {
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

	reactionID, err := ParamInt64(c, "reaction_id")
	if err != nil {
		return BadRequest(c, "Invalid reaction ID")
	}

	if err := h.reactions.DeleteForIssue(c.Context(), owner, repoName, issueNumber, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			return NotFound(c, "Reaction")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListIssueCommentReactions handles GET /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions
func (h *ReactionHandler) ListIssueCommentReactions(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: c.Query("content"),
	}

	reactionList, err := h.reactions.ListForIssueComment(c.Context(), owner, repoName, commentID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reactionList)
}

// CreateIssueCommentReaction handles POST /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions
func (h *ReactionHandler) CreateIssueCommentReaction(c *mizu.Ctx) error {
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
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reaction, err := h.reactions.CreateForIssueComment(c.Context(), owner, repoName, commentID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			return BadRequest(c, "Invalid reaction content")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, reaction)
}

// DeleteIssueCommentReaction handles DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteIssueCommentReaction(c *mizu.Ctx) error {
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

	reactionID, err := ParamInt64(c, "reaction_id")
	if err != nil {
		return BadRequest(c, "Invalid reaction ID")
	}

	if err := h.reactions.DeleteForIssueComment(c.Context(), owner, repoName, commentID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			return NotFound(c, "Reaction")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListPullReviewCommentReactions handles GET /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions
func (h *ReactionHandler) ListPullReviewCommentReactions(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: c.Query("content"),
	}

	reactionList, err := h.reactions.ListForPullReviewComment(c.Context(), owner, repoName, commentID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reactionList)
}

// CreatePullReviewCommentReaction handles POST /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions
func (h *ReactionHandler) CreatePullReviewCommentReaction(c *mizu.Ctx) error {
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
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reaction, err := h.reactions.CreateForPullReviewComment(c.Context(), owner, repoName, commentID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			return BadRequest(c, "Invalid reaction content")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, reaction)
}

// DeletePullReviewCommentReaction handles DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeletePullReviewCommentReaction(c *mizu.Ctx) error {
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

	reactionID, err := ParamInt64(c, "reaction_id")
	if err != nil {
		return BadRequest(c, "Invalid reaction ID")
	}

	if err := h.reactions.DeleteForPullReviewComment(c.Context(), owner, repoName, commentID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			return NotFound(c, "Reaction")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListCommitCommentReactions handles GET /repos/{owner}/{repo}/comments/{comment_id}/reactions
func (h *ReactionHandler) ListCommitCommentReactions(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: c.Query("content"),
	}

	reactionList, err := h.reactions.ListForCommitComment(c.Context(), owner, repoName, commentID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reactionList)
}

// CreateCommitCommentReaction handles POST /repos/{owner}/{repo}/comments/{comment_id}/reactions
func (h *ReactionHandler) CreateCommitCommentReaction(c *mizu.Ctx) error {
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
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reaction, err := h.reactions.CreateForCommitComment(c.Context(), owner, repoName, commentID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			return BadRequest(c, "Invalid reaction content")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, reaction)
}

// DeleteCommitCommentReaction handles DELETE /repos/{owner}/{repo}/comments/{comment_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteCommitCommentReaction(c *mizu.Ctx) error {
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

	reactionID, err := ParamInt64(c, "reaction_id")
	if err != nil {
		return BadRequest(c, "Invalid reaction ID")
	}

	if err := h.reactions.DeleteForCommitComment(c.Context(), owner, repoName, commentID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			return NotFound(c, "Reaction")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListReleaseReactions handles GET /repos/{owner}/{repo}/releases/{release_id}/reactions
func (h *ReactionHandler) ListReleaseReactions(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	pagination := GetPagination(c)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: c.Query("content"),
	}

	reactionList, err := h.reactions.ListForRelease(c.Context(), owner, repoName, releaseID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reactionList)
}

// CreateReleaseReaction handles POST /repos/{owner}/{repo}/releases/{release_id}/reactions
func (h *ReactionHandler) CreateReleaseReaction(c *mizu.Ctx) error {
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

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reaction, err := h.reactions.CreateForRelease(c.Context(), owner, repoName, releaseID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			return BadRequest(c, "Invalid reaction content")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, reaction)
}

// DeleteReleaseReaction handles DELETE /repos/{owner}/{repo}/releases/{release_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteReleaseReaction(c *mizu.Ctx) error {
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

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	reactionID, err := ParamInt64(c, "reaction_id")
	if err != nil {
		return BadRequest(c, "Invalid reaction ID")
	}

	if err := h.reactions.DeleteForRelease(c.Context(), owner, repoName, releaseID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			return NotFound(c, "Reaction")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}
