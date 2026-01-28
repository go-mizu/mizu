package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/bang"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// BangHandler handles bang-related API requests.
type BangHandler struct {
	service *bang.Service
}

// NewBangHandler creates a new bang handler.
func NewBangHandler(st store.Store) *BangHandler {
	return &BangHandler{
		service: bang.NewService(st.Bang()),
	}
}

// List returns all available bangs.
func (h *BangHandler) List(c *mizu.Ctx) error {
	bangs, err := h.service.List(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, bangs)
}

// Parse parses a bang from a query and returns the result.
func (h *BangHandler) Parse(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, map[string]string{"error": "query required"})
	}

	result, err := h.service.Parse(c.Context(), query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result)
}

// Create creates a new custom bang.
func (h *BangHandler) Create(c *mizu.Ctx) error {
	var req struct {
		Trigger     string `json:"trigger"`
		Name        string `json:"name"`
		URLTemplate string `json:"url_template"`
		Category    string `json:"category"`
	}

	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Trigger == "" || req.Name == "" || req.URLTemplate == "" {
		return c.JSON(400, map[string]string{"error": "trigger, name, and url_template required"})
	}

	bang := &types.Bang{
		Trigger:     req.Trigger,
		Name:        req.Name,
		URLTemplate: req.URLTemplate,
		Category:    req.Category,
		IsBuiltin:   false,
	}

	if err := h.service.Create(c.Context(), bang); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, bang)
}

// Delete removes a custom bang.
func (h *BangHandler) Delete(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "deleted"})
}
