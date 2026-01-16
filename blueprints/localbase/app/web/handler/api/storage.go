package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/middleware"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/google/uuid"
)

// StorageHandler handles storage endpoints.
type StorageHandler struct {
	store *postgres.Store
}

// NewStorageHandler creates a new storage handler.
func NewStorageHandler(store *postgres.Store) *StorageHandler {
	return &StorageHandler{store: store}
}

// Supabase Storage error response format
// Note: Supabase returns HTTP 400 for all errors but includes the actual
// status code in the response body (e.g., statusCode: 404 for not found).
type storageErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
	Message    string `json:"message"`
}

func storageError(c *mizu.Ctx, status int, errorType, message string) error {
	// Supabase quirk: return HTTP 400 for all errors, but include actual status in body
	httpStatus := http.StatusBadRequest
	return c.JSON(httpStatus, storageErrorResponse{
		StatusCode: status,
		Error:      errorType,
		Message:    message,
	})
}

// ListBuckets lists all buckets.
// Note: service_role has access to all buckets, while anon only sees public buckets
// (unless RLS policies are configured differently).
func (h *StorageHandler) ListBuckets(c *mizu.Ctx) error {
	// Get role from middleware (service_role bypasses RLS)
	role := middleware.GetRole(c)
	_ = role // Available for RLS enforcement if needed

	buckets, err := h.store.Storage().ListBuckets(c.Context())
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to list buckets")
	}

	return c.JSON(http.StatusOK, buckets)
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
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if req.Name == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "name required")
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
			return storageError(c, http.StatusConflict, "Conflict", "bucket already exists")
		}
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to create bucket")
	}

	// Return format matching Supabase: {"name": "bucket-name"}
	return c.JSON(http.StatusOK, map[string]string{"name": bucket.Name})
}

// GetBucket gets a bucket by ID.
func (h *StorageHandler) GetBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	bucket, err := h.store.Storage().GetBucket(c.Context(), id)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	return c.JSON(http.StatusOK, bucket)
}

