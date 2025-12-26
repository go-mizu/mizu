package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/projects"
)

// Project handles project endpoints.
type Project struct {
	projects  projects.API
	getUserID func(*mizu.Ctx) string
}

// NewProject creates a new project handler.
func NewProject(projects projects.API, getUserID func(*mizu.Ctx) string) *Project {
	return &Project{projects: projects, getUserID: getUserID}
}

// List returns all projects in a team.
func (h *Project) List(c *mizu.Ctx) error {
	teamID := c.Param("teamID")

	list, err := h.projects.ListByTeam(c.Context(), teamID)
	if err != nil {
		return InternalError(c, "failed to list projects")
	}

	return OK(c, list)
}

// Create creates a new project.
func (h *Project) Create(c *mizu.Ctx) error {
	teamID := c.Param("teamID")

	var in projects.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	project, err := h.projects.Create(c.Context(), teamID, &in)
	if err != nil {
		if err == projects.ErrKeyExists {
			return BadRequest(c, "project key already exists")
		}
		return InternalError(c, "failed to create project")
	}

	return Created(c, project)
}

// Get returns a project by ID.
func (h *Project) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	project, err := h.projects.GetByID(c.Context(), id)
	if err != nil {
		if err == projects.ErrNotFound {
			return NotFound(c, "project not found")
		}
		return InternalError(c, "failed to get project")
	}

	return OK(c, project)
}

// Update updates a project.
func (h *Project) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in projects.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	updated, err := h.projects.Update(c.Context(), id, &in)
	if err != nil {
		if err == projects.ErrNotFound {
			return NotFound(c, "project not found")
		}
		return InternalError(c, "failed to update project")
	}

	return OK(c, updated)
}

// Delete deletes a project.
func (h *Project) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.projects.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete project")
	}

	return OK(c, map[string]string{"message": "project deleted"})
}
