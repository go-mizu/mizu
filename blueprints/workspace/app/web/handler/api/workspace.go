package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/members"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
)

// Workspace handles workspace endpoints.
type Workspace struct {
	workspaces workspaces.API
	members    members.API
	getUserID  func(c *mizu.Ctx) string
}

// NewWorkspace creates a new Workspace handler.
func NewWorkspace(workspaces workspaces.API, members members.API, getUserID func(c *mizu.Ctx) string) *Workspace {
	return &Workspace{workspaces: workspaces, members: members, getUserID: getUserID}
}

// List lists workspaces for the current user.
func (h *Workspace) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	list, err := h.workspaces.ListByUser(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, list)
}

// Create creates a new workspace.
func (h *Workspace) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in workspaces.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	ws, err := h.workspaces.Create(c.Request().Context(), userID, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Add creator as owner
	h.members.Add(c.Request().Context(), ws.ID, userID, members.RoleOwner, userID)

	return c.JSON(http.StatusCreated, ws)
}

// Get retrieves a workspace.
func (h *Workspace) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	ws, err := h.workspaces.GetByID(c.Request().Context(), id)
	if err != nil {
		// Try by slug
		ws, err = h.workspaces.GetBySlug(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
		}
	}

	return c.JSON(http.StatusOK, ws)
}

// Update updates a workspace.
func (h *Workspace) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in workspaces.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	ws, err := h.workspaces.Update(c.Request().Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, ws)
}

// Delete deletes a workspace.
func (h *Workspace) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.workspaces.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// ListMembers lists workspace members.
func (h *Workspace) ListMembers(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.members.List(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, list)
}

// AddMember adds a member to the workspace.
func (h *Workspace) AddMember(c *mizu.Ctx) error {
	workspaceID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		UserID string       `json:"user_id"`
		Role   members.Role `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	member, err := h.members.Add(c.Request().Context(), workspaceID, in.UserID, in.Role, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, member)
}