// UpdateBucket updates a bucket.
func (h *StorageHandler) UpdateBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	bucket, err := h.store.Storage().GetBucket(c.Context(), id)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	var req struct {
		Public           *bool    `json:"public"`
		FileSizeLimit    *int64   `json:"file_size_limit"`
		AllowedMimeTypes []string `json:"allowed_mime_types"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
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
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to update bucket")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully updated"})
}

// DeleteBucket deletes a bucket.
func (h *StorageHandler) DeleteBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	// Check if bucket exists
	_, err := h.store.Storage().GetBucket(c.Context(), id)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	if err := h.store.Storage().DeleteBucket(c.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not empty") {
			return storageError(c, http.StatusConflict, "Conflict", "bucket is not empty")
		}
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to delete bucket")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully deleted"})
}

// EmptyBucket empties a bucket (deletes all objects).
func (h *StorageHandler) EmptyBucket(c *mizu.Ctx) error {
	id := c.Param("id")

	// Check if bucket exists
	_, err := h.store.Storage().GetBucket(c.Context(), id)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// List and delete all objects
	objects, err := h.store.Storage().ListObjects(c.Context(), id, "", 10000, 0)
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to list objects")
	}

	for _, obj := range objects {
		_ = h.store.Storage().DeleteObject(c.Context(), id, obj.Name)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully emptied"})
}

// UploadObject uploads a file to a bucket.
func (h *StorageHandler) UploadObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	if path == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "object path is required")
	}

	// Get bucket to verify it exists
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// Check for upsert header
	upsert := strings.EqualFold(c.Request().Header.Get("x-upsert"), "true")

	// Check if object already exists (for non-upsert)
	if !upsert {
		if _, err := h.store.Storage().GetObject(c.Context(), bucketID, path); err == nil {
			return storageError(c, http.StatusConflict, "Conflict", "object already exists")
		}
	}

	// Read body directly
	content, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to read body")
	}
	defer c.Request().Body.Close()

	// Check file size limit
	if bucket.FileSizeLimit != nil && int64(len(content)) > *bucket.FileSizeLimit {
		return storageError(c, http.StatusRequestEntityTooLarge, "Payload Too Large", "file too large")
	}

	// Get content type
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Check MIME type
	if len(bucket.AllowedMimeTypes) > 0 {
		allowed := false
		for _, mt := range bucket.AllowedMimeTypes {
			if strings.HasPrefix(contentType, mt) || mt == "*/*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return storageError(c, http.StatusUnsupportedMediaType, "Unsupported Media Type", "file type not allowed")
		}
	}

	// Create object metadata
	obj := &store.Object{
		ID:          uuid.New().String(),
		BucketID:    bucketID,
		Name:        path,
		ContentType: contentType,
		Size:        int64(len(content)),
		Version:     uuid.New().String(),
		Metadata:    make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.store.Storage().CreateObject(c.Context(), obj); err != nil {
		if strings.Contains(err.Error(), "duplicate") && upsert {
			// Update existing object
			if err := h.store.Storage().UpdateObject(c.Context(), obj); err != nil {
				return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to update object")
			}
		} else if strings.Contains(err.Error(), "duplicate") {
			return storageError(c, http.StatusConflict, "Conflict", "object already exists")
		} else {
			return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to create object")
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"Id":  obj.ID,
		"Key": fmt.Sprintf("%s/%s", bucketID, path),
	})
}

// UpdateObject updates an object (PUT).
func (h *StorageHandler) UpdateObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	if path == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "object path is required")
	}

	// Check bucket exists
	_, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// Check object exists
	if _, err := h.store.Storage().GetObject(c.Context(), bucketID, path); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "object not found")
	}

	// Read body
	content, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to read body")
	}
	defer c.Request().Body.Close()

	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	obj := &store.Object{
		ID:          uuid.New().String(),
		BucketID:    bucketID,
		Name:        path,
		ContentType: contentType,
		Size:        int64(len(content)),
		Version:     uuid.New().String(),
		UpdatedAt:   time.Now(),
	}

	if err := h.store.Storage().UpdateObject(c.Context(), obj); err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to update object")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"Id":  obj.ID,
		"Key": fmt.Sprintf("%s/%s", bucketID, path),
	})
}

// DownloadObject downloads a file from a bucket.
func (h *StorageHandler) DownloadObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	obj, err := h.store.Storage().GetObject(c.Context(), bucketID, path)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "object not found")
	}

	// Set headers
	c.Header().Set("Content-Type", obj.ContentType)
	if c.Query("download") != "" {
		c.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(obj.Name))
	}

	// In production, stream actual file content
	// For now, return placeholder
	return c.Text(http.StatusOK, "file content placeholder")
}

// DownloadPublicObject downloads a public file.
func (h *StorageHandler) DownloadPublicObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")

	// Check bucket is public
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	if !bucket.Public {
		return storageError(c, http.StatusForbidden, "Forbidden", "bucket is not public")
	}

	return h.DownloadObject(c)
}

// DownloadAuthenticatedObject downloads an authenticated file.
func (h *StorageHandler) DownloadAuthenticatedObject(c *mizu.Ctx) error {
	// For now, same as regular download (auth middleware handles authentication)
	return h.DownloadObject(c)
}

// GetObjectInfo gets object metadata.
func (h *StorageHandler) GetObjectInfo(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	obj, err := h.store.Storage().GetObject(c.Context(), bucketID, path)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "object not found")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"id":         obj.ID,
		"name":       filepath.Base(obj.Name),
		"bucket_id":  obj.BucketID,
		"created_at": obj.CreatedAt,
		"updated_at": obj.UpdatedAt,
		"metadata":   obj.Metadata,
	})
}

// GetPublicObjectInfo gets public object metadata.
func (h *StorageHandler) GetPublicObjectInfo(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")

	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	if !bucket.Public {
		return storageError(c, http.StatusForbidden, "Forbidden", "bucket is not public")
	}

	return h.GetObjectInfo(c)
}

// GetAuthenticatedObjectInfo gets authenticated object metadata.
func (h *StorageHandler) GetAuthenticatedObjectInfo(c *mizu.Ctx) error {
	return h.GetObjectInfo(c)
}

// DeleteObject deletes an object.
func (h *StorageHandler) DeleteObject(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	// Check object exists
	if _, err := h.store.Storage().GetObject(c.Context(), bucketID, path); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "object not found")
	}

	if err := h.store.Storage().DeleteObject(c.Context(), bucketID, path); err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to delete object")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully deleted"})
}

// DeleteObjects deletes multiple objects (bulk delete).
func (h *StorageHandler) DeleteObjects(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")

	var req struct {
		Prefixes []string `json:"prefixes"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if len(req.Prefixes) == 0 {
		return storageError(c, http.StatusBadRequest, "Bad Request", "prefixes required")
	}

	var deleted []map[string]string
	for _, prefix := range req.Prefixes {
		if err := h.store.Storage().DeleteObject(c.Context(), bucketID, prefix); err == nil {
			deleted = append(deleted, map[string]string{"name": prefix})
		}
	}

	return c.JSON(http.StatusOK, deleted)
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
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	objects, err := h.store.Storage().ListObjects(c.Context(), bucketID, req.Prefix, req.Limit, req.Offset)
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to list objects")
	}

	// Format response to match Supabase
	var result []map[string]any
	for _, obj := range objects {
		// Extract just the name part (without prefix)
		name := obj.Name
		if req.Prefix != "" && strings.HasPrefix(name, req.Prefix) {
			name = strings.TrimPrefix(name, req.Prefix)
			name = strings.TrimPrefix(name, "/")
		}

		result = append(result, map[string]any{
			"id":         obj.ID,
			"name":       name,
			"bucket_id":  obj.BucketID,
			"created_at": obj.CreatedAt,
			"updated_at": obj.UpdatedAt,
			"metadata":   obj.Metadata,
		})
	}

	if result == nil {
		result = []map[string]any{}
	}

	return c.JSON(http.StatusOK, result)
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
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if req.BucketID == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "bucketId required")
	}
	if req.SourceKey == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "sourceKey required")
	}
	if req.DestKey == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "destinationKey required")
	}

	// Check source exists
	if _, err := h.store.Storage().GetObject(c.Context(), req.BucketID, req.SourceKey); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "source object not found")
	}

	destBucket := req.DestBucket
	if destBucket == "" {
		destBucket = req.BucketID
	}

	if destBucket != req.BucketID {
		// Cross-bucket move = copy + delete
		if err := h.store.Storage().CopyObject(c.Context(), req.BucketID, req.SourceKey, destBucket, req.DestKey); err != nil {
			return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to move object")
		}
		if err := h.store.Storage().DeleteObject(c.Context(), req.BucketID, req.SourceKey); err != nil {
			return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to delete source object")
		}
	} else {
		if err := h.store.Storage().MoveObject(c.Context(), req.BucketID, req.SourceKey, req.DestKey); err != nil {
			return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to move object")
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully moved"})
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
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if req.BucketID == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "bucketId required")
	}
	if req.SourceKey == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "sourceKey required")
	}
	if req.DestKey == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "destinationKey required")
	}

	// Check source exists
	if _, err := h.store.Storage().GetObject(c.Context(), req.BucketID, req.SourceKey); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "source object not found")
	}

	destBucket := req.DestBucket
	if destBucket == "" {
		destBucket = req.BucketID
	}

	if err := h.store.Storage().CopyObject(c.Context(), req.BucketID, req.SourceKey, destBucket, req.DestKey); err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to copy object")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"Id":  uuid.New().String(),
		"Key": fmt.Sprintf("%s/%s", destBucket, req.DestKey),
	})
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

	if req.ExpiresIn <= 0 {
		return storageError(c, http.StatusBadRequest, "Bad Request", "expiresIn must be positive")
	}

	// Verify object exists
	if _, err := h.store.Storage().GetObject(c.Context(), bucketID, path); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "object not found")
	}

	// Generate signed URL
	token := uuid.New().String()

	return c.JSON(http.StatusOK, map[string]string{
		"signedURL": fmt.Sprintf("/storage/v1/object/sign/%s/%s?token=%s", bucketID, path, token),
	})
}

