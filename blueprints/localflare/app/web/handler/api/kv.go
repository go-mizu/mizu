package api

import (
	"fmt"
	"io"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/kv"
)

// KV handles KV namespace requests.
type KV struct {
	svc kv.API
}

// NewKV creates a new KV handler.
func NewKV(svc kv.API) *KV {
	return &KV{svc: svc}
}

// ListNamespaces lists all KV namespaces.
func (h *KV) ListNamespaces(c *mizu.Ctx) error {
	namespaces, err := h.svc.ListNamespaces(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  namespaces,
	})
}

// CreateNamespace creates a new KV namespace.
func (h *KV) CreateNamespace(c *mizu.Ctx) error {
	var input kv.CreateNamespaceIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	ns, err := h.svc.CreateNamespace(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  ns,
	})
}

// DeleteNamespace deletes a KV namespace.
func (h *KV) DeleteNamespace(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.DeleteNamespace(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListKeys lists keys in a KV namespace.
func (h *KV) ListKeys(c *mizu.Ctx) error {
	nsID := c.Param("id")
	opts := kv.ListOpts{
		Prefix: c.Query("prefix"),
		Limit:  1000,
	}

	keys, err := h.svc.ListKeys(c.Request().Context(), nsID, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  keys,
	})
}

// GetValue retrieves a value from a KV namespace.
func (h *KV) GetValue(c *mizu.Ctx) error {
	nsID := c.Param("id")
	key := c.Param("key")

	pair, err := h.svc.Get(c.Request().Context(), nsID, key)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Key not found"})
	}

	c.Writer().Header().Set("Content-Type", "application/octet-stream")
	if pair.Expiration != nil {
		c.Writer().Header().Set("CF-Expiration", pair.Expiration.Format(time.RFC3339))
	}
	c.Writer().Write(pair.Value)
	return nil
}

// PutValue stores a value in a KV namespace.
func (h *KV) PutValue(c *mizu.Ctx) error {
	nsID := c.Param("id")
	key := c.Param("key")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Failed to read body"})
	}

	input := &kv.PutIn{
		Key:   key,
		Value: body,
	}

	// Check for expiration header
	if exp := c.Request().Header.Get("CF-Expiration"); exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil {
			input.Expiration = &t
		}
	}

	// Check for expiration_ttl query param
	if ttl := c.Query("expiration_ttl"); ttl != "" {
		var seconds int
		if _, err := fmt.Sscanf(ttl, "%d", &seconds); err == nil {
			exp := time.Now().Add(time.Duration(seconds) * time.Second)
			input.Expiration = &exp
		}
	}

	if err := h.svc.Put(c.Request().Context(), nsID, input); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}

// DeleteValue deletes a value from a KV namespace.
func (h *KV) DeleteValue(c *mizu.Ctx) error {
	nsID := c.Param("id")
	key := c.Param("key")

	if err := h.svc.Delete(c.Request().Context(), nsID, key); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}
