package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/collaborators"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// CollaboratorHandler handles collaborator endpoints
type CollaboratorHandler struct {
	collaborators collaborators.API
	repos         repos.API
}

// NewCollaboratorHandler creates a new collaborator handler
func NewCollaboratorHandler(collaborators collaborators.API, repos repos.API) *CollaboratorHandler {
	return &CollaboratorHandler{collaborators: collaborators, repos: repos}
}

// ListCollaborators handles GET /repos/{owner}/{repo}/collaborators
func (h *CollaboratorHandler) ListCollaborators(w http.ResponseWriter, r *http.Request) {
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

	pagination := GetPaginationParams(r)
	opts := &collaborators.ListOpts{
		Page:        pagination.Page,
		PerPage:     pagination.PerPage,
		Affiliation: QueryParam(r, "affiliation"),
		Permission:  QueryParam(r, "permission"),
	}

	collabList, err := h.collaborators.List(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, collabList)
}

// CheckCollaborator handles GET /repos/{owner}/{repo}/collaborators/{username}
func (h *CollaboratorHandler) CheckCollaborator(w http.ResponseWriter, r *http.Request) {
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

	username := PathParam(r, "username")

	isCollaborator, err := h.collaborators.IsCollaborator(r.Context(), owner, repoName, username)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isCollaborator {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Collaborator")
	}
}

// AddCollaborator handles PUT /repos/{owner}/{repo}/collaborators/{username}
func (h *CollaboratorHandler) AddCollaborator(w http.ResponseWriter, r *http.Request) {
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

	username := PathParam(r, "username")

	var in struct {
		Permission string `json:"permission,omitempty"`
	}
	DecodeJSON(r, &in) // optional

	if in.Permission == "" {
		in.Permission = "push"
	}

	invitation, err := h.collaborators.Add(r.Context(), owner, repoName, username, in.Permission)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if invitation != nil {
		WriteCreated(w, invitation)
	} else {
		WriteNoContent(w)
	}
}

// RemoveCollaborator handles DELETE /repos/{owner}/{repo}/collaborators/{username}
func (h *CollaboratorHandler) RemoveCollaborator(w http.ResponseWriter, r *http.Request) {
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

	username := PathParam(r, "username")

	if err := h.collaborators.Remove(r.Context(), owner, repoName, username); err != nil {
		if err == collaborators.ErrNotFound {
			WriteNotFound(w, "Collaborator")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetCollaboratorPermission handles GET /repos/{owner}/{repo}/collaborators/{username}/permission
func (h *CollaboratorHandler) GetCollaboratorPermission(w http.ResponseWriter, r *http.Request) {
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

	username := PathParam(r, "username")

	permission, err := h.collaborators.GetPermission(r.Context(), owner, repoName, username)
	if err != nil {
		if err == collaborators.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, permission)
}

// ListInvitations handles GET /repos/{owner}/{repo}/invitations
func (h *CollaboratorHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
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

	pagination := GetPaginationParams(r)
	opts := &collaborators.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	invitations, err := h.collaborators.ListInvitations(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, invitations)
}

// UpdateInvitation handles PATCH /repos/{owner}/{repo}/invitations/{invitation_id}
func (h *CollaboratorHandler) UpdateInvitation(w http.ResponseWriter, r *http.Request) {
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

	invitationID, err := PathParamInt64(r, "invitation_id")
	if err != nil {
		WriteBadRequest(w, "Invalid invitation ID")
		return
	}

	var in struct {
		Permissions string `json:"permissions"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	invitation, err := h.collaborators.UpdateInvitation(r.Context(), owner, repoName, invitationID, in.Permissions)
	if err != nil {
		if err == collaborators.ErrNotFound {
			WriteNotFound(w, "Invitation")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, invitation)
}

// DeleteInvitation handles DELETE /repos/{owner}/{repo}/invitations/{invitation_id}
func (h *CollaboratorHandler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
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

	invitationID, err := PathParamInt64(r, "invitation_id")
	if err != nil {
		WriteBadRequest(w, "Invalid invitation ID")
		return
	}

	if err := h.collaborators.DeleteInvitation(r.Context(), owner, repoName, invitationID); err != nil {
		if err == collaborators.ErrNotFound {
			WriteNotFound(w, "Invitation")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListUserInvitations handles GET /user/repository_invitations
func (h *CollaboratorHandler) ListUserInvitations(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &collaborators.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	invitations, err := h.collaborators.ListUserInvitations(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, invitations)
}

// AcceptInvitation handles PATCH /user/repository_invitations/{invitation_id}
func (h *CollaboratorHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	invitationID, err := PathParamInt64(r, "invitation_id")
	if err != nil {
		WriteBadRequest(w, "Invalid invitation ID")
		return
	}

	if err := h.collaborators.AcceptInvitation(r.Context(), user.ID, invitationID); err != nil {
		if err == collaborators.ErrNotFound {
			WriteNotFound(w, "Invitation")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// DeclineInvitation handles DELETE /user/repository_invitations/{invitation_id}
func (h *CollaboratorHandler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	invitationID, err := PathParamInt64(r, "invitation_id")
	if err != nil {
		WriteBadRequest(w, "Invalid invitation ID")
		return
	}

	if err := h.collaborators.DeclineInvitation(r.Context(), user.ID, invitationID); err != nil {
		if err == collaborators.ErrNotFound {
			WriteNotFound(w, "Invitation")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
