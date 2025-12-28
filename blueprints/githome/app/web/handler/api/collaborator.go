package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/collaborators"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
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
func (h *CollaboratorHandler) ListCollaborators(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &collaborators.ListOpts{
		Page:        pagination.Page,
		PerPage:     pagination.PerPage,
		Affiliation: c.Query("affiliation"),
		Permission:  c.Query("permission"),
	}

	collabList, err := h.collaborators.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, collabList)
}

// CheckCollaborator handles GET /repos/{owner}/{repo}/collaborators/{username}
func (h *CollaboratorHandler) CheckCollaborator(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	username := c.Param("username")

	isCollaborator, err := h.collaborators.IsCollaborator(c.Context(), owner, repoName, username)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isCollaborator {
		return NoContent(c)
	}
	return NotFound(c, "Collaborator")
}

// AddCollaborator handles PUT /repos/{owner}/{repo}/collaborators/{username}
func (h *CollaboratorHandler) AddCollaborator(c *mizu.Ctx) error {
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

	username := c.Param("username")

	var in struct {
		Permission string `json:"permission,omitempty"`
	}
	c.BindJSON(&in, 1<<20) // optional

	if in.Permission == "" {
		in.Permission = "push"
	}

	invitation, err := h.collaborators.Add(c.Context(), owner, repoName, username, in.Permission)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if invitation != nil {
		return Created(c, invitation)
	}
	return NoContent(c)
}

// RemoveCollaborator handles DELETE /repos/{owner}/{repo}/collaborators/{username}
func (h *CollaboratorHandler) RemoveCollaborator(c *mizu.Ctx) error {
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

	username := c.Param("username")

	if err := h.collaborators.Remove(c.Context(), owner, repoName, username); err != nil {
		if err == collaborators.ErrNotFound {
			return NotFound(c, "Collaborator")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetCollaboratorPermission handles GET /repos/{owner}/{repo}/collaborators/{username}/permission
func (h *CollaboratorHandler) GetCollaboratorPermission(c *mizu.Ctx) error {
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

	username := c.Param("username")

	permission, err := h.collaborators.GetPermission(c.Context(), owner, repoName, username)
	if err != nil {
		if err == collaborators.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, permission)
}

// ListInvitations handles GET /repos/{owner}/{repo}/invitations
func (h *CollaboratorHandler) ListInvitations(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &collaborators.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	invitations, err := h.collaborators.ListInvitations(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitations)
}

// UpdateInvitation handles PATCH /repos/{owner}/{repo}/invitations/{invitation_id}
func (h *CollaboratorHandler) UpdateInvitation(c *mizu.Ctx) error {
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

	invitationID, err := ParamInt64(c, "invitation_id")
	if err != nil {
		return BadRequest(c, "Invalid invitation ID")
	}

	var in struct {
		Permissions string `json:"permissions"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	invitation, err := h.collaborators.UpdateInvitation(c.Context(), owner, repoName, invitationID, in.Permissions)
	if err != nil {
		if err == collaborators.ErrNotFound {
			return NotFound(c, "Invitation")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitation)
}

// DeleteInvitation handles DELETE /repos/{owner}/{repo}/invitations/{invitation_id}
func (h *CollaboratorHandler) DeleteInvitation(c *mizu.Ctx) error {
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

	invitationID, err := ParamInt64(c, "invitation_id")
	if err != nil {
		return BadRequest(c, "Invalid invitation ID")
	}

	if err := h.collaborators.DeleteInvitation(c.Context(), owner, repoName, invitationID); err != nil {
		if err == collaborators.ErrNotFound {
			return NotFound(c, "Invitation")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListUserInvitations handles GET /user/repository_invitations
func (h *CollaboratorHandler) ListUserInvitations(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &collaborators.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	invitations, err := h.collaborators.ListUserInvitations(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitations)
}

// AcceptInvitation handles PATCH /user/repository_invitations/{invitation_id}
func (h *CollaboratorHandler) AcceptInvitation(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	invitationID, err := ParamInt64(c, "invitation_id")
	if err != nil {
		return BadRequest(c, "Invalid invitation ID")
	}

	if err := h.collaborators.AcceptInvitation(c.Context(), user.ID, invitationID); err != nil {
		if err == collaborators.ErrNotFound {
			return NotFound(c, "Invitation")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// DeclineInvitation handles DELETE /user/repository_invitations/{invitation_id}
func (h *CollaboratorHandler) DeclineInvitation(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	invitationID, err := ParamInt64(c, "invitation_id")
	if err != nil {
		return BadRequest(c, "Invalid invitation ID")
	}

	if err := h.collaborators.DeclineInvitation(c.Context(), user.ID, invitationID); err != nil {
		if err == collaborators.ErrNotFound {
			return NotFound(c, "Invitation")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}