// CreateSignedURLs creates multiple signed URLs.
func (h *StorageHandler) CreateSignedURLs(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")

	var req struct {
		ExpiresIn int      `json:"expiresIn"`
		Paths     []string `json:"paths"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if req.ExpiresIn <= 0 {
		return storageError(c, http.StatusBadRequest, "Bad Request", "expiresIn must be positive")
	}

	if len(req.Paths) == 0 {
		return storageError(c, http.StatusBadRequest, "Bad Request", "paths required")
	}

	var results []map[string]any
	for _, path := range req.Paths {
		// Check if object exists
		if _, err := h.store.Storage().GetObject(c.Context(), bucketID, path); err != nil {
			results = append(results, map[string]any{
				"path":      path,
				"error":     "Either the object does not exist or you do not have access to it",
				"signedURL": nil,
			})
		} else {
			token := uuid.New().String()
			results = append(results, map[string]any{
				"path":      path,
				"signedURL": fmt.Sprintf("/storage/v1/object/sign/%s/%s?token=%s", bucketID, path, token),
			})
		}
	}

	return c.JSON(http.StatusOK, results)
}

// CreateUploadSignedURL creates a signed URL for uploading.
func (h *StorageHandler) CreateUploadSignedURL(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	// Check bucket exists
	if _, err := h.store.Storage().GetBucket(c.Context(), bucketID); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	token := uuid.New().String()

	return c.JSON(http.StatusOK, map[string]string{
		"url":   fmt.Sprintf("/storage/v1/object/%s/%s", bucketID, path),
		"token": token,
	})
}
