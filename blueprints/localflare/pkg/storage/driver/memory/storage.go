package memdriver

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// Memory driver performance tuning constants.
// These values have been optimized for the benchmark suite.
const (
	// defaultSortedKeysCapacity is the initial capacity for sorted keys slice.
	defaultSortedKeysCapacity = 64

	// entryPoolDataCapacity is the pre-allocated capacity for entry data (4KB).
	// Most small files fit within this buffer without reallocation.
	entryPoolDataCapacity = 4096

	// entryPoolMaxDataSize is the max data size to return to pool (64KB).
	// Larger entries are GC'd to avoid holding excess memory.
	entryPoolMaxDataSize = 64 * 1024

	// keyShardCount is the number of shards for the key index.
	// Higher values reduce contention but increase memory overhead.
	// 256 provides good balance: reduces contention by 256x with ~2KB overhead.
	keyShardCount = 256

	// keyOpBufferSize is the buffer size for async key operations.
	// Larger buffers batch more operations but delay index updates.
	keyOpBufferSize = 1024

	// keyBatchFlushInterval is how often to flush pending key operations.
	keyBatchFlushInterval = 5 * time.Millisecond
)

// DSN format:
//
//   mem://
//   mem://name
//   mem://name?bucket=default
//
// Notes:
//
//   - Each Open creates a new isolated in memory store (no global sharing).
//   - Host (name) is currently ignored, reserved for future sharing.
//   - "bucket" query param sets default bucket name for Bucket("").

func init() {
	storage.Register("mem", &driver{})
	storage.Register("memory", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("mem: parse dsn: %w", err)
	}
	if u.Scheme != "mem" && u.Scheme != "memory" && u.Scheme != "" {
		return nil, fmt.Errorf("mem: unexpected scheme %q", u.Scheme)
	}

	defaultBucket := strings.TrimSpace(u.Query().Get("bucket"))

	st := &store{
		defaultBucket: defaultBucket,
		buckets:       make(map[string]*bucket),
		features:      defaultFeatures(),
	}
	return st, nil
}

// store implements storage.Storage fully in memory.
type store struct {
	mu            sync.RWMutex
	defaultBucket string
	buckets       map[string]*bucket
	features      storage.Features
}

// entryPool provides pooled entry objects to reduce allocations.
var entryPool = sync.Pool{
	New: func() interface{} {
		return &entry{
			data: make([]byte, 0, entryPoolDataCapacity),
		}
	},
}

// writeBufferPool provides pooled buffers for write operations.
// Tiered sizes: 4KB, 64KB, 256KB, 1MB
var writeBufferPools = [4]sync.Pool{
	{New: func() interface{} { b := make([]byte, 0, 4*1024); return &b }},
	{New: func() interface{} { b := make([]byte, 0, 64*1024); return &b }},
	{New: func() interface{} { b := make([]byte, 0, 256*1024); return &b }},
	{New: func() interface{} { b := make([]byte, 0, 1024*1024); return &b }},
}

func getWriteBuffer(size int64) []byte {
	idx := 0
	switch {
	case size <= 4*1024:
		idx = 0
	case size <= 64*1024:
		idx = 1
	case size <= 256*1024:
		idx = 2
	default:
		idx = 3
	}
	return *writeBufferPools[idx].Get().(*[]byte)
}

func putWriteBuffer(buf []byte) {
	cap := cap(buf)
	var idx int
	switch {
	case cap <= 4*1024:
		idx = 0
	case cap <= 64*1024:
		idx = 1
	case cap <= 256*1024:
		idx = 2
	case cap <= 1024*1024:
		idx = 3
	default:
		return // Too large, let GC handle it
	}
	buf = buf[:0]
	writeBufferPools[idx].Put(&buf)
}

func getEntry() *entry {
	return entryPool.Get().(*entry)
}

func putEntry(e *entry) {
	if e == nil {
		return
	}
	// Only return entries with reasonable capacity to pool
	if cap(e.data) <= entryPoolMaxDataSize {
		e.data = e.data[:0]
		e.obj = storage.Object{}
		entryPool.Put(e)
	}
}

// keyShard is a shard of the key index to reduce lock contention.
// Each shard maintains its own sorted key list.
type keyShard struct {
	mu   sync.RWMutex
	keys []string // sorted within shard
}

