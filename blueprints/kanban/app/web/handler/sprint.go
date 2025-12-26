package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/sprints"
)

// Sprint handles sprint endpoints.
type Sprint struct {
	sprints sprints.API
}

// NewSprint creates a new sprint handler.
func NewSprint(sprints sprints.API) *Sprint {
	return &Sprint{sprints: sprints}
}

// List returns all sprints for a project.
func (h *Sprint) List(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	list, err := h.sprints.ListByProject(c.Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list sprints"))
	}

	return c.JSON(http.StatusOK, list)
}

// Create creates a new sprint.
func (h *Sprint) Create(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	var in sprints.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	sprint, err := h.sprints.Create(c.Context(), projectID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to create sprint"))
	}

	return c.JSON(http.StatusCreated, sprint)
}

// Update updates a sprint.
func (h *Sprint) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in sprints.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	sprint, err := h.sprints.Update(c.Context(), id, &in)
	if err != nil {
		if err == sprints.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("sprint not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to update sprint"))
	}

	return c.JSON(http.StatusOK, sprint)
}

// Delete deletes a sprint.
func (h *Sprint) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.sprints.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to delete sprint"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "sprint deleted"})
}

// Start starts a sprint.
func (h *Sprint) Start(c *mizu.Ctx) error {
	id := c.Param("id")

	sprint, err := h.sprints.Start(c.Context(), id)
	if err != nil {
		if err == sprints.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("sprint not found"))
		}
		if err == sprints.ErrActiveExists {
			return c.JSON(http.StatusConflict, errResponse("an active sprint already exists"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to start sprint"))
	}

	return c.JSON(http.StatusOK, sprint)
}

// Complete completes a sprint.
func (h *Sprint) Complete(c *mizu.Ctx) error {
	id := c.Param("id")

	sprint, err := h.sprints.Complete(c.Context(), id)
	if err != nil {
		if err == sprints.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("sprint not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to complete sprint"))
	}

	return c.JSON(http.StatusOK, sprint)
}
