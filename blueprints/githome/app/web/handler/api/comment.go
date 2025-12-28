package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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
func (h *CommentHandler) ListIssueComments(w http.ResponseWriter, r *http.Request) {
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
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListForIssue(r.Context(), owner, repoName, issueNumber, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}

// GetIssueComment handles GET /repos/{owner}/{repo}/issues/comments/{comment_id}
func (h *CommentHandler) GetIssueComment(w http.ResponseWriter, r *http.Request) {
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

	comment, err := h.comments.GetIssueComment(r.Context(), owner, repoName, commentID)
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
		Body string `json:"body"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.CreateIssueComment(r.Context(), owner, repoName, issueNumber, user.ID, in.Body)
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
		Body string `json:"body"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.UpdateIssueComment(r.Context(), owner, repoName, commentID, in.Body)
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

	if err := h.comments.DeleteIssueComment(r.Context(), owner, repoName, commentID); err != nil {
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
	opts := &comments.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	commentList, err := h.comments.ListForRepo(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}

// ListCommitComments handles GET /repos/{owner}/{repo}/commits/{commit_sha}/comments
func (h *CommentHandler) ListCommitComments(w http.ResponseWriter, r *http.Request) {
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

	commitSHA := PathParam(r, "commit_sha")

	pagination := GetPaginationParams(r)
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListForCommit(r.Context(), owner, repoName, commitSHA, opts)
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

	commitSHA := PathParam(r, "commit_sha")

	var in comments.CreateCommitCommentIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.CreateCommitComment(r.Context(), owner, repoName, commitSHA, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, comment)
}

// GetCommitComment handles GET /repos/{owner}/{repo}/comments/{comment_id}
func (h *CommentHandler) GetCommitComment(w http.ResponseWriter, r *http.Request) {
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

	comment, err := h.comments.GetCommitComment(r.Context(), owner, repoName, commentID)
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
		Body string `json:"body"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.comments.UpdateCommitComment(r.Context(), owner, repoName, commentID, in.Body)
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

	if err := h.comments.DeleteCommitComment(r.Context(), owner, repoName, commentID); err != nil {
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
	opts := &comments.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commentList, err := h.comments.ListCommitCommentsForRepo(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commentList)
}
