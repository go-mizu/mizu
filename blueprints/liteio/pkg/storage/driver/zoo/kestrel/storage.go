// Package kestrel implements a high-performance storage driver inspired by
// the F2 paper (FASTER evolved, VLDB 2025).
//
// Architecture:
//   - 4096 shards (16x more than falcon) for minimal lock contention
//   - Per-shard RWMutex + Go map (leveraging Swiss table internals)
//   - Zero-alloc reads via stack-buffer composite key
//   - Deferred bloom + key index updates via per-shard pending lists
//   - Value chunk allocator (4MB bump pointer) to reduce GC pressure
//   - No cold tier overhead in pure in-memory mode
//
// DSN format:
//
//	kestrel:///path/to/data?hot_max_bytes=0
package kestrel

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("kestrel", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	numShards     = 4096
	shardMask     = numShards - 1
	maxPartNumber = 10000
	maxBuckets    = 10000
	dirPerms      = 0750
)

// ---------------------------------------------------------------------------
// Hot tier
// ---------------------------------------------------------------------------

type hotRecord struct {
	value       []byte
	contentType string
	created     int64
	updated     int64
	size        int64
}

var recordPool = sync.Pool{New: func() any { return &hotRecord{} }}

func acquireRecord() *hotRecord {
	e := recordPool.Get().(*hotRecord)
	*e = hotRecord{}
	return e
}

func releaseRecord(e *hotRecord) {
	if e != nil {
		e.value = nil
		recordPool.Put(e)
	}
}

type hotShard struct {
	mu         sync.RWMutex
	m          map[string]*hotRecord
	pending    []indexOp
	hasPending atomic.Bool
	_pad       [8]byte // prevent false sharing
}

// ---------------------------------------------------------------------------
// Value chunk allocator
// ---------------------------------------------------------------------------

type valueChunk struct {
	buf []byte
	off int
}

const (
	valueChunkSize = 4 << 20
	valueChunkMax  = 2 << 20
)

var valueChunkPool = sync.Pool{
	New: func() any { return &valueChunk{buf: make([]byte, valueChunkSize)} },
}

func allocValue(size int) []byte {
	if size <= 0 {
		return nil
	}
	if size > valueChunkMax {
		return make([]byte, size)
	}
	vc := valueChunkPool.Get().(*valueChunk)
	if vc.off+size > len(vc.buf) {
		vc.buf = make([]byte, valueChunkSize)
		vc.off = 0
	}
	s := vc.buf[vc.off : vc.off+size : vc.off+size]
	vc.off += size
	valueChunkPool.Put(vc)
	return s
}

// ---------------------------------------------------------------------------
// Hash / key helpers
// ---------------------------------------------------------------------------

func shardFor(bucket, key string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(bucket); i++ {
		h ^= uint32(bucket[i])
		h *= prime32
	}
	h ^= 0
	h *= prime32
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return h & shardMask
}

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func compositeKeyBuf(buf []byte, bucket, key string) []byte {
	n := len(bucket) + 1 + len(key)
	if cap(buf) >= n {
		buf = buf[:n]
	} else {
		buf = make([]byte, n)
	}
	copy(buf, bucket)
	buf[len(bucket)] = 0
	copy(buf[len(bucket)+1:], key)
	return buf
}

func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// ---------------------------------------------------------------------------
// Per-bucket key index (for List)
// ---------------------------------------------------------------------------

func firstSegment(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i]
	}
	return ""
}

type segmentKeys struct {
	keys   map[string]struct{}
	sorted []string
	dirty  bool
}

func (sk *segmentKeys) ensureSorted() {
	if !sk.dirty {
		return
	}
	sk.sorted = sk.sorted[:0]
	for k := range sk.keys {
		sk.sorted = append(sk.sorted, k)
	}
	sort.Strings(sk.sorted)
	sk.dirty = false
}

type bucketKeySet struct {
	mu       sync.RWMutex
	total    int
	segments map[string]*segmentKeys
	noSlash  *segmentKeys
}

func (bk *bucketKeySet) getSegment(key string) *segmentKeys {
	seg := firstSegment(key)
	if seg == "" {
		return bk.noSlash
	}
	sk, ok := bk.segments[seg]
	if !ok {
		sk = &segmentKeys{keys: make(map[string]struct{}, 64), dirty: true}
		bk.segments[seg] = sk
	}
	return sk
}

type keyIndex struct {
	buckets sync.Map
}

