// Package kestrel implements a high-performance in-memory storage driver
// using pointer-free hash tables + mmap'd data arena (GC-invisible).
//
// Architecture (v6):
//   - 1024 pointer-allocated shards with Robin Hood open-addressing hash tables
//   - 64 mmap'd arena stripes (arena selected by shard index & 63)
//   - Pointer-free htEntry (48 bytes, all non-pointer fields → GC noscan)
//   - mmap'd data arena: all keys, content types, values stored outside Go heap
//   - Zero GC scanning of bulk data (eliminates 39% CPU overhead from v5)
//   - Hash-only matching: 64-bit FNV-1a — no arena key comparison (2^-64 collision)
//   - Backward-shift deletion (no tombstones, constant performance over time)
//   - Per-shard dirty flags for efficient index batch processing
//   - Combined arena reads: single atomic chunk list load for ct + value
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

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("kestrel", &driver{})
}

const (
	maxPartNumber   = 10000
	maxBuckets      = 10000
	dirPerms        = 0750
	numShards       = 1024
	shardMask       = numShards - 1
	numArenaStripes = 64
	arenaStripeMask = numArenaStripes - 1
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


// ---------------------------------------------------------------------------
// Index operation
// ---------------------------------------------------------------------------

type indexOp struct {
	bucket, key string
	remove      bool
}

// ---------------------------------------------------------------------------
// Shard (padded to avoid false sharing on CPU caches)
// ---------------------------------------------------------------------------

type shard struct {
	mu      sync.RWMutex
	ht      htable    // Robin Hood hash table (pointer-free entries)
	pending []indexOp // protected by mu (single lock for data+pending)
	dirty   atomic.Bool
	_       [8]byte // cache-line padding
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

	// Reduce GC frequency. Bulk data lives in mmap'd arena (GC-invisible).
	debug.SetGCPercent(1600)

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
	for i := range st.arenas {
		st.arenas[i] = newDataArena()
	}

	// Initialize shards (each separately heap-allocated for cache isolation).
	for i := range st.shards {
		st.shards[i] = &shard{ht: newHTable(1024)}
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

	arenas [numArenaStripes]*dataArena
	shards [numShards]*shard

	hotCount atomic.Int64
	hotBytes atomic.Int64

	mu       sync.RWMutex
	storBkts map[string]time.Time

	keyIdx     keyIndex
	mp         *multipartRegistry
	indexDirty atomic.Bool
	indexMu    sync.Mutex // serializes processIndexOps

	stopTick chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	bgWg     sync.WaitGroup
}

var _ storage.Storage = (*store)(nil)

// arenaFor returns the arena stripe for a given shard index.
func (s *store) arenaFor(si uint32) *dataArena { return s.arenas[si&arenaStripeMask] }

// hotResult holds data retrieved from the hash table + arena.
type hotResult struct {
	value   []byte // slice into arena (mmap'd, GC-invisible)
	ct      string // unsafe string into arena
	size    int64
	created int64
	updated int64
}

// hotPut stores key → (value, contentType, size, timestamps).
// Data is copied into the mmap arena. The provided value slice can be reused after return.
func (s *store) hotPut(bkt, key string, value []byte, ct string, size int64, created, updated int64) {
	h := htHash64(bkt, key)
	si := uint32(h>>32) & shardMask
	sh := s.shards[si]

	// Write [compositeKey][contentType][value] to arena stripe.
	arena := s.arenaFor(si)
	ckLen := len(bkt) + 1 + len(key)
	total := ckLen + len(ct) + len(value)
	off, buf := arena.alloc(total)
	n := copy(buf, bkt)
	buf[n] = 0
	n++
	n += copy(buf[n:], key)
	copy(buf[n:], ct)
	copy(buf[n+len(ct):], value)

	entry := htEntry{
		hash:     h,
		arenaOff: off,
		keyLen:   uint16(ckLen),
		ctLen:    uint16(len(ct)),
		valueLen: uint32(len(value)),
		size:     size,
		created:  created,
		updated:  updated,
	}

	sh.mu.Lock()
	old, isUpdate := sh.ht.putOrUpdate(h, entry)
	if isUpdate {
		sh.mu.Unlock()
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(int64(len(value)) - int64(old.valueLen))
		}
		return
	}
	sh.pending = append(sh.pending, indexOp{bucket: bkt, key: key})
	sh.dirty.Store(true)
	sh.mu.Unlock()

	s.hotCount.Add(1)
	if s.hotMaxBytes > 0 {
		s.hotBytes.Add(int64(len(value)))
	}
	s.indexDirty.Store(true)
}

// hotPutDirect inserts an entry where arena data is already written.
// Used by bucket.Write to avoid an extra copy (ReadFull directly into arena).
func (s *store) hotPutDirect(bkt, key string, entry htEntry) {
	si := uint32(entry.hash>>32) & shardMask
	sh := s.shards[si]

	sh.mu.Lock()
	old, isUpdate := sh.ht.putOrUpdate(entry.hash, entry)
	if isUpdate {
		sh.mu.Unlock()
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(int64(entry.valueLen) - int64(old.valueLen))
		}
		return
	}
	sh.pending = append(sh.pending, indexOp{bucket: bkt, key: key})
	sh.dirty.Store(true)
	sh.mu.Unlock()

	s.hotCount.Add(1)
	if s.hotMaxBytes > 0 {
		s.hotBytes.Add(int64(entry.valueLen))
	}
	s.indexDirty.Store(true)
}

