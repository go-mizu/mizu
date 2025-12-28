package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Org handles organization endpoints
type Org struct {
	orgs      orgs.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewOrg creates a new org handler
func NewOrg(orgs orgs.API, users users.API, getUserID func(*mizu.Ctx) string) *Org {
	return &Org{
		orgs:      orgs,
		users:     users,
		getUserID: getUserID,
	}
}

// List lists all organizations
func (h *Org) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &orgs.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	orgList, err := h.orgs.List(c.Context(), opts)
	if err != nil {
		return InternalError(c, "failed to list organizations")
	}

	return OK(c, orgList)
}

// Create creates a new organization
func (h *Org) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	var in orgs.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	org, err := h.orgs.Create(c.Context(), userID, &in)
	if err != nil {
		switch err {
		case orgs.ErrExists:
			return Conflict(c, "organization already exists")
		case orgs.ErrMissingName:
			return BadRequest(c, "organization name is required")
		default:
			return InternalError(c, "failed to create organization")
		}
	}

	return Created(c, org)
}

// Get retrieves an organization by slug
func (h *Org) Get(c *mizu.Ctx) error {
	slug := c.Param("org")
	if slug == "" {
		return BadRequest(c, "organization slug is required")
	}

	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	return OK(c, org)
}

// Update updates an organization
func (h *Org) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner
	isOwner, _ := h.orgs.IsOwner(c.Context(), org.ID, userID)
	if !isOwner {
		return Forbidden(c, "insufficient permissions")
	}

	var in orgs.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	org, err = h.orgs.Update(c.Context(), org.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update organization")
	}

	return OK(c, org)
}

// Delete deletes an organization
func (h *Org) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner
	isOwner, _ := h.orgs.IsOwner(c.Context(), org.ID, userID)
	if !isOwner {
		return Forbidden(c, "only owners can delete an organization")
	}

	if err := h.orgs.Delete(c.Context(), org.ID); err != nil {
		return InternalError(c, "failed to delete organization")
	}

	return NoContent(c)
}

// ListMembers lists members of an organization
func (h *Org) ListMembers(c *mizu.Ctx) error {
	slug := c.Param("org")
	org, err := h.orgs.GetBySlug(c.Context(), slug)
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

	opts := &orgs.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	members, err := h.orgs.ListMembers(c.Context(), org.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list members")
	}

	return OK(c, members)
}

// GetMember retrieves a member of an organization
func (h *Org) GetMember(c *mizu.Ctx) error {
	slug := c.Param("org")
	username := c.Param("username")

	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	member, err := h.orgs.GetMember(c.Context(), org.ID, user.ID)
	if err != nil {
		return NotFound(c, "member not found")
	}

	return OK(c, member)
}

// AddMember adds a member to an organization
func (h *Org) AddMember(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	username := c.Param("username")

	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	role := c.Query("role")
	if role == "" {
		role = orgs.RoleMember
	}

	if err := h.orgs.AddMember(c.Context(), org.ID, user.ID, role); err != nil {
		switch err {
		case orgs.ErrMemberExists:
			return Conflict(c, "member already exists")
		default:
			return InternalError(c, "failed to add member")
		}
	}

	return NoContent(c)
}

// UpdateMemberRole updates a member's role
func (h *Org) UpdateMemberRole(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	username := c.Param("username")

	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner
	isOwner, _ := h.orgs.IsOwner(c.Context(), org.ID, userID)
	if !isOwner {
		return Forbidden(c, "only owners can update member roles")
	}

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

	if err := h.orgs.UpdateMemberRole(c.Context(), org.ID, user.ID, in.Role); err != nil {
		return InternalError(c, "failed to update member role")
	}

	return NoContent(c)
}

// RemoveMember removes a member from an organization
func (h *Org) RemoveMember(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	username := c.Param("username")

	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	if err := h.orgs.RemoveMember(c.Context(), org.ID, user.ID); err != nil {
		switch err {
		case orgs.ErrLastOwner:
			return Conflict(c, "cannot remove the last owner")
		case orgs.ErrMemberNotFound:
			return NotFound(c, "member not found")
		default:
			return InternalError(c, "failed to remove member")
		}
	}

	return NoContent(c)
}

// CheckMembership checks if a user is a member of an organization
func (h *Org) CheckMembership(c *mizu.Ctx) error {
	slug := c.Param("org")
	username := c.Param("username")

	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	isMember, err := h.orgs.IsMember(c.Context(), org.ID, user.ID)
	if err != nil {
		return InternalError(c, "failed to check membership")
	}

	if !isMember {
		return NotFound(c, "not a member")
	}

	return NoContent(c)
}

// ListUserOrgs lists organizations for a user
func (h *Org) ListUserOrgs(c *mizu.Ctx) error {
	username := c.Param("username")

	var userID string
	if username == "" {
		// Current user
		userID = h.getUserID(c)
		if userID == "" {
			return Unauthorized(c, "not authenticated")
		}
	} else {
		user, err := h.users.GetByUsername(c.Context(), username)
		if err != nil {
			return NotFound(c, "user not found")
		}
		userID = user.ID
	}

	orgList, err := h.orgs.ListUserOrgs(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to list organizations")
	}

	return OK(c, orgList)
}