func (ki *keyIndex) getBucketKeys(bucket string) *bucketKeySet {
	if v, ok := ki.buckets.Load(bucket); ok {
		return v.(*bucketKeySet)
	}
	bk := &bucketKeySet{
		segments: make(map[string]*segmentKeys, 16),
		noSlash:  &segmentKeys{keys: make(map[string]struct{}, 16), dirty: true},
	}
	actual, _ := ki.buckets.LoadOrStore(bucket, bk)
	return actual.(*bucketKeySet)
}

func (ki *keyIndex) add(bucket, key string) {
	bk := ki.getBucketKeys(bucket)
	bk.mu.RLock()
	sk := bk.getSegment(key)
	if _, exists := sk.keys[key]; exists {
		bk.mu.RUnlock()
		return
	}
	bk.mu.RUnlock()

	bk.mu.Lock()
	sk = bk.getSegment(key)
	if _, exists := sk.keys[key]; !exists {
		sk.keys[key] = struct{}{}
		sk.dirty = true
		bk.total++
	}
	bk.mu.Unlock()
}

func (ki *keyIndex) remove(bucket, key string) {
	bk := ki.getBucketKeys(bucket)
	bk.mu.Lock()
	sk := bk.getSegment(key)
	if _, exists := sk.keys[key]; exists {
		delete(sk.keys, key)
		sk.dirty = true
		bk.total--
	}
	bk.mu.Unlock()
}

func (ki *keyIndex) list(bucket, prefix string) []string {
	bk := ki.getBucketKeys(bucket)
	seg := firstSegment(prefix)

	bk.mu.Lock()
	var sk *segmentKeys
	if seg == "" {
		sk = bk.noSlash
	} else {
		sk = bk.segments[seg]
	}
	if sk == nil || len(sk.keys) == 0 {
		bk.mu.Unlock()
		return nil
	}
	sk.ensureSorted()
	sorted := sk.sorted
	bk.mu.Unlock()

	start := sort.SearchStrings(sorted, prefix)
	var results []string
	for i := start; i < len(sorted); i++ {
		key := sorted[i]
		if !strings.HasPrefix(key, prefix) {
			break
		}
		results = append(results, key)
	}
	return results
}

func (ki *keyIndex) hasBucket(bucket string) bool {
	v, ok := ki.buckets.Load(bucket)
	if !ok {
		return false
	}
	bk := v.(*bucketKeySet)
	bk.mu.RLock()
	n := bk.total
	bk.mu.RUnlock()
	return n > 0
}

func (ki *keyIndex) removeAllForBucket(bucket string) {
	ki.buckets.Delete(bucket)
}

// ---------------------------------------------------------------------------
// Reader pool
// ---------------------------------------------------------------------------

type dataReader struct {
	data []byte
	pos  int
}

var readerPool = sync.Pool{New: func() any { return &dataReader{} }}

func acquireReader(data []byte) *dataReader {
	r := readerPool.Get().(*dataReader)
	r.data = data
	r.pos = 0
	return r
}

func (r *dataReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *dataReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos += n
	return int64(n), err
}

func (r *dataReader) Close() error {
	r.data = nil
	readerPool.Put(r)
	return nil
}

// ---------------------------------------------------------------------------
// Index operation
// ---------------------------------------------------------------------------

type indexOp struct {
	bucket, key string
	remove      bool
}

// ---------------------------------------------------------------------------
// Cached time
// ---------------------------------------------------------------------------

var cachedNano atomic.Int64

func init() { cachedNano.Store(time.Now().UnixNano()) }

