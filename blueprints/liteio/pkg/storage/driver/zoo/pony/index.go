package pony

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// Sharded on-disk hash table via mmap.
//
// Layout per shard: [Header 64B] [Slots N×64B] [StringPool...]
//
// 256 shards, each with its own RWMutex and mmap file. Shard selection via
// FNV-1a hash bitmask. This eliminates the global lock bottleneck from v1.

const (
	idxMagic     = "PONYIDX\x00"
	idxVersion   = 2
	idxHdrSize   = 64
	idxSlotSize  = 64
	hashEmpty    = 0
	hashTombstone = 1

	shardCount           = 256
	shardMask            = shardCount - 1
	defaultSlotsPerShard = 256 // 256 slots × 64B = 16KB per shard
	maxLoadPercent       = 75
	defaultStringPoolCap = 64 * 1024 // 64KB per shard (vs 4MB for single index)
)

// diskSlot is the on-disk format for a hash table entry.
// 64 bytes, cache-line aligned for optimal memory access.
type diskSlot struct {
	Hash    uint64 // FNV-1a of composite key. 0=empty, 1=tombstone
	StrOff  uint64 // offset in string pool for compositeKey+contentType
	StrLen  uint32 // composite key length
	CtLen   uint16 // content type length (ct starts at StrOff+StrLen)
	_pad1   uint16
	ValOff  int64  // value offset in volume
	ValSize int64  // value size
	Created int64  // UnixNano
	Updated int64  // UnixNano
	_pad2   [8]byte
}

// idxHeader is the on-disk header for each shard index file.
type idxHeader struct {
	Magic      [8]byte
	Version    uint32
	_flags     uint32
	SlotCount  uint64
	EntryCount uint64
	StringsPos uint64 // next write offset in string pool (absolute file offset)
	_pad       [24]byte
}

// diskShard is one shard of the sharded hash table.
type diskShard struct {
	mu         sync.RWMutex
	path       string
	fd         *os.File
	data       []byte // mmap'd region
	slotCount  uint64
	entryCount uint64
	stringsPos uint64
	fileSize   int64
}

// shardedIndex manages 256 independent shard indexes.
type shardedIndex struct {
	shards  [shardCount]*diskShard
	dir     string
	version atomic.Uint64 // bumped on every put/remove for list cache invalidation

	// List result cache — avoids rescanning when data hasn't changed.
	listCacheMu sync.RWMutex
	listCache   map[string]listCacheEntry
}

type listCacheEntry struct {
	version uint64
	results []listResult
}

func newShardedIndex(dir string, initialSlotsPerShard uint64) (*shardedIndex, error) {
	if initialSlotsPerShard == 0 {
		initialSlotsPerShard = defaultSlotsPerShard
	}
	initialSlotsPerShard = nextPow2(initialSlotsPerShard)

	idxDir := filepath.Join(dir, "idx")
	if err := os.MkdirAll(idxDir, 0o750); err != nil {
		return nil, fmt.Errorf("pony: mkdir index: %w", err)
	}

	si := &shardedIndex{
		dir:       idxDir,
		listCache: make(map[string]listCacheEntry),
	}

	for i := 0; i < shardCount; i++ {
		path := filepath.Join(idxDir, fmt.Sprintf("s%03d.idx", i))
		shard, err := newDiskShard(path, initialSlotsPerShard)
		if err != nil {
			// Close already-opened shards.
			for j := 0; j < i; j++ {
				si.shards[j].close()
			}
			return nil, err
		}
		si.shards[i] = shard
	}

	return si, nil
}

func newDiskShard(path string, initialSlots uint64) (*diskShard, error) {
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("pony: open shard: %w", err)
	}

	info, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("pony: stat shard: %w", err)
	}

	shard := &diskShard{
		path: path,
		fd:   fd,
	}

	if info.Size() == 0 {
		if err := shard.initNew(initialSlots); err != nil {
			fd.Close()
			return nil, err
		}
	} else {
		if err := shard.loadExisting(); err != nil {
			fd.Close()
			return nil, err
		}
	}

	return shard, nil
}

