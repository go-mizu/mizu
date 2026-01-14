// Package devnull provides a no-op storage driver for benchmark baseline measurement.
//
// The devnull driver accepts all writes (discarding data), returns empty content
// for reads, and provides instant responses for all operations. This establishes
// the theoretical maximum throughput achievable by the benchmark infrastructure.
//
// DSN format:
//
//	devnull://
//	devnull://bucket-name
package devnull

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// devnull driver performance constants.
// These values are used for simulated responses.
const (
	// defaultContentType is returned for all objects.
	defaultContentType = "application/octet-stream"

	// maxTrackedKeys limits memory usage in devnull mode.
	// Keys beyond this limit are silently dropped from tracking.
	maxTrackedKeys = 100000
)

func init() {
	storage.Register("devnull", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("devnull: parse dsn: %w", err)
	}
	if u.Scheme != "devnull" && u.Scheme != "" {
		return nil, fmt.Errorf("devnull: unexpected scheme %q", u.Scheme)
	}

	defaultBucket := strings.TrimSpace(u.Host)
	if defaultBucket == "" {
		defaultBucket = "default"
	}

	return &store{
		defaultBucket: defaultBucket,
		buckets:       make(map[string]*bucket),
	}, nil
}

// store implements storage.Storage with no actual I/O.
type store struct {
	mu            sync.RWMutex
	defaultBucket string
	buckets       map[string]*bucket
}

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		name = s.defaultBucket
	}

	b, ok := s.buckets[name]
	if !ok {
		b = &bucket{
			st:      s,
			name:    name,
			created: time.Now(),
			keys:    make(map[string]*objectMeta),
		}
		s.buckets[name] = b
	}
	return b
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts

	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.buckets))
	for name := range s.buckets {
		names = append(names, name)
	}
	sort.Strings(names)

	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, name := range names {
		b := s.buckets[name]
		infos = append(infos, &storage.BucketInfo{
			Name:      name,
			CreatedAt: b.created,
		})
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(infos) {
		offset = len(infos)
	}
	infos = infos[offset:]

	if limit > 0 && limit < len(infos) {
		infos = infos[:limit]
	}

	return &bucketIter{buckets: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	_ = ctx
	_ = opts

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("devnull: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := time.Now()
	s.buckets[name] = &bucket{
		st:      s,
		name:    name,
		created: now,
		keys:    make(map[string]*objectMeta),
	}

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; !ok {
		return storage.ErrNotExist
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
	}
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets = make(map[string]*bucket)
	return nil
}

// objectMeta tracks minimal metadata for realistic List behavior.
type objectMeta struct {
	size        int64
	contentType string
	created     time.Time
}

// bucket implements storage.Bucket with no actual I/O.
type bucket struct {
	st       *store
	name     string
	created  time.Time
	mu       sync.RWMutex
	keys     map[string]*objectMeta
	keyCount int64
}

var _ storage.Bucket = (*bucket)(nil)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features {
	return b.st.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	_ = ctx
	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: b.created,
	}, nil
}

// Write consumes input data but discards it entirely.
// Tracks key metadata for realistic List behavior.
func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("devnull: key is empty")
	}

	// Consume all data (discard)
	var written int64
	if size > 0 {
		written, _ = io.CopyN(io.Discard, src, size)
	} else {
		written, _ = io.Copy(io.Discard, src)
	}

	now := time.Now()

	// Track key for List (with limit to prevent memory explosion)
	b.mu.Lock()
	if b.keyCount < maxTrackedKeys {
		if b.keys[key] == nil {
			b.keyCount++
		}
		b.keys[key] = &objectMeta{
			size:        written,
			contentType: contentType,
			created:     now,
		}
	}
	b.mu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        written,
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