// hotGet retrieves a record. Returned slices point into mmap'd memory.
// Uses single arena atomic load for both content-type and value.
func (s *store) hotGet(bkt, key string) (hotResult, bool) {
	h := htHash64(bkt, key)
	si := uint32(h>>32) & shardMask
	sh := s.shards[si]

	sh.mu.RLock()
	e, ok := sh.ht.get(h)
	sh.mu.RUnlock()
	if !ok {
		return hotResult{}, false
	}

	arena := s.arenaFor(si)
	ctOff := e.arenaOff + int64(e.keyLen)
	ct, val := arena.readCtAndValue(ctOff, int(e.ctLen), int(e.valueLen))
	return hotResult{
		value:   val,
		ct:      ct,
		size:    e.size,
		created: e.created,
		updated: e.updated,
	}, true
}

// hotStat retrieves only metadata (no value bytes). Faster than hotGet for Stat.
// Uses dedicated readCt for single atomic load (no value access).
func (s *store) hotStat(bkt, key string) (hotResult, bool) {
	h := htHash64(bkt, key)
	si := uint32(h>>32) & shardMask
	sh := s.shards[si]

	sh.mu.RLock()
	e, ok := sh.ht.get(h)
	sh.mu.RUnlock()
	if !ok {
		return hotResult{}, false
	}

	arena := s.arenaFor(si)
	ctOff := e.arenaOff + int64(e.keyLen)
	return hotResult{
		ct:      arena.readCt(ctOff, int(e.ctLen)),
		size:    e.size,
		created: e.created,
		updated: e.updated,
	}, true
}

// hotDelete removes a record from the sharded hash table.
func (s *store) hotDelete(bkt, key string) bool {
	h := htHash64(bkt, key)
	si := uint32(h>>32) & shardMask
	sh := s.shards[si]

	sh.mu.Lock()
	old, found := sh.ht.remove(h)
	if found {
		s.hotCount.Add(-1)
		if s.hotMaxBytes > 0 {
			s.hotBytes.Add(-int64(old.valueLen))
		}
		sh.pending = append(sh.pending, indexOp{bucket: bkt, key: key, remove: true})
		sh.dirty.Store(true)
	}
	sh.mu.Unlock()

	if found {
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
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	for i := range numShards {
		sh := s.shards[i]
		if !sh.dirty.Load() {
			continue
		}
		sh.mu.Lock()
		pending := sh.pending
		if len(pending) > 0 {
			sh.pending = nil
		}
		sh.dirty.Store(false)
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
		rec, ok := s.hotStat(bucketName, key)
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
	for _, a := range s.arenas {
		a.close()
	}
	return nil
}