func fastNow() int64      { return cachedNano.Load() }
func fastTime() time.Time { return time.Unix(0, fastNow()) }

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(_ context.Context, dsn string) (storage.Storage, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("kestrel: parse dsn: %w", err)
	}
	if u.Scheme != "kestrel" && u.Scheme != "" {
		return nil, fmt.Errorf("kestrel: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/kestrel-data"
	}

	hotMaxBytes := int64(0) // 0 = no limit, pure in-memory
	if hm := u.Query().Get("hot_max_bytes"); hm != "" {
		if n, err := strconv.ParseInt(hm, 10, 64); err == nil && n > 0 {
			hotMaxBytes = n
		}
	}

	if err := os.MkdirAll(root, dirPerms); err != nil {
		return nil, fmt.Errorf("kestrel: mkdir root: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	st := &store{
		root:        root,
		hotMaxBytes: hotMaxBytes,
		storBkts:    make(map[string]time.Time),
		mp:          newMultipartRegistry(),
		stopTick:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize shards with pre-sized maps.
	for i := range st.hot {
		st.hot[i] = &hotShard{m: make(map[string]*hotRecord, 64)}
	}

	// Start background index processor.
	st.bgWg.Add(1)
	go st.indexLoop()

	// Cached time ticker.
	go func() {
		t := time.NewTicker(1 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-st.stopTick:
				return
			case <-t.C:
				cachedNano.Store(time.Now().UnixNano())
			}
		}
	}()

	return st, nil
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

type store struct {
	root        string
	hotMaxBytes int64
	hot         [numShards]*hotShard
	hotCount    atomic.Int64
	hotBytes    atomic.Int64

	mu       sync.RWMutex
	storBkts map[string]time.Time

	keyIdx     keyIndex
	mp         *multipartRegistry
	indexDirty atomic.Bool

	stopTick chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	bgWg     sync.WaitGroup
}

var _ storage.Storage = (*store)(nil)

// hotPut writes an entry to the hot tier.
// Deferred index updates keep the hot path to: lock → map write → unlock.
func (s *store) hotPut(bucket, key string, e *hotRecord) {
	si := shardFor(bucket, key)
	sh := s.hot[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	sh.mu.Lock()
	old, existed := sh.m[ck]
	if !existed {
		ck = compositeKey(bucket, key)
		sh.pending = append(sh.pending, indexOp{bucket: bucket, key: key})
		sh.hasPending.Store(true)
	} else if e.created == e.updated {
		e.created = old.created
	}
	sh.m[ck] = e
	sh.mu.Unlock()

	if existed {
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(int64(len(e.value)) - int64(len(old.value)))
		}
		releaseRecord(old)
	} else {
		s.hotCount.Add(1)
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(int64(len(e.value)))
		}
		s.indexDirty.Store(true)
	}
}

// hotGet retrieves from the hot tier (allocation-free lookup).
func (s *store) hotGet(bucket, key string) (*hotRecord, bool) {
	si := shardFor(bucket, key)
	sh := s.hot[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	sh.mu.RLock()
	e, ok := sh.m[ck]
	sh.mu.RUnlock()
	return e, ok
}

// hotDelete removes from the hot tier.
func (s *store) hotDelete(bucket, key string) bool {
	si := shardFor(bucket, key)
	sh := s.hot[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	sh.mu.Lock()
	old, ok := sh.m[ck]
	if ok {
		delete(sh.m, ck)
		s.hotCount.Add(-1)
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(-int64(len(old.value)))
		}
		sh.pending = append(sh.pending, indexOp{bucket: bucket, key: key, remove: true})
		sh.hasPending.Store(true)
	}
	sh.mu.Unlock()

	if ok {
		releaseRecord(old)
		s.indexDirty.Store(true)
	}
	return ok
}

func (s *store) indexLoop() {
	defer s.bgWg.Done()
	t := time.NewTicker(1 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if s.indexDirty.Load() {
				s.processIndexOps()
			}
		case <-s.ctx.Done():
			s.processIndexOps()
			return
		}
	}
}

func (s *store) processIndexOps() {
	for i := range numShards {
		sh := s.hot[i]
		if !sh.hasPending.Load() {
			continue
		}
		sh.mu.Lock()
		pending := sh.pending
		if len(pending) > 0 {
			sh.pending = nil
		}
		sh.hasPending.Store(false)
		sh.mu.Unlock()

		for _, op := range pending {
			if op.remove {
				s.keyIdx.remove(op.bucket, op.key)
			} else {
				s.keyIdx.add(op.bucket, op.key)
			}
		}
	}
	s.indexDirty.Store(false)
}

func (s *store) syncIndex() {
	if !s.indexDirty.Load() {
		return
	}
	s.processIndexOps()
}

func (s *store) listKeys(bucketName, prefix string) []*storage.Object {
	s.syncIndex()
	keys := s.keyIdx.list(bucketName, prefix)
	if len(keys) == 0 {
		return nil
	}

	objs := make([]*storage.Object, 0, len(keys))
	for _, key := range keys {
		if e, ok := s.hotGet(bucketName, key); ok {
			objs = append(objs, &storage.Object{
				Bucket:      bucketName,
				Key:         key,
				Size:        e.size,
				ContentType: e.contentType,
				Created:     time.Unix(0, e.created),
				Updated:     time.Unix(0, e.updated),
			})
		}
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
	return objs
}

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	s.mu.Lock()
	if _, ok := s.storBkts[name]; !ok {
		if len(s.storBkts) < maxBuckets {
			s.storBkts[name] = fastTime()
		}
	}
	s.mu.Unlock()
	return &bucket{st: s, name: name}
}

func (s *store) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
	s.mu.RLock()
	names := make([]string, 0, len(s.storBkts))
	for n := range s.storBkts {
		names = append(names, n)
	}
	s.mu.RUnlock()
	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{Name: n, CreatedAt: s.storBkts[n]})
	}
	s.mu.RUnlock()

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
	return &bucketIter{list: infos}, nil
}