func (s *diskShard) initNew(slotCount uint64) error {
	stringsStart := int64(idxHdrSize) + int64(slotCount)*idxSlotSize
	fileSize := stringsStart + defaultStringPoolCap

	if err := s.fd.Truncate(fileSize); err != nil {
		return fmt.Errorf("pony: truncate shard: %w", err)
	}

	data, err := syscall.Mmap(int(s.fd.Fd()), 0, int(fileSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("pony: mmap shard: %w", err)
	}

	s.data = data
	s.slotCount = slotCount
	s.entryCount = 0
	s.stringsPos = uint64(stringsStart)
	s.fileSize = fileSize
	s.writeHeader()

	return nil
}

func (s *diskShard) loadExisting() error {
	info, err := s.fd.Stat()
	if err != nil {
		return fmt.Errorf("pony: stat shard: %w", err)
	}

	data, err := syscall.Mmap(int(s.fd.Fd()), 0, int(info.Size()),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("pony: mmap shard: %w", err)
	}

	s.data = data
	s.fileSize = info.Size()

	if len(data) < idxHdrSize {
		return fmt.Errorf("pony: shard file too small")
	}
	hdr := (*idxHeader)(unsafe.Pointer(&data[0]))
	if string(hdr.Magic[:]) != idxMagic {
		return fmt.Errorf("pony: invalid shard magic")
	}

	s.slotCount = hdr.SlotCount
	s.entryCount = hdr.EntryCount
	s.stringsPos = hdr.StringsPos

	return nil
}

func (s *diskShard) writeHeader() {
	hdr := (*idxHeader)(unsafe.Pointer(&s.data[0]))
	copy(hdr.Magic[:], idxMagic)
	hdr.Version = idxVersion
	hdr.SlotCount = s.slotCount
	hdr.EntryCount = s.entryCount
	hdr.StringsPos = s.stringsPos
}

func (s *diskShard) updateHeaderCounts() {
	hdr := (*idxHeader)(unsafe.Pointer(&s.data[0]))
	hdr.EntryCount = s.entryCount
	hdr.StringsPos = s.stringsPos
}

func (s *diskShard) slotAt(i uint64) *diskSlot {
	off := idxHdrSize + i*idxSlotSize
	return (*diskSlot)(unsafe.Pointer(&s.data[off]))
}

// appendString writes a string to the string pool and returns its offset.
// Caller must hold the write lock.
func (s *diskShard) appendString(str string) uint64 {
	off := s.stringsPos
	needed := off + uint64(len(str))

	if int64(needed) > s.fileSize {
		s.growFile(int64(needed))
	}

	copy(s.data[off:], str)
	s.stringsPos = needed
	return off
}

// appendCompositeAndCT writes bucket+"\x00"+key+contentType directly to the
// string pool without concatenating them into a Go string first. Returns the
// offset where the composite key starts.
func (s *diskShard) appendCompositeAndCT(bucket, key, contentType string) uint64 {
	total := uint64(len(bucket) + 1 + len(key) + len(contentType))
	off := s.stringsPos
	needed := off + total

	if int64(needed) > s.fileSize {
		s.growFile(int64(needed))
	}

	p := off
	copy(s.data[p:], bucket)
	p += uint64(len(bucket))
	s.data[p] = 0
	p++
	copy(s.data[p:], key)
	p += uint64(len(key))
	copy(s.data[p:], contentType)
	s.stringsPos = needed
	return off
}

func (s *diskShard) growFile(needed int64) {
	newSize := s.fileSize * 2
	for newSize < needed {
		newSize *= 2
	}

	syscall.Munmap(s.data)
	s.fd.Truncate(newSize)

	data, err := syscall.Mmap(int(s.fd.Fd()), 0, int(newSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		panic(fmt.Sprintf("pony: remap shard failed: %v", err))
	}

	s.data = data
	s.fileSize = newSize
}

// readStringView returns a zero-copy string view into mmap'd data.
// The returned string is only valid while the shard's RLock/Lock is held.
func (s *diskShard) readStringView(off uint64, length uint32) string {
	b := s.data[off : off+uint64(length)]
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// readStringCopy returns a copied string from the mmap'd data.
// Safe to use after releasing the lock.
func (s *diskShard) readStringCopy(off uint64, length uint32) string {
	return string(s.data[off : off+uint64(length)])
}

// matchCompositeKey checks if the stored composite key matches bucket+"\x00"+key
// without allocating a new string.
func (s *diskShard) matchCompositeKey(off uint64, strLen uint32, bucket, key string) bool {
	bl := len(bucket)
	kl := len(key)
	expected := bl + 1 + kl
	if int(strLen) != expected {
		return false
	}
	base := off
	// Compare bucket bytes.
	for i := 0; i < bl; i++ {
		if s.data[base+uint64(i)] != bucket[i] {
			return false
		}
	}
	// Check null separator.
	if s.data[base+uint64(bl)] != 0 {
		return false
	}
	// Compare key bytes.
	keyOff := base + uint64(bl) + 1
	for i := 0; i < kl; i++ {
		if s.data[keyOff+uint64(i)] != key[i] {
			return false
		}
	}
	return true
}

func (s *diskShard) close() error {
	s.writeHeader()
	if s.data != nil {
		syscall.Munmap(s.data)
	}
	if s.fd != nil {
		return s.fd.Close()
	}
	return nil
}

// --- shardedIndex methods ---

// shardFor returns the shard index for a given hash.
func (si *shardedIndex) shardFor(h uint64) *diskShard {
	return si.shards[h&shardMask]
}

// hashComposite computes FNV-1a hash of bucket + "\x00" + key.
// Returns hash >= 2 (0=empty, 1=tombstone are reserved).
func hashComposite(bucket, key string) uint64 {
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
	if h <= hashTombstone {
		h = 2
	}
	return h
}

// put inserts or updates an entry in the appropriate shard.
func (si *shardedIndex) put(bucket, key, contentType string, valOff, valSize, created, updated int64) {
	si.version.Add(1)
	h := hashComposite(bucket, key)
	shard := si.shardFor(h)
	ckLen := uint32(len(bucket) + 1 + len(key))
	ctLen := uint16(len(contentType))

	shard.mu.Lock()

	// Check if we need to grow.
	if shard.entryCount*100/shard.slotCount >= maxLoadPercent {
		shard.rehash(shard.slotCount * 2)
	}

	mask := shard.slotCount - 1
	startSlot := h & mask

	for i := uint64(0); i < shard.slotCount; i++ {
		si := (startSlot + i) & mask
		slot := shard.slotAt(si)

		if slot.Hash == hashEmpty || slot.Hash == hashTombstone {
			// Write directly to string pool without Go string concat.
			strOff := shard.appendCompositeAndCT(bucket, key, contentType)
			slot = shard.slotAt(si) // re-obtain after possible remap

			slot.Hash = h
			slot.StrOff = strOff
			slot.StrLen = ckLen
			slot.CtLen = ctLen
			slot.ValOff = valOff
			slot.ValSize = valSize
			slot.Created = created
			slot.Updated = updated

			shard.entryCount++
			shard.updateHeaderCounts()
			shard.mu.Unlock()
			return
		}

		if slot.Hash == h && shard.matchCompositeKey(slot.StrOff, slot.StrLen, bucket, key) {
			strOff := shard.appendCompositeAndCT(bucket, key, contentType)
			slot = shard.slotAt(si) // re-obtain after possible remap
			slot.StrOff = strOff
			slot.StrLen = ckLen
			slot.CtLen = ctLen
			slot.ValOff = valOff
			slot.ValSize = valSize
			slot.Updated = updated

			shard.updateHeaderCounts()
			shard.mu.Unlock()
			return
		}
	}

	shard.mu.Unlock()
}

// indexResult holds the result of a hash table lookup.
type indexResult struct {
	valOff      int64
	valSize     int64
	contentType string
	created     int64
	updated     int64
}

// get looks up an entry by bucket and key.
func (si *shardedIndex) get(bucket, key string) (indexResult, bool) {
	h := hashComposite(bucket, key)
	shard := si.shardFor(h)

	shard.mu.RLock()

	mask := shard.slotCount - 1
	startSlot := h & mask

	for i := uint64(0); i < shard.slotCount; i++ {
		si := (startSlot + i) & mask
		slot := shard.slotAt(si)

		if slot.Hash == hashEmpty {
			shard.mu.RUnlock()
			return indexResult{}, false
		}

		if slot.Hash == hashTombstone {
			continue
		}

		if slot.Hash == h && shard.matchCompositeKey(slot.StrOff, slot.StrLen, bucket, key) {
			ct := shard.readStringCopy(slot.StrOff+uint64(slot.StrLen), uint32(slot.CtLen))
			r := indexResult{
				valOff:      slot.ValOff,
				valSize:     slot.ValSize,
				contentType: ct,
				created:     slot.Created,
				updated:     slot.Updated,
			}
			shard.mu.RUnlock()
			return r, true
		}
	}

	shard.mu.RUnlock()
	return indexResult{}, false
}

// remove marks an entry as deleted.
func (si *shardedIndex) remove(bucket, key string) bool {
	si.version.Add(1)
	h := hashComposite(bucket, key)
	shard := si.shardFor(h)

	shard.mu.Lock()

	mask := shard.slotCount - 1
	startSlot := h & mask

	for i := uint64(0); i < shard.slotCount; i++ {
		si := (startSlot + i) & mask
		slot := shard.slotAt(si)

		if slot.Hash == hashEmpty {
			shard.mu.Unlock()
			return false
		}

		if slot.Hash == hashTombstone {
			continue
		}

		if slot.Hash == h && shard.matchCompositeKey(slot.StrOff, slot.StrLen, bucket, key) {
			slot.Hash = hashTombstone
			shard.entryCount--
			shard.updateHeaderCounts()
			shard.mu.Unlock()
			return true
		}
	}

	shard.mu.Unlock()
	return false
}

// hasBucket returns true if any keys exist for the given bucket.
// Scans all shards.
func (si *shardedIndex) hasBucket(bucket string) bool {
	for i := 0; i < shardCount; i++ {
		shard := si.shards[i]
		shard.mu.RLock()
		for j := uint64(0); j < shard.slotCount; j++ {
			slot := shard.slotAt(j)
			if slot.Hash <= hashTombstone {
				continue
			}
			// Check if composite key starts with bucket + "\x00".
			bl := len(bucket)
			if int(slot.StrLen) > bl && shard.data[slot.StrOff+uint64(bl)] == 0 {
				match := true
				for k := 0; k < bl; k++ {
					if shard.data[slot.StrOff+uint64(k)] != bucket[k] {
						match = false
						break
					}
				}
				if match {
					shard.mu.RUnlock()
					return true
				}
			}
		}
		shard.mu.RUnlock()
	}
	return false
}

// hasPrefix returns true if any key in the bucket starts with prefix.
// Scans shards until first match (early exit for Stat directory checks).
func (si *shardedIndex) hasPrefix(bucket, prefix string) bool {
	for i := 0; i < shardCount; i++ {
		shard := si.shards[i]
		shard.mu.RLock()
		for j := uint64(0); j < shard.slotCount; j++ {
			slot := shard.slotAt(j)
			if slot.Hash <= hashTombstone {
				continue
			}
			b, k := shard.extractBucketKey(slot.StrOff, slot.StrLen)
			if b == bucket && strings.HasPrefix(k, prefix) {
				shard.mu.RUnlock()
				return true
			}
		}
		shard.mu.RUnlock()
	}
	return false
}

// firstMatch returns the first entry matching bucket+prefix.
func (si *shardedIndex) firstMatch(bucket, prefix string) (listResult, bool) {
	for i := 0; i < shardCount; i++ {
		shard := si.shards[i]
		shard.mu.RLock()
		for j := uint64(0); j < shard.slotCount; j++ {
			slot := shard.slotAt(j)
			if slot.Hash <= hashTombstone {
				continue
			}
			b, k := shard.extractBucketKey(slot.StrOff, slot.StrLen)
			if b == bucket && strings.HasPrefix(k, prefix) {
				ct := shard.readStringCopy(slot.StrOff+uint64(slot.StrLen), uint32(slot.CtLen))
				r := listResult{
					key:         k,
					valOff:      slot.ValOff,
					valSize:     slot.ValSize,
					contentType: ct,
					created:     slot.Created,
					updated:     slot.Updated,
				}
				shard.mu.RUnlock()
				return r, true
			}
		}
		shard.mu.RUnlock()
	}
	return listResult{}, false
}

// listScanWorkers is the number of goroutines for parallel shard scanning.
const listScanWorkers = 8

// list returns all entries matching bucket and prefix, sorted by key.
// Uses a version-based cache to avoid rescanning when data hasn't changed.
func (si *shardedIndex) list(bucket, prefix string) []listResult {
	cacheKey := bucket + "\x00" + prefix
	ver := si.version.Load()

	// Check cache.
	si.listCacheMu.RLock()
	if entry, ok := si.listCache[cacheKey]; ok && entry.version == ver {
		result := entry.results
		si.listCacheMu.RUnlock()
		return result
	}
	si.listCacheMu.RUnlock()

	// Cache miss — scan shards in parallel.
	results := si.listScan(bucket, prefix)

	// Update cache.
	si.listCacheMu.Lock()
	si.listCache[cacheKey] = listCacheEntry{version: ver, results: results}
	si.listCacheMu.Unlock()

	return results
}

// listScan performs the actual parallel shard scan.
func (si *shardedIndex) listScan(bucket, prefix string) []listResult {
	perWorker := shardCount / listScanWorkers
	partials := make([][]listResult, listScanWorkers)

	var wg sync.WaitGroup
	wg.Add(listScanWorkers)
	for w := 0; w < listScanWorkers; w++ {
		go func(workerIdx int) {
			defer wg.Done()
			start := workerIdx * perWorker
			end := start + perWorker
			if workerIdx == listScanWorkers-1 {
				end = shardCount
			}
			var local []listResult
			for i := start; i < end; i++ {
				shard := si.shards[i]
				shard.mu.RLock()
				if shard.entryCount == 0 {
					shard.mu.RUnlock()
					continue
				}
				for j := uint64(0); j < shard.slotCount; j++ {
					slot := shard.slotAt(j)
					if slot.Hash <= hashTombstone {
						continue
					}
					b, k := shard.extractBucketKey(slot.StrOff, slot.StrLen)
					if b == bucket && strings.HasPrefix(k, prefix) {
						ct := shard.readStringCopy(slot.StrOff+uint64(slot.StrLen), uint32(slot.CtLen))
						local = append(local, listResult{
							key:         k,
							valOff:      slot.ValOff,
							valSize:     slot.ValSize,
							contentType: ct,
							created:     slot.Created,
							updated:     slot.Updated,
						})
					}
				}
				shard.mu.RUnlock()
			}
			partials[workerIdx] = local
		}(w)
	}
	wg.Wait()

	// Merge partials.
	total := 0
	for _, p := range partials {
		total += len(p)
	}
	results := make([]listResult, 0, total)
	for _, p := range partials {
		results = append(results, p...)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].key < results[j].key
	})

	return results
}

type listResult struct {
	key         string
	valOff      int64
	valSize     int64
	contentType string
	created     int64
	updated     int64
}

// extractBucketKey splits a composite key from the string pool into bucket and key.
// Returns zero-copy views (valid under shard lock).
func (s *diskShard) extractBucketKey(off uint64, strLen uint32) (bucket, key string) {
	data := s.data[off : off+uint64(strLen)]
	for i, b := range data {
		if b == 0 {
			bucket = unsafe.String(unsafe.SliceData(data[:i]), i)
			rest := data[i+1:]
			key = unsafe.String(unsafe.SliceData(rest), len(rest))
			return bucket, key
		}
	}
	return "", ""
}

// rehash grows the shard's hash table to newSlotCount and re-inserts all entries.
// Caller must hold the write lock.
func (s *diskShard) rehash(newSlotCount uint64) {
	newSlotCount = nextPow2(newSlotCount)

	type entry struct {
		hash    uint64
		ckct    string // compositeKey + contentType (copied)
		ckLen   uint32
		ctLen   uint16
		valOff  int64
		valSize int64
		created int64
		updated int64
	}

	entries := make([]entry, 0, s.entryCount)
	for i := uint64(0); i < s.slotCount; i++ {
		slot := s.slotAt(i)
		if slot.Hash <= hashTombstone {
			continue
		}
		totalLen := uint32(slot.StrLen) + uint32(slot.CtLen)
		ckct := s.readStringCopy(slot.StrOff, totalLen)
		entries = append(entries, entry{
			hash:    slot.Hash,
			ckct:    ckct,
			ckLen:   slot.StrLen,
			ctLen:   slot.CtLen,
			valOff:  slot.ValOff,
			valSize: slot.ValSize,
			created: slot.Created,
			updated: slot.Updated,
		})
	}

	syscall.Munmap(s.data)

	stringsStart := int64(idxHdrSize) + int64(newSlotCount)*idxSlotSize
	stringPoolSize := int64(s.stringsPos) - (int64(idxHdrSize) + int64(s.slotCount)*idxSlotSize)
	if stringPoolSize < defaultStringPoolCap {
		stringPoolSize = defaultStringPoolCap
	}
	stringPoolSize *= 2
	fileSize := stringsStart + stringPoolSize

	s.fd.Truncate(fileSize)

	data, err := syscall.Mmap(int(s.fd.Fd()), 0, int(fileSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		panic(fmt.Sprintf("pony: rehash remap failed: %v", err))
	}

	s.data = data
	s.fileSize = fileSize
	s.slotCount = newSlotCount
	s.entryCount = 0
	s.stringsPos = uint64(stringsStart)

	// Clear slots.
	slotsEnd := idxHdrSize + newSlotCount*idxSlotSize
	for i := uint64(idxHdrSize); i < slotsEnd; i++ {
		s.data[i] = 0
	}

	s.writeHeader()

	// Re-insert all entries.
	mask := newSlotCount - 1
	for _, e := range entries {
		startSlot := e.hash & mask
		for j := uint64(0); j < newSlotCount; j++ {
			si := (startSlot + j) & mask
			slot := s.slotAt(si)
			if slot.Hash == hashEmpty {
				strOff := s.appendString(e.ckct)
				slot.Hash = e.hash
				slot.StrOff = strOff
				slot.StrLen = e.ckLen
				slot.CtLen = e.ctLen
				slot.ValOff = e.valOff
				slot.ValSize = e.valSize
				slot.Created = e.created
				slot.Updated = e.updated
				s.entryCount++
				break
			}
		}
	}

	s.updateHeaderCounts()
}

// reset clears a shard for reuse.
func (s *diskShard) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	slotsEnd := uint64(idxHdrSize) + s.slotCount*idxSlotSize
	for i := uint64(idxHdrSize); i < slotsEnd; i++ {
		s.data[i] = 0
	}

	stringsStart := int64(idxHdrSize) + int64(s.slotCount)*idxSlotSize
	s.entryCount = 0
	s.stringsPos = uint64(stringsStart)
	s.writeHeader()
}

// totalEntryCount returns the sum of entries across all shards.
func (si *shardedIndex) totalEntryCount() uint64 {
	var total uint64
	for i := 0; i < shardCount; i++ {
		shard := si.shards[i]
		shard.mu.RLock()
		total += shard.entryCount
		shard.mu.RUnlock()
	}
	return total
}

func (si *shardedIndex) reset() {
	for i := 0; i < shardCount; i++ {
		si.shards[i].reset()
	}
}

func (si *shardedIndex) close() error {
	var firstErr error
	for i := 0; i < shardCount; i++ {
		if err := si.shards[i].close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// nextPow2 returns the smallest power of 2 >= n.
func nextPow2(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}

// slotSize verifies diskSlot is exactly 64 bytes at compile time.
var _ [idxSlotSize]byte = [unsafe.Sizeof(diskSlot{})]byte{}

// hdrSize verifies idxHeader is exactly 64 bytes at compile time.
var _ [idxHdrSize]byte = [unsafe.Sizeof(idxHeader{})]byte{}
