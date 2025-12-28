package api

import (
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Label handles label endpoints
type Label struct {
	labels    labels.API
	repos     repos.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewLabel creates a new label handler
func NewLabel(labels labels.API, repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Label {
	return &Label{
		labels:    labels,
		repos:     repos,
		users:     users,
		getUserID: getUserID,
	}
}

func (h *Label) getRepo(c *mizu.Ctx) (*repos.Repository, error) {
	owner := c.Param("owner")
	name := c.Param("repo")

	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return nil, repos.ErrNotFound
	}

	return h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
}

// List lists labels for a repository
func (h *Label) List(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	labelList, err := h.labels.List(c.Context(), repo.ID)
	if err != nil {
		return InternalError(c, "failed to list labels")
	}

	return OK(c, labelList)
}

// Create creates a new label
func (h *Label) Create(c *mizu.Ctx) error {
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

	var in labels.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	label, err := h.labels.Create(c.Context(), repo.ID, &in)
	if err != nil {
		switch err {
		case labels.ErrExists:
			return Conflict(c, "label already exists")
		case labels.ErrMissingName:
			return BadRequest(c, "label name is required")
		default:
			return InternalError(c, "failed to create label")
		}
	}

	return Created(c, label)
}

// Get retrieves a label by name
func (h *Label) Get(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	name := c.Param("name")
	if name == "" {
		return BadRequest(c, "label name is required")
	}

	label, err := h.labels.GetByName(c.Context(), repo.ID, name)
	if err != nil {
		return NotFound(c, "label not found")
	}

	return OK(c, label)
}

// Update updates a label
func (h *Label) Update(c *mizu.Ctx) error {
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

	name := c.Param("name")
	label, err := h.labels.GetByName(c.Context(), repo.ID, name)
	if err != nil {
		return NotFound(c, "label not found")
	}

	var in labels.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	label, err = h.labels.Update(c.Context(), label.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update label")
	}

	return OK(c, label)
}

// Delete deletes a label
func (h *Label) Delete(c *mizu.Ctx) error {
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

	name := c.Param("name")
	label, err := h.labels.GetByName(c.Context(), repo.ID, name)
	if err != nil {
		return NotFound(c, "label not found")
	}

	if err := h.labels.Delete(c.Context(), label.ID); err != nil {
		return InternalError(c, "failed to delete label")
	}

	return NoContent(c)
}
