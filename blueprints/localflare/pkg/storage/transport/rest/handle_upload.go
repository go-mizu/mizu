package rest

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage"
	"github.com/go-mizu/mizu"
)

const (
	tusVersion       = "1.0.0"
	tusExtensions    = "creation,termination"
	tusMaxSize       = 5368709120 // 5GB default max size
	chunkContentType = "application/offset+octet-stream"
	uploadExpiration = 24 * time.Hour // Default upload expiration
)

// uploadState tracks the state of a resumable upload.
type uploadState struct {
	BucketName    string
	ObjectPath    string
	UploadLength  int64
	CurrentOffset int64
	ContentType   string
	Metadata      map[string]string
	TempFile      string
	CreatedAt     time.Time
	Upsert        bool
}

// uploadStore manages upload states in memory.
// In production, this should be replaced with persistent storage (Redis, DB, etc.)
type uploadStore struct {
	mu      sync.RWMutex
	uploads map[string]*uploadState
}

var globalUploadStore = &uploadStore{
	uploads: make(map[string]*uploadState),
}

// getUploadID generates a unique upload ID from bucket and path.
func getUploadID(bucketName, objectPath string) string {
	return fmt.Sprintf("%s/%s", bucketName, objectPath)
}

// handleTUSOptions implements OPTIONS /upload/resumable/ and /upload/resumable/{path...}
func (s *Server) handleTUSOptions(c *mizu.Ctx) error {
	w := c.Writer()
	w.Header().Set("Tus-Resumable", tusVersion)
	w.Header().Set("Tus-Version", tusVersion)
	w.Header().Set("Tus-Extension", tusExtensions)
	w.Header().Set("Tus-Max-Size", strconv.FormatInt(tusMaxSize, 10))

	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, PATCH, HEAD, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Upload-Length, Upload-Offset, Upload-Metadata, Tus-Resumable, x-upsert, apikey")
	w.Header().Set("Access-Control-Expose-Headers", "Upload-Offset, Location, Upload-Length, Tus-Resumable")
	w.Header().Set("Access-Control-Max-Age", "86400")

	w.WriteHeader(http.StatusOK)
	return nil
}

// handleTUSCreate implements POST /upload/resumable/{bucketName}/{path...}
func (s *Server) handleTUSCreate(c *mizu.Ctx) error {
	// Validate Tus-Resumable header
	tusResumable := c.Request().Header.Get("Tus-Resumable")
	if tusResumable == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("Tus-Resumable header is required"))
	}
	if tusResumable != tusVersion {
		w := c.Writer()
		w.Header().Set("Tus-Version", tusVersion)
		return writeError(c, http.StatusPreconditionFailed, fmt.Errorf("unsupported TUS version: %s", tusResumable))
	}

	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)

	// Get Upload-Length header
	uploadLengthStr := c.Request().Header.Get("Upload-Length")
	uploadDeferLength := c.Request().Header.Get("Upload-Defer-Length")

	var uploadLength int64 = -1
	if uploadLengthStr != "" {
		var err error
		uploadLength, err = strconv.ParseInt(uploadLengthStr, 10, 64)
		if err != nil || uploadLength < 0 {
			return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid Upload-Length header"))
		}

		if uploadLength > tusMaxSize {
			return writeError(c, http.StatusRequestEntityTooLarge, fmt.Errorf("upload exceeds maximum size"))
		}
	} else if uploadDeferLength != "1" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("either Upload-Length or Upload-Defer-Length must be specified"))
	}

	ctx := c.Context()
	bucket := s.store.Bucket(bucketName)

	// Check bucket exists
	if _, err := bucket.Info(ctx); err != nil {
		return writeError(c, mapStorageError(err), err)
	}

	// Parse Upload-Metadata header
	metadata := parseUploadMetadata(c.Request().Header.Get("Upload-Metadata"))
	contentType := metadata["content-type"]
	if contentType == "" {
		contentType = metadata["contentType"]
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Check for upsert
	upsert := strings.EqualFold(c.Request().Header.Get("x-upsert"), "true")

	// Check if object already exists (for non-upsert)
	if !upsert {
		if _, err := bucket.Stat(ctx, objectPath, nil); err == nil {
			return writeError(c, http.StatusConflict, fmt.Errorf("object already exists"))
		}
	}

	// Create temporary file for storing chunks
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "tus-upload-*")
	if err != nil {
		return writeError(c, http.StatusInternalServerError, fmt.Errorf("failed to create temp file: %w", err))
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()

	// Store upload state
	uploadID := getUploadID(bucketName, objectPath)
	state := &uploadState{
		BucketName:    bucketName,
		ObjectPath:    objectPath,
		UploadLength:  uploadLength,
		CurrentOffset: 0,
		ContentType:   contentType,
		Metadata:      metadata,
		TempFile:      tempPath,
		CreatedAt:     time.Now(),
		Upsert:        upsert,
	}

	globalUploadStore.mu.Lock()
	globalUploadStore.uploads[uploadID] = state
	globalUploadStore.mu.Unlock()

	// Set response headers
	w := c.Writer()
	w.Header().Set("Tus-Resumable", tusVersion)
	w.Header().Set("Upload-Offset", "0")
	if uploadLength >= 0 {
		w.Header().Set("Upload-Length", strconv.FormatInt(uploadLength, 10))
	}

	// Generate location URL
	location := fmt.Sprintf("/upload/resumable/%s/%s", bucketName, objectPath)
	w.Header().Set("Location", location)

	// Set expiration
	expiresAt := time.Now().Add(uploadExpiration)
	w.Header().Set("Upload-Expires", expiresAt.Format(http.TimeFormat))

	w.WriteHeader(http.StatusCreated)
	return nil
}

