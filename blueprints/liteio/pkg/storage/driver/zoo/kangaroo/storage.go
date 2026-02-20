// Package kangaroo implements a tiered flash cache storage driver inspired by the
// Kangaroo paper (SOSP 2021, Best Paper). It combines a DRAM LRU (L1), a circular
// append-only log (KLog, L2), and a set-associative page store (KSet, L3) with
// per-page bloom filters and threshold admission control.
//
// DSN format:
//
//	kangaroo:///path/to/data
//	kangaroo:///path/to/data?sync=none&l1_size=10000&klog_mb=64&kset_mb=512&kset_page=4096
package kangaroo

import (
	"bytes"
	"container/list"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
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

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("kangaroo", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultL1Size    = 10000
	defaultKLogMB    = 64
	defaultKSetMB    = 512
	defaultKSetPage  = 4096
	defaultAdmission = 2
	bloomSize        = 256 // bytes per page bloom filter

	dirPerm  = 0750
	filePerm = 0600

	maxPartNumber = 10000
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, opts, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(root, dirPerm); err != nil {
		return nil, fmt.Errorf("kangaroo: mkdir root: %w", err)
	}

	cfg := config{
		root:      root,
		syncMode:  opts.Get("sync"),
		l1Size:    intOpt(opts, "l1_size", defaultL1Size),
		klogBytes: int64(intOpt(opts, "klog_mb", defaultKLogMB)) * 1024 * 1024,
		ksetBytes: int64(intOpt(opts, "kset_mb", defaultKSetMB)) * 1024 * 1024,
		ksetPage:  intOpt(opts, "kset_page", defaultKSetPage),
		admission: intOpt(opts, "admission", defaultAdmission),
	}
	if cfg.syncMode == "" {
		cfg.syncMode = "none"
	}
	if cfg.ksetPage < bloomSize+64 {
		cfg.ksetPage = bloomSize + 64
	}

	st, err := newStore(cfg)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func parseDSN(dsn string) (string, url.Values, error) {
	if dsn == "" {
		return "", nil, errors.New("kangaroo: empty dsn")
	}

	var queryStr string
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		queryStr = dsn[idx+1:]
		dsn = dsn[:idx]
	}

	opts, _ := url.ParseQuery(queryStr)

	if strings.HasPrefix(dsn, "/") {
		return filepath.Clean(dsn), opts, nil
	}

	if strings.HasPrefix(dsn, "kangaroo:") {
		rest := strings.TrimPrefix(dsn, "kangaroo:")
		if strings.HasPrefix(rest, "//") {
			rest = strings.TrimPrefix(rest, "//")
		}
		if rest == "" {
			return "", nil, errors.New("kangaroo: missing path")
		}
		return filepath.Clean(rest), opts, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("kangaroo: parse dsn: %w", err)
	}
	if u.Scheme != "kangaroo" && u.Scheme != "" {
		return "", nil, fmt.Errorf("kangaroo: unsupported scheme %q", u.Scheme)
	}
	p := u.Path
	if u.Host == "." {
		p = "./" + p
	}
	if p == "" {
		return "", nil, errors.New("kangaroo: missing path in dsn")
	}
	return filepath.Clean(p), opts, nil
}

func intOpt(v url.Values, key string, def int) int {
	s := v.Get(key)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type config struct {
	root      string
	syncMode  string
	l1Size    int
	klogBytes int64
	ksetBytes int64
	ksetPage  int
	admission int
}

// ---------------------------------------------------------------------------
// Store (storage.Storage)
// ---------------------------------------------------------------------------

type store struct {
	cfg config

	mu      sync.RWMutex
	buckets map[string]time.Time

	// Three-tier cache
	l1   *dramLRU
	klog *klogFile
	kset *ksetFile

	// Access counter for threshold admission
	accessMu sync.Mutex
	access   map[uint32]uint8

	// Multipart registry (shared across buckets)
	mpMu      sync.Mutex
	mpUploads map[string]*multipartUpload
	mpIDGen   atomic.Int64
}

var _ storage.Storage = (*store)(nil)

func newStore(cfg config) (*store, error) {
	klog, err := newKLog(filepath.Join(cfg.root, "klog.dat"), cfg.klogBytes)
	if err != nil {
		return nil, err
	}

	numPages := int(cfg.ksetBytes / int64(cfg.ksetPage))
	if numPages < 1 {
		numPages = 1
	}

	kset, err := newKSet(filepath.Join(cfg.root, "kset.dat"), cfg.ksetBytes, cfg.ksetPage, numPages)
	if err != nil {
		klog.close()
		return nil, err
	}

	st := &store{
		cfg:       cfg,
		buckets:   make(map[string]time.Time),
		l1:        newDRAMLRU(cfg.l1Size),
		klog:      klog,
		kset:      kset,
		access:    make(map[uint32]uint8),
		mpUploads: make(map[string]*multipartUpload),
	}
	st.mpIDGen.Store(time.Now().UnixNano())

	// Load metadata if present.
	st.loadMeta()

	return st, nil
}

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		s.buckets[name] = time.Now()
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	names := make([]string, 0, len(s.buckets))
	for n := range s.buckets {
		names = append(names, n)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      n,
			CreatedAt: s.buckets[n],
		})
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

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("kangaroo: bucket name required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := time.Now()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("kangaroo: bucket name required")
	}

	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; !ok {
		return storage.ErrNotExist
	}

	if !force {
		// Check if bucket has objects in L1.
		if s.l1.hasBucket(name) {
			return storage.ErrPermission
		}
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
	s.saveMeta()
	s.klog.close()
	s.kset.close()
	return nil
}

// compositeKey builds bucket + "\x00" + key.
func compositeKey(bkt, key string) string {
	return bkt + "\x00" + key
}

// splitCompositeKey splits a composite key back into bucket and key parts.
func splitCompositeKey(ck string) (string, string) {
	idx := strings.IndexByte(ck, 0)
	if idx < 0 {
		return ck, ""
	}
	return ck[:idx], ck[idx+1:]
}

// keyHash returns FNV-1a hash of the composite key.
func keyHash(ck string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(ck))
	return h.Sum32()
}

