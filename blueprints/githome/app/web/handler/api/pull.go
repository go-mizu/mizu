package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/pulls"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
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

// getRepoFromPath gets repository from path parameters
func (h *PullHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListPulls handles GET /repos/{owner}/{repo}/pulls
func (h *PullHandler) ListPulls(w http.ResponseWriter, r *http.Request) {
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
	opts := &pulls.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Head:      QueryParam(r, "head"),
		Base:      QueryParam(r, "base"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	pullList, err := h.pulls.ListForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, pullList)
}

// GetPull handles GET /repos/{owner}/{repo}/pulls/{pull_number}
func (h *PullHandler) GetPull(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, pull)
}

// CreatePull handles POST /repos/{owner}/{repo}/pulls
func (h *PullHandler) CreatePull(w http.ResponseWriter, r *http.Request) {
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

	var in pulls.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	pull, err := h.pulls.Create(r.Context(), repo.ID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, pull)
}

// UpdatePull handles PATCH /repos/{owner}/{repo}/pulls/{pull_number}
func (h *PullHandler) UpdatePull(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in pulls.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.pulls.Update(r.Context(), pull.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// ListPullCommits handles GET /repos/{owner}/{repo}/pulls/{pull_number}/commits
func (h *PullHandler) ListPullCommits(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commits, err := h.pulls.ListCommits(r.Context(), pull.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commits)
}

// ListPullFiles handles GET /repos/{owner}/{repo}/pulls/{pull_number}/files
func (h *PullHandler) ListPullFiles(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	files, err := h.pulls.ListFiles(r.Context(), pull.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, files)
}

// CheckPullMerged handles GET /repos/{owner}/{repo}/pulls/{pull_number}/merge
func (h *PullHandler) CheckPullMerged(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if pull.Merged {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Pull Request not merged")
	}
}

// MergePull handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge
func (h *PullHandler) MergePull(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in pulls.MergeIn
	DecodeJSON(r, &in) // optional body

	result, err := h.pulls.Merge(r.Context(), pull.ID, user.ID, &in)
	if err != nil {
		if err == pulls.ErrNotMergeable {
			WriteError(w, http.StatusMethodNotAllowed, "Pull Request is not mergeable")
			return
		}
		if err == pulls.ErrAlreadyMerged {
			WriteError(w, http.StatusMethodNotAllowed, "Pull Request already merged")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// UpdatePullBranch handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/update-branch
func (h *PullHandler) UpdatePullBranch(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in struct {
		ExpectedHeadSHA string `json:"expected_head_sha,omitempty"`
	}
	DecodeJSON(r, &in) // optional

	if err := h.pulls.UpdateBranch(r.Context(), pull.ID, in.ExpectedHeadSHA); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, map[string]string{"message": "Updating branch"})
}

// ListPullReviews handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews
func (h *PullHandler) ListPullReviews(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	reviews, err := h.pulls.ListReviews(r.Context(), pull.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reviews)
}

// GetPullReview handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) GetPullReview(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	review, err := h.pulls.GetReview(r.Context(), pull.ID, reviewID)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Review")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, review)
}

// CreatePullReview handles POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews
func (h *PullHandler) CreatePullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in pulls.CreateReviewIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	review, err := h.pulls.CreateReview(r.Context(), pull.ID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, review)
}

// UpdatePullReview handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) UpdatePullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	var in struct {
		Body string `json:"body"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	review, err := h.pulls.UpdateReview(r.Context(), pull.ID, reviewID, in.Body)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Review")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, review)
}

// DeletePullReview handles DELETE /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) DeletePullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	if err := h.pulls.DeleteReview(r.Context(), pull.ID, reviewID); err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Review")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Review deleted"})
}

// SubmitPullReview handles POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events
func (h *PullHandler) SubmitPullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	var in struct {
		Body  string `json:"body,omitempty"`
		Event string `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	review, err := h.pulls.SubmitReview(r.Context(), pull.ID, reviewID, in.Event, in.Body)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Review")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, review)
}

// DismissPullReview handles PUT /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/dismissals
func (h *PullHandler) DismissPullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	var in struct {
		Message string `json:"message"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	review, err := h.pulls.DismissReview(r.Context(), pull.ID, reviewID, in.Message)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Review")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, review)
}

// ListReviewComments handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/comments
func (h *PullHandler) ListReviewComments(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	comments, err := h.pulls.ListReviewComments(r.Context(), pull.ID, reviewID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comments)
}

// ListPullReviewComments handles GET /repos/{owner}/{repo}/pulls/{pull_number}/comments
func (h *PullHandler) ListPullReviewComments(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	comments, err := h.pulls.ListPullComments(r.Context(), pull.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comments)
}

// CreatePullReviewComment handles POST /repos/{owner}/{repo}/pulls/{pull_number}/comments
func (h *PullHandler) CreatePullReviewComment(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in pulls.CreateCommentIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.pulls.CreateComment(r.Context(), pull.ID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, comment)
}

// GetPullReviewComment handles GET /repos/{owner}/{repo}/pulls/comments/{comment_id}
func (h *PullHandler) GetPullReviewComment(w http.ResponseWriter, r *http.Request) {
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

	comment, err := h.pulls.GetComment(r.Context(), repo.ID, commentID)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comment)
}

// UpdatePullReviewComment handles PATCH /repos/{owner}/{repo}/pulls/comments/{comment_id}
func (h *PullHandler) UpdatePullReviewComment(w http.ResponseWriter, r *http.Request) {
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

	var in struct {
		Body string `json:"body"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.pulls.UpdateComment(r.Context(), repo.ID, commentID, in.Body)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, comment)
}

// DeletePullReviewComment handles DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}
func (h *PullHandler) DeletePullReviewComment(w http.ResponseWriter, r *http.Request) {
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

	if err := h.pulls.DeleteComment(r.Context(), repo.ID, commentID); err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Comment")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListRequestedReviewers handles GET /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) ListRequestedReviewers(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	reviewers, err := h.pulls.ListRequestedReviewers(r.Context(), pull.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reviewers)
}

// RequestReviewers handles POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) RequestReviewers(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in struct {
		Reviewers     []string `json:"reviewers,omitempty"`
		TeamReviewers []string `json:"team_reviewers,omitempty"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.pulls.RequestReviewers(r.Context(), pull.ID, in.Reviewers, in.TeamReviewers)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, updated)
}

// RemoveRequestedReviewers handles DELETE /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) RemoveRequestedReviewers(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.GetByNumber(r.Context(), repo.ID, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in struct {
		Reviewers     []string `json:"reviewers,omitempty"`
		TeamReviewers []string `json:"team_reviewers,omitempty"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if err := h.pulls.RemoveRequestedReviewers(r.Context(), pull.ID, in.Reviewers, in.TeamReviewers); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Reviewers removed"})
}
