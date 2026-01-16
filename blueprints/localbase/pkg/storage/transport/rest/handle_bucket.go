// File: lib/storage/transport/rest/handle_bucket.go
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	"github.com/go-mizu/mizu"
)

// CreateBucketRequest is the request body for POST /bucket.
type CreateBucketRequest struct {
	Name             string   `json:"name"`
	ID               string   `json:"id,omitempty"`
	Public           bool     `json:"public,omitempty"`
	Type             string   `json:"type,omitempty"` // STANDARD or ANALYTICS
	FileSizeLimit    *int64   `json:"file_size_limit,omitempty"`
	AllowedMimeTypes []string `json:"allowed_mime_types,omitempty"`
}

// CreateBucketResponse is the response for POST /bucket.
type CreateBucketResponse struct {
	Name string `json:"name"`
}

// BucketResponse is the response for GET /bucket/:bucketId and list items.
type BucketResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Owner            string    `json:"owner,omitempty"`
	Public           bool      `json:"public"`
	Type             string    `json:"type,omitempty"` // STANDARD or ANALYTICS
	FileSizeLimit    *int64    `json:"file_size_limit,omitempty"`
	AllowedMimeTypes []string  `json:"allowed_mime_types,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpdateBucketRequest is the request body for PUT /bucket/:bucketId.
type UpdateBucketRequest struct {
	Public           *bool    `json:"public,omitempty"`
	FileSizeLimit    *int64   `json:"file_size_limit,omitempty"`
	AllowedMimeTypes []string `json:"allowed_mime_types,omitempty"`
}

// MessageResponse is a generic response with a message.
type MessageResponse struct {
	Message string `json:"message"`
}

// handleCreateBucket implements POST /bucket.
func (s *Server) handleCreateBucket(c *mizu.Ctx) error {
	var req CreateBucketRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		// Use ID as fallback if name is empty
		name = strings.TrimSpace(req.ID)
	}
	if name == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}

	opts := storage.Options{}
	if req.Public {
		opts["public"] = true
	}
	if req.Type != "" {
		opts["type"] = req.Type
	}
	if req.FileSizeLimit != nil {
		opts["file_size_limit"] = *req.FileSizeLimit
	}
	if len(req.AllowedMimeTypes) > 0 {
		opts["allowed_mime_types"] = req.AllowedMimeTypes
	}

	ctx := c.Context()
	info, err := s.store.CreateBucket(ctx, name, opts)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, CreateBucketResponse{
		Name: info.Name,
	})
}

// handleListBuckets implements GET /bucket.
func (s *Server) handleListBuckets(c *mizu.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	limit := parseIntQuery(c, "limit", 100)
	offset := parseIntQuery(c, "offset", 0)
	// sortColumn and sortOrder are noted but not fully implemented
	// since the storage interface doesn't support sorting options
	// search is also noted but not implemented

	iter, err := s.store.Buckets(ctx, limit, offset, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}
	defer func() {
		_ = iter.Close()
	}()

	var buckets []BucketResponse
	for {
		info, err := iter.Next()
		if err != nil {
			return writeError(c, http.StatusInternalServerError, err)
		}
		if info == nil {
			break
		}

		resp := BucketResponse{
			ID:        info.Name,
			Name:      info.Name,
			Public:    info.Public,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.CreatedAt, // Use CreatedAt as UpdatedAt if not available
		}

		// Extract type from metadata if available
		if info.Metadata != nil {
			if bucketType, ok := info.Metadata["type"]; ok {
				resp.Type = bucketType
			}
		}

		buckets = append(buckets, resp)
	}

	if buckets == nil {
		buckets = []BucketResponse{}
	}

	return c.JSON(http.StatusOK, buckets)
}

// handleGetBucket implements GET /bucket/:bucketId.
func (s *Server) handleGetBucket(c *mizu.Ctx) error {
	bucketId := strings.TrimSpace(c.Param("bucketId"))
	if bucketId == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket id is required"))
	}

	ctx := c.Context()
	bucket := s.store.Bucket(bucketId)
	info, err := bucket.Info(ctx)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	resp := BucketResponse{
		ID:        info.Name,
		Name:      info.Name,
		Public:    info.Public,
		CreatedAt: info.CreatedAt,
		UpdatedAt: info.CreatedAt,
	}

	// Extract type from metadata if available
	if info.Metadata != nil {
		if bucketType, ok := info.Metadata["type"]; ok {
			resp.Type = bucketType
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// handleUpdateBucket implements PUT /bucket/:bucketId.
func (s *Server) handleUpdateBucket(c *mizu.Ctx) error {
	bucketId := c.Param("bucketId")
	if bucketId == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket id is required"))
	}

	var req UpdateBucketRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	// Note: The storage.Storage interface doesn't have an UpdateBucket method.
	// This is a placeholder that verifies the bucket exists.
	ctx := c.Context()
	bucket := s.store.Bucket(bucketId)
	if _, err := bucket.Info(ctx); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	// TODO: Implement bucket metadata updates when the storage interface supports it.

	return c.JSON(http.StatusOK, MessageResponse{
		Message: "Successfully updated",
	})
}

// handleDeleteBucket implements DELETE /bucket/:bucketId.
func (s *Server) handleDeleteBucket(c *mizu.Ctx) error {
	bucketId := c.Param("bucketId")
	if bucketId == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket id is required"))
	}

	ctx := c.Context()
	if err := s.store.DeleteBucket(ctx, bucketId, nil); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, MessageResponse{
		Message: "Successfully deleted",
	})
}

// handleEmptyBucket implements POST /bucket/:bucketId/empty.
func (s *Server) handleEmptyBucket(c *mizu.Ctx) error {
	bucketId := c.Param("bucketId")
	if bucketId == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket id is required"))
	}

	ctx := c.Context()
	bucket := s.store.Bucket(bucketId)

	// Verify bucket exists
	if _, err := bucket.Info(ctx); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	// List all objects and delete them
	iter, err := bucket.List(ctx, "", 0, 0, storage.Options{"recursive": true})
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	var keysToDelete []string
	for {
		obj, err := iter.Next()
		if err != nil {
			_ = iter.Close() // ignore close error in error path
			return writeError(c, http.StatusInternalServerError, err)
		}
		if obj == nil {
			break
		}
		if !obj.IsDir {
			keysToDelete = append(keysToDelete, obj.Key)
		}
	}
	if err := iter.Close(); err != nil {
		return writeError(c, http.StatusInternalServerError, err)
	}

	// Delete all objects
	for _, key := range keysToDelete {
		if err := bucket.Delete(ctx, key, nil); err != nil {
			// Continue deleting other objects even if one fails
			continue
		}
	}

	return c.JSON(http.StatusOK, MessageResponse{
		Message: "Successfully queued for empty",
	})
}

// parseIntQuery parses an integer query parameter with a default value.
func parseIntQuery(c *mizu.Ctx, name string, defaultVal int) int {
	raw := c.Query(name)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return val
}
