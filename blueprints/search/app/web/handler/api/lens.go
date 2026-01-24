package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/store/postgres"
)

// LensHandler handles search lens operations
type LensHandler struct {
	store *postgres.Store
}

// NewLensHandler creates a new lens handler
func NewLensHandler(store *postgres.Store) *LensHandler {
	return &LensHandler{store: store}
}

// List returns all lenses
func (h *LensHandler) List(c *mizu.Ctx) error {
	lenses, err := h.store.Preference().ListLenses(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, lenses)
}

// Create creates a new lens
func (h *LensHandler) Create(c *mizu.Ctx) error {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Domains     []string `json:"domains"`
		Exclude     []string `json:"exclude"`
		Keywords    []string `json:"keywords"`
		IsPublic    bool     `json:"is_public"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}

	lens := &store.SearchLens{
		Name:        req.Name,
		Description: req.Description,
		Domains:     req.Domains,
		Exclude:     req.Exclude,
		Keywords:    req.Keywords,
		IsPublic:    req.IsPublic,
	}

	if err := h.store.Preference().CreateLens(c.Context(), lens); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, lens)
}

// Get returns a lens by ID
func (h *LensHandler) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}

	lens, err := h.store.Preference().GetLens(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "lens not found"})
	}

	return c.JSON(200, lens)
}

// Update updates a lens
func (h *LensHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Domains     []string `json:"domains"`
		Exclude     []string `json:"exclude"`
		Keywords    []string `json:"keywords"`
		IsPublic    bool     `json:"is_public"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	lens := &store.SearchLens{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Domains:     req.Domains,
		Exclude:     req.Exclude,
		Keywords:    req.Keywords,
		IsPublic:    req.IsPublic,
	}

	if err := h.store.Preference().UpdateLens(c.Context(), lens); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, lens)
}

// Delete deletes a lens
func (h *LensHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "id required"})
	}

	if err := h.store.Preference().DeleteLens(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "deleted"})
}
