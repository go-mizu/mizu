package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	store   *postgres.Store
	dataDir string // Directory for storing file content
}

// NewStorageHandler creates a new storage handler.
func NewStorageHandler(store *postgres.Store) *StorageHandler {
	// Default data directory
	dataDir := os.Getenv("LOCALBASE_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data/storage"
	}
	// Ensure directory exists
	os.MkdirAll(dataDir, 0755)

	return &StorageHandler{
		store:   store,
		dataDir: dataDir,
	}
}

// getFilePath returns the filesystem path for a storage object
func (h *StorageHandler) getFilePath(bucketID, objectPath string) string {
	return filepath.Join(h.dataDir, bucketID, objectPath)
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

	// Get bucket to verify it exists (try by ID first, then by name for Supabase compatibility)
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		// Try by name as fallback
		bucket, err = h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
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

	// Save file content to filesystem
	filePath := h.getFilePath(bucketID, path)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to create directory")
	}
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to save file")
	}

	if err := h.store.Storage().CreateObject(c.Context(), obj); err != nil {
		if strings.Contains(err.Error(), "duplicate") && upsert {
			// Update existing object
			if err := h.store.Storage().UpdateObject(c.Context(), obj); err != nil {
				// Clean up file on failure
				os.Remove(filePath)
				return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to update object")
			}
		} else if strings.Contains(err.Error(), "duplicate") {
			// Clean up file on conflict
			os.Remove(filePath)
			return storageError(c, http.StatusConflict, "Conflict", "object already exists")
		} else {
			// Clean up file on failure
			os.Remove(filePath)
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

	// Check bucket exists and get bucket for RLS check (try by ID first, then by name)
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		// Try by name as fallback
		bucket, err = h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
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

	// Get bucket for RLS check (try by ID first, then by name for Supabase compatibility)
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		// Try by name as fallback
		bucket, err = h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
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

	// Try to read actual file content from filesystem
	filePath := h.getFilePath(bucketID, path)
	content, err := os.ReadFile(filePath)
	if err == nil {
		// Serve actual file content
		c.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		return c.Bytes(http.StatusOK, content, obj.ContentType)
	}

	// File doesn't exist on filesystem (seeded files) - generate placeholder content
	placeholder, placeholderContentType := h.generatePlaceholderContent(obj.ContentType, obj.Name, obj.Size)
	c.Header().Set("Content-Length", fmt.Sprintf("%d", len(placeholder)))
	return c.Bytes(http.StatusOK, placeholder, placeholderContentType)
}

// generatePlaceholderContent generates appropriate placeholder content based on content type
// Returns the content and the actual content type to use (may differ for placeholders)
func (h *StorageHandler) generatePlaceholderContent(contentType, name string, size int64) ([]byte, string) {
	// For images, generate an SVG placeholder (returns SVG content type)
	if strings.HasPrefix(contentType, "image/") {
		return h.generateImagePlaceholder(name, contentType), "image/svg+xml"
	}

	// For text files, generate sample text
	if strings.HasPrefix(contentType, "text/") || contentType == "application/json" ||
		contentType == "application/xml" || contentType == "application/x-yaml" ||
		contentType == "application/sql" || contentType == "application/toml" {
		return h.generateTextPlaceholder(name, contentType), contentType
	}

	// For other types, return a generic placeholder as plain text
	return []byte(fmt.Sprintf("Placeholder content for: %s\nContent-Type: %s\nSize: %d bytes", name, contentType, size)), "text/plain"
}

// generateImagePlaceholder generates an SVG placeholder for image files
func (h *StorageHandler) generateImagePlaceholder(name, contentType string) []byte {
	// Get filename without path
	filename := filepath.Base(name)

	// Generate a color based on filename hash
	hash := 0
	for _, c := range filename {
		hash = hash*31 + int(c)
	}
	colors := []string{"#3B82F6", "#10B981", "#F59E0B", "#EF4444", "#8B5CF6", "#EC4899", "#06B6D4", "#84CC16"}
	bgColor := colors[abs(hash)%len(colors)]

	// Create SVG placeholder
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="300" viewBox="0 0 400 300">
  <rect width="400" height="300" fill="%s"/>
  <rect x="160" y="100" width="80" height="60" fill="white" opacity="0.3" rx="8"/>
  <circle cx="185" cy="118" r="8" fill="white" opacity="0.5"/>
  <polygon points="165,150 200,125 235,150" fill="white" opacity="0.5"/>
  <text x="200" y="200" font-family="system-ui, sans-serif" font-size="14" fill="white" text-anchor="middle" opacity="0.8">%s</text>
</svg>`, bgColor, filename)

	return []byte(svg)
}

// generateTextPlaceholder generates sample text content based on file type
func (h *StorageHandler) generateTextPlaceholder(name, contentType string) []byte {
	filename := filepath.Base(name)
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".json":
		return []byte(`{
  "name": "sample",
  "version": "1.0.0",
  "description": "Sample JSON file",
  "data": {
    "items": ["item1", "item2", "item3"],
    "count": 3
  }
}`)
	case ".yaml", ".yml":
		return []byte(`# Sample YAML configuration
name: sample
version: 1.0.0
settings:
  debug: true
  timeout: 30
items:
  - item1
  - item2
  - item3
`)
	case ".md":
		return []byte(`# Sample Markdown

This is a sample markdown file.

## Features

- Feature 1
- Feature 2
- Feature 3

## Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `
`)
	case ".sql":
		return []byte(`-- Sample SQL file
SELECT * FROM users WHERE active = true;

CREATE TABLE IF NOT EXISTS sample (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
`)
	case ".go":
		return []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`)
	case ".ts", ".tsx":
		return []byte(`import React from 'react';

interface Props {
  title: string;
}

export const Component: React.FC<Props> = ({ title }) => {
  return <div>{title}</div>;
};
`)
	case ".py":
		return []byte(`#!/usr/bin/env python3
"""Sample Python script."""

def main():
    print("Hello, World!")

if __name__ == "__main__":
    main()
`)
	case ".css":
		return []byte(`:root {
  --primary-color: #3b82f6;
  --background: #ffffff;
}

body {
  font-family: system-ui, sans-serif;
  background: var(--background);
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem;
}
`)
	case ".html":
		return []byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sample HTML</title>
</head>
<body>
  <h1>Hello, World!</h1>
  <p>This is a sample HTML file.</p>
</body>
</html>
`)
	case ".svg":
		return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
  <circle cx="50" cy="50" r="40" fill="#3B82F6"/>
  <text x="50" y="55" font-family="system-ui" font-size="12" fill="white" text-anchor="middle">SVG</text>
</svg>`)
	default:
		return []byte(fmt.Sprintf("// Sample content for: %s\n// Content-Type: %s\n\nThis is placeholder content.", filename, contentType))
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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

	// Try bucket lookup by ID first, then by name for Supabase compatibility
	if _, err := h.store.Storage().GetBucket(c.Context(), bucketID); err != nil {
		bucket, err := h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
	}

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

	// Get bucket for RLS check (try by ID first, then by name for Supabase compatibility)
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		// Try by name as fallback
		bucket, err = h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
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

	// Try bucket lookup by ID first, then by name for Supabase compatibility
	if _, err := h.store.Storage().GetBucket(c.Context(), bucketID); err != nil {
		bucket, err := h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
	}

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

	// Try bucket lookup by ID first, then by name for Supabase compatibility
	if _, err := h.store.Storage().GetBucket(c.Context(), bucketID); err != nil {
		bucket, err := h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
	}

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

	// Check bucket exists (try by ID first, then by name for Supabase compatibility)
	if _, err := h.store.Storage().GetBucket(c.Context(), bucketID); err != nil {
		bucket, err := h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
	}

	token := uuid.New().String()

	return c.JSON(http.StatusOK, map[string]string{
		"url":   fmt.Sprintf("/storage/v1/object/%s/%s", bucketID, path),
		"token": token,
	})
}

