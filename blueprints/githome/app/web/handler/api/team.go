package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/mizu"
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
func (h *TeamHandler) ListOrgTeams(c *mizu.Ctx) error {
	org := c.Param("org")
	pagination := GetPagination(c)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	teamList, err := h.teams.List(c.Context(), org, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, teamList)
}

// GetOrgTeam handles GET /orgs/{org}/teams/{team_slug}
func (h *TeamHandler) GetOrgTeam(c *mizu.Ctx) error {
	org := c.Param("org")
	teamSlug := c.Param("team_slug")

	team, err := h.teams.GetBySlug(c.Context(), org, teamSlug)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, team)
}

// CreateTeam handles POST /orgs/{org}/teams
func (h *TeamHandler) CreateTeam(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")

	var in teams.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	team, err := h.teams.Create(c.Context(), org, &in)
	if err != nil {
		if err == teams.ErrTeamExists {
			return Conflict(c, "Team already exists")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, team)
}

// UpdateTeam handles PATCH /orgs/{org}/teams/{team_slug}
func (h *TeamHandler) UpdateTeam(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")

	var in teams.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.teams.Update(c.Context(), org, teamSlug, &in)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteTeam handles DELETE /orgs/{org}/teams/{team_slug}
func (h *TeamHandler) DeleteTeam(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")

	if err := h.teams.Delete(c.Context(), org, teamSlug); err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListTeamMembers handles GET /orgs/{org}/teams/{team_slug}/members
func (h *TeamHandler) ListTeamMembers(c *mizu.Ctx) error {
	org := c.Param("org")
	teamSlug := c.Param("team_slug")

	pagination := GetPagination(c)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
		Role:    c.Query("role"),
	}

	members, err := h.teams.ListMembers(c.Context(), org, teamSlug, opts)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, members)
}

// GetTeamMembership handles GET /orgs/{org}/teams/{team_slug}/memberships/{username}
func (h *TeamHandler) GetTeamMembership(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")
	username := c.Param("username")

	membership, err := h.teams.GetMembership(c.Context(), org, teamSlug, username)
	if err != nil {
		if err == teams.ErrNotFound || err == teams.ErrNotMember {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, membership)
}

// AddTeamMember handles PUT /orgs/{org}/teams/{team_slug}/memberships/{username}
func (h *TeamHandler) AddTeamMember(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")
	username := c.Param("username")

	var in struct {
		Role string `json:"role,omitempty"`
	}
	c.BindJSON(&in, 1<<20) // optional

	if in.Role == "" {
		in.Role = "member"
	}

	membership, err := h.teams.AddMembership(c.Context(), org, teamSlug, username, in.Role)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, membership)
}

// RemoveTeamMember handles DELETE /orgs/{org}/teams/{team_slug}/memberships/{username}
func (h *TeamHandler) RemoveTeamMember(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")
	username := c.Param("username")

	if err := h.teams.RemoveMembership(c.Context(), org, teamSlug, username); err != nil {
		if err == teams.ErrNotFound || err == teams.ErrNotMember {
			return NotFound(c, "Membership")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListTeamRepos handles GET /orgs/{org}/teams/{team_slug}/repos
func (h *TeamHandler) ListTeamRepos(c *mizu.Ctx) error {
	org := c.Param("org")
	teamSlug := c.Param("team_slug")

	pagination := GetPagination(c)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	repos, err := h.teams.ListRepos(c.Context(), org, teamSlug, opts)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repos)
}

// CheckTeamRepoPermission handles GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *TeamHandler) CheckTeamRepoPermission(c *mizu.Ctx) error {
	org := c.Param("org")
	teamSlug := c.Param("team_slug")
	owner := c.Param("owner")
	repo := c.Param("repo")

	repoPerms, err := h.teams.CheckRepoPermission(c.Context(), org, teamSlug, owner, repo)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoPerms)
}

// AddTeamRepo handles PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *TeamHandler) AddTeamRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")
	owner := c.Param("owner")
	repo := c.Param("repo")

	var in struct {
		Permission string `json:"permission,omitempty"`
	}
	c.BindJSON(&in, 1<<20) // optional

	if in.Permission == "" {
		in.Permission = "push"
	}

	if err := h.teams.AddRepo(c.Context(), org, teamSlug, owner, repo, in.Permission); err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team or Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// RemoveTeamRepo handles DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *TeamHandler) RemoveTeamRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	teamSlug := c.Param("team_slug")
	owner := c.Param("owner")
	repo := c.Param("repo")

	if err := h.teams.RemoveRepo(c.Context(), org, teamSlug, owner, repo); err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team or Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListChildTeams handles GET /orgs/{org}/teams/{team_slug}/teams
func (h *TeamHandler) ListChildTeams(c *mizu.Ctx) error {
	org := c.Param("org")
	teamSlug := c.Param("team_slug")

	pagination := GetPagination(c)
	opts := &teams.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	children, err := h.teams.ListChildren(c.Context(), org, teamSlug, opts)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "Team")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, children)
}

// ListAuthenticatedUserTeams handles GET /user/teams
func (h *TeamHandler) ListAuthenticatedUserTeams(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	// TODO: Implement when teams.API.ListForUser is available
	// For now, return empty list
	return c.JSON(http.StatusOK, []*teams.Team{})
}
