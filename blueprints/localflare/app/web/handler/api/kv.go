package api

import (
	"fmt"
	"io"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// KV handles KV namespace requests.
type KV struct {
	store store.KVStore
}

// NewKV creates a new KV handler.
func NewKV(store store.KVStore) *KV {
	return &KV{store: store}
}

// ListNamespaces lists all KV namespaces.
func (h *KV) ListNamespaces(c *mizu.Ctx) error {
	namespaces, err := h.store.ListNamespaces(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  namespaces,
	})
}

// CreateNamespaceInput is the input for creating a KV namespace.
type CreateNamespaceInput struct {
	Title string `json:"title"`
}

// CreateNamespace creates a new KV namespace.
func (h *KV) CreateNamespace(c *mizu.Ctx) error {
	var input CreateNamespaceInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Title == "" {
		return c.JSON(400, map[string]string{"error": "Title is required"})
	}

	ns := &store.KVNamespace{
		ID:        ulid.Make().String(),
		Title:     input.Title,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateNamespace(c.Request().Context(), ns); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  ns,
	})
}

// DeleteNamespace deletes a KV namespace.
func (h *KV) DeleteNamespace(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteNamespace(c.Request().Context(), id); err != nil {
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
	prefix := c.Query("prefix")
	limit := 1000

	pairs, err := h.store.List(c.Request().Context(), nsID, prefix, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Return only keys, not values
	keys := make([]map[string]interface{}, len(pairs))
	for i, pair := range pairs {
		keys[i] = map[string]interface{}{
			"name":       pair.Key,
			"expiration": pair.Expiration,
			"metadata":   pair.Metadata,
		}
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

	pair, err := h.store.Get(c.Request().Context(), nsID, key)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Key not found"})
	}

	// Return raw value
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

	pair := &store.KVPair{
		Key:   key,
		Value: body,
	}

	// Check for expiration header
	if exp := c.Request().Header.Get("CF-Expiration"); exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil {
			pair.Expiration = &t
		}
	}

	// Check for expiration_ttl query param
	if ttl := c.Query("expiration_ttl"); ttl != "" {
		var seconds int
		if _, err := fmt.Sscanf(ttl, "%d", &seconds); err == nil {
			exp := time.Now().Add(time.Duration(seconds) * time.Second)
			pair.Expiration = &exp
		}
	}

	if err := h.store.Put(c.Request().Context(), nsID, pair); err != nil {
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

	if err := h.store.Delete(c.Request().Context(), nsID, key); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}