// RenameObject renames an object (changes its path within the same bucket).
func (h *StorageHandler) RenameObject(c *mizu.Ctx) error {
	var req struct {
		BucketID string `json:"bucketId"`
		OldPath  string `json:"oldPath"`
		NewPath  string `json:"newPath"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return storageError(c, http.StatusBadRequest, "Bad Request", "invalid request body")
	}

	if req.BucketID == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "bucketId required")
	}
	if req.OldPath == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "oldPath required")
	}
	if req.NewPath == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "newPath required")
	}

	// Sanitize paths
	req.OldPath = sanitizeStoragePath(req.OldPath)
	req.NewPath = sanitizeStoragePath(req.NewPath)

	// Get bucket for RLS check
	bucket, err := h.store.Storage().GetBucket(c.Context(), req.BucketID)
	if err != nil {
		bucket, err = h.store.Storage().GetBucketByName(c.Context(), req.BucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		req.BucketID = bucket.ID
	}

	// Check access for both old and new paths
	if err := h.checkStorageAccess(c, bucket, req.OldPath, StorageAccessWrite); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
	}
	if err := h.checkStorageAccess(c, bucket, req.NewPath, StorageAccessWrite); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
	}

	// Check source exists
	if _, err := h.store.Storage().GetObject(c.Context(), req.BucketID, req.OldPath); err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "source object not found")
	}

	// Check destination doesn't exist
	if _, err := h.store.Storage().GetObject(c.Context(), req.BucketID, req.NewPath); err == nil {
		return storageError(c, http.StatusConflict, "Conflict", "destination object already exists")
	}

	// Move file on filesystem if it exists
	oldFilePath := h.getFilePath(req.BucketID, req.OldPath)
	newFilePath := h.getFilePath(req.BucketID, req.NewPath)
	if _, err := os.Stat(oldFilePath); err == nil {
		if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
			return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to create directory")
		}
		if err := os.Rename(oldFilePath, newFilePath); err != nil {
			return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to rename file")
		}
	}

	// Move in database
	if err := h.store.Storage().MoveObject(c.Context(), req.BucketID, req.OldPath, req.NewPath); err != nil {
		// Try to rollback filesystem change
		if _, statErr := os.Stat(newFilePath); statErr == nil {
			os.Rename(newFilePath, oldFilePath)
		}
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to rename object")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully renamed",
		"oldPath": req.OldPath,
		"newPath": req.NewPath,
	})
}

// DeleteFolder recursively deletes a folder and all its contents.
func (h *StorageHandler) DeleteFolder(c *mizu.Ctx) error {
	bucketID := c.Param("bucket")
	path := c.Param("path")

	if path == "" {
		return storageError(c, http.StatusBadRequest, "Bad Request", "folder path is required")
	}

	// Ensure path ends with / for folder
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	// Sanitize path
	path = sanitizeStoragePath(path)
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	// Get bucket for RLS check
	bucket, err := h.store.Storage().GetBucket(c.Context(), bucketID)
	if err != nil {
		bucket, err = h.store.Storage().GetBucketByName(c.Context(), bucketID)
		if err != nil {
			return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
		}
		bucketID = bucket.ID
	}

	// Check delete access
	if err := h.checkStorageAccess(c, bucket, path, StorageAccessDelete); err != nil {
		return storageError(c, http.StatusForbidden, "Forbidden", err.Error())
	}

	// List all objects with this prefix
	objects, err := h.store.Storage().ListObjects(c.Context(), bucketID, path, 10000, 0)
	if err != nil {
		return storageError(c, http.StatusInternalServerError, "Internal Server Error", "failed to list objects")
	}

	if len(objects) == 0 {
		return storageError(c, http.StatusNotFound, "Not Found", "folder not found or empty")
	}

	// Delete all objects
	var deleted []string
	for _, obj := range objects {
		// Delete from filesystem
		filePath := h.getFilePath(bucketID, obj.Name)
		os.Remove(filePath)

		// Delete from database
		if err := h.store.Storage().DeleteObject(c.Context(), bucketID, obj.Name); err == nil {
			deleted = append(deleted, obj.Name)
		}
	}

	// Try to remove empty directories
	folderPath := h.getFilePath(bucketID, path)
	os.RemoveAll(folderPath)

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Successfully deleted folder",
		"deleted": len(deleted),
		"files":   deleted,
	})
}

// GetBucketByName gets a bucket by name (for Supabase compatibility).
func (h *StorageHandler) GetBucketByName(c *mizu.Ctx) error {
	name := c.Param("name")

	bucket, err := h.store.Storage().GetBucketByName(c.Context(), name)
	if err != nil {
		return storageError(c, http.StatusNotFound, "Not Found", "bucket not found")
	}

	return c.JSON(http.StatusOK, bucket)
}