// handleTUSPatch implements PATCH /upload/resumable/{bucketName}/{path...}
func (s *Server) handleTUSPatch(c *mizu.Ctx) error {
	// Validate Tus-Resumable header
	tusResumable := c.Request().Header.Get("Tus-Resumable")
	if tusResumable != tusVersion {
		return writeError(c, http.StatusPreconditionFailed, fmt.Errorf("unsupported TUS version"))
	}

	// Validate Content-Type
	contentType := c.Request().Header.Get("Content-Type")
	if contentType != chunkContentType {
		return writeError(c, http.StatusUnsupportedMediaType, fmt.Errorf("Content-Type must be %s", chunkContentType))
	}

	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)
	uploadID := getUploadID(bucketName, objectPath)

	// Get upload state
	globalUploadStore.mu.RLock()
	state, exists := globalUploadStore.uploads[uploadID]
	globalUploadStore.mu.RUnlock()

	if !exists {
		return writeError(c, http.StatusNotFound, fmt.Errorf("upload not found"))
	}

	// Validate Upload-Offset
	uploadOffsetStr := c.Request().Header.Get("Upload-Offset")
	if uploadOffsetStr == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("Upload-Offset header is required"))
	}

	uploadOffset, err := strconv.ParseInt(uploadOffsetStr, 10, 64)
	if err != nil || uploadOffset < 0 {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("invalid Upload-Offset header"))
	}

	// Check if offset matches current offset
	if uploadOffset != state.CurrentOffset {
		return writeError(c, http.StatusConflict, fmt.Errorf("Upload-Offset mismatch: expected %d, got %d", state.CurrentOffset, uploadOffset))
	}

	// Open temp file for appending
	tempFile, err := os.OpenFile(state.TempFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return writeError(c, http.StatusInternalServerError, fmt.Errorf("failed to open temp file: %w", err))
	}
	defer func() {
		_ = tempFile.Close()
	}()

	// Write chunk to temp file
	written, err := io.Copy(tempFile, c.Request().Body)
	if err != nil {
		return writeError(c, http.StatusInternalServerError, fmt.Errorf("failed to write chunk: %w", err))
	}

	// Update state
	globalUploadStore.mu.Lock()
	state.CurrentOffset += written
	globalUploadStore.mu.Unlock()

	// Check if upload is complete
	complete := state.UploadLength >= 0 && state.CurrentOffset >= state.UploadLength

	if complete {
		// Upload is complete, move to final location
		ctx := c.Context()
		bucket := s.store.Bucket(bucketName)

		// Open temp file for reading
		uploadFile, err := os.Open(state.TempFile)
		if err != nil {
			return writeError(c, http.StatusInternalServerError, fmt.Errorf("failed to open upload file: %w", err))
		}
		defer func() {
			_ = uploadFile.Close()
			// Clean up temp file
			_ = os.Remove(state.TempFile)
		}()

		// Write to final location
		opts := storage.Options{}
		if state.Upsert {
			opts["upsert"] = true
		}

		_, err = bucket.Write(ctx, state.ObjectPath, uploadFile, state.UploadLength, state.ContentType, opts)
		if err != nil {
			return writeError(c, mapStorageError(err), err)
		}

		// Clean up upload state
		globalUploadStore.mu.Lock()
		delete(globalUploadStore.uploads, uploadID)
		globalUploadStore.mu.Unlock()
	}

	// Set response headers
	w := c.Writer()
	w.Header().Set("Tus-Resumable", tusVersion)
	w.Header().Set("Upload-Offset", strconv.FormatInt(state.CurrentOffset, 10))
	if state.UploadLength >= 0 {
		w.Header().Set("Upload-Length", strconv.FormatInt(state.UploadLength, 10))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// handleTUSHead implements HEAD /upload/resumable/{bucketName}/{path...}
func (s *Server) handleTUSHead(c *mizu.Ctx) error {
	// Validate Tus-Resumable header
	tusResumable := c.Request().Header.Get("Tus-Resumable")
	if tusResumable != tusVersion {
		return writeError(c, http.StatusPreconditionFailed, fmt.Errorf("unsupported TUS version"))
	}

	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)
	uploadID := getUploadID(bucketName, objectPath)

	// Get upload state
	globalUploadStore.mu.RLock()
	state, exists := globalUploadStore.uploads[uploadID]
	globalUploadStore.mu.RUnlock()

	if !exists {
		return writeError(c, http.StatusNotFound, fmt.Errorf("upload not found"))
	}

	// Set response headers
	w := c.Writer()
	w.Header().Set("Tus-Resumable", tusVersion)
	w.Header().Set("Upload-Offset", strconv.FormatInt(state.CurrentOffset, 10))
	if state.UploadLength >= 0 {
		w.Header().Set("Upload-Length", strconv.FormatInt(state.UploadLength, 10))
	}
	w.Header().Set("Cache-Control", "no-store")

	w.WriteHeader(http.StatusOK)
	return nil
}

