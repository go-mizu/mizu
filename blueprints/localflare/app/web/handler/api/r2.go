package api

import (
	"io"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/r2"
)

// R2 handles R2 bucket requests.
type R2 struct {
	svc r2.API
}

// NewR2 creates a new R2 handler.
func NewR2(svc r2.API) *R2 {
	return &R2{svc: svc}
}

// ListBuckets lists all R2 buckets.
func (h *R2) ListBuckets(c *mizu.Ctx) error {
	buckets, err := h.svc.ListBuckets(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"buckets": buckets,
		},
	})
}

// CreateBucket creates a new R2 bucket.
func (h *R2) CreateBucket(c *mizu.Ctx) error {
	var input r2.CreateBucketIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	bucket, err := h.svc.CreateBucket(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  bucket,
	})
}

// GetBucket retrieves a bucket.
func (h *R2) GetBucket(c *mizu.Ctx) error {
	id := c.Param("id")
	bucket, err := h.svc.GetBucket(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Bucket not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  bucket,
	})
}

// DeleteBucket deletes a bucket.
func (h *R2) DeleteBucket(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.DeleteBucket(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListObjects lists objects in a bucket.
func (h *R2) ListObjects(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	opts := r2.ListObjectsOpts{
		Prefix:    c.Query("prefix"),
		Delimiter: c.Query("delimiter"),
		Limit:     1000,
	}

	objects, err := h.svc.ListObjects(c.Request().Context(), bucketID, opts)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"objects":             objects,
			"truncated":           len(objects) >= opts.Limit,
			"cursor":              "",
			"delimited_prefixes":  []string{},
		},
	})
}

// PutObject stores an object.
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

	input := &r2.PutObjectIn{
		Key:         key,
		Data:        body,
		ContentType: contentType,
	}

	if err := h.svc.PutObject(c.Request().Context(), bucketID, input); err != nil {
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

// GetObject retrieves an object.
func (h *R2) GetObject(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	key := c.Param("key")

	data, obj, err := h.svc.GetObject(c.Request().Context(), bucketID, key)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Object not found"})
	}

	c.Writer().Header().Set("Content-Type", obj.ContentType)
	c.Writer().Header().Set("ETag", obj.ETag)
	c.Writer().Header().Set("Last-Modified", obj.LastModified.Format(http.TimeFormat))
	c.Writer().Write(data)
	return nil
}

// DeleteObject deletes an object.
func (h *R2) DeleteObject(c *mizu.Ctx) error {
	bucketID := c.Param("id")
	key := c.Param("key")

	if err := h.svc.DeleteObject(c.Request().Context(), bucketID, key); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  nil,
	})
}
