package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Team handles team endpoints
type Team struct {
	teams     teams.API
	orgs      orgs.API
	repos     repos.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewTeam creates a new team handler
func NewTeam(teams teams.API, orgs orgs.API, repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Team {
	return &Team{
		teams:     teams,
		orgs:      orgs,
		repos:     repos,
		users:     users,
		getUserID: getUserID,
	}
}

func (h *Team) getOrg(c *mizu.Ctx) (*orgs.Organization, error) {
	slug := c.Param("org")
	return h.orgs.GetBySlug(c.Context(), slug)
}

// List lists teams for an organization
func (h *Team) List(c *mizu.Ctx) error {
	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &teams.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	teamList, err := h.teams.List(c.Context(), org.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list teams")
	}

	return OK(c, teamList)
}

// Create creates a new team
func (h *Team) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	var in teams.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	team, err := h.teams.Create(c.Context(), org.ID, &in)
	if err != nil {
		switch err {
		case teams.ErrExists:
			return Conflict(c, "team already exists")
		case teams.ErrMissingName:
			return BadRequest(c, "team name is required")
		default:
			return InternalError(c, "failed to create team")
		}
	}

	return Created(c, team)
}

// Get retrieves a team by slug
func (h *Team) Get(c *mizu.Ctx) error {
	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	teamSlug := c.Param("team")
	if teamSlug == "" {
		return BadRequest(c, "team slug is required")
	}

	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	return OK(c, team)
}

// Update updates a team
func (h *Team) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	var in teams.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	team, err = h.teams.Update(c.Context(), team.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update team")
	}

	return OK(c, team)
}

// Delete deletes a team
func (h *Team) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner
	isOwner, _ := h.orgs.IsOwner(c.Context(), org.ID, userID)
	if !isOwner {
		return Forbidden(c, "only owners can delete teams")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	if err := h.teams.Delete(c.Context(), team.ID); err != nil {
		return InternalError(c, "failed to delete team")
	}

	return NoContent(c)
}

// ListMembers lists members of a team
func (h *Team) ListMembers(c *mizu.Ctx) error {
	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &teams.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	members, err := h.teams.ListMembers(c.Context(), team.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list members")
	}

	return OK(c, members)
}

// GetMember retrieves a team member
func (h *Team) GetMember(c *mizu.Ctx) error {
	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	username := c.Param("username")
	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	member, err := h.teams.GetMember(c.Context(), team.ID, user.ID)
	if err != nil {
		return NotFound(c, "member not found")
	}

	return OK(c, member)
}

// AddMember adds a member to a team
func (h *Team) AddMember(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	username := c.Param("username")
	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	role := c.Query("role")
	if role == "" {
		role = teams.RoleMember
	}

	if err := h.teams.AddMember(c.Context(), team.ID, user.ID, role); err != nil {
		switch err {
		case teams.ErrMemberExists:
			return Conflict(c, "member already exists")
		default:
			return InternalError(c, "failed to add member")
		}
	}

	return NoContent(c)
}

// UpdateMemberRole updates a team member's role
func (h *Team) UpdateMemberRole(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	username := c.Param("username")
	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	var in struct {
		Role string `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.teams.UpdateMemberRole(c.Context(), team.ID, user.ID, in.Role); err != nil {
		return InternalError(c, "failed to update member role")
	}

	return NoContent(c)
}

// RemoveMember removes a member from a team
func (h *Team) RemoveMember(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	username := c.Param("username")
	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	if err := h.teams.RemoveMember(c.Context(), team.ID, user.ID); err != nil {
		switch err {
		case teams.ErrMemberNotFound:
			return NotFound(c, "member not found")
		default:
			return InternalError(c, "failed to remove member")
		}
	}

	return NoContent(c)
}

// ListRepos lists repositories accessible to a team
func (h *Team) ListRepos(c *mizu.Ctx) error {
	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &teams.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	teamRepos, err := h.teams.ListRepos(c.Context(), team.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list repositories")
	}

	return OK(c, teamRepos)
}

// GetRepoAccess gets team access to a repository
func (h *Team) GetRepoAccess(c *mizu.Ctx) error {
	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	teamRepo, err := h.teams.GetRepoAccess(c.Context(), team.ID, repo.ID)
	if err != nil {
		return NotFound(c, "team does not have access to this repository")
	}

	return OK(c, teamRepo)
}

// AddRepo adds a repository to a team
func (h *Team) AddRepo(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	permission := c.Query("permission")
	if permission == "" {
		permission = teams.PermissionRead
	}

	if err := h.teams.AddRepo(c.Context(), team.ID, repo.ID, permission); err != nil {
		switch err {
		case teams.ErrRepoExists:
			return Conflict(c, "repository already added to team")
		default:
			return InternalError(c, "failed to add repository")
		}
	}

	return NoContent(c)
}

// UpdateRepoPermission updates team permission for a repository
func (h *Team) UpdateRepoPermission(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	var in struct {
		Permission string `json:"permission"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.teams.UpdateRepoPermission(c.Context(), team.ID, repo.ID, in.Permission); err != nil {
		return InternalError(c, "failed to update repository permission")
	}

	return NoContent(c)
}

// RemoveRepo removes a repository from a team
func (h *Team) RemoveRepo(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	teamSlug := c.Param("team")
	team, err := h.teams.GetBySlug(c.Context(), org.ID, teamSlug)
	if err != nil {
		return NotFound(c, "team not found")
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	if err := h.teams.RemoveRepo(c.Context(), team.ID, repo.ID); err != nil {
		switch err {
		case teams.ErrRepoNotFound:
			return NotFound(c, "repository not found in team")
		default:
			return InternalError(c, "failed to remove repository")
		}
	}

	return NoContent(c)
}

// ListUserTeams lists teams for the current user in an organization
func (h *Team) ListUserTeams(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	org, err := h.getOrg(c)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	teamList, err := h.teams.ListUserTeams(c.Context(), org.ID, userID)
	if err != nil {
		return InternalError(c, "failed to list teams")
	}

	return OK(c, teamList)
}
