package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/teams"
)

// TeamHandler handles team endpoints
type TeamHandler struct {
	teams teams.API
}

// NewTeamHandler creates a new team handler
func NewTeamHandler(teams teams.API) *TeamHandler {
	return &TeamHandler{teams: teams}
}

// ListOrgTeams handles GET /orgs/{org}/teams
func (h *TeamHandler) ListOrgTeams(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	pagination := GetPaginationParams(r)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	teamList, err := h.teams.ListForOrg(r.Context(), org, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, teamList)
}

// GetOrgTeam handles GET /orgs/{org}/teams/{team_slug}
func (h *TeamHandler) GetOrgTeam(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, team)
}

// CreateTeam handles POST /orgs/{org}/teams
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")

	var in teams.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	team, err := h.teams.Create(r.Context(), org, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, team)
}

// UpdateTeam handles PATCH /orgs/{org}/teams/{team_slug}
func (h *TeamHandler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in teams.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.teams.Update(r.Context(), team.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// DeleteTeam handles DELETE /orgs/{org}/teams/{team_slug}
func (h *TeamHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.teams.Delete(r.Context(), team.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListTeamMembers handles GET /orgs/{org}/teams/{team_slug}/members
func (h *TeamHandler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &teams.MemberListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Role:    QueryParam(r, "role"),
	}

	members, err := h.teams.ListMembers(r.Context(), team.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, members)
}

// GetTeamMembership handles GET /orgs/{org}/teams/{team_slug}/memberships/{username}
func (h *TeamHandler) GetTeamMembership(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")
	username := PathParam(r, "username")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	membership, err := h.teams.GetMembership(r.Context(), team.ID, username)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, membership)
}

// AddTeamMember handles PUT /orgs/{org}/teams/{team_slug}/memberships/{username}
func (h *TeamHandler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")
	username := PathParam(r, "username")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in struct {
		Role string `json:"role,omitempty"`
	}
	DecodeJSON(r, &in) // optional

	if in.Role == "" {
		in.Role = "member"
	}

	membership, err := h.teams.AddMember(r.Context(), team.ID, username, in.Role)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, membership)
}

// RemoveTeamMember handles DELETE /orgs/{org}/teams/{team_slug}/memberships/{username}
func (h *TeamHandler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")
	username := PathParam(r, "username")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.teams.RemoveMember(r.Context(), team.ID, username); err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Membership")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListTeamRepos handles GET /orgs/{org}/teams/{team_slug}/repos
func (h *TeamHandler) ListTeamRepos(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	repos, err := h.teams.ListRepos(r.Context(), team.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repos)
}

// CheckTeamRepoPermission handles GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *TeamHandler) CheckTeamRepoPermission(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")
	owner := PathParam(r, "owner")
	repo := PathParam(r, "repo")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	repoPerms, err := h.teams.GetRepoPermission(r.Context(), team.ID, owner, repo)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoPerms)
}

// AddTeamRepo handles PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *TeamHandler) AddTeamRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")
	owner := PathParam(r, "owner")
	repo := PathParam(r, "repo")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in struct {
		Permission string `json:"permission,omitempty"`
	}
	DecodeJSON(r, &in) // optional

	if in.Permission == "" {
		in.Permission = "push"
	}

	if err := h.teams.AddRepo(r.Context(), team.ID, owner, repo, in.Permission); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// RemoveTeamRepo handles DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *TeamHandler) RemoveTeamRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")
	owner := PathParam(r, "owner")
	repo := PathParam(r, "repo")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.teams.RemoveRepo(r.Context(), team.ID, owner, repo); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListChildTeams handles GET /orgs/{org}/teams/{team_slug}/teams
func (h *TeamHandler) ListChildTeams(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	teamSlug := PathParam(r, "team_slug")

	team, err := h.teams.GetBySlug(r.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			WriteNotFound(w, "Team")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	children, err := h.teams.ListChildren(r.Context(), team.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, children)
}

// ListAuthenticatedUserTeams handles GET /user/teams
func (h *TeamHandler) ListAuthenticatedUserTeams(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	teamList, err := h.teams.ListForUser(r.Context(), user.Login, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, teamList)
}
