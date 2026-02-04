package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// LabelHandler handles label API endpoints.
type LabelHandler struct {
	store store.Store
}

// NewLabelHandler creates a new label handler.
func NewLabelHandler(st store.Store) *LabelHandler {
	return &LabelHandler{store: st}
}

// List returns all labels with their unread and total counts.
func (h *LabelHandler) List(c *mizu.Ctx) error {
	labels, err := h.store.ListLabels(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list labels"})
	}

	return c.JSON(http.StatusOK, labels)
}

// Create creates a new user label.
func (h *LabelHandler) Create(c *mizu.Ctx) error {
	var req struct {
		Name    string `json:"name"`
		Color   string `json:"color"`
		Visible *bool  `json:"visible"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "label name is required"})
	}

	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}

	label := &types.Label{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Color:     req.Color,
		Type:      types.LabelTypeUser,
		Visible:   visible,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.store.CreateLabel(c.Context(), label); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create label"})
	}

	return c.JSON(http.StatusCreated, label)
}

// Update updates an existing label's properties.
func (h *LabelHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "label id is required"})
	}

	var updates map[string]any
	if err := c.BindJSON(&updates, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Only allow specific fields to be updated
	allowed := map[string]bool{
		"name":     true,
		"color":    true,
		"visible":  true,
		"position": true,
	}
	filtered := make(map[string]any)
	for k, v := range updates {
		if allowed[k] {
			filtered[k] = v
		}
	}

	if len(filtered) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no valid fields to update"})
	}

	if err := h.store.UpdateLabel(c.Context(), id, filtered); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "label not found"})
	}

	// Return updated labels list
	labels, err := h.store.ListLabels(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch labels"})
	}

	// Find the updated label to return
	for _, l := range labels {
		if l.ID == id {
			return c.JSON(http.StatusOK, l)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "label updated"})
}

// Delete removes a label by ID. System labels cannot be deleted.
func (h *LabelHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "label id is required"})
	}

	// Check if label exists and is not a system label
	labels, err := h.store.ListLabels(c.Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to check label"})
	}

	found := false
	for _, l := range labels {
		if l.ID == id {
			found = true
			if l.Type == types.LabelTypeSystem {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot delete system labels"})
			}
			break
		}
	}
	if !found {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "label not found"})
	}

	if err := h.store.DeleteLabel(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete label"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "label deleted"})
}
