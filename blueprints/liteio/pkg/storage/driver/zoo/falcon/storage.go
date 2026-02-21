// Package falcon implements a two-tier hash-indexed storage driver inspired by
// the F2 paper (FASTER evolved, VLDB 2025).
//
// Architecture:
//   - Hot tier: sharded in-memory concurrent hash map for recently written/accessed entries
//   - Cold tier: on-disk hash-indexed file with 256-byte aligned slots
//   - Promotion: reading a cold entry copies it to the hot tier
//   - Demotion: when hot tier exceeds capacity, oldest entries flush to cold tier
//
// Optimizations over baseline:
//   - Zero-copy reads: Open() returns pooled reader referencing entry's value slice directly
//   - Bloom filter: lock-free concurrent bloom filter for fast negative lookups
//   - Per-bucket key index: segmented sorted keys for O(matching) List operations
//   - In-memory cold directory: map lookup instead of linear probing for cold tier
//   - Allocation-free lookups: stack buffer + unsafe.String for map lookups
//   - Reader pooling: sync.Pool for dataReader to eliminate GC pressure
//   - Entry pooling: sync.Pool for hotEntry structs
//
// DSN format:
//
//	falcon:///path/to/data?sync=none&hot_size=1048576
package falcon

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
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
	storage.Register("falcon", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	numShards     = 256
	slotSize      = 256
	coldHeaderLen = 64
	maxPartNumber = 10000

	// Slot flags.
	flagEmpty     = 0x00
	flagOccupied  = 0x01
	flagTombstone = 0x02
	flagOverflow  = 0x04

	// Slot field sizes.
	fieldHash    = 8
	fieldKeyLen  = 2
	fieldCTLen   = 2
	fieldValLen  = 8
	fieldCreated = 8
	fieldUpdated = 8
	fieldFlags   = 1

	// Fixed overhead inside a 256B slot (before variable-length key/ct/val).
	slotFixedOverhead = fieldHash + fieldKeyLen + fieldCTLen + fieldValLen + fieldCreated + fieldUpdated + fieldFlags // 37

	defaultHotSize     = 1_048_576
	defaultHotMaxBytes = 0 // 0 = no auto-demote; hot tier grows freely. Cold used only on Close/flushAll.
	defaultColdSlot    = 1 << 22                 // 4M slots (~1 GB cold file) — avoids grow events
	maxColdSlots       = 1 << 24           // 16M slots

	maxBuckets = 10000

	dirPerms  = 0750
	filePerms = 0600
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(_ context.Context, dsn string) (storage.Storage, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("falcon: parse dsn: %w", err)
	}
	if u.Scheme != "falcon" && u.Scheme != "" {
		return nil, fmt.Errorf("falcon: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/falcon-data"
	}

	syncMode := u.Query().Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	hotSize := defaultHotSize
	if hs := u.Query().Get("hot_size"); hs != "" {
		if n, err := strconv.Atoi(hs); err == nil && n > 0 {
			hotSize = n
		}
	}

	if err := os.MkdirAll(root, dirPerms); err != nil {
		return nil, fmt.Errorf("falcon: mkdir root: %w", err)
	}

	cold, err := openColdFile(filepath.Join(root, "cold.dat"))
	if err != nil {
		return nil, err
	}

	over, err := openOverflowFile(filepath.Join(root, "overflow.dat"))
	if err != nil {
		cold.close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	st := &store{
		root:        root,
		syncMode:    syncMode,
		hotSize:     hotSize,
		hotMaxBytes: defaultHotMaxBytes,
		cold:        cold,
		overflow:    over,
		buckets:     make(map[string]time.Time),
		bloom:       newBloomFilter(hotSize),
		coldDir:     newColdDirectory(),
		mp:          newMultipartRegistry(),
		demoteCh: make(chan struct{}, 1),
		stopTick: make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := range st.hot {
		st.hot[i] = &hotShard{m: make(map[string]*hotEntry, 256)}
	}

	// Build cold directory from existing cold file.
	st.buildColdDirectory()

	// Start background goroutines.
	st.bgWg.Add(2)
	go st.demoteLoop()
	go st.indexLoop()

	// Start per-store ticker for cachedNano.
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
// Cached time
// ---------------------------------------------------------------------------

var cachedNano atomic.Int64

func init() {
	cachedNano.Store(time.Now().UnixNano())
}

func fastNow() int64     { return cachedNano.Load() }
func fastTime() time.Time { return time.Unix(0, fastNow()) }

// ---------------------------------------------------------------------------
// FNV helpers (allocation-free)
// ---------------------------------------------------------------------------

// shardForParts computes shard index from bucket+key without allocation.
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
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return h % numShards
}

func fnv1a64Parts(bucket, key string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(bucket); i++ {
		h ^= uint64(bucket[i])
		h *= prime64
	}
	h ^= 0 // null separator
	h *= prime64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
}

func fnv1a64Str(s string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
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

func splitCompositeKey(ck string) (bucket, key string) {
	if i := strings.IndexByte(ck, 0); i >= 0 {
		return ck[:i], ck[i+1:]
	}
	return ck, ""
}

// ---------------------------------------------------------------------------
// Bloom filter (lock-free)
// ---------------------------------------------------------------------------

type bloomFilter struct {
	bits    []atomic.Uint64
	numBits uint64
	numHash int
}

func newBloomFilter(expectedItems int) *bloomFilter {
	if expectedItems < 1024 {
		expectedItems = 1024
	}
	numBits := uint64(expectedItems) * 10
	numBits = (numBits + 63) &^ 63

	return &bloomFilter{
		bits:    make([]atomic.Uint64, numBits/64),
		numBits: numBits,
		numHash: 7,
	}
}

func (bf *bloomFilter) add(bucket, key string) {
	h1, h2 := bloomHash(bucket, key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		bf.bits[bit/64].Or(uint64(1) << (bit % 64))
	}
}

func (bf *bloomFilter) mayContain(bucket, key string) bool {
	h1, h2 := bloomHash(bucket, key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		if bf.bits[bit/64].Load()&(1<<(bit%64)) == 0 {
			return false
		}
	}
	return true
}

func bloomHash(bucket, key string) (uint64, uint64) {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211

	h1 := uint64(offset64)
	for i := 0; i < len(bucket); i++ {
		h1 ^= uint64(bucket[i])
		h1 *= prime64
	}
	h1 ^= 0
	h1 *= prime64
	for i := 0; i < len(key); i++ {
		h1 ^= uint64(key[i])
		h1 *= prime64
	}

	h2 := h1 ^ 0xDEADBEEFCAFEBABE
	for i := 0; i < len(key); i++ {
		h2 ^= uint64(key[i])
		h2 *= prime64
	}
	h2 ^= 0xFF
	h2 *= prime64
	for i := 0; i < len(bucket); i++ {
		h2 ^= uint64(bucket[i])
		h2 *= prime64
	}
	h2 |= 1

	return h1, h2
}

// ---------------------------------------------------------------------------
// Per-bucket key index (for fast List)
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
	buckets sync.Map // bucket name → *bucketKeySet
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
	// Optimistic check: skip write lock if key already tracked.
	bk.mu.RLock()
	sk := bk.getSegment(key)
	if _, exists := sk.keys[key]; exists {
		bk.mu.RUnlock()
		return
	}
	bk.mu.RUnlock()

	// Upgrade to write lock for insertion.
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

// removeAllForBucket removes all keys for a bucket from the index.
func (ki *keyIndex) removeAllForBucket(bucket string) {
	ki.buckets.Delete(bucket)
}

// ---------------------------------------------------------------------------
// In-memory cold directory
// ---------------------------------------------------------------------------

type coldLocation struct {
	slotIndex int64
}

type coldDirectory struct {
	mu      sync.RWMutex
	entries map[string]coldLocation // composite key → slot location
}

func newColdDirectory() *coldDirectory {
	return &coldDirectory{
		entries: make(map[string]coldLocation, 1024),
	}
}

func (cd *coldDirectory) get(ck string) (coldLocation, bool) {
	cd.mu.RLock()
	loc, ok := cd.entries[ck]
	cd.mu.RUnlock()
	return loc, ok
}

func (cd *coldDirectory) put(ck string, loc coldLocation) {
	cd.mu.Lock()
	cd.entries[ck] = loc
	cd.mu.Unlock()
}

func (cd *coldDirectory) remove(ck string) {
	cd.mu.Lock()
	delete(cd.entries, ck)
	cd.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Hot tier
// ---------------------------------------------------------------------------

type hotEntry struct {
	value       []byte
	contentType string
	created     int64
	updated     int64
	size        int64
}

var entryPool = sync.Pool{
	New: func() any { return &hotEntry{} },
}

func acquireEntry() *hotEntry {
	e := entryPool.Get().(*hotEntry)
	*e = hotEntry{}
	return e
}

func releaseEntry(e *hotEntry) {
	if e != nil {
		e.value = nil
		entryPool.Put(e)
	}
}

type hotShard struct {
	mu      sync.RWMutex
	m       map[string]*hotEntry
	pending []indexOp // deferred bloom+keyIndex adds, processed by indexLoop
	_       [24]byte  // padding to avoid false sharing
}

// indexOp is a deferred bloom+keyIndex update, processed by indexLoop.
type indexOp struct {
	bucket, key string
	remove      bool // true = keyIdx.remove, false = bloom.add + keyIdx.add
}

// ---------------------------------------------------------------------------
// Reader pool (zero-copy)
// ---------------------------------------------------------------------------

type dataReader struct {
	data []byte
	pos  int
}

var readerPool = sync.Pool{
	New: func() any { return &dataReader{} },
}

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
// Cold file
// ---------------------------------------------------------------------------
//
// Layout: [header 64B] [slot0 256B] [slot1 256B] ...
//
// Each 256B slot:
//
//	hash(8) | keyLen(2) | key(...) | ctLen(2) | ct(...) | valLen(8) |
//	value(...) or overflowOffset(8)+overflowLen(8) | created(8) | updated(8) | flags(1)
//
// Flags: 0x01=occupied 0x02=tombstone 0x04=overflow

type coldFile struct {
	mu       sync.RWMutex
	f        *os.File
	numSlots int64
	count    int64
}

func openColdFile(path string) (*coldFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return nil, fmt.Errorf("falcon: open cold file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("falcon: stat cold file: %w", err)
	}

	cf := &coldFile{f: f}

	if info.Size() < coldHeaderLen {
		// Initialize header + initial slots.
		cf.numSlots = int64(defaultColdSlot)
		totalSize := coldHeaderLen + cf.numSlots*slotSize
		if err := f.Truncate(totalSize); err != nil {
			f.Close()
			return nil, fmt.Errorf("falcon: truncate cold file: %w", err)
		}
		// Write slot count into header.
		var hdr [coldHeaderLen]byte
		binary.LittleEndian.PutUint64(hdr[0:8], uint64(cf.numSlots))
		if _, err := f.WriteAt(hdr[:], 0); err != nil {
			f.Close()
			return nil, fmt.Errorf("falcon: write cold header: %w", err)
		}
	} else {
		var hdr [coldHeaderLen]byte
		if _, err := f.ReadAt(hdr[:], 0); err != nil {
			f.Close()
			return nil, fmt.Errorf("falcon: read cold header: %w", err)
		}
		cf.numSlots = int64(binary.LittleEndian.Uint64(hdr[0:8]))
		if cf.numSlots == 0 {
			cf.numSlots = int64(defaultColdSlot)
		}

		// Count existing entries.
		cf.count = cf.countOccupied()
	}

	return cf, nil
}

func (cf *coldFile) close() error {
	if cf.f != nil {
		return cf.f.Close()
	}
	return nil
}

func (cf *coldFile) slotOffset(idx int64) int64 {
	return coldHeaderLen + idx*slotSize
}

// readSlot reads a single 256B slot.
func (cf *coldFile) readSlot(idx int64) ([slotSize]byte, error) {
	var buf [slotSize]byte
	_, err := cf.f.ReadAt(buf[:], cf.slotOffset(idx))
	return buf, err
}

// writeSlot writes a single 256B slot.
func (cf *coldFile) writeSlot(idx int64, buf [slotSize]byte) error {
	_, err := cf.f.WriteAt(buf[:], cf.slotOffset(idx))
	return err
}

func (cf *coldFile) countOccupied() int64 {
	var count int64
	var buf [slotSize]byte
	for i := int64(0); i < cf.numSlots; i++ {
		if _, err := cf.f.ReadAt(buf[:], cf.slotOffset(i)); err != nil {
			break
		}
		flags := buf[slotSize-1]
		if flags&flagOccupied != 0 && flags&flagTombstone == 0 {
			count++
		}
	}
	return count
}

// probe finds the slot index for a composite key, or the first empty slot.
// Returns (slotIndex, found).
func (cf *coldFile) probe(ck string, hash uint64) (int64, bool, error) {
	start := int64(hash % uint64(cf.numSlots))
	for i := int64(0); i < cf.numSlots; i++ {
		idx := (start + i) % cf.numSlots
		buf, err := cf.readSlot(idx)
		if err != nil {
			return 0, false, err
		}
		flags := buf[slotSize-1]
		if flags == flagEmpty {
			return idx, false, nil
		}
		if flags&flagTombstone != 0 {
			continue
		}
		if flags&flagOccupied != 0 {
			slotHash := binary.LittleEndian.Uint64(buf[0:8])
			if slotHash == hash {
				kl := int(binary.LittleEndian.Uint16(buf[8:10]))
				if kl <= len(ck) && kl > 0 {
					slotKey := string(buf[10 : 10+kl])
					if slotKey == ck {
						return idx, true, nil
					}
				}
			}
		}
	}
	return 0, false, fmt.Errorf("falcon: cold file full")
}

// encodeSlot creates a 256B slot buffer for the given entry.
var errSlotTooSmall = fmt.Errorf("falcon: entry too large for cold slot")

func (cf *coldFile) encodeSlot(ck string, hash uint64, e *hotEntry, over *overflowFile) ([slotSize]byte, error) {
	var buf [slotSize]byte

	kl := len(ck)
	cl := len(e.contentType)

	// Minimum space: hash(8) + keyLen(2) + key + ctLen(2) + ct + valLen(8) + created(8) + updated(8) + flags(1) = 37 + key + ct
	// Plus at least 16 bytes for overflow pointer (offset 8 + length 8) if value doesn't fit inline.
	minSpace := 10 + kl + 2 + cl + fieldValLen + fieldCreated + fieldUpdated + fieldFlags
	if minSpace > slotSize {
		return buf, errSlotTooSmall
	}

	binary.LittleEndian.PutUint64(buf[0:8], hash)

	binary.LittleEndian.PutUint16(buf[8:10], uint16(kl))
	pos := 10
	copy(buf[pos:], ck)
	pos += kl

	binary.LittleEndian.PutUint16(buf[pos:pos+2], uint16(cl))
	pos += 2
	copy(buf[pos:], e.contentType)
	pos += cl

	// Calculate available space for inline value.
	// Need: valLen(8) + created(8) + updated(8) + flags(1) = 25 bytes at the end.
	tailFixed := fieldValLen + fieldCreated + fieldUpdated + fieldFlags // 25
	availableForValue := slotSize - pos - tailFixed

	binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(e.size))
	pos += 8

	if int(e.size) <= availableForValue {
		// Inline value.
		copy(buf[pos:], e.value)
		pos += int(e.size)
	} else {
		// Overflow: write value to overflow file and store offset+length.
		off, err := over.write(e.value)
		if err != nil {
			return buf, err
		}
		binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(off))
		pos += 8
		binary.LittleEndian.PutUint64(buf[pos:pos+8], uint64(e.size))
		pos += 8
		buf[slotSize-1] = flagOccupied | flagOverflow
		binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated-fieldCreated:slotSize-1-fieldUpdated], uint64(e.created))
		binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated:slotSize-1], uint64(e.updated))
		return buf, nil
	}

	// Write timestamps at known positions from the end.
	binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated-fieldCreated:slotSize-1-fieldUpdated], uint64(e.created))
	binary.LittleEndian.PutUint64(buf[slotSize-1-fieldUpdated:slotSize-1], uint64(e.updated))
	buf[slotSize-1] = flagOccupied

	return buf, nil
}

