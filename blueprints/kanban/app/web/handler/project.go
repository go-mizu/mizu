package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// Project handles project endpoints.
type Project struct {
	projects   projects.API
	workspaces workspaces.API
	getUserID  func(*mizu.Ctx) string
}

// NewProject creates a new project handler.
func NewProject(projects projects.API, workspaces workspaces.API, getUserID func(*mizu.Ctx) string) *Project {
	return &Project{projects: projects, workspaces: workspaces, getUserID: getUserID}
}

// List returns all projects in a workspace.
func (h *Project) List(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
	}

	list, err := h.projects.ListByWorkspace(c.Context(), ws.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list projects"))
	}

	return c.JSON(http.StatusOK, list)
}

// Create creates a new project.
func (h *Project) Create(c *mizu.Ctx) error {
	slug := c.Param("slug")
	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
	}

	var in projects.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	project, err := h.projects.Create(c.Context(), ws.ID, &in)
	if err != nil {
		if err == projects.ErrKeyExists {
			return c.JSON(http.StatusConflict, errResponse("project key already exists"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to create project"))
	}

	return c.JSON(http.StatusCreated, project)
}

// Get returns a project by key.
func (h *Project) Get(c *mizu.Ctx) error {
	slug := c.Param("slug")
	key := c.Param("key")

	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
	}

	project, err := h.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		if err == projects.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("project not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get project"))
	}

	return c.JSON(http.StatusOK, project)
}

// Update updates a project.
func (h *Project) Update(c *mizu.Ctx) error {
	slug := c.Param("slug")
	key := c.Param("key")

	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
	}

	project, err := h.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		if err == projects.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("project not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get project"))
	}

	var in projects.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	updated, err := h.projects.Update(c.Context(), project.ID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to update project"))
	}

	return c.JSON(http.StatusOK, updated)
}

// Delete deletes a project.
func (h *Project) Delete(c *mizu.Ctx) error {
	slug := c.Param("slug")
	key := c.Param("key")

	ws, err := h.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, errResponse("workspace not found"))
	}

	project, err := h.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		if err == projects.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("project not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to get project"))
	}

	if err := h.projects.Delete(c.Context(), project.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to delete project"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "project deleted"})
}
