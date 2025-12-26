package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/labels"
)

// Label handles label endpoints.
type Label struct {
	labels labels.API
}

// NewLabel creates a new label handler.
func NewLabel(labels labels.API) *Label {
	return &Label{labels: labels}
}

// List returns all labels for a project.
func (h *Label) List(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	list, err := h.labels.ListByProject(c.Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to list labels"))
	}

	return c.JSON(http.StatusOK, list)
}

// Create creates a new label.
func (h *Label) Create(c *mizu.Ctx) error {
	projectID := c.Param("projectID")

	var in labels.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	label, err := h.labels.Create(c.Context(), projectID, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to create label"))
	}

	return c.JSON(http.StatusCreated, label)
}

// Update updates a label.
func (h *Label) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in labels.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	label, err := h.labels.Update(c.Context(), id, &in)
	if err != nil {
		if err == labels.ErrNotFound {
			return c.JSON(http.StatusNotFound, errResponse("label not found"))
		}
		return c.JSON(http.StatusInternalServerError, errResponse("failed to update label"))
	}

	return c.JSON(http.StatusOK, label)
}

// Delete deletes a label.
func (h *Label) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.labels.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("failed to delete label"))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "label deleted"})
}