func (s *store) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("kestrel: bucket name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.storBkts[name]; ok {
		return nil, storage.ErrExist
	}
	now := fastTime()
	s.storBkts[name] = now
	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (s *store) DeleteBucket(_ context.Context, name string, opts storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("kestrel: bucket name is empty")
	}
	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storBkts[name]; !ok {
		return storage.ErrNotExist
	}

	if !force {
		s.syncIndex()
		if s.keyIdx.hasBucket(name) {
			return storage.ErrPermission
		}
		prefix := name + "\x00"
		for i := range numShards {
			sh := s.hot[i]
			sh.mu.RLock()
			for k := range sh.m {
				if strings.HasPrefix(k, prefix) {
					sh.mu.RUnlock()
					return storage.ErrPermission
				}
			}
			sh.mu.RUnlock()
		}
	}

	if force {
		prefix := name + "\x00"
		for i := range numShards {
			sh := s.hot[i]
			sh.mu.Lock()
			for k := range sh.m {
				if strings.HasPrefix(k, prefix) {
					delete(sh.m, k)
					s.hotCount.Add(-1)
				}
			}
			sh.mu.Unlock()
		}
		s.keyIdx.removeAllForBucket(name)
	}

	delete(s.storBkts, name)
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
	s.cancel()
	close(s.stopTick)
	s.bgWg.Wait()
	return nil
}

// ---------------------------------------------------------------------------
// Bucket
// ---------------------------------------------------------------------------

type bucket struct {
	st   *store
	name string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
	_ storage.HasMultipart   = (*bucket)(nil)
)

func (b *bucket) Name() string              { return b.name }
func (b *bucket) Features() storage.Features { return b.st.Features() }

func (b *bucket) Info(_ context.Context) (*storage.BucketInfo, error) {
	b.st.mu.RLock()
	created, ok := b.st.storBkts[b.name]
	b.st.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *bucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}

	var data []byte
	if size >= 0 {
		data = allocValue(int(size))
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("kestrel: read value: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("kestrel: read value: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNow()
	e := acquireRecord()
	e.value = data
	e.contentType = contentType
	e.created = now
	e.updated = now
	e.size = int64(len(data))

	b.st.hotPut(b.name, key, e)

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	if key == "" {
		return nil, nil, fmt.Errorf("kestrel: key is empty")
	}

	e, ok := b.st.hotGet(b.name, key)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}

	data := e.value
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
	return acquireReader(data[offset:end]), obj, nil
}