// incrAccess bumps the access counter and returns the new count.
func (s *store) incrAccess(ck string) uint8 {
	h := keyHash(ck)
	s.accessMu.Lock()
	c := s.access[h] + 1
	s.access[h] = c
	s.accessMu.Unlock()
	return c
}

// clearAccess removes the access counter entry.
func (s *store) clearAccess(ck string) {
	h := keyHash(ck)
	s.accessMu.Lock()
	delete(s.access, h)
	s.accessMu.Unlock()
}

// putToCache inserts an entry into the tiered cache.
//
// Lock ordering: l1.mu (via put) is released before klog.mu/kset.mu are acquired
// in evictToKLog. No cross-tier locks are held simultaneously.
func (s *store) putToCache(ck string, val []byte, ct string, created, updated time.Time) {
	evicted := s.l1.put(ck, val, ct, created, updated)
	// l1.mu is now released (defer in l1.put).
	if evicted != nil {
		s.evictToKLog(evicted)
	}
}

// evictToKLog writes an L1-evicted entry to the KLog. When KLog wraps, entries are
// promoted to KSet if they meet the threshold.
//
// Lock ordering invariant: klog.mu is fully released (via defer in klog.append)
// before kset.mu is acquired (via kset.put). We NEVER hold both locks simultaneously.
// The access counter check uses its own independent lock (accessMu).
func (s *store) evictToKLog(e *lruEntry) {
	// Step 1: Append to klog. klog.mu is acquired+released inside append.
	// Any entries overwritten by the circular wrap are returned for promotion.
	promoted := s.klog.append(e.key, e.value, e.contentType, e.created, e.updated)
	// klog.mu is now released.

	// Step 2: Promote overwritten entries to kset if they meet the admission threshold.
	// Each kset.put acquires kset.mu independently -- no cross-tier locking.
	for _, p := range promoted {
		h := keyHash(p.key)
		s.accessMu.Lock()
		count := s.access[h]
		s.accessMu.Unlock()

		if int(count) >= s.cfg.admission {
			s.kset.put(p.key, p.value, p.contentType, p.created, p.updated)
		}
		// Entry leaves KLog regardless.
	}
}

// lookupAll searches all three tiers. Returns value, contentType, timestamps, and found.
func (s *store) lookupAll(ck string) ([]byte, string, time.Time, time.Time, int64, bool) {
	// L1: DRAM LRU
	if e, ok := s.l1.get(ck); ok {
		return e.value, e.contentType, e.created, e.updated, int64(len(e.value)), true
	}

	// L2: KLog
	if val, ct, created, updated, ok := s.klog.get(ck); ok {
		s.incrAccess(ck)
		return val, ct, created, updated, int64(len(val)), true
	}

	// L3: KSet
	if val, ct, created, updated, ok := s.kset.get(ck); ok {
		s.incrAccess(ck)
		return val, ct, created, updated, int64(len(val)), true
	}

	return nil, "", time.Time{}, time.Time{}, 0, false
}

// deleteAll removes the key from all tiers.
//
// Lock ordering: each tier's lock is acquired and released independently.
// We NEVER hold klog.mu and kset.mu simultaneously.
//
// 1. Clear the access counter FIRST so that any concurrent evictToKLog
//    promotion will fail the admission threshold check and skip kset.put.
// 2. Remove from L1 (l1.mu) -- no cross-tier interaction.
// 3. Remove from klog (klog.mu) -- klog.mu released before step 4.
// 4. Remove from kset (kset.mu) -- independent lock.
func (s *store) deleteAll(ck string) bool {
	// Step 1: Clear access counter first to prevent concurrent promotion
	// from evictToKLog re-inserting this key into kset after we delete it.
	s.clearAccess(ck)

	// Step 2: Remove from L1 (DRAM LRU). l1.mu acquired+released.
	foundL1 := s.l1.remove(ck)

	// Step 3: Remove from klog. klog.mu acquired+released.
	foundKLog := s.klog.remove(ck)

	// Step 4: Remove from kset. kset.mu acquired+released.
	// klog.mu is guaranteed released before this point.
	foundKSet := s.kset.remove(ck)

	return foundL1 || foundKLog || foundKSet
}

// listAll returns all live composite keys with their metadata.
func (s *store) listAll() []entryMeta {
	seen := make(map[string]struct{})
	var results []entryMeta

	// L1
	for _, e := range s.l1.allItems() {
		if _, dup := seen[e.key]; dup {
			continue
		}
		seen[e.key] = struct{}{}
		results = append(results, entryMeta{
			key:         e.key,
			size:        int64(len(e.value)),
			contentType: e.contentType,
			created:     e.created,
			updated:     e.updated,
		})
	}

	// L2
	for _, e := range s.klog.allEntries() {
		if _, dup := seen[e.key]; dup {
			continue
		}
		seen[e.key] = struct{}{}
		results = append(results, e)
	}

	// L3
	for _, e := range s.kset.allEntries() {
		if _, dup := seen[e.key]; dup {
			continue
		}
		seen[e.key] = struct{}{}
		results = append(results, e)
	}

	return results
}

// entryMeta holds metadata for listing.
type entryMeta struct {
	key         string
	size        int64
	contentType string
	created     time.Time
	updated     time.Time
}

