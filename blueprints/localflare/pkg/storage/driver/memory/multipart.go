// File: lib/storage/driver/memory/multipart.go
package memdriver

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// Ensure bucket implements storage.HasMultipart.
var _ storage.HasMultipart = (*bucket)(nil)

// multipartUpload tracks an in progress multipart upload in memory.
type multipartUpload struct {
	mu *storage.MultipartUpload

	contentType string
	metadata    map[string]string
	createdAt   time.Time

	// parts maps part number to part data.
	parts map[int]*partData
}

type partData struct {
	number       int
	data         []byte
	etag         string
	lastModified time.Time
}

// InitMultipart starts a multipart upload for the given key.
func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = ctx

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("mem: key is empty")
	}

	// Extract metadata from options.
	meta := extractMetadata(opts)

	uploadID := newUploadID()

	mu := &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: uploadID,
		Metadata: meta,
	}

	b.mpMu.Lock()
	defer b.mpMu.Unlock()

	b.mpUploads[uploadID] = &multipartUpload{
		mu:          mu,
		contentType: contentType,
		metadata:    cloneStringMap(meta),
		createdAt:   time.Now().UTC(),
		parts:       make(map[int]*partData),
	}

	return mu, nil
}

// UploadPart uploads a single part for an existing multipart upload.
func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	if number <= 0 || number > 10000 {
		return nil, fmt.Errorf("mem: part number %d out of range (1-10000)", number)
	}

	// Read part data outside lock with pre-allocation.
	var data []byte
	if size >= 0 {
		// Pre-allocate exact capacity for known size.
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, err
		}
		data = buf.Bytes()
	}

	now := time.Now().UTC()

	// Compute ETag for this part as MD5 hex.
	sum := md5.Sum(data)
	etag := hex.EncodeToString(sum[:])

	pd := &partData{
		number:       number,
		data:         data,
		etag:         etag,
		lastModified: now,
	}

	// Short lock to update upload state.
	b.mpMu.Lock()
	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		b.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	upload.parts[number] = pd
	b.mpMu.Unlock()

	return &storage.PartInfo{
		Number:       number,
		Size:         int64(len(data)),
		ETag:         etag,
		LastModified: &now,
	}, nil
}

// CopyPart is not implemented for memory driver.
func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = mu
	_ = number
	_ = opts
	return nil, storage.ErrUnsupported
}

// ListParts lists already uploaded parts for a multipart upload.
func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	b.mpMu.RLock()
	defer b.mpMu.RUnlock()

	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Collect parts and sort by number.
	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, pd := range upload.parts {
		lastMod := pd.lastModified
		parts = append(parts, &storage.PartInfo{
			Number:       pd.number,
			Size:         int64(len(pd.data)),
			ETag:         pd.etag,
			LastModified: &lastMod,
		})
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Apply pagination.
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
	_ = ctx
	_ = opts

	if len(parts) == 0 {
		return nil, fmt.Errorf("mem: no parts to complete")
	}

	b.mpMu.Lock()
	defer b.mpMu.Unlock()

	upload, ok := b.mpUploads[mu.UploadID]
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Sort parts by number according to the order requested by the client.
	sortedParts := make([]*storage.PartInfo, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool {
		return sortedParts[i].Number < sortedParts[j].Number
	})

	// Pre-calculate total size for single allocation.
	totalSize := 0
	for _, part := range sortedParts {
		pd, ok := upload.parts[part.Number]
		if !ok {
			return nil, fmt.Errorf("mem: part %d not found", part.Number)
		}
		totalSize += len(pd.data)
	}

	// Single allocation for final data.
	data := make([]byte, 0, totalSize)
	hash := md5.New()

	// Concatenate part data in order and compute MD5.
	for _, part := range sortedParts {
		pd := upload.parts[part.Number]
		data = append(data, pd.data...)
		hash.Write(pd.data)
	}

	sum := hash.Sum(nil)
	objETag := hex.EncodeToString(sum)

	// Create final object using sync.Map.
	now := time.Now().UTC()

	e := &entry{
		obj: storage.Object{
			Bucket:      b.name,
			Key:         upload.mu.Key,
			Size:        int64(len(data)),
			ContentType: upload.contentType,
			Created:     now,
			Updated:     now,
			Metadata:    cloneStringMap(upload.metadata),
			IsDir:       false,
			ETag:        objETag,
			Hash: map[string]string{
				"md5":  objETag,
				"etag": objETag,
			},
		},
		data: data,
	}

	// Store using sync.Map.
	b.obj.Store(upload.mu.Key, e)

	// Update sharded key index.
	b.addKey(upload.mu.Key)

	// Clean up multipart upload.
	delete(b.mpUploads, mu.UploadID)

	objCopy := e.obj
	return &objCopy, nil
}

// AbortMultipart aborts the multipart upload and discards all parts.
func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = ctx
	_ = opts

	b.mpMu.Lock()
	defer b.mpMu.Unlock()

	if _, ok := b.mpUploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}

	delete(b.mpUploads, mu.UploadID)
	return nil
}

// newUploadID generates a random upload id.
func newUploadID() string {
	now := time.Now().UTC().UnixNano()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to timestamp-only ID if random fails.
		return fmt.Sprintf("%x-0", now)
	}
	r := binary.LittleEndian.Uint64(b[:])
	return fmt.Sprintf("%x-%x", now, r)
}
