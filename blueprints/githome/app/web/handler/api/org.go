package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/mizu"
)

// OrgHandler handles organization endpoints
type OrgHandler struct {
	orgs orgs.API
}

// NewOrgHandler creates a new org handler
func NewOrgHandler(orgs orgs.API) *OrgHandler {
	return &OrgHandler{orgs: orgs}
}

// ListOrgs handles GET /organizations
func (h *OrgHandler) ListOrgs(c *mizu.Ctx) error {
	pagination := GetPagination(c)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	if since := c.Query("since"); since != "" {
		if n, err := strconv.ParseInt(since, 10, 64); err == nil {
			opts.Since = n
		}
	}

	orgList, err := h.orgs.List(c.Context(), opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, orgList)
}

// GetOrg handles GET /orgs/{org}
func (h *OrgHandler) GetOrg(c *mizu.Ctx) error {
	orgLogin := c.Param("org")

	org, err := h.orgs.Get(c.Context(), orgLogin)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, org)
}

// UpdateOrg handles PATCH /orgs/{org}
func (h *OrgHandler) UpdateOrg(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")

	var in orgs.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.orgs.Update(c.Context(), orgLogin, &in)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// ListAuthenticatedUserOrgs handles GET /user/orgs
func (h *OrgHandler) ListAuthenticatedUserOrgs(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	orgList, err := h.orgs.ListForUser(c.Context(), user.Login, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, orgList)
}

// ListUserOrgs handles GET /users/{username}/orgs
func (h *OrgHandler) ListUserOrgs(c *mizu.Ctx) error {
	username := c.Param("username")
	pagination := GetPagination(c)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	orgList, err := h.orgs.ListForUser(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, orgList)
}

// ListOrgMembers handles GET /orgs/{org}/members
func (h *OrgHandler) ListOrgMembers(c *mizu.Ctx) error {
	orgLogin := c.Param("org")
	pagination := GetPagination(c)
	opts := &orgs.ListMembersOpts{
		ListOpts: orgs.ListOpts{
			Page:    pagination.Page,
			PerPage: pagination.PerPage,
		},
		Filter: c.Query("filter"),
		Role:   c.Query("role"),
	}

	members, err := h.orgs.ListMembers(c.Context(), orgLogin, opts)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, members)
}

// CheckOrgMember handles GET /orgs/{org}/members/{username}
func (h *OrgHandler) CheckOrgMember(c *mizu.Ctx) error {
	orgLogin := c.Param("org")
	username := c.Param("username")

	isMember, err := h.orgs.IsMember(c.Context(), orgLogin, username)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isMember {
		return NoContent(c)
	}
	return NotFound(c, "Member")
}

// RemoveOrgMember handles DELETE /orgs/{org}/members/{username}
func (h *OrgHandler) RemoveOrgMember(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")
	username := c.Param("username")

	if err := h.orgs.RemoveMember(c.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Member")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetOrgMembership handles GET /orgs/{org}/memberships/{username}
func (h *OrgHandler) GetOrgMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")
	username := c.Param("username")

	membership, err := h.orgs.GetMembership(c.Context(), orgLogin, username)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, membership)
}

// SetOrgMembership handles PUT /orgs/{org}/memberships/{username}
func (h *OrgHandler) SetOrgMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")
	username := c.Param("username")

	var in struct {
		Role string `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	membership, err := h.orgs.SetMembership(c.Context(), orgLogin, username, in.Role)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, membership)
}

// RemoveOrgMembership handles DELETE /orgs/{org}/memberships/{username}
func (h *OrgHandler) RemoveOrgMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")
	username := c.Param("username")

	if err := h.orgs.RemoveMember(c.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListPublicOrgMembers handles GET /orgs/{org}/public_members
func (h *OrgHandler) ListPublicOrgMembers(c *mizu.Ctx) error {
	orgLogin := c.Param("org")
	pagination := GetPagination(c)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	members, err := h.orgs.ListPublicMembers(c.Context(), orgLogin, opts)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, members)
}

// CheckPublicOrgMember handles GET /orgs/{org}/public_members/{username}
func (h *OrgHandler) CheckPublicOrgMember(c *mizu.Ctx) error {
	orgLogin := c.Param("org")
	username := c.Param("username")

	isPublic, err := h.orgs.IsPublicMember(c.Context(), orgLogin, username)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isPublic {
		return NoContent(c)
	}
	return NotFound(c, "Member")
}

// PublicizeMembership handles PUT /orgs/{org}/public_members/{username}
func (h *OrgHandler) PublicizeMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")
	username := c.Param("username")

	if err := h.orgs.PublicizeMembership(c.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ConcealMembership handles DELETE /orgs/{org}/public_members/{username}
func (h *OrgHandler) ConcealMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")
	username := c.Param("username")

	if err := h.orgs.ConcealMembership(c.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetAuthenticatedUserOrgMembership handles GET /user/memberships/orgs/{org}
func (h *OrgHandler) GetAuthenticatedUserOrgMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")

	membership, err := h.orgs.GetMembership(c.Context(), orgLogin, user.Login)
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, membership)
}

// UpdateAuthenticatedUserOrgMembership handles PATCH /user/memberships/orgs/{org}
func (h *OrgHandler) UpdateAuthenticatedUserOrgMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	orgLogin := c.Param("org")

	var in struct {
		State string `json:"state"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	// Use SetMembership - when accepting an invite, state becomes "active"
	membership, err := h.orgs.SetMembership(c.Context(), orgLogin, user.Login, "member")
	if err != nil {
		if err == orgs.ErrNotFound {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, membership)
}

// ListOutsideCollaborators handles GET /orgs/{org}/outside_collaborators
func (h *OrgHandler) ListOutsideCollaborators(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	// TODO: Implement when orgs.API.ListOutsideCollaborators is available
	// For now, return empty list
	return c.JSON(http.StatusOK, []any{})
}