// metaFile holds serialized store metadata for persistence across restarts.
type metaFile struct {
	Buckets  map[string]time.Time `json:"buckets"`
	KLogPos  int64                `json:"klog_pos"`
	KLogUsed int64                `json:"klog_used"`
}

func (s *store) loadMeta() {
	data, err := os.ReadFile(filepath.Join(s.cfg.root, "meta.json"))
	if err != nil {
		return
	}
	var m metaFile
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	if m.Buckets != nil {
		s.mu.Lock()
		for k, v := range m.Buckets {
			s.buckets[k] = v
		}
		s.mu.Unlock()
	}
	s.klog.mu.Lock()
	s.klog.writePos = m.KLogPos
	s.klog.used = m.KLogUsed
	s.klog.mu.Unlock()
}

func (s *store) saveMeta() {
	s.mu.RLock()
	bkts := make(map[string]time.Time, len(s.buckets))
	for k, v := range s.buckets {
		bkts[k] = v
	}
	s.mu.RUnlock()

	s.klog.mu.Lock()
	pos := s.klog.writePos
	used := s.klog.used
	s.klog.mu.Unlock()

	m := metaFile{
		Buckets:  bkts,
		KLogPos:  pos,
		KLogUsed: used,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(s.cfg.root, "meta.json"), data, filePerm)
}

// ---------------------------------------------------------------------------
// DRAM LRU (L1)
// ---------------------------------------------------------------------------

type dramLRU struct {
	mu       sync.Mutex
	maxSize  int
	index    map[string]*list.Element
	eviction *list.List
}

type lruEntry struct {
	key         string
	value       []byte
	contentType string
	created     time.Time
	updated     time.Time
}

func newDRAMLRU(maxSize int) *dramLRU {
	return &dramLRU{
		maxSize:  maxSize,
		index:    make(map[string]*list.Element, maxSize),
		eviction: list.New(),
	}
}

// put inserts or updates an entry. Returns the evicted entry or nil.
func (lru *dramLRU) put(ck string, val []byte, ct string, created, updated time.Time) *lruEntry {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	// Copy value to avoid caller mutation.
	valCopy := make([]byte, len(val))
	copy(valCopy, val)

	if elem, ok := lru.index[ck]; ok {
		e := elem.Value.(*lruEntry)
		e.value = valCopy
		e.contentType = ct
		e.updated = updated
		lru.eviction.MoveToFront(elem)
		return nil
	}

	e := &lruEntry{
		key:         ck,
		value:       valCopy,
		contentType: ct,
		created:     created,
		updated:     updated,
	}
	elem := lru.eviction.PushFront(e)
	lru.index[ck] = elem

	// Evict if over capacity.
	if lru.eviction.Len() > lru.maxSize {
		tail := lru.eviction.Back()
		if tail != nil {
			lru.eviction.Remove(tail)
			evicted := tail.Value.(*lruEntry)
			delete(lru.index, evicted.key)
			return evicted
		}
	}
	return nil
}

// get retrieves an entry and moves it to front (LRU touch).
func (lru *dramLRU) get(ck string) (*lruEntry, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	elem, ok := lru.index[ck]
	if !ok {
		return nil, false
	}
	lru.eviction.MoveToFront(elem)
	e := elem.Value.(*lruEntry)
	return e, true
}

// remove removes an entry by key. Returns true if found.
func (lru *dramLRU) remove(ck string) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	elem, ok := lru.index[ck]
	if !ok {
		return false
	}
	lru.eviction.Remove(elem)
	delete(lru.index, ck)
	return true
}

// hasBucket returns true if any entry belongs to the given bucket.
func (lru *dramLRU) hasBucket(bkt string) bool {
	prefix := bkt + "\x00"
	lru.mu.Lock()
	defer lru.mu.Unlock()
	for ck := range lru.index {
		if strings.HasPrefix(ck, prefix) {
			return true
		}
	}
	return false
}

// allItems returns a snapshot of all entries (for listing).
func (lru *dramLRU) allItems() []lruEntry {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	result := make([]lruEntry, 0, len(lru.index))
	for elem := lru.eviction.Front(); elem != nil; elem = elem.Next() {
		e := elem.Value.(*lruEntry)
		result = append(result, *e)
	}
	return result
}

// ---------------------------------------------------------------------------
// KLog (L2) -- Circular Append-Only Log
// ---------------------------------------------------------------------------

type klogEntry struct {
	offset int64
	size   int64
}

type klogFile struct {
	mu       sync.Mutex
	f        *os.File
	capacity int64
	writePos int64
	used     int64

	// In-memory index: compositeKey -> offset+size in the log.
	index map[string]klogEntry

	// Cached entry data for reads (avoids re-reading from file).
	data map[string]*klogCached
}

type klogCached struct {
	value       []byte
	contentType string
	created     time.Time
	updated     time.Time
}

func newKLog(path string, capacity int64) (*klogFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, fmt.Errorf("kangaroo: open klog: %w", err)
	}

	// Ensure file is at least as large as capacity for circular WriteAt.
	info, _ := f.Stat()
	if info.Size() < capacity {
		if err := f.Truncate(capacity); err != nil {
			f.Close()
			return nil, fmt.Errorf("kangaroo: truncate klog: %w", err)
		}
	}

	return &klogFile{
		f:        f,
		capacity: capacity,
		index:    make(map[string]klogEntry),
		data:     make(map[string]*klogCached),
	}, nil
}

