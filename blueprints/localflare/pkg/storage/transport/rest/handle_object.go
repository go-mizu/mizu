// File: lib/storage/transport/rest/handle_object.go
package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage"
	"github.com/go-mizu/mizu"
)

// UploadResponse is the response for POST/PUT /object/:bucketName/*path.
type UploadResponse struct {
	ID  string `json:"Id"`
	Key string `json:"Key"`
}

// ListObjectsRequest is the request body for POST /object/list/:bucketName.
type ListObjectsRequest struct {
	Prefix string      `json:"prefix"`
	Limit  int         `json:"limit,omitempty"`
	Offset int         `json:"offset,omitempty"`
	SortBy *SortConfig `json:"sortBy,omitempty"`
	Search string      `json:"search,omitempty"`
}

// SortConfig configures sorting for list operations.
type SortConfig struct {
	Column string `json:"column,omitempty"`
	Order  string `json:"order,omitempty"`
}

// ObjectInfo is the response item for list operations.
type ObjectInfo struct {
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name"`
	BucketID       string            `json:"bucket_id,omitempty"`
	Owner          string            `json:"owner,omitempty"`
	CreatedAt      time.Time         `json:"created_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
	LastAccessedAt *time.Time        `json:"last_accessed_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DeleteObjectsRequest is the request body for DELETE /object/:bucketName (bulk delete).
type DeleteObjectsRequest struct {
	Prefixes []string `json:"prefixes"`
}

// MoveObjectRequest is the request body for POST /object/move.
type MoveObjectRequest struct {
	BucketID          string `json:"bucketId"`
	SourceKey         string `json:"sourceKey"`
	DestinationBucket string `json:"destinationBucket,omitempty"`
	DestinationKey    string `json:"destinationKey"`
}

// CopyObjectRequest is the request body for POST /object/copy.
type CopyObjectRequest struct {
	BucketID          string            `json:"bucketId"`
	SourceKey         string            `json:"sourceKey"`
	DestinationBucket string            `json:"destinationBucket,omitempty"`
	DestinationKey    string            `json:"destinationKey"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	CopyMetadata      bool              `json:"copyMetadata,omitempty"`
}

// CopyObjectResponse is the response for POST /object/copy.
type CopyObjectResponse struct {
	ID       string            `json:"Id"`
	Key      string            `json:"Key"`
	Name     string            `json:"name,omitempty"`
	BucketID string            `json:"bucket_id,omitempty"`
	Owner    string            `json:"owner,omitempty"`
	OwnerID  string            `json:"owner_id,omitempty"`
	Version  string            `json:"version,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SignURLRequest is the request body for POST /object/sign/:bucketName/*path.
type SignURLRequest struct {
	ExpiresIn int            `json:"expiresIn"`
	Transform *TransformOpts `json:"transform,omitempty"`
}

// TransformOpts for image transformations.
type TransformOpts struct {
	Height  int    `json:"height,omitempty"`
	Width   int    `json:"width,omitempty"`
	Resize  string `json:"resize,omitempty"`
	Format  string `json:"format,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

// SignURLResponse is the response for POST /object/sign/:bucketName/*path.
type SignURLResponse struct {
	SignedURL string `json:"signedURL"`
}

// SignURLsRequest is the request body for POST /object/sign/:bucketName.
type SignURLsRequest struct {
	ExpiresIn int      `json:"expiresIn"`
	Paths     []string `json:"paths"`
}

// SignURLsResponseItem is an item in the response for POST /object/sign/:bucketName.
type SignURLsResponseItem struct {
	Error     string `json:"error,omitempty"`
	Path      string `json:"path"`
	SignedURL string `json:"signedURL,omitempty"`
}

// UploadSignedURLResponse is the response for POST /object/upload/sign/:bucketName/*path.
type UploadSignedURLResponse struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

// handleCreateUploadSignedURL implements POST /object/upload/sign/:bucketName/*path.
func (s *Server) handleCreateUploadSignedURL(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	// Generate a signed URL for uploading
	// Default expiry of 1 hour (3600 seconds)
	expires := 3600 * time.Second

	signedURL, err := bucket.SignedURL(ctx, objectPath, "PUT", expires, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, UploadSignedURLResponse{
		URL:   signedURL,
		Token: "", // Token is optional and driver-specific
	})
}

// handleUploadObject implements POST /object/:bucketName/*path.
func (s *Server) handleUploadObject(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	// Clean path
	objectPath = cleanObjectPath(objectPath)

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	// Check bucket exists
	if _, err := bucket.Info(ctx); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	// Get content type from header
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Get content length
	contentLength := c.Request().ContentLength

	opts := storage.Options{}
	// Check for upsert header (Supabase uses x-upsert)
	if strings.EqualFold(c.Request().Header.Get("x-upsert"), "true") {
		opts["upsert"] = true
	}

	// Check if object already exists (for non-upsert)
	if !boolOpt(opts, "upsert") {
		if _, err := bucket.Stat(ctx, objectPath, nil); err == nil {
			return writeError(c, http.StatusConflict, fmt.Errorf("object already exists"))
		}
	}

	obj, err := bucket.Write(ctx, objectPath, c.Request().Body, contentLength, contentType, opts)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, UploadResponse{
		ID:  objectPath,
		Key: path.Join(bucketName, obj.Key),
	})
}

// handleDownloadObject implements GET /object/:bucketName/*path.
func (s *Server) handleDownloadObject(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	// Parse range header for partial content
	var offset, length int64 = 0, -1
	rangeHeader := c.Request().Header.Get("Range")
	partial := false
	if rangeHeader != "" {
		offset, length = parseRangeHeader(rangeHeader)
		partial = length >= 0 || offset > 0
	}

	rc, obj, err := bucket.Open(ctx, objectPath, offset, length, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}
	defer func() {
		_ = rc.Close()
	}()

	// Set headers
	w := c.Writer()
	if obj.ContentType != "" {
		w.Header().Set("Content-Type", obj.ContentType)
	}
	if obj.ETag != "" {
		w.Header().Set("ETag", obj.ETag)
	}

	// Handle download query param
	if c.Query("download") != "" {
		filename := path.Base(objectPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	// Handle range response
	if partial {
		end := obj.Size - 1
		if length > 0 {
			if offset+length-1 < end {
				end = offset + length - 1
			}
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, end, obj.Size))
		w.Header().Set("Content-Length", strconv.FormatInt(end-offset+1, 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(obj.Size, 10))
		w.WriteHeader(http.StatusOK)
	}

	_, _ = io.Copy(w, rc)
	return nil
}

// handleUpdateObject implements PUT /object/:bucketName/*path.
func (s *Server) handleUpdateObject(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	// Check bucket exists
	if _, err := bucket.Info(ctx); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	// For update, the object must exist
	if _, err := bucket.Stat(ctx, objectPath, nil); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	contentLength := c.Request().ContentLength

	obj, err := bucket.Write(ctx, objectPath, c.Request().Body, contentLength, contentType, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, UploadResponse{
		ID:  objectPath,
		Key: path.Join(bucketName, obj.Key),
	})
}

// handleDeleteObject implements DELETE /object/:bucketName/*path.
func (s *Server) handleDeleteObject(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	if err := bucket.Delete(ctx, objectPath, nil); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, MessageResponse{
		Message: "Successfully deleted",
	})
}

// handleDeleteObjects implements DELETE /object/:bucketName (bulk delete).
func (s *Server) handleDeleteObjects(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}

	var req DeleteObjectsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	if len(req.Prefixes) == 0 {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("prefixes is required"))
	}

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	var deleted []ObjectInfo
	for _, prefix := range req.Prefixes {
		prefix = cleanObjectPath(prefix)
		if err := bucket.Delete(ctx, prefix, nil); err == nil {
			deleted = append(deleted, ObjectInfo{
				Name: prefix,
			})
		}
	}

	return c.JSON(http.StatusOK, deleted)
}

// handleListObjects implements POST /object/list/:bucketName.
func (s *Server) handleListObjects(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}

	var req ListObjectsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Non-recursive by default to match Supabase behavior (list only direct children)
	opts := storage.Options{
		"recursive": false,
	}

	prefix := cleanObjectPath(req.Prefix)
	iter, err := bucket.List(ctx, prefix, limit, offset, opts)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}
	defer func() {
		_ = iter.Close()
	}()

	var objects []ObjectInfo
	for {
		obj, err := iter.Next()
		if err != nil {
			return writeError(c, http.StatusInternalServerError, err)
		}
		if obj == nil {
			break
		}

		// Extract just the name part (without prefix)
		name := obj.Key
		if prefix != "" && strings.HasPrefix(name, prefix+"/") {
			name = strings.TrimPrefix(name, prefix+"/")
		} else if prefix != "" && strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			name = strings.TrimPrefix(name, "/")
		}

		objects = append(objects, ObjectInfo{
			ID:        obj.Key,
			Name:      name,
			BucketID:  bucketName,
			CreatedAt: obj.Created,
			UpdatedAt: obj.Updated,
			Metadata:  obj.Metadata,
		})
	}

	if objects == nil {
		objects = []ObjectInfo{}
	}

	return c.JSON(http.StatusOK, objects)
}

// handleMoveObject implements POST /object/move.
func (s *Server) handleMoveObject(c *mizu.Ctx) error {
	var req MoveObjectRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	if req.BucketID == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucketId is required"))
	}
	if req.SourceKey == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("sourceKey is required"))
	}
	if req.DestinationKey == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("destinationKey is required"))
	}

	srcBucket := req.BucketID
	dstBucket := req.DestinationBucket
	if dstBucket == "" {
		dstBucket = srcBucket
	}

	srcKey := cleanObjectPath(req.SourceKey)
	dstKey := cleanObjectPath(req.DestinationKey)

	ctx := c.Context()
	bucket := s.store.Bucket(dstBucket)

	_, err := bucket.Move(ctx, dstKey, srcBucket, srcKey, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, MessageResponse{
		Message: "Successfully moved",
	})
}

// handleCopyObject implements POST /object/copy.
func (s *Server) handleCopyObject(c *mizu.Ctx) error {
	var req CopyObjectRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	if req.BucketID == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucketId is required"))
	}
	if req.SourceKey == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("sourceKey is required"))
	}
	if req.DestinationKey == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("destinationKey is required"))
	}

	srcBucket := req.BucketID
	dstBucket := req.DestinationBucket
	if dstBucket == "" {
		dstBucket = srcBucket
	}

	srcKey := cleanObjectPath(req.SourceKey)
	dstKey := cleanObjectPath(req.DestinationKey)

	ctx := c.Context()
	bucket := s.store.Bucket(dstBucket)

	opts := storage.Options{}
	if len(req.Metadata) > 0 {
		opts["metadata"] = req.Metadata
	}

	obj, err := bucket.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, CopyObjectResponse{
		ID:       obj.Key,
		Key:      path.Join(dstBucket, obj.Key),
		Name:     path.Base(obj.Key),
		BucketID: dstBucket,
		Metadata: obj.Metadata,
	})
}

// handleCreateSignedURL implements POST /object/sign/:bucketName/*path.
func (s *Server) handleCreateSignedURL(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	var req SignURLRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	if req.ExpiresIn <= 0 {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("expiresIn must be positive"))
	}

	objectPath = cleanObjectPath(objectPath)
	expires := time.Duration(req.ExpiresIn) * time.Second

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	signedURL, err := bucket.SignedURL(ctx, objectPath, "GET", expires, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, SignURLResponse{
		SignedURL: signedURL,
	})
}

// handleCreateSignedURLs implements POST /object/sign/:bucketName.
func (s *Server) handleCreateSignedURLs(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}

	var req SignURLsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}
	defer func() {
		_ = c.Request().Body.Close()
	}()

	if req.ExpiresIn <= 0 {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("expiresIn must be positive"))
	}
	if len(req.Paths) == 0 {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("paths is required"))
	}

	expires := time.Duration(req.ExpiresIn) * time.Second

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	var results []SignURLsResponseItem
	for _, p := range req.Paths {
		objectPath := cleanObjectPath(p)
		signedURL, err := bucket.SignedURL(ctx, objectPath, "GET", expires, nil)
		if err != nil {
			results = append(results, SignURLsResponseItem{
				Path:  p,
				Error: err.Error(),
			})
		} else {
			results = append(results, SignURLsResponseItem{
				Path:      p,
				SignedURL: signedURL,
			})
		}
	}

	return c.JSON(http.StatusOK, results)
}

// handlePublicObject implements GET /object/public/:bucketName/*path.
func (s *Server) handlePublicObject(c *mizu.Ctx) error {
	// Same as download but for public buckets
	return s.handleDownloadObject(c)
}

// handleAuthenticatedObject implements GET /object/authenticated/:bucketName/*path.
func (s *Server) handleAuthenticatedObject(c *mizu.Ctx) error {
	// Same as download but requires authentication (handled by middleware)
	return s.handleDownloadObject(c)
}

// handleObjectInfo implements GET /object/info/:bucketName/*path.
func (s *Server) handleObjectInfo(c *mizu.Ctx) error {
	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	obj, err := bucket.Stat(ctx, objectPath, nil)
	if err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	return c.JSON(http.StatusOK, ObjectInfo{
		ID:        obj.Key,
		Name:      path.Base(obj.Key),
		BucketID:  bucketName,
		CreatedAt: obj.Created,
		UpdatedAt: obj.Updated,
		Metadata:  obj.Metadata,
	})
}

// handlePublicObjectInfo implements GET /object/info/public/:bucketName/*path.
func (s *Server) handlePublicObjectInfo(c *mizu.Ctx) error {
	return s.handleObjectInfo(c)
}

// handleAuthenticatedObjectInfo implements GET /object/info/authenticated/:bucketName/*path.
func (s *Server) handleAuthenticatedObjectInfo(c *mizu.Ctx) error {
	return s.handleObjectInfo(c)
}

// cleanObjectPath normalizes an object path.
func cleanObjectPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "/")
	p = path.Clean(p)
	if p == "." {
		return ""
	}
	return p
}

// parseRangeHeader parses HTTP Range header and returns offset and length.
// Format: bytes=start-end or bytes=start-
func parseRangeHeader(h string) (offset, length int64) {
	h = strings.TrimSpace(h)
	if !strings.HasPrefix(h, "bytes=") {
		return 0, -1
	}
	h = strings.TrimPrefix(h, "bytes=")
	parts := strings.Split(h, "-")
	if len(parts) != 2 {
		return 0, -1
	}
	start, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, -1
	}
	offset = start
	if parts[1] != "" {
		end, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			return start, -1
		}
		length = end - start + 1
	}
	return offset, length
}

// boolOpt extracts a boolean option from storage.Options.
func boolOpt(opts storage.Options, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