// decodeSlot extracts entry data from a 256B slot.
func (cf *coldFile) decodeSlot(buf [slotSize]byte, over *overflowFile) (ck string, e *hotEntry, err error) {
	flags := buf[slotSize-1]
	if flags&flagOccupied == 0 || flags&flagTombstone != 0 {
		return "", nil, fmt.Errorf("falcon: slot not occupied")
	}

	kl := int(binary.LittleEndian.Uint16(buf[8:10]))
	pos := 10
	ck = string(buf[pos : pos+kl])
	pos += kl

	cl := int(binary.LittleEndian.Uint16(buf[pos : pos+2]))
	pos += 2
	ct := string(buf[pos : pos+cl])
	pos += cl

	valLen := int64(binary.LittleEndian.Uint64(buf[pos : pos+8]))
	pos += 8

	created := int64(binary.LittleEndian.Uint64(buf[slotSize-1-fieldUpdated-fieldCreated : slotSize-1-fieldUpdated]))
	updated := int64(binary.LittleEndian.Uint64(buf[slotSize-1-fieldUpdated : slotSize-1]))

	var value []byte
	if flags&flagOverflow != 0 {
		off := int64(binary.LittleEndian.Uint64(buf[pos : pos+8]))
		pos += 8
		overLen := int64(binary.LittleEndian.Uint64(buf[pos : pos+8]))
		value, err = over.read(off, overLen)
		if err != nil {
			return "", nil, err
		}
	} else {
		value = make([]byte, valLen)
		copy(value, buf[pos:pos+int(valLen)])
	}

	e = acquireEntry()
	e.value = value
	e.contentType = ct
	e.created = created
	e.updated = updated
	e.size = valLen
	return ck, e, nil
}