// append writes an entry to the circular log. Returns any entries that were overwritten
// and need promotion consideration.
func (kl *klogFile) append(ck string, val []byte, ct string, created, updated time.Time) []promotionCandidate {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	entryBytes := encodeEntry(ck, val, ct, created, updated)
	entrySize := int64(len(entryBytes))

	if entrySize > kl.capacity {
		// Entry too large for klog; skip.
		return nil
	}

	var promoted []promotionCandidate

	// Check if we will wrap around and overwrite existing entries.
	endPos := kl.writePos + entrySize
	if endPos > kl.capacity {
		// Wrap: collect entries in the overwritten region.
		promoted = kl.collectOverwritten(kl.writePos, kl.capacity)
		promoted = append(promoted, kl.collectOverwritten(0, endPos-kl.capacity)...)

		// Write in two parts: end of file + beginning.
		firstPart := kl.capacity - kl.writePos
		kl.f.WriteAt(entryBytes[:firstPart], kl.writePos)
		kl.f.WriteAt(entryBytes[firstPart:], 0)
		kl.writePos = endPos - kl.capacity
	} else {
		if kl.used >= kl.capacity {
			promoted = kl.collectOverwritten(kl.writePos, endPos)
		}
		kl.f.WriteAt(entryBytes, kl.writePos)
		kl.writePos = endPos
		if kl.writePos >= kl.capacity {
			kl.writePos = 0
		}
	}

	kl.used += entrySize
	if kl.used > kl.capacity {
		kl.used = kl.capacity
	}

	// Update index.
	kl.index[ck] = klogEntry{offset: kl.writePos - entrySize, size: entrySize}
	if kl.index[ck].offset < 0 {
		kl.index[ck] = klogEntry{offset: kl.capacity + (kl.writePos - entrySize), size: entrySize}
	}
	kl.data[ck] = &klogCached{
		value:       val,
		contentType: ct,
		created:     created,
		updated:     updated,
	}

	return promoted
}

// collectOverwritten returns promotion candidates for entries in [start, end) that
// are still referenced by the index.
func (kl *klogFile) collectOverwritten(start, end int64) []promotionCandidate {
	var result []promotionCandidate
	for ck, e := range kl.index {
		entryEnd := e.offset + e.size
		// Check overlap with [start, end).
		if e.offset >= start && e.offset < end {
			if cached, ok := kl.data[ck]; ok {
				result = append(result, promotionCandidate{
					key:         ck,
					value:       cached.value,
					contentType: cached.contentType,
					created:     cached.created,
					updated:     cached.updated,
				})
			}
			delete(kl.index, ck)
			delete(kl.data, ck)
		} else if entryEnd > start && entryEnd <= end {
			if cached, ok := kl.data[ck]; ok {
				result = append(result, promotionCandidate{
					key:         ck,
					value:       cached.value,
					contentType: cached.contentType,
					created:     cached.created,
					updated:     cached.updated,
				})
			}
			delete(kl.index, ck)
			delete(kl.data, ck)
		}
	}
	return result
}

// get reads an entry from the in-memory cache.
func (kl *klogFile) get(ck string) ([]byte, string, time.Time, time.Time, bool) {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	cached, ok := kl.data[ck]
	if !ok {
		return nil, "", time.Time{}, time.Time{}, false
	}

	// Return a copy.
	val := make([]byte, len(cached.value))
	copy(val, cached.value)
	return val, cached.contentType, cached.created, cached.updated, true
}

// remove removes an entry from the klog index.
func (kl *klogFile) remove(ck string) bool {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	_, ok := kl.index[ck]
	if !ok {
		return false
	}
	delete(kl.index, ck)
	delete(kl.data, ck)
	return true
}

// allEntries returns metadata for all entries in the klog.
func (kl *klogFile) allEntries() []entryMeta {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	result := make([]entryMeta, 0, len(kl.data))
	for ck, cached := range kl.data {
		result = append(result, entryMeta{
			key:         ck,
			size:        int64(len(cached.value)),
			contentType: cached.contentType,
			created:     cached.created,
			updated:     cached.updated,
		})
	}
	return result
}

func (kl *klogFile) close() {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	if kl.f != nil {
		kl.f.Close()
	}
}

type promotionCandidate struct {
	key         string
	value       []byte
	contentType string
	created     time.Time
	updated     time.Time
}

// ---------------------------------------------------------------------------
// KSet (L3) -- Set-Associative Pages with Bloom Filters
// ---------------------------------------------------------------------------

type ksetFile struct {
	mu       sync.Mutex
	f        *os.File
	capacity int64
	pageSize int
	numPages int
}

func newKSet(path string, capacity int64, pageSize, numPages int) (*ksetFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, fmt.Errorf("kangaroo: open kset: %w", err)
	}

	// Ensure file is at least as large as capacity for page-based random access.
	info, _ := f.Stat()
	if info.Size() < capacity {
		if err := f.Truncate(capacity); err != nil {
			f.Close()
			return nil, fmt.Errorf("kangaroo: truncate kset: %w", err)
		}
	}

	return &ksetFile{
		f:        f,
		capacity: capacity,
		pageSize: pageSize,
		numPages: numPages,
	}, nil
}

// pageOffset returns the byte offset for a given page number.
func (ks *ksetFile) pageOffset(pageNum int) int64 {
	return int64(pageNum) * int64(ks.pageSize)
}

// pageForKey returns the page number for a composite key.
func (ks *ksetFile) pageForKey(ck string) int {
	return int(keyHash(ck) % uint32(ks.numPages))
}

