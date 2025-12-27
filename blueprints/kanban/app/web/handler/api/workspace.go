package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// Workspace handles workspace endpoints.
type Workspace struct {
	workspaces workspaces.API
	getUserID  func(*mizu.Ctx) string
}

// NewWorkspace creates a new workspace handler.
func NewWorkspace(workspaces workspaces.API, getUserID func(*mizu.Ctx) string) *Workspace {
	return &Workspace{workspaces: workspaces, getUserID: getUserID}
}

// List returns all workspaces for the current user.
func (h *Workspace) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	list, err := h.workspaces.ListByUser(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list workspaces"))
	}
	return c.JSON(http.StatusOK, list)
}

// Create creates a new workspace.
func (h *Workspace) Create(c *mizu.Ctx) error {
	var in workspaces.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	userID := h.getUserID(c)
	ws, err := h.workspaces.Create(c.Context(), userID, &in)
	if err != nil {
		if err == workspaces.ErrSlugExists {
			return c.JSON(http.StatusConflict, errResponse("workspace slug already exists"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to create workspace"))
	}

	return c.JSON(http.StatusCreated, ws)
}

// Get returns a workspace by slug.
func (h *Workspace) Get(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == workspaces.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get workspace"))
	}
	return c.JSON(http.StatusOK, ws)
}

// Update updates a workspace.
func (h *Workspace) Update(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == workspaces.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get workspace"))
	}

	var in workspaces.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	updated, err := h.workspaces.Update(c.Context(), ws.ID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to update workspace"))
	}

	return c.JSON(http.StatusOK, updated)
}

// Delete deletes a workspace.
func (h *Workspace) Delete(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == workspaces.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get workspace"))
	}

	if err := h.workspaces.Delete(c.Context(), ws.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to delete workspace"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "workspace deleted"})
}

// ListMembers returns all members of a workspace.
func (h *Workspace) ListMembers(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == workspaces.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get workspace"))
	}

	members, err := h.workspaces.ListMembers(c.Context(), ws.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list members"))
	}

	return c.JSON(http.StatusOK, members)
}

// AddMember adds a member to a workspace.
func (h *Workspace) AddMember(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == workspaces.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get workspace"))
	}

	var in struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	member, err := h.workspaces.AddMember(c.Context(), ws.ID, in.UserID, in.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to add member"))
	}

	return c.JSON(http.StatusCreated, member)
}
