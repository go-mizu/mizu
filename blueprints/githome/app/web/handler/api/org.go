package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
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
func (h *OrgHandler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	pagination := GetPaginationParams(r)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	if since := QueryParam(r, "since"); since != "" {
		if n, err := PathParamInt64(r, "since"); err == nil {
			opts.Since = n
		}
	}

	orgList, err := h.orgs.List(r.Context(), opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, orgList)
}

// GetOrg handles GET /orgs/{org}
func (h *OrgHandler) GetOrg(w http.ResponseWriter, r *http.Request) {
	orgLogin := PathParam(r, "org")

	org, err := h.orgs.Get(r.Context(), orgLogin)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, org)
}

// UpdateOrg handles PATCH /orgs/{org}
func (h *OrgHandler) UpdateOrg(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")

	var in orgs.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.orgs.Update(r.Context(), orgLogin, &in)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// ListAuthenticatedUserOrgs handles GET /user/orgs
func (h *OrgHandler) ListAuthenticatedUserOrgs(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	orgList, err := h.orgs.ListForUser(r.Context(), user.Login, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, orgList)
}

// ListUserOrgs handles GET /users/{username}/orgs
func (h *OrgHandler) ListUserOrgs(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	pagination := GetPaginationParams(r)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	orgList, err := h.orgs.ListForUser(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, orgList)
}

// ListOrgMembers handles GET /orgs/{org}/members
func (h *OrgHandler) ListOrgMembers(w http.ResponseWriter, r *http.Request) {
	orgLogin := PathParam(r, "org")
	pagination := GetPaginationParams(r)
	opts := &orgs.ListMembersOpts{
		ListOpts: orgs.ListOpts{
			Page:    pagination.Page,
			PerPage: pagination.PerPage,
		},
		Filter: QueryParam(r, "filter"),
		Role:   QueryParam(r, "role"),
	}

	members, err := h.orgs.ListMembers(r.Context(), orgLogin, opts)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, members)
}

// CheckOrgMember handles GET /orgs/{org}/members/{username}
func (h *OrgHandler) CheckOrgMember(w http.ResponseWriter, r *http.Request) {
	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	isMember, err := h.orgs.IsMember(r.Context(), orgLogin, username)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isMember {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Member")
	}
}

// RemoveOrgMember handles DELETE /orgs/{org}/members/{username}
func (h *OrgHandler) RemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	if err := h.orgs.RemoveMember(r.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Member")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetOrgMembership handles GET /orgs/{org}/memberships/{username}
func (h *OrgHandler) GetOrgMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	membership, err := h.orgs.GetMembership(r.Context(), orgLogin, username)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, membership)
}

// SetOrgMembership handles PUT /orgs/{org}/memberships/{username}
func (h *OrgHandler) SetOrgMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	var in struct {
		Role string `json:"role"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	membership, err := h.orgs.SetMembership(r.Context(), orgLogin, username, in.Role)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, membership)
}

// RemoveOrgMembership handles DELETE /orgs/{org}/memberships/{username}
func (h *OrgHandler) RemoveOrgMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	if err := h.orgs.RemoveMember(r.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListPublicOrgMembers handles GET /orgs/{org}/public_members
func (h *OrgHandler) ListPublicOrgMembers(w http.ResponseWriter, r *http.Request) {
	orgLogin := PathParam(r, "org")
	pagination := GetPaginationParams(r)
	opts := &orgs.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	members, err := h.orgs.ListPublicMembers(r.Context(), orgLogin, opts)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, members)
}

// CheckPublicOrgMember handles GET /orgs/{org}/public_members/{username}
func (h *OrgHandler) CheckPublicOrgMember(w http.ResponseWriter, r *http.Request) {
	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	isPublic, err := h.orgs.IsPublicMember(r.Context(), orgLogin, username)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isPublic {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Member")
	}
}

// PublicizeMembership handles PUT /orgs/{org}/public_members/{username}
func (h *OrgHandler) PublicizeMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	if err := h.orgs.PublicizeMembership(r.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ConcealMembership handles DELETE /orgs/{org}/public_members/{username}
func (h *OrgHandler) ConcealMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")
	username := PathParam(r, "username")

	if err := h.orgs.ConcealMembership(r.Context(), orgLogin, username); err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetAuthenticatedUserOrgMembership handles GET /user/memberships/orgs/{org}
func (h *OrgHandler) GetAuthenticatedUserOrgMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")

	membership, err := h.orgs.GetMembership(r.Context(), orgLogin, user.Login)
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, membership)
}

// UpdateAuthenticatedUserOrgMembership handles PATCH /user/memberships/orgs/{org}
func (h *OrgHandler) UpdateAuthenticatedUserOrgMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	orgLogin := PathParam(r, "org")

	var in struct {
		State string `json:"state"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	// Use SetMembership - when accepting an invite, state becomes "active"
	// The API accepts state to transition pending -> active
	membership, err := h.orgs.SetMembership(r.Context(), orgLogin, user.Login, "member")
	if err != nil {
		if err == orgs.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, membership)
}
