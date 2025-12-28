package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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
func (h *PullHandler) ListPulls(w http.ResponseWriter, r *http.Request) {
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
	opts := &pulls.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Head:      QueryParam(r, "head"),
		Base:      QueryParam(r, "base"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	pullList, err := h.pulls.List(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, pullList)
}

// GetPull handles GET /repos/{owner}/{repo}/pulls/{pull_number}
func (h *PullHandler) GetPull(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pull, err := h.pulls.Get(r.Context(), owner, repoName, pullNumber)
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

	var in pulls.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	pull, err := h.pulls.Create(r.Context(), owner, repoName, user.ID, &in)
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	var in pulls.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.pulls.Update(r.Context(), owner, repoName, pullNumber, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// ListPullCommits handles GET /repos/{owner}/{repo}/pulls/{pull_number}/commits
func (h *PullHandler) ListPullCommits(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	commits, err := h.pulls.ListCommits(r.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commits)
}

// ListPullFiles handles GET /repos/{owner}/{repo}/pulls/{pull_number}/files
func (h *PullHandler) ListPullFiles(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	files, err := h.pulls.ListFiles(r.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, files)
}

// CheckPullMerged handles GET /repos/{owner}/{repo}/pulls/{pull_number}/merge
func (h *PullHandler) CheckPullMerged(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	merged, err := h.pulls.IsMerged(r.Context(), owner, repoName, pullNumber)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if merged {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	var in pulls.MergeIn
	DecodeJSON(r, &in) // optional body

	result, err := h.pulls.Merge(r.Context(), owner, repoName, pullNumber, &in)
	if err != nil {
		if err == pulls.ErrNotMergeable {
			WriteError(w, http.StatusMethodNotAllowed, "Pull Request is not mergeable")
			return
		}
		if err == pulls.ErrAlreadyMerged {
			WriteError(w, http.StatusMethodNotAllowed, "Pull Request already merged")
			return
		}
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	if err := h.pulls.UpdateBranch(r.Context(), owner, repoName, pullNumber); err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, map[string]string{"message": "Updating branch"})
}

// ListPullReviews handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews
func (h *PullHandler) ListPullReviews(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	reviews, err := h.pulls.ListReviews(r.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reviews)
}

// GetPullReview handles GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}
func (h *PullHandler) GetPullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	review, err := h.pulls.GetReview(r.Context(), owner, repoName, pullNumber, reviewID)
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	var in pulls.CreateReviewIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	review, err := h.pulls.CreateReview(r.Context(), owner, repoName, pullNumber, user.ID, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
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

	review, err := h.pulls.UpdateReview(r.Context(), owner, repoName, pullNumber, reviewID, in.Body)
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

// SubmitPullReview handles POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events
func (h *PullHandler) SubmitPullReview(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	reviewID, err := PathParamInt64(r, "review_id")
	if err != nil {
		WriteBadRequest(w, "Invalid review ID")
		return
	}

	var in pulls.SubmitReviewIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	review, err := h.pulls.SubmitReview(r.Context(), owner, repoName, pullNumber, reviewID, &in)
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
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

	review, err := h.pulls.DismissReview(r.Context(), owner, repoName, pullNumber, reviewID, in.Message)
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

// ListPullReviewComments handles GET /repos/{owner}/{repo}/pulls/{pull_number}/comments
func (h *PullHandler) ListPullReviewComments(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &pulls.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	comments, err := h.pulls.ListReviewComments(r.Context(), owner, repoName, pullNumber, opts)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
		return
	}

	var in pulls.CreateReviewCommentIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	comment, err := h.pulls.CreateReviewComment(r.Context(), owner, repoName, pullNumber, user.ID, &in)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, comment)
}

// RequestReviewers handles POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers
func (h *PullHandler) RequestReviewers(w http.ResponseWriter, r *http.Request) {
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
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

	updated, err := h.pulls.RequestReviewers(r.Context(), owner, repoName, pullNumber, in.Reviewers, in.TeamReviewers)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
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

	pullNumber, err := PathParamInt(r, "pull_number")
	if err != nil {
		WriteBadRequest(w, "Invalid pull number")
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

	updated, err := h.pulls.RemoveReviewers(r.Context(), owner, repoName, pullNumber, in.Reviewers, in.TeamReviewers)
	if err != nil {
		if err == pulls.ErrNotFound {
			WriteNotFound(w, "Pull Request")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}