// readPage reads the full page into a buffer.
func (ks *ksetFile) readPage(pageNum int) ([]byte, error) {
	buf := make([]byte, ks.pageSize)
	off := ks.pageOffset(pageNum)
	_, err := ks.f.ReadAt(buf, off)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

// writePage writes the full page buffer back to disk.
func (ks *ksetFile) writePage(pageNum int, buf []byte) error {
	off := ks.pageOffset(pageNum)
	_, err := ks.f.WriteAt(buf, off)
	return err
}

// usablePageSize is the data area excluding the bloom filter.
func (ks *ksetFile) usablePageSize() int {
	return ks.pageSize - bloomSize
}

// put inserts an entry into its assigned KSet page.
func (ks *ksetFile) put(ck string, val []byte, ct string, created, updated time.Time) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	pageNum := ks.pageForKey(ck)
	page, err := ks.readPage(pageNum)
	if err != nil {
		return
	}

	usable := ks.usablePageSize()
	if usable < 4 {
		return
	}

	// Parse used bytes count from the start of the page.
	used := int(binary.LittleEndian.Uint16(page[:2]))
	if used > usable-2 {
		used = 0
	}

	entryBytes := encodeEntry(ck, val, ct, created, updated)
	entrySize := len(entryBytes)

	dataStart := 2 // after usedBytes field
	available := usable - 2 - used

	if entrySize > usable-2 {
		// Entry does not fit in a single page even if empty. Skip.
		return
	}

	// If not enough space, evict the oldest entry (first one in the page).
	for available < entrySize && used > 0 {
		// Remove first entry from the page data area.
		removed := removeFirstEntry(page[dataStart : dataStart+used])
		if removed <= 0 {
			break
		}
		// Shift remaining data.
		copy(page[dataStart:], page[dataStart+removed:dataStart+used])
		used -= removed
		// Zero out freed region.
		for i := dataStart + used; i < dataStart+used+removed; i++ {
			if i < usable {
				page[i] = 0
			}
		}
		available = usable - 2 - used
	}

	if available < entrySize {
		// Still not enough; reset page.
		used = 0
	}

	// Write entry.
	copy(page[dataStart+used:], entryBytes)
	used += entrySize
	binary.LittleEndian.PutUint16(page[:2], uint16(used))

	// Rebuild bloom filter for this page.
	ks.rebuildBloom(page, dataStart, used)

	ks.writePage(pageNum, page)
}

// get reads an entry from its KSet page.
func (ks *ksetFile) get(ck string) ([]byte, string, time.Time, time.Time, bool) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	pageNum := ks.pageForKey(ck)
	page, err := ks.readPage(pageNum)
	if err != nil {
		return nil, "", time.Time{}, time.Time{}, false
	}

	usable := ks.usablePageSize()
	if usable < 4 {
		return nil, "", time.Time{}, time.Time{}, false
	}

	// Check bloom filter first.
	bloom := page[usable:]
	if !bloomMayContain(bloom, ck) {
		return nil, "", time.Time{}, time.Time{}, false
	}

	used := int(binary.LittleEndian.Uint16(page[:2]))
	if used == 0 || used > usable-2 {
		return nil, "", time.Time{}, time.Time{}, false
	}

	// Scan entries in the data area.
	dataStart := 2
	return scanPageForKey(page[dataStart:dataStart+used], ck)
}

// remove marks an entry as deleted in its KSet page.
func (ks *ksetFile) remove(ck string) bool {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	pageNum := ks.pageForKey(ck)
	page, err := ks.readPage(pageNum)
	if err != nil {
		return false
	}

	usable := ks.usablePageSize()
	if usable < 4 {
		return false
	}

	used := int(binary.LittleEndian.Uint16(page[:2]))
	if used == 0 || used > usable-2 {
		return false
	}

	dataStart := 2
	newUsed, found := removeEntryByKey(page[dataStart:dataStart+used], ck)
	if !found {
		return false
	}

	binary.LittleEndian.PutUint16(page[:2], uint16(newUsed))

	// Rebuild bloom.
	ks.rebuildBloom(page, dataStart, newUsed)

	ks.writePage(pageNum, page)
	return true
}

// allEntries scans all pages for entries (for listing).
func (ks *ksetFile) allEntries() []entryMeta {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	var result []entryMeta
	for p := 0; p < ks.numPages; p++ {
		page, err := ks.readPage(p)
		if err != nil {
			continue
		}
		usable := ks.usablePageSize()
		if usable < 4 {
			continue
		}
		used := int(binary.LittleEndian.Uint16(page[:2]))
		if used == 0 || used > usable-2 {
			continue
		}
		dataStart := 2
		entries := parsePageEntries(page[dataStart : dataStart+used])
		result = append(result, entries...)
	}
	return result
}

func (ks *ksetFile) close() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if ks.f != nil {
		ks.f.Close()
	}
}

// rebuildBloom rebuilds the bloom filter for a page.
func (ks *ksetFile) rebuildBloom(page []byte, dataStart, used int) {
	usable := ks.usablePageSize()
	bloom := page[usable:]
	// Clear bloom.
	for i := range bloom {
		bloom[i] = 0
	}
	// Scan entries and add each key.
	data := page[dataStart : dataStart+used]
	offset := 0
	for offset < len(data) {
		ck, advance := readEntryKey(data[offset:])
		if advance <= 0 {
			break
		}
		bloomAdd(bloom, ck)
		offset += advance
	}
}

// ---------------------------------------------------------------------------
// Entry Encoding
// ---------------------------------------------------------------------------

