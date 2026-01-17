package api

import (
	"encoding/json"
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

// StorageAccessLevel defines the type of access being requested
type StorageAccessLevel int

const (
	StorageAccessRead StorageAccessLevel = iota
	StorageAccessWrite
	StorageAccessDelete
)

// checkStorageAccess verifies if the current user has access to perform the operation.
// Service role bypasses all checks.
// For anon users, only public buckets allow read access.
// For authenticated users, access depends on bucket visibility and path ownership.
func (h *StorageHandler) checkStorageAccess(c *mizu.Ctx, bucket *store.Bucket, path string, level StorageAccessLevel) error {
	role := middleware.GetRole(c)
	userID := middleware.GetUserID(c)

	// Service role bypasses all RLS
	if role == "service_role" {
		return nil
	}

	// For public buckets, anyone can read
	if bucket.Public && level == StorageAccessRead {
		return nil
	}

	// For private buckets or write operations, more checks are needed
	if role == "anon" {
		// Anon users cannot access private buckets
		if !bucket.Public {
			return fmt.Errorf("access denied: private bucket")
		}
		// Anon users cannot write
		if level != StorageAccessRead {
			return fmt.Errorf("access denied: authentication required for write access")
		}
		return nil
	}

	// Authenticated users (role == "authenticated")
	// Check if the path is in a user-specific folder
	// Supabase storage folder patterns:
	// - path starts with user ID (e.g., "abc123/file.txt")
	// - path starts with "user/" + user ID (e.g., "user/abc123/file.txt")
	// - path contains user ID anywhere (legacy support)
	if userID != "" {
		pathParts := strings.Split(path, "/")
		// Check if first path segment is user ID
		if len(pathParts) > 0 && pathParts[0] == userID {
			return nil
		}
		// Check if path follows "user/{uid}/*" pattern
		if len(pathParts) > 1 && pathParts[0] == "user" && pathParts[1] == userID {
			return nil
		}
		// Check if path contains user ID (broader match)
		if strings.Contains(path, userID) {
			return nil
		}
	}

	// For public buckets, authenticated users can read anything
	if bucket.Public && level == StorageAccessRead {
		return nil
	}

	// For private buckets without user-specific path, deny by default
	// (Real Supabase would check storage.policies here)
	if !bucket.Public {
		return fmt.Errorf("access denied: no policy allows this operation")
	}

	// Public bucket write requires more specific policies
	// For now, allow authenticated users to write to public buckets
	// (Real Supabase would check policies)
	return nil
}

// sanitizeStoragePath removes path traversal sequences and normalizes the path.
// This prevents access to files outside the bucket directory.
func sanitizeStoragePath(path string) string {
	// Remove any path traversal sequences
	path = strings.ReplaceAll(path, "..", "")
	// Remove null bytes
	path = strings.ReplaceAll(path, "\x00", "")
	// Clean the path to normalize it
	path = filepath.Clean(path)
	// Remove leading slashes
	path = strings.TrimPrefix(path, "/")
	// Ensure no leading dots remain
	for strings.HasPrefix(path, ".") {
		path = strings.TrimPrefix(path, ".")
		path = strings.TrimPrefix(path, "/")
	}
	return path
}

// sanitizeFilename removes dangerous characters from filenames to prevent header injection.
func sanitizeFilename(filename string) string {
	// Remove characters that could cause header injection or path traversal
	replacer := strings.NewReplacer(
		"\n", "",
		"\r", "",
		"\"", "",
		"\\", "",
		"/", "",
		"\x00", "",
	)
	return replacer.Replace(filename)
}

// checkObjectOwnership verifies if the current user owns the object.
// Returns true if the user has ownership access.
func (h *StorageHandler) checkObjectOwnership(c *mizu.Ctx, obj *store.Object) bool {
	role := middleware.GetRole(c)

	// Service role always has access
	if role == "service_role" {
		return true
	}

	// If object has an owner, check it matches the current user
	userID := middleware.GetUserID(c)
	if obj.Owner != "" && userID != "" {
		return obj.Owner == userID
	}

	// No owner set or no user ID, rely on other checks
	return false
}

// ListBuckets lists all buckets.
// Service role sees all buckets, while anon only sees public buckets.
func (h *StorageHandler) ListBuckets(c *mizu.Ctx) error {
	role := middleware.GetRole(c)

	buckets, err := h.store.Storage().ListBuckets(c.Context())
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to list buckets")
	}

	// Service role sees all buckets
	if role == "service_role" {
		return c.JSON(http.StatusOK, buckets)
	}

	// Non-service roles only see public buckets
	var publicBuckets []*store.Bucket
	for _, b := range buckets {
		if b.Public {
			publicBuckets = append(publicBuckets, b)
		}
	}

	return c.JSON(http.StatusOK, publicBuckets)
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

	// Sanitize path to prevent path traversal attacks
	path = sanitizeStoragePath(path)

	// Get bucket to verify it exists
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// Check storage access (RLS enforcement)
	if err := h.checkStorageAccess(c, bucket, path, StorageAccessWrite); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
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

	// Get owner from JWT claims (for RLS-style ownership tracking)
	owner := middleware.GetUserID(c)

	// Create object metadata with ownership tracking (Supabase-compatible)
	obj := &store.Object{
		ID:          uuid.New().String(),
		BucketID:    bucketID,
		Name:        path,
		Owner:       owner, // Track who uploaded this object
		ContentType: contentType,
		Size:        int64(len(content)),
		Version:     uuid.New().String(),
		Metadata:    make(map[string]any),
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

	// Sanitize path to prevent path traversal attacks
	path = sanitizeStoragePath(path)

	// Check bucket exists and get bucket for RLS check
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// Check storage access (RLS enforcement) - SEC-008 fix
	if err := h.checkStorageAccess(c, bucket, path, StorageAccessWrite); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
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

	// Get bucket for RLS check
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// Check storage access (RLS enforcement)
	if err := h.checkStorageAccess(c, bucket, path, StorageAccessRead); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
	}

	obj, err := h.store.Storage().GetObject(c.Context(), bucketID, path)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "object not found")
	}

	// Set headers
	c.Header().Set("Content-Type", obj.ContentType)
	if c.Query("download") != "" {
		// Sanitize filename to prevent header injection attacks
		filename := sanitizeFilename(filepath.Base(obj.Name))
		c.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
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

	// Return comprehensive metadata (Supabase-compatible)
	return c.JSON(http.StatusOK, map[string]any{
		"id":               obj.ID,
		"name":             filepath.Base(obj.Name),
		"bucket_id":        obj.BucketID,
		"owner":            obj.Owner,
		"content_type":     obj.ContentType,
		"size":             obj.Size,
		"version":          obj.Version,
		"created_at":       obj.CreatedAt,
		"updated_at":       obj.UpdatedAt,
		"last_accessed_at": obj.LastAccessedAt,
		"metadata":         obj.Metadata,
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

	// Get bucket for RLS check
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	// Check storage access (RLS enforcement)
	if err := h.checkStorageAccess(c, bucket, path, StorageAccessDelete); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
	}

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

	// First check if bucket exists (by ID or name for Supabase compatibility)
	if _, err := h.store.Storage().GetBucket(c.Context(), bucketID); err != nil {
		// Try by name as fallback
		bucket, err := h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
	}

	// Read and parse request body manually to avoid DisallowUnknownFields issues
	var req struct {
		Prefix string `json:"prefix"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
		Search string `json:"search"`
	}

	// Use standard JSON decoder without DisallowUnknownFields
	if c.Request().Body != nil {
		dec := json.NewDecoder(c.Request().Body)
		if err := dec.Decode(&req); err != nil && err != io.EOF {
			return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body: "+err.Error())
		}
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	// If search is provided, we need to get all objects and filter
	// Otherwise use the standard list with prefix
	var objects []*store.Object
	var err error

	if req.Search != "" {
		// Get all objects from root and filter by search term
		allObjects, listErr := h.store.Storage().ListObjects(c.Context(), bucketID, "", 1000, 0)
		if listErr != nil {
			err = listErr
		} else {
			// Filter by search term (case-insensitive)
			searchLower := strings.ToLower(req.Search)
			for _, obj := range allObjects {
				if strings.Contains(strings.ToLower(obj.Name), searchLower) {
					objects = append(objects, obj)
				}
			}
			// Apply offset and limit
			if req.Offset < len(objects) {
				objects = objects[req.Offset:]
			} else {
				objects = []*store.Object{}
			}
			if req.Limit < len(objects) {
				objects = objects[:req.Limit]
			}
		}
	} else {
		objects, err = h.store.Storage().ListObjects(c.Context(), bucketID, req.Prefix, req.Limit, req.Offset)
	}
	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("ListObjects error for bucket %s: %v\n", bucketID, err)
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to list objects: "+err.Error())
	}

	// Format response to match Supabase (include content_type and size for UI)
	var result []map[string]any
	for _, obj := range objects {
		// Extract just the name part (without prefix)
		name := obj.Name
		if req.Prefix != "" && strings.HasPrefix(name, req.Prefix) {
			name = strings.TrimPrefix(name, req.Prefix)
			name = strings.TrimPrefix(name, "/")
		}

		result = append(result, map[string]any{
			"id":           obj.ID,
			"name":         name,
			"bucket_id":    obj.BucketID,
			"owner":        obj.Owner,
			"content_type": obj.ContentType,
			"size":         obj.Size,
			"version":      obj.Version,
			"created_at":   obj.CreatedAt,
			"updated_at":   obj.UpdatedAt,
			"metadata":     obj.Metadata,
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
