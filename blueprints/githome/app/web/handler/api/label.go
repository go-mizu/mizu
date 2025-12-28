package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/labels"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
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

// getRepoFromPath gets repository from path parameters
func (h *LabelHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListRepoLabels handles GET /repos/{owner}/{repo}/labels
func (h *LabelHandler) ListRepoLabels(w http.ResponseWriter, r *http.Request) {
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
	opts := &labels.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	labelList, err := h.labels.ListForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, labelList)
}

// GetLabel handles GET /repos/{owner}/{repo}/labels/{name}
func (h *LabelHandler) GetLabel(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	name := PathParam(r, "name")

	label, err := h.labels.GetByName(r.Context(), repo.ID, name)
	if err != nil {
		if err == labels.ErrNotFound {
			WriteNotFound(w, "Label")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, label)
}

// CreateLabel handles POST /repos/{owner}/{repo}/labels
func (h *LabelHandler) CreateLabel(w http.ResponseWriter, r *http.Request) {
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

	var in labels.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	label, err := h.labels.Create(r.Context(), repo.ID, &in)
	if err != nil {
		if err == labels.ErrExists {
			WriteConflict(w, "Label already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, label)
}

// UpdateLabel handles PATCH /repos/{owner}/{repo}/labels/{name}
func (h *LabelHandler) UpdateLabel(w http.ResponseWriter, r *http.Request) {
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

	name := PathParam(r, "name")

	label, err := h.labels.GetByName(r.Context(), repo.ID, name)
	if err != nil {
		if err == labels.ErrNotFound {
			WriteNotFound(w, "Label")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in labels.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.labels.Update(r.Context(), label.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// DeleteLabel handles DELETE /repos/{owner}/{repo}/labels/{name}
func (h *LabelHandler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
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

	name := PathParam(r, "name")

	label, err := h.labels.GetByName(r.Context(), repo.ID, name)
	if err != nil {
		if err == labels.ErrNotFound {
			WriteNotFound(w, "Label")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.labels.Delete(r.Context(), label.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListIssueLabels handles GET /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) ListIssueLabels(w http.ResponseWriter, r *http.Request) {
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
	opts := &labels.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	labelList, err := h.labels.ListForIssue(r.Context(), repo.ID, issueNumber, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, labelList)
}

// AddIssueLabels handles POST /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) AddIssueLabels(w http.ResponseWriter, r *http.Request) {
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

	var in struct {
		Labels []string `json:"labels"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	labelList, err := h.labels.AddToIssue(r.Context(), repo.ID, issueNumber, in.Labels)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, labelList)
}

// SetIssueLabels handles PUT /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) SetIssueLabels(w http.ResponseWriter, r *http.Request) {
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

	var in struct {
		Labels []string `json:"labels"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	labelList, err := h.labels.SetForIssue(r.Context(), repo.ID, issueNumber, in.Labels)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, labelList)
}

// RemoveAllIssueLabels handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels
func (h *LabelHandler) RemoveAllIssueLabels(w http.ResponseWriter, r *http.Request) {
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

	if err := h.labels.RemoveAllFromIssue(r.Context(), repo.ID, issueNumber); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// RemoveIssueLabel handles DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels/{name}
func (h *LabelHandler) RemoveIssueLabel(w http.ResponseWriter, r *http.Request) {
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

	name := PathParam(r, "name")

	labelList, err := h.labels.RemoveFromIssue(r.Context(), repo.ID, issueNumber, name)
	if err != nil {
		if err == labels.ErrNotFound {
			WriteNotFound(w, "Label")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, labelList)
}

// ListLabelsForMilestone handles GET /repos/{owner}/{repo}/milestones/{milestone_number}/labels
func (h *LabelHandler) ListLabelsForMilestone(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	milestoneNumber, err := PathParamInt(r, "milestone_number")
	if err != nil {
		WriteBadRequest(w, "Invalid milestone number")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &labels.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	labelList, err := h.labels.ListForMilestone(r.Context(), repo.ID, milestoneNumber, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, labelList)
}
