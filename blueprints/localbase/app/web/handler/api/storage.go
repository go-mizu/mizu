package api

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/oklog/ulid/v2"
)

// StorageHandler handles storage endpoints.
type StorageHandler struct {
	store *postgres.Store
}

// NewStorageHandler creates a new storage handler.
func NewStorageHandler(store *postgres.Store) *StorageHandler {
	return &StorageHandler{store: store}
}

// ListBuckets lists all buckets.
func (h *StorageHandler) ListBuckets(c *mizu.Ctx) error {
	buckets, err := h.store.Storage().ListBuckets(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list buckets"})
	}

	return c.JSON(200, buckets)
}

// CreateBucket creates a new bucket.
func (h *StorageHandler) CreateBucket(c *mizu.Ctx) error {
	var req struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		Public           bool     `json:"public"`
		FileSizeLimit    *int64   `json:"file_size_limit"`
		AllowedMimeTypes []string `json:"allowed_mime_types"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}

	id := req.ID
	if id == "" {
		id = strings.ToLower(req.Name)
	}

	bucket := &store.Bucket{
		ID:               id,
		Name:             req.Name,
		Public:           req.Public,
		FileSizeLimit:    req.FileSizeLimit,
		AllowedMimeTypes: req.AllowedMimeTypes,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.store.Storage().CreateBucket(c.Context(), bucket); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return c.JSON(400, map[string]string{"error": "bucket already exists"})
		}
		return c.JSON(500, map[string]string{"error": "failed to create bucket"})
	}

	return c.JSON(201, bucket)
}

// GetBucket gets a bucket by ID.
func (h *StorageHandler) GetBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	bucket, err := h.store.Storage().GetBucket(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "bucket not found"})
	}

	return c.JSON(200, bucket)
}

// UpdateBucket updates a bucket.
func (h *StorageHandler) UpdateBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	bucket, err := h.store.Storage().GetBucket(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "bucket not found"})
	}

	var req struct {
		Public           *bool    `json:"public"`
		FileSizeLimit    *int64   `json:"file_size_limit"`
		AllowedMimeTypes []string `json:"allowed_mime_types"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Public != nil {
		bucket.Public = *req.Public
	}
	if req.FileSizeLimit != nil {
		bucket.FileSizeLimit = req.FileSizeLimit
	}
	if req.AllowedMimeTypes != nil {
		bucket.AllowedMimeTypes = req.AllowedMimeTypes
	}
	bucket.UpdatedAt = time.Now()

	if err := h.store.Storage().UpdateBucket(c.Context(), bucket); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update bucket"})
	}

	return c.JSON(200, bucket)
}

// DeleteBucket deletes a bucket.
func (h *StorageHandler) DeleteBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Storage().DeleteBucket(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete bucket"})
	}

	return c.NoContent()
}

// UploadObject uploads a file to a bucket.
func (h *StorageHandler) UploadObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")

	// Get bucket to verify it exists
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "bucket not found"})
	}

	// Parse multipart form
	form, cleanup, err := c.MultipartForm(32 << 20) // 32 MB max
	if err != nil {
		return c.JSON(400, map[string]string{"error": "failed to parse multipart form"})
	}
	defer cleanup()

	files := form.File["file"]
	if len(files) == 0 {
		return c.JSON(400, map[string]string{"error": "file required"})
	}

	header := files[0]

	// Check file size limit
	if bucket.FileSizeLimit != nil && header.Size > *bucket.FileSizeLimit {
		return c.JSON(413, map[string]string{"error": "file too large"})
	}

	// Get filename from form or header
	filename := ""
	if names, ok := form.Value["name"]; ok && len(names) > 0 {
		filename = names[0]
	}
	if filename == "" {
		filename = header.Filename
	}

	// Check MIME type
	contentType := header.Header.Get("Content-Type")
	if len(bucket.AllowedMimeTypes) > 0 {
		allowed := false
		for _, mt := range bucket.AllowedMimeTypes {
			if strings.HasPrefix(contentType, mt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return c.JSON(415, map[string]string{"error": "file type not allowed"})
		}
	}

	// Open file
	file, err := header.Open()
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to open file"})
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to read file"})
	}

	// Create object metadata
	obj := &store.Object{
		ID:          ulid.Make().String(),
		BucketID:    bucketID,
		Name:        filename,
		ContentType: contentType,
		Size:        int64(len(content)),
		Version:     ulid.Make().String(),
		Metadata:    make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.store.Storage().CreateObject(c.Context(), obj); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			// Update existing object
			if err := h.store.Storage().UpdateObject(c.Context(), obj); err != nil {
				return c.JSON(500, map[string]string{"error": "failed to update object"})
			}
		} else {
			return c.JSON(500, map[string]string{"error": "failed to create object"})
		}
	}

	// In a real implementation, we'd store the file content to disk or S3
	// For now, we just return the object metadata

	return c.JSON(201, map[string]any{
		"id":       obj.ID,
		"key":      obj.Name,
		"bucket":   bucketID,
		"size":     obj.Size,
		"mimeType": obj.ContentType,
	})
}

