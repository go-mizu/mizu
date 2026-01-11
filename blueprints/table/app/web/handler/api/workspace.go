package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
)

// Workspace handles workspace endpoints.
type Workspace struct {
	workspaces *workspaces.Service
	bases      *bases.Service
	getUserID  func(*mizu.Ctx) string
}

// NewWorkspace creates a new workspace handler.
func NewWorkspace(ws *workspaces.Service, bases *bases.Service, getUserID func(*mizu.Ctx) string) *Workspace {
	return &Workspace{workspaces: ws, bases: bases, getUserID: getUserID}
}

// List returns all workspaces for the current user.
func (h *Workspace) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	list, err := h.workspaces.ListByUser(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to list workspaces")
	}

	return OK(c, map[string]any{"workspaces": list})
}

// Create creates a new workspace.
func (h *Workspace) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in workspaces.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	ws, err := h.workspaces.Create(c.Context(), userID, in)
	if err != nil {
		if err == workspaces.ErrSlugTaken {
			return BadRequest(c, "workspace slug already taken")
		}
		return InternalError(c, "failed to create workspace")
	}

	return Created(c, map[string]any{"workspace": ws})
}

// Get returns a workspace by ID.
func (h *Workspace) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	ws, err := h.workspaces.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "workspace not found")
	}

	return OK(c, map[string]any{"workspace": ws})
}

// Update updates a workspace.
func (h *Workspace) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in workspaces.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	ws, err := h.workspaces.Update(c.Context(), id, in)
	if err != nil {
		if err == workspaces.ErrNotFound {
			return NotFound(c, "workspace not found")
		}
		return InternalError(c, "failed to update workspace")
	}

	return OK(c, map[string]any{"workspace": ws})
}

// Delete deletes a workspace.
func (h *Workspace) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.workspaces.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete workspace")
	}

	return NoContent(c)
}

// ListBases returns all bases in a workspace.
func (h *Workspace) ListBases(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.bases.ListByWorkspace(c.Context(), id)
	if err != nil {
		return InternalError(c, "failed to list bases")
	}

	return OK(c, map[string]any{"bases": list})
}