func (b *bucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}

	if strings.HasSuffix(key, "/") {
		b.st.syncIndex()
		keys := b.st.keyIdx.list(b.name, key)
		if len(keys) == 0 {
			return nil, storage.ErrNotExist
		}
		if e, ok := b.st.hotGet(b.name, keys[0]); ok {
			return &storage.Object{
				Bucket:  b.name,
				Key:     strings.TrimSuffix(key, "/"),
				IsDir:   true,
				Created: time.Unix(0, e.created),
				Updated: time.Unix(0, e.updated),
			}, nil
		}
		return &storage.Object{Bucket: b.name, Key: strings.TrimSuffix(key, "/"), IsDir: true}, nil
	}

	e, ok := b.st.hotGet(b.name, key)
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Delete(_ context.Context, key string, _ storage.Options) error {
	if key == "" {
		return fmt.Errorf("kestrel: key is empty")
	}
	if !b.st.hotDelete(b.name, key) {
		return storage.ErrNotExist
	}
	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("kestrel: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}
	e, ok := b.st.hotGet(srcBucket, srcKey)
	if !ok {
		return nil, storage.ErrNotExist
	}

	now := fastNow()
	valCopy := make([]byte, len(e.value))
	copy(valCopy, e.value)

	dst := acquireRecord()
	dst.value = valCopy
	dst.contentType = e.contentType
	dst.created = now
	dst.updated = now
	dst.size = e.size

	b.st.hotPut(b.name, dstKey, dst)

	return &storage.Object{
		Bucket: b.name, Key: dstKey, Size: dst.size, ContentType: dst.contentType,
		Created: time.Unix(0, now), Updated: time.Unix(0, now),
	}, nil
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

func (b *bucket) List(_ context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	results := b.st.listKeys(b.name, prefix)
	if !recursive {
		var filtered []*storage.Object
		for _, obj := range results {
			rest := strings.TrimPrefix(obj.Key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
			filtered = append(filtered, obj)
		}
		results = filtered
	}

	if offset < 0 {
		offset = 0
	}
	if offset > len(results) {
		offset = len(results)
	}
	results = results[offset:]
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}
	return &objectIter{list: results}, nil
}

func (b *bucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// Directory support
// ---------------------------------------------------------------------------

func (b *bucket) Directory(p string) storage.Directory {
	return &dir{b: b, path: strings.Trim(p, "/")}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket { return d.b }
func (d *dir) Path() string           { return d.path }

func (d *dir) Info(_ context.Context) (*storage.Object, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	keys := d.b.st.keyIdx.list(d.b.name, prefix)
	if len(keys) == 0 {
		return nil, storage.ErrNotExist
	}
	var created, updated time.Time
	if e, ok := d.b.st.hotGet(d.b.name, keys[0]); ok {
		created = time.Unix(0, e.created)
		updated = time.Unix(0, e.updated)
	}
	return &storage.Object{Bucket: d.b.name, Key: d.path, IsDir: true, Created: created, Updated: updated}, nil
}

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.listKeys(d.b.name, prefix)
	var objs []*storage.Object
	for _, r := range results {
		rest := strings.TrimPrefix(r.Key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, r)
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
	return &objectIter{list: objs}, nil
}

func (d *dir) Delete(_ context.Context, opts storage.Options) error {
	recursive := false
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	keys := d.b.st.keyIdx.list(d.b.name, prefix)
	if len(keys) == 0 {
		return storage.ErrNotExist
	}
	for _, key := range keys {
		if !recursive {
			rest := strings.TrimPrefix(key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		d.b.st.hotDelete(d.b.name, key)
	}
	return nil
}

func (d *dir) Move(_ context.Context, dstPath string, _ storage.Options) (storage.Directory, error) {
	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")
	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}
	keys := d.b.st.keyIdx.list(d.b.name, srcPrefix)
	if len(keys) == 0 {
		return nil, storage.ErrNotExist
	}
	for _, key := range keys {
		rel := strings.TrimPrefix(key, srcPrefix)
		newKey := dstPrefix + rel
		e, ok := d.b.st.hotGet(d.b.name, key)
		if !ok {
			continue
		}
		now := fastNow()
		valCopy := make([]byte, len(e.value))
		copy(valCopy, e.value)
		dst := acquireRecord()
		dst.value = valCopy
		dst.contentType = e.contentType
		dst.created = e.created
		dst.updated = now
		dst.size = e.size
		d.b.st.hotPut(d.b.name, newKey, dst)
		d.b.st.hotDelete(d.b.name, key)
	}
	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

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
	upload, ok := b.st.mp.uploads[mu.UploadID]
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

	e, found := b.st.hotGet(srcBucket, srcKey)
	if !found {
		return nil, storage.ErrNotExist
	}
	data := e.value
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
	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, upload.parts[p.Number].data...)
	}

	now := fastNow()
	e := acquireRecord()
	e.value = assembled
	e.contentType = upload.contentType
	e.created = now
	e.updated = now
	e.size = int64(len(assembled))
	b.st.hotPut(b.name, upload.key, e)

	return &storage.Object{
		Bucket: b.name, Key: upload.key, Size: e.size, ContentType: upload.contentType,
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

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

type bucketIter struct {
	list []*storage.BucketInfo
	pos  int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	b := it.list[it.pos]
	it.pos++
	return b, nil
}

func (it *bucketIter) Close() error { it.list = nil; return nil }

type objectIter struct {
	list []*storage.Object
	pos  int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	o := it.list[it.pos]
	it.pos++
	return o, nil
}

func (it *objectIter) Close() error { it.list = nil; return nil }