// put writes an entry to the cold file. Returns the slot index where it was written.
func (cf *coldFile) put(ck string, hash uint64, e *hotEntry, over *overflowFile) (int64, error) {
	// Check load factor and grow if needed.
	if float64(cf.count+1)/float64(cf.numSlots) > 0.7 {
		if err := cf.grow(over); err != nil {
			return 0, err
		}
	}

	idx, found, err := cf.probe(ck, hash)
	if err != nil {
		return 0, err
	}

	buf, err := cf.encodeSlot(ck, hash, e, over)
	if err != nil {
		return 0, err
	}

	if err := cf.writeSlot(idx, buf); err != nil {
		return 0, err
	}

	if !found {
		cf.count++
	}
	return idx, nil
}

// get reads an entry from the cold file.
func (cf *coldFile) get(ck string, hash uint64, over *overflowFile) (*hotEntry, bool, error) {
	idx, found, err := cf.probe(ck, hash)
	if err != nil || !found {
		return nil, false, err
	}

	buf, err := cf.readSlot(idx)
	if err != nil {
		return nil, false, err
	}

	_, e, err := cf.decodeSlot(buf, over)
	if err != nil {
		return nil, false, err
	}

	return e, true, nil
}

// getAt reads an entry from a known slot index (used with cold directory).
func (cf *coldFile) getAt(idx int64, over *overflowFile) (string, *hotEntry, error) {
	buf, err := cf.readSlot(idx)
	if err != nil {
		return "", nil, err
	}
	return cf.decodeSlot(buf, over)
}

