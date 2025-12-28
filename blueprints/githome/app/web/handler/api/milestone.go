package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Milestone handles milestone endpoints
type Milestone struct {
	milestones milestones.API
	repos      repos.API
	users      users.API
	getUserID  func(*mizu.Ctx) string
}

// NewMilestone creates a new milestone handler
func NewMilestone(milestones milestones.API, repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Milestone {
	return &Milestone{
		milestones: milestones,
		repos:      repos,
		users:      users,
		getUserID:  getUserID,
	}
}

func (h *Milestone) getRepo(c *mizu.Ctx) (*repos.Repository, error) {
	owner := c.Param("owner")
	name := c.Param("repo")

	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return nil, repos.ErrNotFound
	}

	return h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
}

// List lists milestones for a repository
func (h *Milestone) List(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	opts := &milestones.ListOpts{
		State:     c.Query("state"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	milestoneList, err := h.milestones.List(c.Context(), repo.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list milestones")
	}

	return OK(c, milestoneList)
}

// Create creates a new milestone
func (h *Milestone) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in milestones.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	milestone, err := h.milestones.Create(c.Context(), repo.ID, &in)
	if err != nil {
		switch err {
		case milestones.ErrMissingTitle:
			return BadRequest(c, "milestone title is required")
		default:
			return InternalError(c, "failed to create milestone")
		}
	}

	return Created(c, milestone)
}

// Get retrieves a milestone by number
func (h *Milestone) Get(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid milestone number")
	}

	milestone, err := h.milestones.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "milestone not found")
	}

	return OK(c, milestone)
}

// Update updates a milestone
func (h *Milestone) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid milestone number")
	}

	milestone, err := h.milestones.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "milestone not found")
	}

	var in milestones.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	milestone, err = h.milestones.Update(c.Context(), milestone.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update milestone")
	}

	return OK(c, milestone)
}

// Delete deletes a milestone
func (h *Milestone) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid milestone number")
	}

	milestone, err := h.milestones.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "milestone not found")
	}

	if err := h.milestones.Delete(c.Context(), milestone.ID); err != nil {
		return InternalError(c, "failed to delete milestone")
	}

	return NoContent(c)
}