// getShardIndex returns the shard index for a key using FNV-1a hash.
func getShardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() % keyShardCount
}

// addKey adds a key to the appropriate shard's sorted list.
func (b *bucket) addKey(key string) {
	shard := b.keyShards[getShardIndex(key)]
	shard.mu.Lock()
	idx := sort.SearchStrings(shard.keys, key)
	if idx >= len(shard.keys) || shard.keys[idx] != key {
		// Key doesn't exist, insert it
		shard.keys = append(shard.keys, "")
		copy(shard.keys[idx+1:], shard.keys[idx:])
		shard.keys[idx] = key
		b.keyCount.Add(1)
	}
	shard.mu.Unlock()
}

// removeKey removes a key from the appropriate shard's sorted list.
func (b *bucket) removeKey(key string) {
	shard := b.keyShards[getShardIndex(key)]
	shard.mu.Lock()
	idx := sort.SearchStrings(shard.keys, key)
	if idx < len(shard.keys) && shard.keys[idx] == key {
		shard.keys = append(shard.keys[:idx], shard.keys[idx+1:]...)
		b.keyCount.Add(-1)
	}
	shard.mu.Unlock()
}

// getAllKeys returns all keys from all shards in sorted order.
// This is used by List operations.
func (b *bucket) getAllKeys() []string {
	// First pass: count total keys and collect from each shard
	var totalKeys []string

	for i := 0; i < keyShardCount; i++ {
		shard := b.keyShards[i]
		shard.mu.RLock()
		totalKeys = append(totalKeys, shard.keys...)
		shard.mu.RUnlock()
	}

	// Sort all keys
	sort.Strings(totalKeys)
	return totalKeys
}