// markTombstone marks a cold slot as deleted.
func (cf *coldFile) markTombstone(ck string, hash uint64) error {
	idx, found, err := cf.probe(ck, hash)
	if err != nil || !found {
		return err
	}

	buf, err := cf.readSlot(idx)
	if err != nil {
		return err
	}

	buf[slotSize-1] = flagTombstone
	if err := cf.writeSlot(idx, buf); err != nil {
		return err
	}
	cf.count--
	return nil
}

// markTombstoneAt marks a cold slot at a known index as deleted (used with cold directory).
func (cf *coldFile) markTombstoneAt(idx int64) error {
	buf, err := cf.readSlot(idx)
	if err != nil {
		return err
	}
	buf[slotSize-1] = flagTombstone
	if err := cf.writeSlot(idx, buf); err != nil {
		return err
	}
	cf.count--
	return nil
}

// grow doubles the cold file capacity and rehashes all entries.
func (cf *coldFile) grow(over *overflowFile) error {
	oldSlots := cf.numSlots
	newSlots := oldSlots * 2

	if newSlots > maxColdSlots {
		return fmt.Errorf("falcon: cold file would exceed max capacity (%d slots)", maxColdSlots)
	}

	// Read all existing entries.
	type entry struct {
		ck   string
		hash uint64
		buf  [slotSize]byte
	}
	var entries []entry

	for i := int64(0); i < oldSlots; i++ {
		buf, err := cf.readSlot(i)
		if err != nil {
			continue
		}
		flags := buf[slotSize-1]
		if flags&flagOccupied == 0 || flags&flagTombstone != 0 {
			continue
		}

		kl := int(binary.LittleEndian.Uint16(buf[8:10]))
		ck := string(buf[10 : 10+kl])
		hash := binary.LittleEndian.Uint64(buf[0:8])

		entries = append(entries, entry{ck: ck, hash: hash, buf: buf})
	}

	// Expand file.
	cf.numSlots = newSlots
	totalSize := coldHeaderLen + cf.numSlots*slotSize
	if err := cf.f.Truncate(totalSize); err != nil {
		return fmt.Errorf("falcon: grow cold file: %w", err)
	}

	// Update header.
	var hdr [coldHeaderLen]byte
	binary.LittleEndian.PutUint64(hdr[0:8], uint64(cf.numSlots))
	if _, err := cf.f.WriteAt(hdr[:], 0); err != nil {
		return fmt.Errorf("falcon: write cold header: %w", err)
	}

	// Zero out all slots.
	zero := make([]byte, slotSize)
	for i := int64(0); i < newSlots; i++ {
		if _, err := cf.f.WriteAt(zero, cf.slotOffset(i)); err != nil {
			return fmt.Errorf("falcon: zero slot: %w", err)
		}
	}

	// Reinsert entries.
	cf.count = 0
	for _, ent := range entries {
		start := int64(ent.hash % uint64(cf.numSlots))
		for j := int64(0); j < cf.numSlots; j++ {
			idx := (start + j) % cf.numSlots
			var check [slotSize]byte
			if _, err := cf.f.ReadAt(check[:], cf.slotOffset(idx)); err != nil {
				continue
			}
			if check[slotSize-1] == flagEmpty {
				if err := cf.writeSlot(idx, ent.buf); err != nil {
					return err
				}
				cf.count++
				break
			}
		}
	}

	return nil
}

// allEntries returns all live (ck, slotIndex) pairs from cold storage.
// Used only during startup to build the cold directory.
func (cf *coldFile) allSlotEntries() []coldSlotEntry {
	var result []coldSlotEntry
	for i := int64(0); i < cf.numSlots; i++ {
		buf, err := cf.readSlot(i)
		if err != nil {
			continue
		}
		flags := buf[slotSize-1]
		if flags&flagOccupied == 0 || flags&flagTombstone != 0 {
			continue
		}
		kl := int(binary.LittleEndian.Uint16(buf[8:10]))
		ck := string(buf[10 : 10+kl])
		result = append(result, coldSlotEntry{ck: ck, slotIndex: i})
	}
	return result
}

