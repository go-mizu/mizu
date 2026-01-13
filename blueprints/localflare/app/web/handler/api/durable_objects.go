package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// DurableObjects handles Durable Objects requests.
type DurableObjects struct {
	store store.DurableObjectStore
}

// NewDurableObjects creates a new DurableObjects handler.
func NewDurableObjects(store store.DurableObjectStore) *DurableObjects {
	return &DurableObjects{store: store}
}

// ListNamespaces lists all DO namespaces.
func (h *DurableObjects) ListNamespaces(c *mizu.Ctx) error {
	namespaces, err := h.store.ListNamespaces(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  namespaces,
	})
}

// CreateDONamespaceInput is the input for creating a DO namespace.
type CreateDONamespaceInput struct {
	Name      string `json:"name"`
	Script    string `json:"script"`
	ClassName string `json:"class"`
}

// CreateNamespace creates a new DO namespace.
func (h *DurableObjects) CreateNamespace(c *mizu.Ctx) error {
	var input CreateDONamespaceInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" || input.ClassName == "" {
		return c.JSON(400, map[string]string{"error": "Name and class are required"})
	}

	ns := &store.DurableObjectNamespace{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		Script:    input.Script,
		ClassName: input.ClassName,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateNamespace(c.Request().Context(), ns); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  ns,
	})
}

// GetNamespace retrieves a namespace by ID.
func (h *DurableObjects) GetNamespace(c *mizu.Ctx) error {
	id := c.Param("id")
	ns, err := h.store.GetNamespace(c.Request().Context(), id)
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
	if err := h.store.DeleteNamespace(c.Request().Context(), id); err != nil {
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
	objects, err := h.store.ListInstances(c.Request().Context(), nsID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  objects,
	})
}
