package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/teams"
)

// Team handles team endpoints.
type Team struct {
	teams     teams.API
	getUserID func(*mizu.Ctx) string
}

// NewTeam creates a new team handler.
func NewTeam(teams teams.API, getUserID func(*mizu.Ctx) string) *Team {
	return &Team{teams: teams, getUserID: getUserID}
}

// List returns all teams for a workspace.
func (h *Team) List(c *mizu.Ctx) error {
	workspaceID := c.Param("workspaceID")

	list, err := h.teams.ListByWorkspace(c.Context(), workspaceID)
	if err != nil {
		return InternalError(c, "failed to list teams")
	}

	return OK(c, list)
}

// Create creates a new team.
func (h *Team) Create(c *mizu.Ctx) error {
	workspaceID := c.Param("workspaceID")

	var in teams.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	team, err := h.teams.Create(c.Context(), workspaceID, &in)
	if err != nil {
		if err == teams.ErrKeyExists {
			return BadRequest(c, "team key already exists")
		}
		return InternalError(c, "failed to create team")
	}

	// Add creator as lead
	userID := h.getUserID(c)
	if userID != "" {
		_ = h.teams.AddMember(c.Context(), team.ID, userID, teams.RoleLead)
	}

	return Created(c, team)
}

// Get returns a team by ID.
func (h *Team) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	team, err := h.teams.GetByID(c.Context(), id)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "team not found")
		}
		return InternalError(c, "failed to get team")
	}

	return OK(c, team)
}

// Update updates a team.
func (h *Team) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in teams.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	team, err := h.teams.Update(c.Context(), id, &in)
	if err != nil {
		if err == teams.ErrNotFound {
			return NotFound(c, "team not found")
		}
		return InternalError(c, "failed to update team")
	}

	return OK(c, team)
}

// Delete deletes a team.
func (h *Team) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.teams.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete team")
	}

	return OK(c, map[string]string{"message": "team deleted"})
}

// ListMembers returns all members of a team.
func (h *Team) ListMembers(c *mizu.Ctx) error {
	id := c.Param("id")

	members, err := h.teams.ListMembers(c.Context(), id)
	if err != nil {
		return InternalError(c, "failed to list members")
	}

	return OK(c, members)
}

// AddMember adds a member to a team.
func (h *Team) AddMember(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if in.Role == "" {
		in.Role = teams.RoleMember
	}

	if err := h.teams.AddMember(c.Context(), id, in.UserID, in.Role); err != nil {
		if err == teams.ErrMemberExists {
			return BadRequest(c, "user is already a member")
		}
		return InternalError(c, "failed to add member")
	}

	return Created(c, map[string]string{"message": "member added"})
}

// UpdateMemberRole updates a member's role.
func (h *Team) UpdateMemberRole(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := c.Param("userID")

	var in struct {
		Role string `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.teams.UpdateMemberRole(c.Context(), id, userID, in.Role); err != nil {
		return InternalError(c, "failed to update role")
	}

	return OK(c, map[string]string{"message": "role updated"})
}

// RemoveMember removes a member from a team.
func (h *Team) RemoveMember(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := c.Param("userID")

	if err := h.teams.RemoveMember(c.Context(), id, userID); err != nil {
		return InternalError(c, "failed to remove member")
	}

	return OK(c, map[string]string{"message": "member removed"})
}