type coldSlotEntry struct {
	ck        string
	slotIndex int64
}

// ---------------------------------------------------------------------------
// Overflow file
// ---------------------------------------------------------------------------

type overflowFile struct {
	mu   sync.Mutex
	f    *os.File
	tail int64
}

func openOverflowFile(path string) (*overflowFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return nil, fmt.Errorf("falcon: open overflow file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("falcon: stat overflow file: %w", err)
	}

	return &overflowFile{f: f, tail: info.Size()}, nil
}

func (of *overflowFile) write(data []byte) (int64, error) {
	of.mu.Lock()
	defer of.mu.Unlock()

	off := of.tail
	if _, err := of.f.WriteAt(data, off); err != nil {
		return 0, fmt.Errorf("falcon: write overflow: %w", err)
	}
	of.tail += int64(len(data))
	return off, nil
}

func (of *overflowFile) read(off, length int64) ([]byte, error) {
	buf := make([]byte, length)
	if _, err := of.f.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("falcon: read overflow: %w", err)
	}
	return buf, nil
}

func (of *overflowFile) close() error {
	if of.f != nil {
		return of.f.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Store (storage.Storage)
// ---------------------------------------------------------------------------

type store struct {
	root        string
	syncMode    string
	hotSize     int
	hotMaxBytes int64
	cold        *coldFile
	overflow    *overflowFile
	hot         [numShards]*hotShard
	hotCount    atomic.Int64
	hotBytes    atomic.Int64 // total value bytes in hot tier

	mu      sync.RWMutex
	buckets map[string]time.Time

	// New: bloom filter for fast negative lookups.
	bloom *bloomFilter

	// New: per-bucket key index for fast List.
	keyIdx keyIndex

	// New: in-memory cold directory for O(1) cold lookups.
	coldDir *coldDirectory

	mp *multipartRegistry

	// Epoch counter for safe concurrent demotion.
	epoch   atomic.Int64
	flushMu sync.Mutex

	// Background demote goroutine channel.
	demoteCh chan struct{}

	// Async index: dirty flag signals indexLoop to sweep per-shard pending lists.
	indexDirty atomic.Bool

	// Per-store stoppable ticker for cachedNano.
	stopTick chan struct{}

	// Context for stopping background goroutines.
	ctx    context.Context
	cancel context.CancelFunc
	bgWg   sync.WaitGroup // tracks background goroutines (demoteLoop, indexLoop)
}

var _ storage.Storage = (*store)(nil)

// buildColdDirectory scans the cold file once at startup to populate the
// in-memory cold directory, key index, and bloom filter.
func (s *store) buildColdDirectory() {
	entries := s.cold.allSlotEntries()
	for _, e := range entries {
		s.coldDir.put(e.ck, coldLocation{slotIndex: e.slotIndex})
		bucket, key := splitCompositeKey(e.ck)
		if bucket != "" && key != "" {
			s.keyIdx.add(bucket, key)
			s.bloom.add(bucket, key)
		}
	}
}

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		if len(s.buckets) < maxBuckets {
			s.buckets[name] = fastTime()
		}
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
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

func (s *store) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("falcon: bucket name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	now := fastTime()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(_ context.Context, name string, opts storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("falcon: bucket name is empty")
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
		// Flush any pending index updates before checking.
		s.syncIndex()
		// Check via key index first (fast path).
		if s.keyIdx.hasBucket(name) {
			return storage.ErrPermission
		}
		// Fallback: check hot shards directly.
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

	// If force, delete all keys for this bucket from hot tier.
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
	// Stop background goroutines and wait for them to finish.
	s.cancel()
	close(s.stopTick)
	s.bgWg.Wait()

	// Flush hot tier to cold.
	s.flushAll()

	if s.syncMode != "none" {
		s.cold.mu.Lock()
		s.cold.f.Sync()
		s.cold.mu.Unlock()
		s.overflow.mu.Lock()
		s.overflow.f.Sync()
		s.overflow.mu.Unlock()
	}

	err1 := s.cold.close()
	err2 := s.overflow.close()
	if err1 != nil {
		return err1
	}
	return err2
}

// hotPut writes an entry to the hot tier.
// Bloom filter and key index updates are deferred to per-shard pending lists
// and processed asynchronously by indexLoop. This keeps the write hot path
// to just: shard lock → map write → unlock (no bloom/keyIndex overhead).
func (s *store) hotPut(bucket, key string, e *hotEntry) {
	si := shardForParts(bucket, key)
	sh := s.hot[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	sh.mu.Lock()
	old, existed := sh.m[ck]
	if !existed {
		ck = compositeKey(bucket, key)
		// Defer bloom+keyIndex update — zero overhead since we hold the lock.
		sh.pending = append(sh.pending, indexOp{bucket: bucket, key: key})
	} else if e.created == e.updated {
		// Preserve created time when updating an existing key.
		e.created = old.created
	}
	sh.m[ck] = e
	sh.mu.Unlock()

	newValBytes := int64(len(e.value))
	if existed {
		oldBytes := int64(len(old.value))
		s.hotBytes.Add(newValBytes - oldBytes)
		releaseEntry(old)
	} else {
		s.hotCount.Add(1)
		s.hotBytes.Add(newValBytes)
		s.indexDirty.Store(true)
	}

	// Signal background demote goroutine if hot tier exceeds soft limit.
	if s.hotMaxBytes > 0 && s.hotBytes.Load() > s.hotMaxBytes {
		select {
		case s.demoteCh <- struct{}{}:
		default:
		}
	}
}

// hotGet retrieves an entry from the hot tier (allocation-free lookup).
func (s *store) hotGet(bucket, key string) (*hotEntry, bool) {
	si := shardForParts(bucket, key)
	sh := s.hot[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	sh.mu.RLock()
	e, ok := sh.m[ck]
	sh.mu.RUnlock()
	return e, ok
}

// hotDelete removes an entry from the hot tier.
// keyIdx.remove is deferred to per-shard pending list (same as hotPut).
func (s *store) hotDelete(bucket, key string) bool {
	si := shardForParts(bucket, key)
	sh := s.hot[si]

	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	sh.mu.Lock()
	old, ok := sh.m[ck]
	if ok {
		delete(sh.m, ck)
		s.hotCount.Add(-1)
		s.hotBytes.Add(-int64(len(old.value)))
		sh.pending = append(sh.pending, indexOp{bucket: bucket, key: key, remove: true})
	}
	sh.mu.Unlock()

	if ok {
		releaseEntry(old)
		s.indexDirty.Store(true)
	}
	return ok
}

// coldGet retrieves an entry from the cold tier and promotes it to hot.
func (s *store) coldGet(bucket, key string) (*hotEntry, bool) {
	// Check bloom filter first.
	if !s.bloom.mayContain(bucket, key) {
		return nil, false
	}

	// Stack-buffer compositeKey for directory lookup.
	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	// Check in-memory cold directory first.
	loc, found := s.coldDir.get(ck)
	if found {
		s.cold.mu.RLock()
		_, e, err := s.cold.getAt(loc.slotIndex, s.overflow)
		s.cold.mu.RUnlock()
		if err == nil && e != nil {
			// Promote to hot tier.
			s.hotPut(bucket, key, e)
			return e, true
		}
	}

	// Fallback: probe cold file directly.
	heapCK := compositeKey(bucket, key)
	hash := fnv1a64Str(heapCK)
	s.cold.mu.RLock()
	e, ok, err := s.cold.get(heapCK, hash, s.overflow)
	s.cold.mu.RUnlock()

	if err != nil || !ok {
		return nil, false
	}

	// Promote to hot tier.
	s.hotPut(bucket, key, e)
	return e, true
}

// coldStat retrieves metadata from the cold tier without promotion.
func (s *store) coldStat(bucket, key string) (*hotEntry, bool) {
	// Check bloom filter first.
	if !s.bloom.mayContain(bucket, key) {
		return nil, false
	}

	// Stack-buffer compositeKey for directory lookup.
	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], bucket, key))

	// Check in-memory cold directory first.
	loc, found := s.coldDir.get(ck)
	if found {
		s.cold.mu.RLock()
		_, e, err := s.cold.getAt(loc.slotIndex, s.overflow)
		s.cold.mu.RUnlock()
		if err == nil && e != nil {
			return e, true
		}
	}

	// Fallback: probe cold file directly.
	heapCK := compositeKey(bucket, key)
	hash := fnv1a64Str(heapCK)
	s.cold.mu.RLock()
	e, ok, err := s.cold.get(heapCK, hash, s.overflow)
	s.cold.mu.RUnlock()

	if err != nil || !ok {
		return nil, false
	}
	return e, true
}

// demote evicts entries from the hot tier to cold until hotBytes drops below half the limit.
// Per-shard three-phase approach:
//   Phase A: RLock(shard) — snapshot entries to evict (no deletion)
//   Phase B: Lock(cold) — write snapshots to cold file + cold directory
//   Phase C: Lock(shard) — delete from hot (pointer compare to skip concurrent updates)
// This ensures no visibility gap (entries exist in BOTH hot and cold during B+C),
// and minimizes cold.mu hold time (one shard batch at a time, not all shards).
func (s *store) demote() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.epoch.Add(1)

	currentBytes := s.hotBytes.Load()
	targetBytes := s.hotMaxBytes / 2
	if currentBytes <= targetBytes {
		return
	}
	bytesToEvict := currentBytes - targetBytes

	type evictItem struct {
		ck   string
		hash uint64
		e    *hotEntry
	}

	var evictedBytes int64
	for i := range numShards {
		if evictedBytes >= bytesToEvict {
			break
		}
		sh := s.hot[i]

		// Phase A: Snapshot entries from this shard (RLock — doesn't block writers).
		sh.mu.RLock()
		var batch []evictItem
		for ck, e := range sh.m {
			batch = append(batch, evictItem{ck: ck, hash: fnv1a64Str(ck), e: e})
			evictedBytes += int64(len(e.value))
			if evictedBytes >= bytesToEvict {
				break
			}
		}
		sh.mu.RUnlock()

		if len(batch) == 0 {
			continue
		}

		// Phase B: Write to cold (brief Lock — one shard's batch only).
		s.cold.mu.Lock()
		for _, item := range batch {
			slotIdx, err := s.cold.put(item.ck, item.hash, item.e, s.overflow)
			if err != nil {
				continue
			}
			s.coldDir.put(item.ck, coldLocation{slotIndex: slotIdx})
		}
		s.cold.mu.Unlock()

		// Phase C: Remove from hot (only if entry pointer unchanged).
		sh.mu.Lock()
		for _, item := range batch {
			if cur, ok := sh.m[item.ck]; ok && cur == item.e {
				delete(sh.m, item.ck)
				valBytes := int64(len(item.e.value))
				s.hotCount.Add(-1)
				s.hotBytes.Add(-valBytes)
			}
		}
		sh.mu.Unlock()
	}

	if s.syncMode == "full" {
		s.cold.f.Sync()
		s.overflow.mu.Lock()
		s.overflow.f.Sync()
		s.overflow.mu.Unlock()
	}
}

