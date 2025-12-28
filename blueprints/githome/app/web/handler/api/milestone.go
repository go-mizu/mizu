package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// MilestoneHandler handles milestone endpoints
type MilestoneHandler struct {
	milestones milestones.API
	repos      repos.API
}

// NewMilestoneHandler creates a new milestone handler
func NewMilestoneHandler(milestones milestones.API, repos repos.API) *MilestoneHandler {
	return &MilestoneHandler{milestones: milestones, repos: repos}
}

// ListMilestones handles GET /repos/{owner}/{repo}/milestones
func (h *MilestoneHandler) ListMilestones(w http.ResponseWriter, r *http.Request) {
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
	opts := &milestones.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		State:     QueryParam(r, "state"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	milestoneList, err := h.milestones.List(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, milestoneList)
}

// GetMilestone handles GET /repos/{owner}/{repo}/milestones/{milestone_number}
func (h *MilestoneHandler) GetMilestone(w http.ResponseWriter, r *http.Request) {
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

	milestoneNumber, err := PathParamInt(r, "milestone_number")
	if err != nil {
		WriteBadRequest(w, "Invalid milestone number")
		return
	}

	milestone, err := h.milestones.Get(r.Context(), owner, repoName, milestoneNumber)
	if err != nil {
		if err == milestones.ErrNotFound {
			WriteNotFound(w, "Milestone")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, milestone)
}

// CreateMilestone handles POST /repos/{owner}/{repo}/milestones
func (h *MilestoneHandler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
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

	var in milestones.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	milestone, err := h.milestones.Create(r.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, milestone)
}

// UpdateMilestone handles PATCH /repos/{owner}/{repo}/milestones/{milestone_number}
func (h *MilestoneHandler) UpdateMilestone(w http.ResponseWriter, r *http.Request) {
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

	milestoneNumber, err := PathParamInt(r, "milestone_number")
	if err != nil {
		WriteBadRequest(w, "Invalid milestone number")
		return
	}

	var in milestones.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.milestones.Update(r.Context(), owner, repoName, milestoneNumber, &in)
	if err != nil {
		if err == milestones.ErrNotFound {
			WriteNotFound(w, "Milestone")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// DeleteMilestone handles DELETE /repos/{owner}/{repo}/milestones/{milestone_number}
func (h *MilestoneHandler) DeleteMilestone(w http.ResponseWriter, r *http.Request) {
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

	milestoneNumber, err := PathParamInt(r, "milestone_number")
	if err != nil {
		WriteBadRequest(w, "Invalid milestone number")
		return
	}

	if err := h.milestones.Delete(r.Context(), owner, repoName, milestoneNumber); err != nil {
		if err == milestones.ErrNotFound {
			WriteNotFound(w, "Milestone")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
