package api

import (
	"github.com/go-mizu/mizu"

	do "github.com/go-mizu/blueprints/localflare/feature/durable_objects"
)

// DurableObjects handles Durable Objects requests.
type DurableObjects struct {
	svc do.API
}

// NewDurableObjects creates a new DurableObjects handler.
func NewDurableObjects(svc do.API) *DurableObjects {
	return &DurableObjects{svc: svc}
}

// ListNamespaces lists all DO namespaces.
func (h *DurableObjects) ListNamespaces(c *mizu.Ctx) error {
	namespaces, err := h.svc.ListNamespaces(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"namespaces": namespaces,
		},
	})
}

// CreateNamespace creates a new DO namespace.
func (h *DurableObjects) CreateNamespace(c *mizu.Ctx) error {
	var input do.CreateNamespaceIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	ns, err := h.svc.CreateNamespace(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  ns,
	})
}

// GetNamespace retrieves a namespace by ID.
func (h *DurableObjects) GetNamespace(c *mizu.Ctx) error {
	id := c.Param("id")
	ns, err := h.svc.GetNamespace(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Namespace not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  ns,
	})
}

// DeleteNamespace deletes a DO namespace.
func (h *DurableObjects) DeleteNamespace(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.DeleteNamespace(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListObjects lists DO instances in a namespace.
func (h *DurableObjects) ListObjects(c *mizu.Ctx) error {
	nsID := c.Param("id")
	objects, err := h.svc.ListObjects(c.Request().Context(), nsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"objects": objects,
		},
	})
}
