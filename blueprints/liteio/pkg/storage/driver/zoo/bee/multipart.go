package bee

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

// Ensure bucket supports multipart APIs.
var _ storage.HasMultipart = (*bucket)(nil)

type multipartUpload struct {
	mu          *storage.MultipartUpload
	contentType string
	createdAt   time.Time
	parts       map[int]*partData
}

type partData struct {
	number       int
	data         []byte
	etag         string
	lastModified time.Time
}

type multipartRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*multipartUpload
}

func newMultipartRegistry() *multipartRegistry {
	return &multipartRegistry{uploads: make(map[string]*multipartUpload)}
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("bee: key is empty")
	}

	uploadID := newUploadID()
	mu := &storage.MultipartUpload{Bucket: b.name, Key: key, UploadID: uploadID}

	b.st.mp.mu.Lock()
	b.st.mp.uploads[uploadID] = &multipartUpload{
		mu:          mu,
		contentType: contentType,
		createdAt:   time.Now(),
		parts:       make(map[int]*partData),
	}
	b.st.mp.mu.Unlock()

	return mu, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if mu == nil || mu.UploadID == "" {
		return nil, fmt.Errorf("bee: invalid multipart upload")
	}
	if number <= 0 || number > 10000 {
		return nil, fmt.Errorf("bee: part number %d out of range (1-10000)", number)
	}

	data, err := readAllSized(src, size)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sum := md5.Sum(data)
	etag := hex.EncodeToString(sum[:])

	pd := &partData{
		number:       number,
		data:         data,
		etag:         etag,
		lastModified: now,
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	upload.parts[number] = pd
	b.st.mp.mu.Unlock()

	return &storage.PartInfo{
		Number:       number,
		Size:         int64(len(data)),
		ETag:         etag,
		LastModified: &now,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = mu
	_ = number
	_ = opts
	return nil, storage.ErrUnsupported
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if mu == nil || mu.UploadID == "" {
		return nil, fmt.Errorf("bee: invalid multipart upload")
	}

	b.st.mp.mu.RLock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.RUnlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, pd := range upload.parts {
		lm := pd.lastModified
		parts = append(parts, &storage.PartInfo{
			Number:       pd.number,
			Size:         int64(len(pd.data)),
			ETag:         pd.etag,
			LastModified: &lm,
		})
	}
	b.st.mp.mu.RUnlock()

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })
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

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	_ = opts
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if mu == nil || mu.UploadID == "" {
		return nil, fmt.Errorf("bee: invalid multipart upload")
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("bee: no parts to complete")
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}

	sortedParts := make([]*storage.PartInfo, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool { return sortedParts[i].Number < sortedParts[j].Number })

	total := 0
	for _, part := range sortedParts {
		pd, exists := upload.parts[part.Number]
		if !exists {
			b.st.mp.mu.Unlock()
			return nil, fmt.Errorf("bee: part %d not found", part.Number)
		}
		total += len(pd.data)
	}

	data := make([]byte, 0, total)
	for _, part := range sortedParts {
		data = append(data, upload.parts[part.Number].data...)
	}

	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	return b.Write(ctx, upload.mu.Key, bytes.NewReader(data), int64(len(data)), upload.contentType, nil)
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = opts
	if err := ctx.Err(); err != nil {
		return err
	}
	if mu == nil || mu.UploadID == "" {
		return fmt.Errorf("bee: invalid multipart upload")
	}

	b.st.mp.mu.Lock()
	defer b.st.mp.mu.Unlock()
	if _, ok := b.st.mp.uploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	return nil
}

func newUploadID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("upload-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
