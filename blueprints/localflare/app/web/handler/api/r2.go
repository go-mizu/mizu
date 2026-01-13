package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// R2 handles R2 storage requests.
type R2 struct {
	store store.R2Store
}

// NewR2 creates a new R2 handler.
func NewR2(store store.R2Store) *R2 {
	return &R2{store: store}
}

// ListBuckets lists all R2 buckets.
func (h *R2) ListBuckets(c *mizu.Ctx) error {
	buckets, err := h.store.ListBuckets(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  buckets,
	})
}

// GetBucket retrieves a bucket by ID.
func (h *R2) GetBucket(c *mizu.Ctx) error {
	id := c.Param("id")
	bucket, err := h.store.GetBucket(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Bucket not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  bucket,
	})
}

// CreateBucketInput is the input for creating an R2 bucket.
type CreateBucketInput struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

// CreateBucket creates a new R2 bucket.
func (h *R2) CreateBucket(c *mizu.Ctx) error {
	var input CreateBucketInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "Name is required"})
	}

	bucket := &store.R2Bucket{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		Location:  input.Location,
		CreatedAt: time.Now(),
	}

	if bucket.Location == "" {
		bucket.Location = "auto"
	}

	if err := h.store.CreateBucket(c.Request().Context(), bucket); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  bucket,
	})
}

// DeleteBucket deletes an R2 bucket.
func (h *R2) DeleteBucket(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteBucket(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListObjects lists objects in an R2 bucket.
func (h *R2) ListObjects(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	prefix := c.Query("prefix")
	delimiter := c.Query("delimiter")
	limit := 1000

	objects, err := h.store.ListObjects(c.Request().Context(), bucketID, prefix, delimiter, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"objects":      objects,
			"truncated":    len(objects) >= limit,
			"cursor":       "",
			"delimited_prefixes": []string{},
		},
	})
}

// PutObject uploads an object to an R2 bucket.
func (h *R2) PutObject(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	key := c.Param("key")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Failed to read body"})
	}

	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	metadata := map[string]string{
		"content-type": contentType,
	}

	if err := h.store.PutObject(c.Request().Context(), bucketID, key, body, metadata); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"key":  key,
			"size": len(body),
		},
	})
}

// GetObject retrieves an object from an R2 bucket.
func (h *R2) GetObject(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	key := c.Param("key")

	data, obj, err := h.store.GetObject(c.Request().Context(), bucketID, key)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Object not found"})
	}

	c.Writer().Header().Set("Content-Type", obj.ContentType)
	c.Writer().Header().Set("ETag", obj.ETag)
	c.Writer().Header().Set("Last-Modified", obj.LastModified.Format(http.TimeFormat))
	c.Writer().Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))

	c.Writer().Write(data)
	return nil
}

// DeleteObject deletes an object from an R2 bucket.
func (h *R2) DeleteObject(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	key := c.Param("key")

	if err := h.store.DeleteObject(c.Request().Context(), bucketID, key); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}