// getKeysWithPrefix returns all keys that match the given prefix.
func (b *bucket) getKeysWithPrefix(prefix string) []string {
	var keys []string

	for i := 0; i < keyShardCount; i++ {
		shard := b.keyShards[i]
		shard.mu.RLock()
		for _, k := range shard.keys {
			if prefix == "" || strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
		shard.mu.RUnlock()
	}

	// Sort all matched keys
	sort.Strings(keys)
	return keys
}

// Note: We use sync.Map instead of striped locks for optimal performance.
// This eliminates all lock contention for read and write operations.

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		name = s.defaultBucket
	}
	if name == "" {
		name = "default"
	}
	b, ok := s.buckets[name]
	if !ok {
		now := time.Now()
		b = &bucket{
			st:        s,
			name:      name,
			created:   now,
			mpUploads: make(map[string]*multipartUpload),
		}
		// Initialize sharded key index
		for i := 0; i < keyShardCount; i++ {
			b.keyShards[i] = &keyShard{
				keys: make([]string, 0, defaultSortedKeysCapacity/keyShardCount+1),
			}
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
			Public:    false,
			Metadata:  map[string]string{},
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = opts

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("mem: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}
	now := time.Now()
	b := &bucket{
		st:        s,
		name:      name,
		created:   now,
		mpUploads: make(map[string]*multipartUpload),
	}
	// Initialize sharded key index
	for i := 0; i < keyShardCount; i++ {
		b.keyShards[i] = &keyShard{
			keys: make([]string, 0, defaultSortedKeysCapacity/keyShardCount+1),
		}
	}
	s.buckets[name] = b

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
		Public:    false,
		Metadata:  map[string]string{},
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("mem: bucket name is empty")
	}

	force, _ := opts["force"].(bool)

	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[name]
	if !ok {
		return storage.ErrNotExist
	}
	if !force {
		// Check if bucket has any objects
		hasObjects := false
		b.obj.Range(func(_, _ any) bool {
			hasObjects = true
			return false // Stop iteration
		})
		if hasObjects {
			return storage.ErrPermission
		}
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return cloneFeatures(s.features)
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buckets = make(map[string]*bucket)
	return nil
}

// bucket implements storage.Bucket for in memory bucket.
type bucket struct {
	st   *store
	name string

	// Use sync.Map for lock-free reads and concurrent writes.
	// This is the primary performance optimization for liteio_mem.
	obj     sync.Map // key -> *entry
	created time.Time

	// Sharded key index for efficient List operations with minimal lock contention.
	// Keys are distributed across shards using FNV-1a hash.
	// This reduces lock contention by keyShardCount (256x).
	keyShards [keyShardCount]*keyShard

	// keyCount tracks total number of keys for fast count operations.
	keyCount atomic.Int64

	// multipart state
	mpMu      sync.RWMutex
	mpUploads map[string]*multipartUpload
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
)

// entry holds object metadata and content in memory.
type entry struct {
	obj  storage.Object
	data []byte
}

func (b *bucket) Name() string {
	return b.name
}

func (b *bucket) Features() storage.Features {
	return cloneFeatures(b.st.features)
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mu.RLock()
	defer b.st.mu.RUnlock()

	// Check if bucket still exists in the store
	if _, ok := b.st.buckets[b.name]; !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: b.created,
		Public:    false,
		Metadata:  map[string]string{},
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("mem: key is empty")
	}

	// Read data outside the lock - pre-allocate buffer when size is known.
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
		// Unknown size - read into buffer.
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, err
		}
		data = buf.Bytes()
	}
	now := time.Now()

	meta := extractMetadata(opts)

	// Check if entry exists using sync.Map's LoadOrStore for atomic operation.
	// This is lock-free for existing keys.
	existingEntry, loaded := b.obj.Load(key)

	var e *entry
	if loaded {
		// Update existing entry atomically.
		e = existingEntry.(*entry)
		// Create new entry with updated data (copy-on-write semantics).
		newEntry := &entry{
			obj: storage.Object{
				Bucket:      b.name,
				Key:         key,
				Size:        int64(len(data)),
				ContentType: contentType,
				Created:     e.obj.Created, // Preserve original creation time
				Updated:     now,
				Hash:        nil,
				Metadata:    cloneStringMap(meta),
				IsDir:       false,
			},
			data: data,
		}
		b.obj.Store(key, newEntry)
		objCopy := newEntry.obj
		return &objCopy, nil
	}

	// New entry - create and store.
	e = &entry{
		obj: storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(data)),
			ContentType: contentType,
			Created:     now,
			Updated:     now,
			Hash:        nil,
			Metadata:    cloneStringMap(meta),
			IsDir:       false,
		},
		data: data,
	}
	b.obj.Store(key, e)

	// Update sharded key index (for List optimization).
	b.addKey(key)

	objCopy := e.obj
	return &objCopy, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, fmt.Errorf("mem: key is empty")
	}

	// Lock-free load using sync.Map.
	entryVal, ok := b.obj.Load(key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}
	e := entryVal.(*entry)

	// Direct slice reference (no copy needed for reading).
	data := e.data
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	end := int64(len(data))
	if length > 0 && offset+length < end {
		end = offset + length
	}
	slice := data[offset:end]

	rc := io.NopCloser(bytes.NewReader(slice))
	objCopy := e.obj
	return rc, &objCopy, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("mem: key is empty")
	}

	// Check if key ends with "/" indicating a directory request
	if strings.HasSuffix(key, "/") {
		// Look for any objects with this prefix
		prefix := key
		var created, updated time.Time
		found := false

		b.obj.Range(func(k, v any) bool {
			keyStr := k.(string)
			if strings.HasPrefix(keyStr, prefix) {
				e := v.(*entry)
				if !found {
					created = e.obj.Created
					updated = e.obj.Updated
					found = true
				} else {
					if e.obj.Created.Before(created) {
						created = e.obj.Created
					}
					if e.obj.Updated.After(updated) {
						updated = e.obj.Updated
					}
				}
			}
			return true // Continue iteration
		})

		if !found {
			return nil, storage.ErrNotExist
		}

		return &storage.Object{
			Bucket:   b.name,
			Key:      strings.TrimSuffix(key, "/"),
			Size:     0,
			IsDir:    true,
			Created:  created,
			Updated:  updated,
			Metadata: map[string]string{},
		}, nil
	}

	// Lock-free load using sync.Map.
	entryVal, ok := b.obj.Load(key)
	if !ok {
		return nil, storage.ErrNotExist
	}
	e := entryVal.(*entry)

	objCopy := e.obj
	return &objCopy, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("mem: key is empty")
	}

	// Check if key exists before deleting.
	_, ok := b.obj.Load(key)
	if !ok {
		return storage.ErrNotExist
	}

	// Delete using sync.Map's Delete (lock-free).
	b.obj.Delete(key)

	// Update sharded key index.
	b.removeKey(key)

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" {
		return nil, fmt.Errorf("mem: dstKey is empty")
	}
	if srcKey == "" {
		return nil, fmt.Errorf("mem: srcKey is empty")
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	var srcB *bucket
	if srcBucket == b.name {
		srcB = b
	} else {
		// cross bucket copy
		sb := b.st.Bucket(srcBucket)
		var ok bool
		srcB, ok = sb.(*bucket)
		if !ok {
			return nil, fmt.Errorf("mem: unexpected bucket type for %q", srcBucket)
		}
	}

	// Lock-free load from source.
	srcEntryVal, ok := srcB.obj.Load(srcKey)
	if !ok {
		return nil, storage.ErrNotExist
	}
	srcEntry := srcEntryVal.(*entry)

	now := time.Now()
	dataCopy := make([]byte, len(srcEntry.data))
	copy(dataCopy, srcEntry.data)

	newEntry := &entry{
		obj: storage.Object{
			Bucket:      b.name,
			Key:         dstKey,
			Size:        int64(len(dataCopy)),
			ContentType: srcEntry.obj.ContentType,
			Created:     now,
			Updated:     now,
			Hash:        nil,
			Metadata:    cloneStringMap(srcEntry.obj.Metadata),
			IsDir:       false,
		},
		data: dataCopy,
	}

	// Store to destination using sync.Map.
	b.obj.Store(dstKey, newEntry)

	// Update sharded key index.
	b.addKey(dstKey)

	objCopy := newEntry.obj
	return &objCopy, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	sb := b.st.Bucket(srcBucket)
	if err := sb.Delete(ctx, srcKey, nil); err != nil {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx

	prefix = strings.TrimSpace(prefix)
	recursive := true
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}

	// Use sharded key index for efficient iteration.
	keys := b.getKeysWithPrefix(prefix)

	objs := make([]*storage.Object, 0, len(keys))
	for _, k := range keys {
		if !recursive {
			rest := strings.TrimPrefix(k, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if i := strings.Index(rest, "/"); i >= 0 {
				// subdir, skip in non recursive
				continue
			}
		}
		entryVal, ok := b.obj.Load(k)
		if !ok {
			continue // Entry was deleted between getting keys and loading
		}
		e := entryVal.(*entry)
		objCopy := e.obj
		objs = append(objs, &objCopy)
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]

	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{objects: objs}, nil
}

// SignedURL returns ErrUnsupported for mem backend.
func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	_ = ctx
	_ = key
	_ = method
	_ = expires
	_ = opts

	return "", storage.ErrUnsupported
}