// encodeEntry encodes a cache entry as: keyLen(2B) | key | ctLen(2B) | ct | valLen(8B) | val | created(8B) | updated(8B)
func encodeEntry(ck string, val []byte, ct string, created, updated time.Time) []byte {
	keyBytes := []byte(ck)
	ctBytes := []byte(ct)
	size := 2 + len(keyBytes) + 2 + len(ctBytes) + 8 + len(val) + 8 + 8
	buf := make([]byte, size)
	pos := 0

	binary.LittleEndian.PutUint16(buf[pos:], uint16(len(keyBytes)))
	pos += 2
	copy(buf[pos:], keyBytes)
	pos += len(keyBytes)

	binary.LittleEndian.PutUint16(buf[pos:], uint16(len(ctBytes)))
	pos += 2
	copy(buf[pos:], ctBytes)
	pos += len(ctBytes)

	binary.LittleEndian.PutUint64(buf[pos:], uint64(len(val)))
	pos += 8
	copy(buf[pos:], val)
	pos += len(val)

	binary.LittleEndian.PutUint64(buf[pos:], uint64(created.UnixNano()))
	pos += 8
	binary.LittleEndian.PutUint64(buf[pos:], uint64(updated.UnixNano()))

	return buf
}

// decodeEntry decodes an entry from buf. Returns key, val, ct, created, updated, bytesConsumed.
func decodeEntry(buf []byte) (string, []byte, string, time.Time, time.Time, int) {
	if len(buf) < 2 {
		return "", nil, "", time.Time{}, time.Time{}, 0
	}
	pos := 0

	keyLen := int(binary.LittleEndian.Uint16(buf[pos:]))
	pos += 2
	if pos+keyLen > len(buf) {
		return "", nil, "", time.Time{}, time.Time{}, 0
	}
	key := string(buf[pos : pos+keyLen])
	pos += keyLen

	if pos+2 > len(buf) {
		return "", nil, "", time.Time{}, time.Time{}, 0
	}
	ctLen := int(binary.LittleEndian.Uint16(buf[pos:]))
	pos += 2
	if pos+ctLen > len(buf) {
		return "", nil, "", time.Time{}, time.Time{}, 0
	}
	ct := string(buf[pos : pos+ctLen])
	pos += ctLen

	if pos+8 > len(buf) {
		return "", nil, "", time.Time{}, time.Time{}, 0
	}
	valLen := int(binary.LittleEndian.Uint64(buf[pos:]))
	pos += 8
	if pos+valLen > len(buf) {
		return "", nil, "", time.Time{}, time.Time{}, 0
	}
	val := make([]byte, valLen)
	copy(val, buf[pos:pos+valLen])
	pos += valLen

	if pos+16 > len(buf) {
		return key, val, ct, time.Time{}, time.Time{}, 0
	}
	created := time.Unix(0, int64(binary.LittleEndian.Uint64(buf[pos:])))
	pos += 8
	updated := time.Unix(0, int64(binary.LittleEndian.Uint64(buf[pos:])))
	pos += 8

	return key, val, ct, created, updated, pos
}

// readEntryKey reads just the composite key and returns it along with the total entry size.
func readEntryKey(buf []byte) (string, int) {
	if len(buf) < 2 {
		return "", 0
	}
	pos := 0

	keyLen := int(binary.LittleEndian.Uint16(buf[pos:]))
	pos += 2
	if pos+keyLen > len(buf) {
		return "", 0
	}
	key := string(buf[pos : pos+keyLen])
	pos += keyLen

	if pos+2 > len(buf) {
		return key, 0
	}
	ctLen := int(binary.LittleEndian.Uint16(buf[pos:]))
	pos += 2
	pos += ctLen

	if pos+8 > len(buf) {
		return key, 0
	}
	valLen := int(binary.LittleEndian.Uint64(buf[pos:]))
	pos += 8
	pos += valLen
	pos += 16 // created + updated

	return key, pos
}

// removeFirstEntry returns the size of the first entry in data.
func removeFirstEntry(data []byte) int {
	_, _, _, _, _, consumed := decodeEntry(data)
	return consumed
}

// scanPageForKey scans page data for a matching composite key.
func scanPageForKey(data []byte, ck string) ([]byte, string, time.Time, time.Time, bool) {
	offset := 0
	for offset < len(data) {
		key, val, ct, created, updated, consumed := decodeEntry(data[offset:])
		if consumed <= 0 {
			break
		}
		if key == ck {
			return val, ct, created, updated, true
		}
		offset += consumed
	}
	return nil, "", time.Time{}, time.Time{}, false
}

// removeEntryByKey removes an entry from page data. Returns new used size and whether found.
func removeEntryByKey(data []byte, ck string) (int, bool) {
	offset := 0
	for offset < len(data) {
		key, _, _, _, _, consumed := decodeEntry(data[offset:])
		if consumed <= 0 {
			break
		}
		if key == ck {
			// Remove by shifting.
			remaining := len(data) - offset - consumed
			copy(data[offset:], data[offset+consumed:])
			// Zero out freed area.
			for i := offset + remaining; i < len(data); i++ {
				data[i] = 0
			}
			return len(data) - consumed, true
		}
		offset += consumed
	}
	return len(data), false
}

// parsePageEntries parses all entries in a page data area.
func parsePageEntries(data []byte) []entryMeta {
	var result []entryMeta
	offset := 0
	for offset < len(data) {
		key, val, ct, created, updated, consumed := decodeEntry(data[offset:])
		if consumed <= 0 {
			break
		}
		result = append(result, entryMeta{
			key:         key,
			size:        int64(len(val)),
			contentType: ct,
			created:     created,
			updated:     updated,
		})
		offset += consumed
	}
	return result
}

// ---------------------------------------------------------------------------
// Bloom Filter (256-byte, 2-hash)
// ---------------------------------------------------------------------------

func bloomHash1(key string) uint32 {
	return keyHash(key)
}

func bloomHash2(key string) uint32 {
	h := keyHash(key)
	// Secondary hash via bit mixing.
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = h ^ (h >> 16)
	return h
}