// handleTUSDelete implements DELETE /upload/resumable/{bucketName}/{path...}
func (s *Server) handleTUSDelete(c *mizu.Ctx) error {
	// Validate Tus-Resumable header
	tusResumable := c.Request().Header.Get("Tus-Resumable")
	if tusResumable != tusVersion {
		return writeError(c, http.StatusPreconditionFailed, fmt.Errorf("unsupported TUS version"))
	}

	bucketName := c.Param("bucketName")
	objectPath := c.Param("path")

	if bucketName == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("bucket name is required"))
	}
	if objectPath == "" {
		return writeError(c, http.StatusBadRequest, fmt.Errorf("object path is required"))
	}

	objectPath = cleanObjectPath(objectPath)
	uploadID := getUploadID(bucketName, objectPath)

	// Get and remove upload state
	globalUploadStore.mu.Lock()
	state, exists := globalUploadStore.uploads[uploadID]
	if exists {
		delete(globalUploadStore.uploads, uploadID)
	}
	globalUploadStore.mu.Unlock()

	if !exists {
		return writeError(c, http.StatusNotFound, fmt.Errorf("upload not found"))
	}

	// Clean up temp file
	if state.TempFile != "" {
		_ = os.Remove(state.TempFile)
	}

	// Set response headers
	w := c.Writer()
	w.Header().Set("Tus-Resumable", tusVersion)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// parseUploadMetadata parses the Upload-Metadata header.
// Format: key1 base64value1,key2 base64value2
func parseUploadMetadata(header string) map[string]string {
	metadata := make(map[string]string)
	if header == "" {
		return metadata
	}

	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, " ", 2)
		if len(parts) < 1 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}

		var value string
		if len(parts) == 2 {
			// Decode base64 value
			decoded, err := base64.StdEncoding.DecodeString(parts[1])
			if err == nil {
				value = string(decoded)
			} else {
				// If decode fails, use raw value
				value = parts[1]
			}
		}

		metadata[key] = value
	}

	return metadata
}

// cleanupExpiredUploads removes expired upload states.
// This should be called periodically by a background job.
func cleanupExpiredUploads() {
	now := time.Now()

	globalUploadStore.mu.Lock()
	defer globalUploadStore.mu.Unlock()

	for uploadID, state := range globalUploadStore.uploads {
		if now.Sub(state.CreatedAt) > uploadExpiration {
			// Clean up temp file
			if state.TempFile != "" {
				_ = os.Remove(state.TempFile)
			}
			delete(globalUploadStore.uploads, uploadID)
		}
	}
}

// StartUploadCleanup starts a background goroutine to clean up expired uploads.
func StartUploadCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredUploads()
		}
	}()
}

// GetUploadState retrieves the current state of an upload (for testing).
func GetUploadState(bucketName, objectPath string) *uploadState {
	uploadID := getUploadID(bucketName, objectPath)
	globalUploadStore.mu.RLock()
	defer globalUploadStore.mu.RUnlock()
	return globalUploadStore.uploads[uploadID]
}

// ClearUploadStates clears all upload states (for testing).
func ClearUploadStates() {
	globalUploadStore.mu.Lock()
	defer globalUploadStore.mu.Unlock()

	// Clean up temp files
	for _, state := range globalUploadStore.uploads {
		if state.TempFile != "" {
			_ = os.Remove(state.TempFile)
		}
	}

	globalUploadStore.uploads = make(map[string]*uploadState)
}

// getTempFilePath returns the temp file path for an upload (for testing).
func getTempFilePath(bucketName, objectPath string) string {
	state := GetUploadState(bucketName, objectPath)
	if state == nil {
		return ""
	}
	return state.TempFile
}