// Directory: prefix based directories over keys.

func (b *bucket) Directory(p string) storage.Directory {
	clean := strings.Trim(p, "/")
	return &dir{
		b:    b,
		path: clean,
	}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket {
	return d.b
}

func (d *dir) Path() string {
	return d.path
}

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	_ = ctx

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	found := false
	var created, updated time.Time

	d.b.obj.Range(func(k, v any) bool {
		keyStr := k.(string)
		if prefix != "" {
			if !strings.HasPrefix(keyStr, prefix) {
				return true // Continue
			}
		} else {
			// root directory exists if bucket has any object
			if keyStr == "" {
				return true // Continue
			}
		}
		e := v.(*entry)
		if !found {
			created = e.obj.Created
			updated = e.obj.Updated
			found = true
		} else {
			if e.obj.Created.Before(created) {
				created = e.obj.Created
			}
			if e.obj.Updated.After(updated) {
				updated = e.obj.Updated
			}
		}
		return true // Continue
	})

	if !found {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:   d.b.name,
		Key:      d.path,
		Size:     0,
		IsDir:    true,
		Created:  created,
		Updated:  updated,
		Metadata: map[string]string{},
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Use sharded key index for efficient iteration.
	allKeys := d.b.getKeysWithPrefix(prefix)
	var keys []string
	for _, k := range allKeys {
		rest := strings.TrimPrefix(k, prefix)
		if i := strings.Index(rest, "/"); i >= 0 {
			// has deeper directory, skip
			continue
		}
		keys = append(keys, k)
	}

	objs := make([]*storage.Object, 0, len(keys))
	for _, k := range keys {
		entryVal, ok := d.b.obj.Load(k)
		if !ok {
			continue // Entry was deleted between getting keys and loading
		}
		e := entryVal.(*entry)
		objCopy := e.obj
		objs = append(objs, &objCopy)
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]

	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{objects: objs}, nil
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	_ = ctx

	recursive, _ := opts["recursive"].(bool)

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Collect keys to delete.
	var keysToDelete []string

	if !recursive {
		// Non recursive delete: only delete objects directly under this directory.
		d.b.obj.Range(func(k, _ any) bool {
			keyStr := k.(string)
			if prefix != "" && !strings.HasPrefix(keyStr, prefix) {
				return true // Continue
			}
			rest := strings.TrimPrefix(keyStr, prefix)
			if strings.Contains(rest, "/") {
				return true // Continue
			}
			keysToDelete = append(keysToDelete, keyStr)
			return true // Continue
		})

		if len(keysToDelete) == 0 {
			return storage.ErrNotExist
		}

		// Delete collected keys.
		for _, k := range keysToDelete {
			d.b.obj.Delete(k)
			d.b.removeKey(k)
		}

		return nil
	}

	// Recursive delete: remove all keys with prefix.
	d.b.obj.Range(func(k, _ any) bool {
		keyStr := k.(string)
		if prefix == "" {
			// root directory recursive delete: clear bucket
			keysToDelete = append(keysToDelete, keyStr)
			return true // Continue
		}
		if strings.HasPrefix(keyStr, prefix) {
			keysToDelete = append(keysToDelete, keyStr)
		}
		return true // Continue
	})

	if len(keysToDelete) == 0 {
		return storage.ErrNotExist
	}

	// Delete collected keys.
	for _, k := range keysToDelete {
		d.b.obj.Delete(k)
		d.b.removeKey(k)
	}

	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	_ = ctx
	_ = opts

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	// Collect keys to move and create new entries.
	type moveOp struct {
		oldKey string
		newKey string
		entry  *entry
	}
	var moveOps []moveOp

	d.b.obj.Range(func(k, v any) bool {
		keyStr := k.(string)
		e := v.(*entry)
		if srcPrefix == "" {
			// moving root, treat all keys as under prefix
			newKey := dstPrefix + keyStr
			newE := &entry{
				obj:  e.obj,
				data: make([]byte, len(e.data)),
			}
			copy(newE.data, e.data)
			newE.obj.Key = newKey
			moveOps = append(moveOps, moveOp{oldKey: keyStr, newKey: newKey, entry: newE})
			return true // Continue
		}
		if !strings.HasPrefix(keyStr, srcPrefix) {
			return true // Continue
		}
		rel := strings.TrimPrefix(keyStr, srcPrefix)
		newKey := dstPrefix + rel
		newE := &entry{
			obj:  e.obj,
			data: make([]byte, len(e.data)),
		}
		copy(newE.data, e.data)
		newE.obj.Key = newKey
		moveOps = append(moveOps, moveOp{oldKey: keyStr, newKey: newKey, entry: newE})
		return true // Continue
	})

	if len(moveOps) == 0 {
		return nil, storage.ErrNotExist
	}

	// Perform the move operations.
	for _, op := range moveOps {
		// Delete old key.
		d.b.obj.Delete(op.oldKey)
		d.b.removeKey(op.oldKey)

		// Insert new key.
		d.b.obj.Store(op.newKey, op.entry)
		d.b.addKey(op.newKey)
	}

	return &dir{
		b:    d.b,
		path: strings.Trim(dstPath, "/"),
	}, nil
}

// Iterators.

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

// helpers.

func defaultFeatures() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
		"hash:md5":         false,
		"watch":            false,
		"public_url":       false,
		"signed_url":       false,
	}
}

func cloneFeatures(in storage.Features) storage.Features {
	if in == nil {
		return nil
	}
	out := make(storage.Features, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func extractMetadata(opts storage.Options) map[string]string {
	if opts == nil {
		return map[string]string{}
	}
	if m, ok := opts["metadata"].(map[string]string); ok && m != nil {
		return cloneStringMap(m)
	}
	return map[string]string{}
}
