package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/reactions"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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
func (h *ReactionHandler) ListIssueReactions(w http.ResponseWriter, r *http.Request) {
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

	reactionList, err := h.reactions.ListForIssue(r.Context(), owner, repoName, issueNumber, opts)
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

	issueNumber, err := PathParamInt(r, "issue_number")
	if err != nil {
		WriteBadRequest(w, "Invalid issue number")
		return
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForIssue(r.Context(), owner, repoName, issueNumber, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			WriteBadRequest(w, "Invalid reaction content")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, reaction)
}

// DeleteIssueReaction handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}
func (h *ReactionHandler) DeleteIssueReaction(w http.ResponseWriter, r *http.Request) {
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

	if err := h.reactions.DeleteForIssue(r.Context(), owner, repoName, issueNumber, reactionID); err != nil {
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

	reactionList, err := h.reactions.ListForIssueComment(r.Context(), owner, repoName, commentID, opts)
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForIssueComment(r.Context(), owner, repoName, commentID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			WriteBadRequest(w, "Invalid reaction content")
			return
		}
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

	if err := h.reactions.DeleteForIssueComment(r.Context(), owner, repoName, commentID, reactionID); err != nil {
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

	reactionList, err := h.reactions.ListForPullReviewComment(r.Context(), owner, repoName, commentID, opts)
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForPullReviewComment(r.Context(), owner, repoName, commentID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			WriteBadRequest(w, "Invalid reaction content")
			return
		}
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

	if err := h.reactions.DeleteForPullReviewComment(r.Context(), owner, repoName, commentID, reactionID); err != nil {
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

	reactionList, err := h.reactions.ListForCommitComment(r.Context(), owner, repoName, commentID, opts)
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

	commentID, err := PathParamInt64(r, "comment_id")
	if err != nil {
		WriteBadRequest(w, "Invalid comment ID")
		return
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForCommitComment(r.Context(), owner, repoName, commentID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			WriteBadRequest(w, "Invalid reaction content")
			return
		}
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

	if err := h.reactions.DeleteForCommitComment(r.Context(), owner, repoName, commentID, reactionID); err != nil {
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

	reactionList, err := h.reactions.ListForRelease(r.Context(), owner, repoName, releaseID, opts)
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

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reaction, err := h.reactions.CreateForRelease(r.Context(), owner, repoName, releaseID, user.ID, in.Content)
	if err != nil {
		if err == reactions.ErrInvalidContent {
			WriteBadRequest(w, "Invalid reaction content")
			return
		}
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

	if err := h.reactions.DeleteForRelease(r.Context(), owner, repoName, releaseID, reactionID); err != nil {
		if err == reactions.ErrNotFound {
			WriteNotFound(w, "Reaction")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
