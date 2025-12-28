package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/reactions"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
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

// getRepoFromPath gets repository from path parameters
func (h *ReactionHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListIssueReactions handles GET /repos/{owner}/{repo}/issues/{issue_number}/reactions
func (h *ReactionHandler) ListIssueReactions(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
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
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: QueryParam(r, "content"),
	}

	reactionList, err := h.reactions.ListForIssue(r.Context(), repo.ID, issueNumber, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reactionList)
}

// CreateIssueReaction handles POST /repos/{owner}/{repo}/issues/{issue_number}/reactions
func (h *ReactionHandler) CreateIssueReaction(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	var in reactions.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForIssue(r.Context(), repo.ID, issueNumber, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return 200 if already exists, 201 if new
	WriteCreated(w, reaction)
}

// DeleteIssueReaction handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteIssueReaction(w http.ResponseWriter, r *http.Request) {
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	reactionID, err := PathParamInt64(r, "reaction_id")
	if err != nil {
		WriteBadRequest(w, "Invalid reaction ID")
		return
	}

	if err := h.reactions.DeleteForIssue(r.Context(), repo.ID, issueNumber, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			WriteNotFound(w, "Reaction")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListIssueCommentReactions handles GET /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions
func (h *ReactionHandler) ListIssueCommentReactions(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: QueryParam(r, "content"),
	}

	reactionList, err := h.reactions.ListForIssueComment(r.Context(), repo.ID, commentID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reactionList)
}

// CreateIssueCommentReaction handles POST /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions
func (h *ReactionHandler) CreateIssueCommentReaction(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	var in reactions.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForIssueComment(r.Context(), repo.ID, commentID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, reaction)
}

// DeleteIssueCommentReaction handles DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteIssueCommentReaction(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	reactionID, err := PathParamInt64(r, "reaction_id")
	if err != nil {
		WriteBadRequest(w, "Invalid reaction ID")
		return
	}

	if err := h.reactions.DeleteForIssueComment(r.Context(), repo.ID, commentID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			WriteNotFound(w, "Reaction")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListPullReviewCommentReactions handles GET /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions
func (h *ReactionHandler) ListPullReviewCommentReactions(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: QueryParam(r, "content"),
	}

	reactionList, err := h.reactions.ListForPullComment(r.Context(), repo.ID, commentID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reactionList)
}

// CreatePullReviewCommentReaction handles POST /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions
func (h *ReactionHandler) CreatePullReviewCommentReaction(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	var in reactions.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForPullComment(r.Context(), repo.ID, commentID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, reaction)
}

// DeletePullReviewCommentReaction handles DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeletePullReviewCommentReaction(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	reactionID, err := PathParamInt64(r, "reaction_id")
	if err != nil {
		WriteBadRequest(w, "Invalid reaction ID")
		return
	}

	if err := h.reactions.DeleteForPullComment(r.Context(), repo.ID, commentID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			WriteNotFound(w, "Reaction")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListCommitCommentReactions handles GET /repos/{owner}/{repo}/comments/{comment_id}/reactions
func (h *ReactionHandler) ListCommitCommentReactions(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: QueryParam(r, "content"),
	}

	reactionList, err := h.reactions.ListForCommitComment(r.Context(), repo.ID, commentID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reactionList)
}

// CreateCommitCommentReaction handles POST /repos/{owner}/{repo}/comments/{comment_id}/reactions
func (h *ReactionHandler) CreateCommitCommentReaction(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	var in reactions.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForCommitComment(r.Context(), repo.ID, commentID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, reaction)
}

// DeleteCommitCommentReaction handles DELETE /repos/{owner}/{repo}/comments/{comment_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteCommitCommentReaction(w http.ResponseWriter, r *http.Request) {
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	reactionID, err := PathParamInt64(r, "reaction_id")
	if err != nil {
		WriteBadRequest(w, "Invalid reaction ID")
		return
	}

	if err := h.reactions.DeleteForCommitComment(r.Context(), repo.ID, commentID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			WriteNotFound(w, "Reaction")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListReleaseReactions handles GET /repos/{owner}/{repo}/releases/{release_id}/reactions
func (h *ReactionHandler) ListReleaseReactions(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &reactions.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Content: QueryParam(r, "content"),
	}

	reactionList, err := h.reactions.ListForRelease(r.Context(), repo.ID, releaseID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reactionList)
}

// CreateReleaseReaction handles POST /repos/{owner}/{repo}/releases/{release_id}/reactions
func (h *ReactionHandler) CreateReleaseReaction(w http.ResponseWriter, r *http.Request) {
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

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	var in reactions.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForRelease(r.Context(), repo.ID, releaseID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, reaction)
}

// DeleteReleaseReaction handles DELETE /repos/{owner}/{repo}/releases/{release_id}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteReleaseReaction(w http.ResponseWriter, r *http.Request) {
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

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	reactionID, err := PathParamInt64(r, "reaction_id")
	if err != nil {
		WriteBadRequest(w, "Invalid reaction ID")
		return
	}

	if err := h.reactions.DeleteForRelease(r.Context(), repo.ID, releaseID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			WriteNotFound(w, "Reaction")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