// demoteLoop runs in a dedicated goroutine, processing demote signals.
func (s *store) demoteLoop() {
	defer s.bgWg.Done()
	for {
		select {
		case <-s.demoteCh:
			s.demote()
		case <-s.ctx.Done():
			return
		}
	}
}

// indexLoop runs in a dedicated goroutine, periodically sweeping per-shard
// pending lists and applying bloom+keyIndex updates in batches.
// This decouples index maintenance from the write hot path.
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
			// Final drain — ensure all pending ops are processed before shutdown.
			s.processIndexOps()
			return
		}
	}
}

// processIndexOps sweeps all shards and processes their pending bloom+keyIndex ops.
func (s *store) processIndexOps() {
	for i := range numShards {
		sh := s.hot[i]
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
				s.bloom.add(op.bucket, op.key)
				s.keyIdx.add(op.bucket, op.key)
			}
		}
	}
	s.indexDirty.Store(false)
}

// syncIndex ensures all pending index ops are flushed before a read operation
// that depends on the key index (List, DeleteBucket, directory Stat).
func (s *store) syncIndex() {
	if !s.indexDirty.Load() {
		return
	}
	s.processIndexOps()
}

// flushAll writes all hot tier entries to cold.
func (s *store) flushAll() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.cold.mu.Lock()
	defer s.cold.mu.Unlock()

	for i := range numShards {
		sh := s.hot[i]
		sh.mu.Lock()
		for ck, e := range sh.m {
			hash := fnv1a64Str(ck)
			slotIdx, err := s.cold.put(ck, hash, e, s.overflow)
			if err == nil {
				s.coldDir.put(ck, coldLocation{slotIndex: slotIdx})
			}
		}
		sh.mu.Unlock()
	}
}