// DownloadObject downloads a file from a bucket.
func (h *StorageHandler) DownloadObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	obj, err := h.store.Storage().GetObject(c.Context(), bucketID, path)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "object not found"})
	}

	// In a real implementation, we'd stream the file content
	// For now, return object metadata as placeholder
	c.Header().Set("Content-Type", obj.ContentType)
	c.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(obj.Name))

	return c.JSON(200, obj)
}

// DeleteObject deletes an object.
func (h *StorageHandler) DeleteObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	if err := h.store.Storage().DeleteObject(c.Context(), bucketID, path); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete object"})
	}

	return c.NoContent()
}

// ListObjects lists objects in a bucket.
func (h *StorageHandler) ListObjects(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")

	var req struct {
		Prefix string `json:"prefix"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	objects, err := h.store.Storage().ListObjects(c.Context(), bucketID, req.Prefix, req.Limit, req.Offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list objects"})
	}

	return c.JSON(200, objects)
}

// MoveObject moves/renames an object.
func (h *StorageHandler) MoveObject(c *mizu.Ctx) error {
	var req struct {
		BucketID   string `json:"bucketId"`
		SourceKey  string `json:"sourceKey"`
		DestKey    string `json:"destinationKey"`
		DestBucket string `json:"destinationBucket,omitempty"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	destBucket := req.DestBucket
	if destBucket == "" {
		destBucket = req.BucketID
	}

	if destBucket != req.BucketID {
		// Cross-bucket move = copy + delete
		if err := h.store.Storage().CopyObject(c.Context(), req.BucketID, req.SourceKey, destBucket, req.DestKey); err != nil {
			return c.JSON(500, map[string]string{"error": "failed to move object"})
		}
		if err := h.store.Storage().DeleteObject(c.Context(), req.BucketID, req.SourceKey); err != nil {
			return c.JSON(500, map[string]string{"error": "failed to delete source object"})
		}
	} else {
		if err := h.store.Storage().MoveObject(c.Context(), req.BucketID, req.SourceKey, req.DestKey); err != nil {
			return c.JSON(500, map[string]string{"error": "failed to move object"})
		}
	}

	return c.JSON(200, map[string]string{"message": "moved"})
}

// CopyObject copies an object.
func (h *StorageHandler) CopyObject(c *mizu.Ctx) error {
	var req struct {
		BucketID   string `json:"bucketId"`
		SourceKey  string `json:"sourceKey"`
		DestKey    string `json:"destinationKey"`
		DestBucket string `json:"destinationBucket,omitempty"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	destBucket := req.DestBucket
	if destBucket == "" {
		destBucket = req.BucketID
	}

	if err := h.store.Storage().CopyObject(c.Context(), req.BucketID, req.SourceKey, destBucket, req.DestKey); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to copy object"})
	}

	return c.JSON(200, map[string]string{"message": "copied"})
}

// CreateSignedURL creates a signed URL for an object.
func (h *StorageHandler) CreateSignedURL(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	var req struct {
		ExpiresIn int `json:"expiresIn"` // seconds
	}
	if err := c.BindJSON(&req, 0); err != nil {
		req.ExpiresIn = 3600 // default 1 hour
	}

	// Verify object exists
	if _, err := h.store.Storage().GetObject(c.Context(), bucketID, path); err != nil {
		return c.JSON(404, map[string]string{"error": "object not found"})
	}

	// In production, generate actual signed URL
	// For now, return a placeholder
	expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)

	return c.JSON(200, map[string]any{
		"signedURL": "/storage/v1/object/" + bucketID + "/" + path + "?token=xxx",
		"expiresAt": expiresAt.Unix(),
	})
}
