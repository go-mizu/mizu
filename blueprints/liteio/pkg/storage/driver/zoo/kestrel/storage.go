// Package kestrel implements a high-performance in-memory storage driver
// using a hybrid architecture: sharded Go maps + per-P value allocation.
//
// Architecture (v4):
//   - 256 pointer-allocated sharded Go maps (cache isolation, matches falcon)
//   - Per-P value allocation via sync.Pool chunks (zero lock contention on writes)
//   - Pooled hotRecord structs via sync.Pool (eliminates per-record heap alloc)
//   - Stack-buffer composite keys for reads (allocation-free lookups)
//   - Pointer returns from hotGet (zero-copy field access)
//   - Embedded pending index ops in shard (single lock per write)
//
// DSN format:
//
//	kestrel:///path/to/data?hot_max_bytes=0
package kestrel

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
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

const (
	maxPartNumber = 10000
	maxBuckets    = 10000
	dirPerms      = 0750
	numShards     = 256
	shardMask     = numShards - 1
)

// ---------------------------------------------------------------------------
// Cached time
// ---------------------------------------------------------------------------

var cachedNano atomic.Int64

func init() { cachedNano.Store(time.Now().UnixNano()) }

func fastNow() int64      { return cachedNano.Load() }
func fastTime() time.Time { return time.Unix(0, fastNow()) }

// ---------------------------------------------------------------------------
// Composite key helpers (allocation-free for read path)
// ---------------------------------------------------------------------------

func compositeKey(bucket, key string) string { return bucket + "\x00" + key }

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
// Index operation
// ---------------------------------------------------------------------------

type indexOp struct {
	bucket, key string
	remove      bool
}

// ---------------------------------------------------------------------------
// Record (stored in shard map, value data in sync.Pool chunks)
// ---------------------------------------------------------------------------

type hotRecord struct {
	value   []byte
	ct      string
	size    int64
	created int64
	updated int64
}

var recordPool = sync.Pool{
	New: func() any { return &hotRecord{} },
}

func acquireRecord() *hotRecord {
	return recordPool.Get().(*hotRecord)
}

func releaseRecord(r *hotRecord) {
	if r != nil {
		r.value = nil
		recordPool.Put(r)
	}
}

// ---------------------------------------------------------------------------
// Shard (padded to avoid false sharing on CPU caches)
// ---------------------------------------------------------------------------

type shard struct {
	mu      sync.RWMutex
	data    map[string]*hotRecord
	pending []indexOp // protected by mu (single lock for data+pending)
	_       [16]byte  // cache-line padding
}

// shardForParts computes shard index from bucket+key without allocation.
// For keys > 16 bytes, samples first+last 8 bytes for O(1) hashing.
func shardForParts(bucket, key string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(bucket); i++ {
		h ^= uint32(bucket[i])
		h *= prime32
	}
	h ^= 0 // null separator
	h *= prime32
	if len(key) <= 16 {
		for i := 0; i < len(key); i++ {
			h ^= uint32(key[i])
			h *= prime32
		}
	} else {
		// Sample first 8 + last 8 bytes + length for O(1) shard selection.
		for i := 0; i < 8; i++ {
			h ^= uint32(key[i])
			h *= prime32
		}
		for i := len(key) - 8; i < len(key); i++ {
			h ^= uint32(key[i])
			h *= prime32
		}
		h ^= uint32(len(key))
		h *= prime32
	}
	return h & shardMask
}

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

	hotMaxBytes := int64(0)
	if hm := u.Query().Get("hot_max_bytes"); hm != "" {
		if n, err := strconv.ParseInt(hm, 10, 64); err == nil && n > 0 {
			hotMaxBytes = n
		}
	}

	if err := os.MkdirAll(root, dirPerms); err != nil {
		return nil, fmt.Errorf("kestrel: mkdir root: %w", err)
	}

	// Reduce GC frequency. Bulk data lives in sync.Pool chunks.
	debug.SetGCPercent(800)

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

	// Initialize shards (each separately heap-allocated for cache isolation).
	for i := range st.shards {
		st.shards[i] = &shard{data: make(map[string]*hotRecord, 256)}
	}

	st.bgWg.Add(1)
	go st.indexLoop()

	go func() {
		t := time.NewTicker(500 * time.Microsecond)
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

	shards [numShards]*shard

	hotCount atomic.Int64
	hotBytes atomic.Int64

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

// hotPut stores a pre-populated record.
// Value data should already be allocated (via allocValue).
func (s *store) hotPut(bkt, key string, rec *hotRecord) {
	si := shardForParts(bkt, key)
	sh := s.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bkt, key))

	sh.mu.Lock()
	old, existed := sh.data[ck]
	if !existed {
		ck = compositeKey(bkt, key)
		sh.pending = append(sh.pending, indexOp{bucket: bkt, key: key})
	} else if rec.created == rec.updated {
		rec.created = old.created
	}
	sh.data[ck] = rec
	sh.mu.Unlock()

	if existed {
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(int64(len(rec.value)) - int64(len(old.value)))
		}
		releaseRecord(old)
	} else {
		s.hotCount.Add(1)
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(int64(len(rec.value)))
		}
		s.indexDirty.Store(true)
	}
}

// hotGet retrieves a record pointer (allocation-free lookup).
// The returned pointer is valid as long as the key exists in the map.
func (s *store) hotGet(bkt, key string) (*hotRecord, bool) {
	si := shardForParts(bkt, key)
	sh := s.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bkt, key))

	sh.mu.RLock()
	rec, ok := sh.data[ck]
	sh.mu.RUnlock()
	return rec, ok
}

// hotDelete removes a record from the sharded map.
func (s *store) hotDelete(bkt, key string) bool {
	si := shardForParts(bkt, key)
	sh := s.shards[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bkt, key))

	sh.mu.Lock()
	rec, found := sh.data[ck]
	if found {
		valLen := int64(len(rec.value))
		delete(sh.data, ck)
		s.hotCount.Add(-1)
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(-valLen)
		}
		sh.pending = append(sh.pending, indexOp{bucket: bkt, key: key, remove: true})
	}
	sh.mu.Unlock()

	if found {
		releaseRecord(rec)
		s.indexDirty.Store(true)
	}
	return found
}

// ---------------------------------------------------------------------------
// Background index processing
// ---------------------------------------------------------------------------

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
		sh := s.shards[i]
		sh.mu.Lock()
		pending := sh.pending
		if len(pending) > 0 {
			sh.pending = nil
		}
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
		rec, ok := s.hotGet(bucketName, key)
		if ok {
			objs = append(objs, &storage.Object{
				Bucket:      bucketName,
				Key:         key,
				Size:        rec.size,
				ContentType: rec.ct,
				Created:     time.Unix(0, rec.created),
				Updated:     time.Unix(0, rec.updated),
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
	}

	if force {
		s.syncIndex()
		keys := s.keyIdx.list(name, "")
		for _, key := range keys {
			s.hotDelete(name, key)
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