// listKeys returns all keys (from both tiers) matching bucket and prefix.
// Uses per-bucket key index for O(matching) instead of scanning cold file.
func (s *store) listKeys(bucketName, prefix string) []*storage.Object {
	s.syncIndex()
	keys := s.keyIdx.list(bucketName, prefix)
	if len(keys) == 0 {
		return nil
	}

	objs := make([]*storage.Object, 0, len(keys))
	for _, key := range keys {
		e, ok := s.hotGet(bucketName, key)
		if !ok {
			e, ok = s.coldStat(bucketName, key)
			if !ok {
				continue
			}
		}
		objs = append(objs, &storage.Object{
			Bucket:      bucketName,
			Key:         key,
			Size:        e.size,
			ContentType: e.contentType,
			Created:     time.Unix(0, e.created),
			Updated:     time.Unix(0, e.updated),
		})
	}

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
	return objs
}


// ---------------------------------------------------------------------------
// Bucket (storage.Bucket + HasDirectories + HasMultipart)
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

func (b *bucket) Features() storage.Features { return b.st.Features() }

func (b *bucket) Info(_ context.Context) (*storage.BucketInfo, error) {
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

func (b *bucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	if key == "" {
		return nil, fmt.Errorf("falcon: key is empty")
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("falcon: read value: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("falcon: read value: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNow()

	e := acquireEntry()
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
		return nil, nil, fmt.Errorf("falcon: key is empty")
	}

	// Hot tier first (allocation-free lookup).
	e, ok := b.st.hotGet(b.name, key)
	if !ok {
		// Cold tier with promotion.
		e, ok = b.st.coldGet(b.name, key)
		if !ok {
			return nil, nil, storage.ErrNotExist
		}
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}

	// Zero-copy: reference entry's value slice directly.
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
		return nil, fmt.Errorf("falcon: key is empty")
	}

	// Check for directory stat via key index (no cold scan).
	if strings.HasSuffix(key, "/") {
		b.st.syncIndex()
		keys := b.st.keyIdx.list(b.name, key)
		if len(keys) == 0 {
			return nil, storage.ErrNotExist
		}
		// Get first key's timestamps.
		if e, ok := b.st.hotGet(b.name, keys[0]); ok {
			return &storage.Object{
				Bucket:  b.name,
				Key:     strings.TrimSuffix(key, "/"),
				IsDir:   true,
				Created: time.Unix(0, e.created),
				Updated: time.Unix(0, e.updated),
			}, nil
		}
		if e, ok := b.st.coldStat(b.name, keys[0]); ok {
			return &storage.Object{
				Bucket:  b.name,
				Key:     strings.TrimSuffix(key, "/"),
				IsDir:   true,
				Created: time.Unix(0, e.created),
				Updated: time.Unix(0, e.updated),
			}, nil
		}
		return &storage.Object{
			Bucket: b.name,
			Key:    strings.TrimSuffix(key, "/"),
			IsDir:  true,
		}, nil
	}

	// Hot tier first (allocation-free lookup).
	e, ok := b.st.hotGet(b.name, key)
	if !ok {
		// Cold tier stat (no promotion).
		e, ok = b.st.coldStat(b.name, key)
		if !ok {
			return nil, storage.ErrNotExist
		}
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
		return fmt.Errorf("falcon: key is empty")
	}

	// Hot-only delete when possible (no cold probe if found in hot).
	if b.st.hotDelete(b.name, key) {
		// Key was in hot. Only check cold if bloom filter says it might exist there.
		if !b.st.bloom.mayContain(b.name, key) {
			return nil // Definitely not in cold.
		}
		// Stack-buffer compositeKey (no heap allocation).
		var buf [256]byte
		ck := unsafeString(compositeKeyBuf(buf[:0], b.name, key))
		if loc, found := b.st.coldDir.get(ck); found {
			b.st.cold.mu.Lock()
			b.st.cold.markTombstoneAt(loc.slotIndex)
			b.st.cold.mu.Unlock()
			// Need heap-allocated key for coldDir.remove (map key).
			b.st.coldDir.remove(compositeKey(b.name, key))
		}
		return nil
	}

	// Not in hot. Check cold via bloom filter first.
	if !b.st.bloom.mayContain(b.name, key) {
		return storage.ErrNotExist
	}

	// Stack-buffer for cold directory lookup.
	var buf [256]byte
	ck := unsafeString(compositeKeyBuf(buf[:0], b.name, key))

	// Check cold directory.
	if loc, found := b.st.coldDir.get(ck); found {
		b.st.cold.mu.Lock()
		b.st.cold.markTombstoneAt(loc.slotIndex)
		b.st.cold.mu.Unlock()
		heapCK := compositeKey(b.name, key)
		b.st.coldDir.remove(heapCK)
		b.st.keyIdx.remove(b.name, key)
		return nil
	}

	// Fallback: probe cold file.
	heapCK := compositeKey(b.name, key)
	hash := fnv1a64Str(heapCK)
	b.st.cold.mu.Lock()
	coldErr := b.st.cold.markTombstone(heapCK, hash)
	b.st.cold.mu.Unlock()

	if coldErr != nil {
		return storage.ErrNotExist
	}

	b.st.keyIdx.remove(b.name, key)
	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("falcon: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	e, ok := b.st.hotGet(srcBucket, srcKey)
	if !ok {
		e, ok = b.st.coldGet(srcBucket, srcKey)
		if !ok {
			return nil, storage.ErrNotExist
		}
	}

	now := fastNow()

	// Copy value bytes.
	valCopy := make([]byte, len(e.value))
	copy(valCopy, e.value)

	dst := acquireEntry()
	dst.value = valCopy
	dst.contentType = e.contentType
	dst.created = now
	dst.updated = now
	dst.size = e.size

	b.st.hotPut(b.name, dstKey, dst)

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        dst.size,
		ContentType: dst.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
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
	// Use key index instead of cold file scan.
	keys := d.b.st.keyIdx.list(d.b.name, prefix)
	if len(keys) == 0 {
		return nil, storage.ErrNotExist
	}
	// Get first key's timestamps.
	var created, updated time.Time
	if e, ok := d.b.st.hotGet(d.b.name, keys[0]); ok {
		created = time.Unix(0, e.created)
		updated = time.Unix(0, e.updated)
	} else if e, ok := d.b.st.coldStat(d.b.name, keys[0]); ok {
		created = time.Unix(0, e.created)
		updated = time.Unix(0, e.updated)
	}
	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: created,
		Updated: updated,
	}, nil
}

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := d.b.st.listKeys(d.b.name, prefix)

	// Filter to direct children only.
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

	// Use key index instead of cold file scan.
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

		ck := compositeKey(d.b.name, key)
		if loc, found := d.b.st.coldDir.get(ck); found {
			d.b.st.cold.mu.Lock()
			d.b.st.cold.markTombstoneAt(loc.slotIndex)
			d.b.st.cold.mu.Unlock()
			d.b.st.coldDir.remove(ck)
		}
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

	// Use key index instead of cold file scan.
	keys := d.b.st.keyIdx.list(d.b.name, srcPrefix)
	if len(keys) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, key := range keys {
		rel := strings.TrimPrefix(key, srcPrefix)
		newKey := dstPrefix + rel

		e, ok := d.b.st.hotGet(d.b.name, key)
		if !ok {
			e, ok = d.b.st.coldGet(d.b.name, key)
		}
		if !ok {
			continue
		}

		now := fastNow()
		valCopy := make([]byte, len(e.value))
		copy(valCopy, e.value)

		dst := acquireEntry()
		dst.value = valCopy
		dst.contentType = e.contentType
		dst.created = e.created
		dst.updated = now
		dst.size = e.size

		d.b.st.hotPut(d.b.name, newKey, dst)

		// Delete old.
		d.b.st.hotDelete(d.b.name, key)
		ck := compositeKey(d.b.name, key)
		if loc, found := d.b.st.coldDir.get(ck); found {
			d.b.st.cold.mu.Lock()
			d.b.st.cold.markTombstoneAt(loc.slotIndex)
			d.b.st.cold.mu.Unlock()
			d.b.st.coldDir.remove(ck)
		}
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
	r := &multipartRegistry{
		uploads: make(map[string]*multipartUpload),
	}
	r.counter.Store(time.Now().UnixNano())
	return r
}

type multipartUpload struct {
	id          string
	bucket      string
	key         string
	contentType string
	parts       map[int]*partData
	metadata    map[string]string
	created     time.Time
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
		return nil, fmt.Errorf("falcon: key is empty")
	}

	id := strconv.FormatInt(b.st.mp.counter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:          id,
		bucket:      b.name,
		key:         key,
		contentType: contentType,
		parts:       make(map[int]*partData),
		metadata:    metadata,
		created:     fastTime(),
	}

	b.st.mp.mu.Lock()
	b.st.mp.uploads[id] = upload
	b.st.mp.mu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(_ context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, _ storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("falcon: part number %d out of range [1, %d]", number, maxPartNumber)
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
				return nil, fmt.Errorf("falcon: read part: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("falcon: read part: %w", err)
		}
		data = buf.Bytes()
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.st.mp.mu.Lock()
	upload.parts[number] = &partData{
		number: number,
		data:   data,
		size:   int64(len(data)),
		etag:   etag,
	}
	b.st.mp.mu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   int64(len(data)),
		ETag:   etag,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("falcon: part number %d out of range", number)
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
		return nil, fmt.Errorf("falcon: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	// Open source object.
	e, found := b.st.hotGet(srcBucket, srcKey)
	if !found {
		e, found = b.st.coldGet(srcBucket, srcKey)
		if !found {
			return nil, storage.ErrNotExist
		}
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
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   p.size,
			ETag:   p.etag,
		})
	}
	b.st.mp.mu.RUnlock()

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

func (b *bucket) CompleteMultipart(_ context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, _ storage.Options) (*storage.Object, error) {
	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Sort parts by number.
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Verify all parts exist.
	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("falcon: part %d not found", p.Number)
		}
	}

	// Assemble final value.
	var totalSize int64
	for _, p := range parts {
		totalSize += upload.parts[p.Number].size
	}

	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, upload.parts[p.Number].data...)
	}

	now := fastNow()

	e := acquireEntry()
	e.value = assembled
	e.contentType = upload.contentType
	e.created = now
	e.updated = now
	e.size = int64(len(assembled))

	b.st.hotPut(b.name, upload.key, e)

	return &storage.Object{
		Bucket:      b.name,
		Key:         upload.key,
		Size:        e.size,
		ContentType: upload.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
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

func (it *bucketIter) Close() error {
	it.list = nil
	return nil
}

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

func (it *objectIter) Close() error {
	it.list = nil
	return nil
}