// Open returns an empty reader (zero bytes).
// For benchmarks measuring read throughput, this measures infrastructure overhead only.
func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts
	_ = offset
	_ = length

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("devnull: key is empty")
	}

	// Check if key was written (for realistic ErrNotExist behavior)
	b.mu.RLock()
	meta, exists := b.keys[key]
	b.mu.RUnlock()

	if !exists {
		return nil, nil, storage.ErrNotExist
	}

	// Return empty reader - measures infrastructure overhead only
	return io.NopCloser(bytes.NewReader(nil)), &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        meta.size,
		ContentType: meta.contentType,
		Created:     meta.created,
		Updated:     meta.created,
	}, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("devnull: key is empty")
	}

	b.mu.RLock()
	meta, exists := b.keys[key]
	b.mu.RUnlock()

	if !exists {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        meta.size,
		ContentType: meta.contentType,
		Created:     meta.created,
		Updated:     meta.created,
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("devnull: key is empty")
	}

	b.mu.Lock()
	if _, exists := b.keys[key]; exists {
		delete(b.keys, key)
		b.keyCount--
	}
	b.mu.Unlock()

	// devnull ignores not-found errors
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("devnull: key is empty")
	}

	// Simulate instant copy
	now := time.Now()

	b.mu.Lock()
	if b.keyCount < maxTrackedKeys {
		if b.keys[dstKey] == nil {
			b.keyCount++
		}
		b.keys[dstKey] = &objectMeta{
			size:        0,
			contentType: defaultContentType,
			created:     now,
		}
	}
	b.mu.Unlock()

	return &storage.Object{
		Bucket:  b.name,
		Key:     dstKey,
		Size:    0,
		Created: now,
		Updated: now,
	}, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	_ = b.Delete(ctx, srcKey, opts)
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts

	prefix = strings.TrimSpace(prefix)

	b.mu.RLock()
	var keys []string
	for k := range b.keys {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	b.mu.RUnlock()

	sort.Strings(keys)

	if offset < 0 {
		offset = 0
	}
	if offset > len(keys) {
		offset = len(keys)
	}
	keys = keys[offset:]

	if limit > 0 && limit < len(keys) {
		keys = keys[:limit]
	}

	objs := make([]*storage.Object, 0, len(keys))
	b.mu.RLock()
	for _, k := range keys {
		if meta, ok := b.keys[k]; ok {
			objs = append(objs, &storage.Object{
				Bucket:      b.name,
				Key:         k,
				Size:        meta.size,
				ContentType: meta.contentType,
				Created:     meta.created,
				Updated:     meta.created,
			})
		}
	}
	b.mu.RUnlock()

	return &objectIter{objects: objs}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// Multipart support for devnull

var _ storage.HasMultipart = (*bucket)(nil)

type multipartUpload struct {
	key         string
	uploadID    string
	parts       map[int]int64
	contentType string
	created     time.Time
}

var (
	multipartUploads   = make(map[string]*multipartUpload)
	multipartMu        sync.RWMutex
	multipartIDCounter uint64
)

func (b *bucket) InitMultipart(ctx context.Context, key, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = ctx
	_ = opts

	uploadID := fmt.Sprintf("devnull-%d", atomic.AddUint64(&multipartIDCounter, 1))
	now := time.Now()

	multipartMu.Lock()
	multipartUploads[uploadID] = &multipartUpload{
		key:         key,
		uploadID:    uploadID,
		parts:       make(map[int]int64),
		contentType: contentType,
		created:     now,
	}
	multipartMu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: uploadID,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, partNum int, r io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	// Consume data
	var written int64
	if size > 0 {
		written, _ = io.CopyN(io.Discard, r, size)
	} else {
		written, _ = io.Copy(io.Discard, r)
	}

	multipartMu.Lock()
	if up, ok := multipartUploads[mu.UploadID]; ok {
		up.parts[partNum] = written
	}
	multipartMu.Unlock()

	return &storage.PartInfo{
		Number: partNum,
		Size:   written,
		ETag:   fmt.Sprintf("devnull-part-%d", partNum),
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	// devnull simulates instant copy
	multipartMu.Lock()
	if up, ok := multipartUploads[mu.UploadID]; ok {
		up.parts[number] = 0
	}
	multipartMu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   0,
		ETag:   fmt.Sprintf("devnull-copypart-%d", number),
	}, nil
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	multipartMu.RLock()
	up, ok := multipartUploads[mu.UploadID]
	if !ok {
		multipartMu.RUnlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(up.parts))
	for num, size := range up.parts {
		parts = append(parts, &storage.PartInfo{
			Number: num,
			Size:   size,
			ETag:   fmt.Sprintf("devnull-part-%d", num),
		})
	}
	multipartMu.RUnlock()

	// Sort by part number
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

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
	_ = ctx
	_ = opts

	multipartMu.Lock()
	up, ok := multipartUploads[mu.UploadID]
	if ok {
		delete(multipartUploads, mu.UploadID)
	}
	multipartMu.Unlock()

	if !ok {
		return nil, storage.ErrNotExist
	}

	var totalSize int64
	for _, p := range parts {
		totalSize += p.Size
	}

	now := time.Now()

	// Track completed object
	b.mu.Lock()
	if b.keyCount < maxTrackedKeys {
		if b.keys[up.key] == nil {
			b.keyCount++
		}
		b.keys[up.key] = &objectMeta{
			size:        totalSize,
			contentType: up.contentType,
			created:     now,
		}
	}
	b.mu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         up.key,
		Size:        totalSize,
		ContentType: up.contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = ctx
	_ = opts

	multipartMu.Lock()
	delete(multipartUploads, mu.UploadID)
	multipartMu.Unlock()

	return nil
}

// Iterators

type bucketIter struct {
	buckets []*storage.BucketInfo
	index   int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.index >= len(it.buckets) {
		return nil, nil
	}
	b := it.buckets[it.index]
	it.index++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.buckets = nil
	return nil
}

type objectIter struct {
	objects []*storage.Object
	index   int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.index >= len(it.objects) {
		return nil, nil
	}
	o := it.objects[it.index]
	it.index++
	return o, nil
}

func (it *objectIter) Close() error {
	it.objects = nil
	return nil
}
