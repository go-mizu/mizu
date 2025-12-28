package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/comments"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
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

// getRepoFromPath gets repository from path parameters
func (h *CommentHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListIssueComments handles GET /repos/{owner}/{repo}/issues/{issue_number}/comments
func (h *CommentHandler) ListIssueComments(w http.ResponseWriter, r *http.Request) {
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
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListForIssue(r.Context(), repo.ID, issueNumber, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}

// GetIssueComment handles GET /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) GetIssueComment(w http.ResponseWriter, r *http.Request) {
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

	comment, err := h.comments.GetByID(r.Context(), repo.ID, commentID)
	if err != nil {
		if err == comments.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comment)
}

// CreateIssueComment handles POST /repos/{owner}/{repo}/issues/{issue_number}/comments
func (h *CommentHandler) CreateIssueComment(w http.ResponseWriter, r *http.Request) {
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

	var in comments.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.CreateForIssue(r.Context(), repo.ID, issueNumber, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, comment)
}

// UpdateIssueComment handles PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) UpdateIssueComment(w http.ResponseWriter, r *http.Request) {
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

	var in comments.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.Update(r.Context(), repo.ID, commentID, &in)
	if err != nil {
		if err == comments.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comment)
}

// DeleteIssueComment handles DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) DeleteIssueComment(w http.ResponseWriter, r *http.Request) {
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

	if err := h.comments.Delete(r.Context(), repo.ID, commentID); err != nil {
		if err == comments.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListRepoComments handles GET /repos/{owner}/{repo}/issues/comments
func (h *CommentHandler) ListRepoComments(w http.ResponseWriter, r *http.Request) {
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
	opts := &comments.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	commentList, err := h.comments.ListForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}

// ListCommitComments handles GET /repos/{owner}/{repo}/commits/{commit_sha}/comments
func (h *CommentHandler) ListCommitComments(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	commitSHA := PathParam(r, "commit_sha")

	pagination := GetPaginationParams(r)
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListForCommit(r.Context(), repo.ID, commitSHA, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}

// CreateCommitComment handles POST /repos/{owner}/{repo}/commits/{commit_sha}/comments
func (h *CommentHandler) CreateCommitComment(w http.ResponseWriter, r *http.Request) {
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

	commitSHA := PathParam(r, "commit_sha")

	var in comments.CreateCommitCommentIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.CreateForCommit(r.Context(), repo.ID, commitSHA, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, comment)
}

// GetCommitComment handles GET /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) GetCommitComment(w http.ResponseWriter, r *http.Request) {
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

	comment, err := h.comments.GetCommitComment(r.Context(), repo.ID, commentID)
	if err != nil {
		if err == comments.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comment)
}

// UpdateCommitComment handles PATCH /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) UpdateCommitComment(w http.ResponseWriter, r *http.Request) {
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

	var in comments.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.UpdateCommitComment(r.Context(), repo.ID, commentID, &in)
	if err != nil {
		if err == comments.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comment)
}

// DeleteCommitComment handles DELETE /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) DeleteCommitComment(w http.ResponseWriter, r *http.Request) {
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

	if err := h.comments.DeleteCommitComment(r.Context(), repo.ID, commentID); err != nil {
		if err == comments.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListRepoCommitComments handles GET /repos/{owner}/{repo}/comments
func (h *CommentHandler) ListRepoCommitComments(w http.ResponseWriter, r *http.Request) {
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
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListCommitCommentsForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}
