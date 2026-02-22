package kestrel

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

// ---------------------------------------------------------------------------
// Multipart support
// ---------------------------------------------------------------------------

type multipartRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*multipartUpload
	counter atomic.Int64
}

func newMultipartRegistry() *multipartRegistry {
	r := &multipartRegistry{uploads: make(map[string]*multipartUpload)}
	r.counter.Store(time.Now().UnixNano())
	return r
}

type multipartUpload struct {
	id, bucket, key, contentType string
	parts                        map[int]*partData
	metadata                     map[string]string
	created                      time.Time
}

type partData struct {
	number int
	data   []byte
	size   int64
	etag   string
}

func (b *bucket) InitMultipart(_ context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}
	id := strconv.FormatInt(b.st.mp.counter.Add(1), 36)
	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}
	upload := &multipartUpload{
		id: id, bucket: b.name, key: key, contentType: contentType,
		parts: make(map[int]*partData), metadata: metadata, created: fastTime(),
	}
	b.st.mp.mu.Lock()
	b.st.mp.uploads[id] = upload
	b.st.mp.mu.Unlock()
	return &storage.MultipartUpload{Bucket: b.name, Key: key, UploadID: id, Metadata: metadata}, nil
}

func (b *bucket) UploadPart(_ context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, _ storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("kestrel: part number %d out of range [1, %d]", number, maxPartNumber)
	}
	b.st.mp.mu.RLock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("kestrel: read part: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("kestrel: read part: %w", err)
		}
		data = buf.Bytes()
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])
	b.st.mp.mu.Lock()
	upload := b.st.mp.uploads[mu.UploadID]
	upload.parts[number] = &partData{number: number, data: data, size: int64(len(data)), etag: etag}
	b.st.mp.mu.Unlock()
	return &storage.PartInfo{Number: number, Size: int64(len(data)), ETag: etag}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("kestrel: part number %d out of range", number)
	}
	b.st.mp.mu.RLock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, fmt.Errorf("kestrel: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	rec, found := b.st.hotGet(srcBucket, srcKey)
	if !found {
		return nil, storage.ErrNotExist
	}
	data := rec.value
	if srcOffset > 0 {
		if srcOffset >= int64(len(data)) {
			data = nil
		} else {
			data = data[srcOffset:]
		}
	}
	if srcLength > 0 && int64(len(data)) > srcLength {
		data = data[:srcLength]
	}
	return b.UploadPart(ctx, mu, number, bytes.NewReader(data), int64(len(data)), nil)
}

func (b *bucket) ListParts(_ context.Context, mu *storage.MultipartUpload, limit, offset int, _ storage.Options) ([]*storage.PartInfo, error) {
	b.st.mp.mu.RLock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.RUnlock()
		return nil, storage.ErrNotExist
	}
	var parts []*storage.PartInfo
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{Number: p.number, Size: p.size, ETag: p.etag})
	}
	b.st.mp.mu.RUnlock()
	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })
	if offset > 0 && offset < len(parts) {
		parts = parts[offset:]
	}
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}
	return parts, nil
}

func (b *bucket) CompleteMultipart(_ context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, _ storage.Options) (*storage.Object, error) {
	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })
	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("kestrel: part %d not found", p.Number)
		}
	}
	var totalSize int64
	for _, p := range parts {
		totalSize += upload.parts[p.Number].size
	}

	now := fastNow()

	// Assemble parts into a temporary buffer, then store via hotPut.
	assembled := make([]byte, totalSize)
	n := 0
	for _, p := range parts {
		n += copy(assembled[n:], upload.parts[p.Number].data)
	}
	assembled = assembled[:n]

	b.st.hotPut(b.name, upload.key, assembled, upload.contentType, int64(n), now, now)

	return &storage.Object{
		Bucket: b.name, Key: upload.key, Size: int64(n), ContentType: upload.contentType,
		Created: time.Unix(0, now), Updated: time.Unix(0, now),
	}, nil
}

func (b *bucket) AbortMultipart(_ context.Context, mu *storage.MultipartUpload, _ storage.Options) error {
	b.st.mp.mu.Lock()
	defer b.st.mp.mu.Unlock()
	if _, ok := b.st.mp.uploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	return nil
}