func bloomAdd(bloom []byte, key string) {
	bits := uint32(len(bloom) * 8)
	if bits == 0 {
		return
	}
	h1 := bloomHash1(key) % bits
	h2 := bloomHash2(key) % bits
	bloom[h1/8] |= 1 << (h1 % 8)
	bloom[h2/8] |= 1 << (h2 % 8)
}

func bloomMayContain(bloom []byte, key string) bool {
	bits := uint32(len(bloom) * 8)
	if bits == 0 {
		return true
	}
	h1 := bloomHash1(key) % bits
	h2 := bloomHash2(key) % bits
	return (bloom[h1/8]&(1<<(h1%8))) != 0 && (bloom[h2/8]&(1<<(h2%8))) != 0
}

// ---------------------------------------------------------------------------
// Bucket (storage.Bucket + storage.HasDirectories + storage.HasMultipart)
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

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features {
	return b.st.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mu.RLock()
	created, ok := b.st.buckets[b.name]
	b.st.mu.RUnlock()

	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: created,
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Read value.
	var val []byte
	if size == 0 {
		val = nil
	} else if size > 0 {
		val = make([]byte, size)
		n, readErr := io.ReadFull(src, val)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("kangaroo: read: %w", readErr)
		}
		val = val[:n]
	} else {
		// Unknown size.
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("kangaroo: read: %w", err)
		}
		val = buf.Bytes()
	}

	now := time.Now()
	ck := compositeKey(b.name, key)
	b.st.putToCache(ck, val, contentType, now, now)

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(val)),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, nil, err
	}

	ck := compositeKey(b.name, key)
	val, ct, created, updated, sz, ok := b.st.lookupAll(ck)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        sz,
		ContentType: ct,
		Created:     created,
		Updated:     updated,
	}

	// Apply range.
	data := val
	if offset > 0 {
		if offset > int64(len(data)) {
			offset = int64(len(data))
		}
		data = data[offset:]
	}
	if length > 0 && length < int64(len(data)) {
		data = data[:length]
	}

	return io.NopCloser(bytes.NewReader(data)), obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Check for directory stat.
	if strings.HasSuffix(key, "/") {
		prefix := compositeKey(b.name, key)
		all := b.st.listAll()
		for _, e := range all {
			if strings.HasPrefix(e.key, prefix) {
				return &storage.Object{
					Bucket:  b.name,
					Key:     strings.TrimSuffix(key, "/"),
					IsDir:   true,
					Created: e.created,
					Updated: e.updated,
				}, nil
			}
		}
		return nil, storage.ErrNotExist
	}

	ck := compositeKey(b.name, key)
	_, ct, created, updated, sz, ok := b.st.lookupAll(ck)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        sz,
		ContentType: ct,
		Created:     created,
		Updated:     updated,
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key, err := cleanKey(key)
	if err != nil {
		return err
	}

	recursive := false
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	if recursive {
		prefix := compositeKey(b.name, key)
		all := b.st.listAll()
		found := false
		for _, e := range all {
			if strings.HasPrefix(e.key, prefix) {
				b.st.deleteAll(e.key)
				found = true
			}
		}
		if !found {
			ck := compositeKey(b.name, key)
			if !b.st.deleteAll(ck) {
				return storage.ErrNotExist
			}
		}
		return nil
	}

	ck := compositeKey(b.name, key)
	if !b.st.deleteAll(ck) {
		return storage.ErrNotExist
	}
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dstKey, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}
	srcKey, err = cleanKey(srcKey)
	if err != nil {
		return nil, err
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	srcCK := compositeKey(srcBucket, srcKey)
	val, ct, created, _, _, ok := b.st.lookupAll(srcCK)
	if !ok {
		return nil, storage.ErrNotExist
	}

	now := time.Now()
	dstCK := compositeKey(b.name, dstKey)
	b.st.putToCache(dstCK, val, ct, created, now)

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        int64(len(val)),
		ContentType: ct,
		Created:     created,
		Updated:     now,
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
	srcCK := compositeKey(srcBucket, srcKey)
	b.st.deleteAll(srcCK)
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = cleanPrefix(prefix)

	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	bucketPrefix := compositeKey(b.name, "")
	all := b.st.listAll()

	var objs []*storage.Object
	seen := make(map[string]struct{})
	for _, e := range all {
		if !strings.HasPrefix(e.key, bucketPrefix) {
			continue
		}
		_, objKey := splitCompositeKey(e.key)
		if prefix != "" && !strings.HasPrefix(objKey, prefix) {
			continue
		}
		if !recursive {
			rest := strings.TrimPrefix(objKey, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				// Add directory entry.
				dirPart := prefix
				if dirPart != "" {
					dirPart += "/"
				}
				parts := strings.SplitN(rest, "/", 2)
				dirKey := dirPart + parts[0]
				if _, dup := seen[dirKey]; !dup {
					seen[dirKey] = struct{}{}
					objs = append(objs, &storage.Object{
						Bucket: b.name,
						Key:    dirKey,
						IsDir:  true,
					})
				}
				continue
			}
		}
		if _, dup := seen[objKey]; dup {
			continue
		}
		seen[objKey] = struct{}{}
		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        e.size,
			ContentType: e.contentType,
			Created:     e.created,
			Updated:     e.updated,
		})
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

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

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// Directory Support (storage.HasDirectories)
// ---------------------------------------------------------------------------

