// File: lib/storage/driver/local/multipart.go
package local

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// Ensure bucket implements storage.HasMultipart
var _ storage.HasMultipart = (*bucket)(nil)

const multipartPrefix = "_multipart"

// multipartMeta describes one in progress multipart upload.
type multipartMeta struct {
	Bucket      string            `json:"bucket"`
	Key         string            `json:"key"`
	ContentType string            `json:"content_type,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// InitMultipart starts a multipart upload for the given key.
func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Extract metadata from options
	meta := map[string]string{}
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			meta = m
		}
	}

	uploadID := newUploadID()

	m := multipartMeta{
		Bucket:      b.name,
		Key:         relKey,
		ContentType: contentType,
		Metadata:    meta,
		CreatedAt:   time.Now().UTC(),
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("local: marshal multipart meta: %w", err)
	}

	// Store metadata as a small file
	metaPath, err := b.multipartMetaPath(uploadID)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(metaPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("local: mkdir multipart meta dir: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("local: write multipart meta: %w", err)
	}

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      relKey,
		UploadID: uploadID,
		Metadata: meta,
	}, nil
}

// UploadPart uploads a single part for an existing multipart upload.
func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number <= 0 || number > 10000 {
		return nil, fmt.Errorf("local: part number %d out of range (1-10000)", number)
	}

	// Verify metadata exists
	if _, err := b.loadMultipartMeta(mu.UploadID); err != nil {
		return nil, err
	}

	partPath, err := b.multipartPartPath(mu.UploadID, number)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(partPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("local: mkdir part dir: %w", err)
	}

	// Write part to temporary file then rename
	tmp, err := os.CreateTemp(dir, ".lake-part-tmp-*")
	if err != nil {
		return nil, fmt.Errorf("local: create temp part file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	written, err := io.Copy(tmp, src)
	if err != nil {
		return nil, fmt.Errorf("local: write part %d: %w", number, err)
	}

	if err := tmp.Sync(); err != nil {
		return nil, fmt.Errorf("local: fsync part %d: %w", number, err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("local: close temp part %d: %w", number, err)
	}

	if err := os.Rename(tmpName, partPath); err != nil {
		return nil, fmt.Errorf("local: rename part %d: %w", number, err)
	}

	info, err := os.Stat(partPath)
	if err != nil {
		return nil, fmt.Errorf("local: stat part %d: %w", number, err)
	}

	modTime := info.ModTime()
	return &storage.PartInfo{
		Number:       number,
		Size:         written,
		ETag:         fmt.Sprintf("%d-%x", number, written), // Simple etag
		LastModified: &modTime,
	}, nil
}

// CopyPart is not implemented for local driver
func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	return nil, storage.ErrUnsupported
}

// ListParts lists already uploaded parts for a multipart upload.
func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Verify metadata exists
	if _, err := b.loadMultipartMeta(mu.UploadID); err != nil {
		return nil, err
	}

	partDir, err := b.multipartPartDir(mu.UploadID)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(partDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*storage.PartInfo{}, nil
		}
		return nil, fmt.Errorf("local: read part dir: %w", err)
	}

	var parts []*storage.PartInfo
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "part-") {
			continue
		}
		nStr := strings.TrimPrefix(e.Name(), "part-")
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime()
		parts = append(parts, &storage.PartInfo{
			Number:       n,
			Size:         info.Size(),
			ETag:         fmt.Sprintf("%d-%x", n, info.Size()),
			LastModified: &modTime,
		})
	}

	// Sort by part number
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Apply pagination
	if offset < 0 {
		offset = 0
	}
	if offset > len(parts) {
		offset = len(parts)
	}
	parts = parts[offset:]

	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

// CompleteMultipart completes a multipart upload and assembles the final object.
func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	meta, err := b.loadMultipartMeta(mu.UploadID)
	if err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("local: no parts to complete")
	}

	// Sort parts by number
	sortedParts := make([]*storage.PartInfo, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool {
		return sortedParts[i].Number < sortedParts[j].Number
	})

	// Prepare final object path
	relKey, err := cleanKey(meta.Key)
	if err != nil {
		return nil, err
	}

	full, err := joinUnderRoot(b.root, relKey)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(full)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("local: mkdir final object dir: %w", err)
	}

	// Create temp file for final object
	tmp, err := os.CreateTemp(dir, ".lake-complete-tmp-*")
	if err != nil {
		return nil, fmt.Errorf("local: create temp final file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	// Concatenate parts
	var totalSize int64
	for _, part := range sortedParts {
		partPath, err := b.multipartPartPath(mu.UploadID, part.Number)
		if err != nil {
			return nil, err
		}

		// #nosec G304 -- path is validated
		pf, err := os.Open(partPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("local: part %d not found", part.Number)
			}
			return nil, fmt.Errorf("local: open part %d: %w", part.Number, err)
		}

		written, err := io.Copy(tmp, pf)
		_ = pf.Close()
		if err != nil {
			return nil, fmt.Errorf("local: copy part %d: %w", part.Number, err)
		}
		totalSize += written
	}

	if err := tmp.Sync(); err != nil {
		return nil, fmt.Errorf("local: fsync final object: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("local: close temp final object: %w", err)
	}

	// Rename to final location
	if err := os.Rename(tmpName, full); err != nil {
		return nil, fmt.Errorf("local: rename to final object: %w", err)
	}

	// Clean up multipart files
	_ = b.cleanupMultipart(mu.UploadID)

	info, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("local: stat final object: %w", err)
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        info.Size(),
		ContentType: meta.ContentType,
		Metadata:    meta.Metadata,
		Created:     info.ModTime(),
		Updated:     info.ModTime(),
	}, nil
}

// AbortMultipart aborts the multipart upload and discards all parts.
func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Check if metadata exists
	if _, err := b.loadMultipartMeta(mu.UploadID); err != nil {
		// Already not found, treat as success
		if errors.Is(err, storage.ErrNotExist) {
			return nil
		}
		return err
	}

	// Clean up all multipart files
	return b.cleanupMultipart(mu.UploadID)
}

// Helper methods

func (b *bucket) multipartMetaPath(uploadID string) (string, error) {
	rel := filepath.Join(multipartPrefix, uploadID, "meta.json")
	return joinUnderRoot(b.root, rel)
}

func (b *bucket) multipartPartDir(uploadID string) (string, error) {
	rel := filepath.Join(multipartPrefix, uploadID)
	return joinUnderRoot(b.root, rel)
}

func (b *bucket) multipartPartPath(uploadID string, partNum int) (string, error) {
	rel := filepath.Join(multipartPrefix, uploadID, fmt.Sprintf("part-%05d", partNum))
	return joinUnderRoot(b.root, rel)
}

func (b *bucket) loadMultipartMeta(uploadID string) (*multipartMeta, error) {
	metaPath, err := b.multipartMetaPath(uploadID)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- path is validated
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("local: read multipart meta: %w", err)
	}

	var m multipartMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("local: unmarshal multipart meta: %w", err)
	}

	return &m, nil
}

func (b *bucket) cleanupMultipart(uploadID string) error {
	uploadDir, err := b.multipartPartDir(uploadID)
	if err != nil {
		return err
	}

	// Remove entire upload directory
	if err := os.RemoveAll(uploadDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("local: cleanup multipart: %w", err)
	}

	return nil
}

// newUploadID generates a random upload id.
func newUploadID() string {
	now := time.Now().UTC().UnixNano()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to timestamp-only ID if random fails
		return fmt.Sprintf("%x-0", now)
	}
	r := binary.LittleEndian.Uint64(b[:])
	return fmt.Sprintf("%x-%x", now, r)
}