func (b *bucket) Directory(path string) storage.Directory {
	return &dir{b: b, path: strings.Trim(path, "/")}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket { return d.b }
func (d *dir) Path() string           { return d.path }

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	bucketPrefix := compositeKey(d.b.name, prefix)
	all := d.b.st.listAll()
	for _, e := range all {
		if strings.HasPrefix(e.key, bucketPrefix) {
			return &storage.Object{
				Bucket:  d.b.name,
				Key:     d.path,
				IsDir:   true,
				Created: e.created,
				Updated: e.updated,
			}, nil
		}
	}
	return nil, storage.ErrNotExist
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	bucketPrefix := compositeKey(d.b.name, prefix)
	all := d.b.st.listAll()

	var objs []*storage.Object
	for _, e := range all {
		if !strings.HasPrefix(e.key, bucketPrefix) {
			continue
		}
		_, objKey := splitCompositeKey(e.key)
		rest := strings.TrimPrefix(objKey, prefix)
		if strings.Contains(rest, "/") {
			continue // Not direct child.
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         objKey,
			Size:        e.size,
			ContentType: e.contentType,
			Created:     e.created,
			Updated:     e.updated,
		})
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

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

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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

	bucketPrefix := compositeKey(d.b.name, prefix)
	all := d.b.st.listAll()
	found := false
	for _, e := range all {
		if !strings.HasPrefix(e.key, bucketPrefix) {
			continue
		}
		if !recursive {
			_, objKey := splitCompositeKey(e.key)
			rest := strings.TrimPrefix(objKey, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		d.b.st.deleteAll(e.key)
		found = true
	}

	if !found {
		return storage.ErrNotExist
	}
	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	srcBucketPrefix := compositeKey(d.b.name, srcPrefix)
	all := d.b.st.listAll()

	found := false
	for _, e := range all {
		if !strings.HasPrefix(e.key, srcBucketPrefix) {
			continue
		}
		found = true

		_, objKey := splitCompositeKey(e.key)
		rel := strings.TrimPrefix(objKey, srcPrefix)
		newKey := dstPrefix + rel

		// Read and re-insert with new key.
		val, ct, created, updated, _, ok := d.b.st.lookupAll(e.key)
		if !ok {
			continue
		}
		newCK := compositeKey(d.b.name, newKey)
		d.b.st.putToCache(newCK, val, ct, created, updated)
		d.b.st.deleteAll(e.key)
	}

	if !found {
		return nil, storage.ErrNotExist
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart Support (storage.HasMultipart)
// ---------------------------------------------------------------------------

type multipartUpload struct {
	id          string
	bucketName  string
	key         string
	contentType string
	parts       map[int]*uploadPart
	created     time.Time
	metadata    map[string]string
}

type uploadPart struct {
	number int
	data   []byte
	size   int64
	etag   string
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	id := strconv.FormatInt(b.st.mpIDGen.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:          id,
		bucketName:  b.name,
		key:         key,
		contentType: contentType,
		parts:       make(map[int]*uploadPart),
		created:     time.Now(),
		metadata:    metadata,
	}

	b.st.mpMu.Lock()
	b.st.mpUploads[id] = upload
	b.st.mpMu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("kangaroo: part number %d out of range [1, %d]", number, maxPartNumber)
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	b.st.mpMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Read part data.
	var data []byte
	if size > 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("kangaroo: read part: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("kangaroo: read part: %w", err)
		}
		data = buf.Bytes()
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.st.mpMu.Lock()
	upload.parts[number] = &uploadPart{
		number: number,
		data:   data,
		size:   int64(len(data)),
		etag:   etag,
	}
	b.st.mpMu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   int64(len(data)),
		ETag:   etag,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("kangaroo: part number %d out of range", number)
	}

	b.st.mpMu.Lock()
	_, ok := b.st.mpUploads[mu.UploadID]
	b.st.mpMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if opts != nil {
		if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
			srcBucket = sb
		}
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("kangaroo: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	srcCK := compositeKey(srcBucket, srcKey)
	val, _, _, _, _, found := b.st.lookupAll(srcCK)
	if !found {
		return nil, storage.ErrNotExist
	}

	// Apply range.
	if srcOffset > 0 {
		if srcOffset > int64(len(val)) {
			srcOffset = int64(len(val))
		}
		val = val[srcOffset:]
	}
	if srcLength > 0 && srcLength < int64(len(val)) {
		val = val[:srcLength]
	}

	return b.UploadPart(ctx, mu, number, bytes.NewReader(val), int64(len(val)), opts)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   p.size,
			ETag:   p.etag,
		})
	}
	b.st.mpMu.Unlock()

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	if offset > 0 && offset < len(parts) {
		parts = parts[offset:]
	}
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mpUploads, mu.UploadID)
	b.st.mpMu.Unlock()

	// Sort parts.
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Verify all parts exist.
	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("kangaroo: part %d not found", p.Number)
		}
	}

	// Assemble value.
	var totalSize int64
	for _, p := range parts {
		totalSize += upload.parts[p.Number].size
	}

	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, upload.parts[p.Number].data...)
	}

	now := time.Now()
	ck := compositeKey(upload.bucketName, upload.key)
	b.st.putToCache(ck, assembled, upload.contentType, now, now)

	return &storage.Object{
		Bucket:      upload.bucketName,
		Key:         upload.key,
		Size:        int64(len(assembled)),
		ContentType: upload.contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.st.mpMu.Lock()
	_, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
		return storage.ErrNotExist
	}
	delete(b.st.mpUploads, mu.UploadID)
	b.st.mpMu.Unlock()

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

func (it *bucketIter) Close() error { return nil }

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

func (it *objectIter) Close() error { return nil }

// ---------------------------------------------------------------------------
// Key Helpers
// ---------------------------------------------------------------------------

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("kangaroo: empty key")
	}
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("kangaroo: empty key")
	}
	// Reject path traversal.
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return key, nil
}

func cleanPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	prefix = strings.TrimPrefix(prefix, "/")
	return prefix
}
