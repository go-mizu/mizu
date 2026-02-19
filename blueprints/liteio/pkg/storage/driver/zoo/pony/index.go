package pony

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// On-disk hash table via mmap.
//
// Layout: [Header 64B] [Slots N×64B] [StringPool...]
//
// Uses open-addressing with linear probing. The string pool stores composite
// keys (bucket\x00key) and content types for each entry. Only mmap'd pages
// that are accessed consume physical memory — the OS manages eviction.

const (
	idxMagic     = "PONYIDX\x00"
	idxVersion   = 1
	idxHdrSize   = 64
	idxSlotSize  = 64
	hashEmpty    = 0
	hashTombstone = 1

	defaultSlotCount     = 1 << 16 // 65536 slots
	maxLoadPercent       = 75
	defaultStringPoolCap = 4 * 1024 * 1024 // 4MB initial string pool
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

// idxHeader is the on-disk header for the index file.
type idxHeader struct {
	Magic      [8]byte
	Version    uint32
	_flags     uint32
	SlotCount  uint64
	EntryCount uint64
	StringsPos uint64 // next write offset in string pool (absolute file offset)
	_pad       [24]byte
}

// diskIndex manages the mmap'd on-disk hash table.
type diskIndex struct {
	mu   sync.RWMutex
	path string
	fd   *os.File
	data []byte // mmap'd region

	slotCount  uint64
	entryCount uint64
	stringsPos uint64
	fileSize   int64

	// In-memory per-bucket key lists for fast List operations.
	bucketKeys sync.Map // bucket name → *bucketKeyList
}

// bucketKeyList maintains a sorted key list for one bucket.
type bucketKeyList struct {
	mu     sync.RWMutex
	keys   map[string]struct{}
	sorted []string
	dirty  bool
}

func newDiskIndex(path string, initialSlots uint64) (*diskIndex, error) {
	if initialSlots == 0 {
		initialSlots = defaultSlotCount
	}
	// Round up to power of 2.
	initialSlots = nextPow2(initialSlots)

	dir := path
	if i := lastSlash(path); i >= 0 {
		dir = path[:i]
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("pony: mkdir index: %w", err)
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("pony: open index: %w", err)
	}

	info, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("pony: stat index: %w", err)
	}

	idx := &diskIndex{
		path: path,
		fd:   fd,
	}

	if info.Size() == 0 {
		// New index file — initialize.
		if err := idx.initNew(initialSlots); err != nil {
			fd.Close()
			return nil, err
		}
	} else {
		// Existing index file — load.
		if err := idx.loadExisting(); err != nil {
			fd.Close()
			return nil, err
		}
	}

	return idx, nil
}

func (idx *diskIndex) initNew(slotCount uint64) error {
	stringsStart := int64(idxHdrSize) + int64(slotCount)*idxSlotSize
	fileSize := stringsStart + defaultStringPoolCap

	if err := idx.fd.Truncate(fileSize); err != nil {
		return fmt.Errorf("pony: truncate index: %w", err)
	}

	data, err := syscall.Mmap(int(idx.fd.Fd()), 0, int(fileSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("pony: mmap index: %w", err)
	}

	idx.data = data
	idx.slotCount = slotCount
	idx.entryCount = 0
	idx.stringsPos = uint64(stringsStart)
	idx.fileSize = fileSize

	// Write header.
	idx.writeHeader()

	return nil
}

func (idx *diskIndex) loadExisting() error {
	info, err := idx.fd.Stat()
	if err != nil {
		return fmt.Errorf("pony: stat index: %w", err)
	}

	data, err := syscall.Mmap(int(idx.fd.Fd()), 0, int(info.Size()),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("pony: mmap index: %w", err)
	}

	idx.data = data
	idx.fileSize = info.Size()

	// Read header.
	if len(data) < idxHdrSize {
		return fmt.Errorf("pony: index file too small")
	}
	hdr := (*idxHeader)(unsafe.Pointer(&data[0]))
	if string(hdr.Magic[:]) != idxMagic {
		return fmt.Errorf("pony: invalid index magic")
	}
	if hdr.Version != idxVersion {
		return fmt.Errorf("pony: unsupported index version %d", hdr.Version)
	}

	idx.slotCount = hdr.SlotCount
	idx.entryCount = hdr.EntryCount
	idx.stringsPos = hdr.StringsPos

	// Rebuild in-memory key lists from the hash table.
	idx.rebuildKeyLists()

	return nil
}

func (idx *diskIndex) writeHeader() {
	hdr := (*idxHeader)(unsafe.Pointer(&idx.data[0]))
	copy(hdr.Magic[:], idxMagic)
	hdr.Version = idxVersion
	hdr.SlotCount = idx.slotCount
	hdr.EntryCount = idx.entryCount
	hdr.StringsPos = idx.stringsPos
}

func (idx *diskIndex) updateHeaderCounts() {
	hdr := (*idxHeader)(unsafe.Pointer(&idx.data[0]))
	hdr.EntryCount = idx.entryCount
	hdr.StringsPos = idx.stringsPos
}

// slotAt returns a pointer to the i-th slot in the mmap'd hash table.
func (idx *diskIndex) slotAt(i uint64) *diskSlot {
	off := idxHdrSize + i*idxSlotSize
	return (*diskSlot)(unsafe.Pointer(&idx.data[off]))
}

// readString reads a string from the mmap'd data at given offset and length.
func (idx *diskIndex) readString(off uint64, length uint32) string {
	return string(idx.data[off : off+uint64(length)])
}

// appendString writes a string to the string pool and returns its offset.
// Caller must hold the write lock.
func (idx *diskIndex) appendString(s string) uint64 {
	off := idx.stringsPos
	needed := off + uint64(len(s))

	// Grow file if string pool is full.
	if int64(needed) > idx.fileSize {
		idx.growFile(int64(needed))
	}

	copy(idx.data[off:], s)
	idx.stringsPos = needed
	return off
}

// growFile extends the index file and remaps.
func (idx *diskIndex) growFile(needed int64) {
	newSize := idx.fileSize * 2
	for newSize < needed {
		newSize *= 2
	}

	// munmap old region.
	syscall.Munmap(idx.data)

	idx.fd.Truncate(newSize)

	data, err := syscall.Mmap(int(idx.fd.Fd()), 0, int(newSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		// Fatal — index is corrupted.
		panic(fmt.Sprintf("pony: remap index failed: %v", err))
	}

	idx.data = data
	idx.fileSize = newSize
}

// compositeKey returns bucket + "\x00" + key.
func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// hashComposite computes FNV-1a hash of the composite key.
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

// put inserts or updates an entry in the hash table.
func (idx *diskIndex) put(bucket, key, contentType string, valOff, valSize, created, updated int64) {
	h := hashComposite(bucket, key)
	ck := compositeKey(bucket, key)

	idx.mu.Lock()

	// Check if we need to grow.
	if idx.entryCount*100/idx.slotCount >= maxLoadPercent {
		idx.rehash(idx.slotCount * 2)
	}

	mask := idx.slotCount - 1
	startSlot := h & mask

	// Linear probe.
	for i := uint64(0); i < idx.slotCount; i++ {
		si := (startSlot + i) & mask
		s := idx.slotAt(si)

		if s.Hash == hashEmpty || s.Hash == hashTombstone {
			// Found empty slot — insert new entry.
			// appendString may call growFile (munmap+mmap), invalidating s.
			strOff := idx.appendString(ck + contentType)
			s = idx.slotAt(si) // re-obtain after possible remap

			s.Hash = h
			s.StrOff = strOff
			s.StrLen = uint32(len(ck))
			s.CtLen = uint16(len(contentType))
			s.ValOff = valOff
			s.ValSize = valSize
			s.Created = created
			s.Updated = updated

			idx.entryCount++
			idx.updateHeaderCounts()
			idx.mu.Unlock()

			// Update in-memory key list.
			idx.addBucketKey(bucket, key)
			return
		}

		if s.Hash == h {
			// Possible match — verify key.
			storedKey := idx.readString(s.StrOff, s.StrLen)
			if storedKey == ck {
				// Update existing entry.
				// appendString may call growFile (munmap+mmap), invalidating s.
				strOff := idx.appendString(ck + contentType)
				s = idx.slotAt(si) // re-obtain after possible remap
				s.StrOff = strOff
				s.StrLen = uint32(len(ck))
				s.CtLen = uint16(len(contentType))
				s.ValOff = valOff
				s.ValSize = valSize
				s.Updated = updated

				idx.updateHeaderCounts()
				idx.mu.Unlock()
				return
			}
		}
	}

	// Table is full — should not happen with load factor check.
	idx.mu.Unlock()
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
func (idx *diskIndex) get(bucket, key string) (indexResult, bool) {
	h := hashComposite(bucket, key)
	ck := compositeKey(bucket, key)

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	mask := idx.slotCount - 1
	startSlot := h & mask

	for i := uint64(0); i < idx.slotCount; i++ {
		si := (startSlot + i) & mask
		s := idx.slotAt(si)

		if s.Hash == hashEmpty {
			return indexResult{}, false
		}

		if s.Hash == hashTombstone {
			continue
		}

		if s.Hash == h {
			storedKey := idx.readString(s.StrOff, s.StrLen)
			if storedKey == ck {
				ct := idx.readString(s.StrOff+uint64(s.StrLen), uint32(s.CtLen))
				return indexResult{
					valOff:      s.ValOff,
					valSize:     s.ValSize,
					contentType: ct,
					created:     s.Created,
					updated:     s.Updated,
				}, true
			}
		}
	}

	return indexResult{}, false
}

// remove marks an entry as deleted.
func (idx *diskIndex) remove(bucket, key string) bool {
	h := hashComposite(bucket, key)
	ck := compositeKey(bucket, key)

	idx.mu.Lock()

	mask := idx.slotCount - 1
	startSlot := h & mask

	for i := uint64(0); i < idx.slotCount; i++ {
		si := (startSlot + i) & mask
		s := idx.slotAt(si)

		if s.Hash == hashEmpty {
			idx.mu.Unlock()
			return false
		}

		if s.Hash == hashTombstone {
			continue
		}

		if s.Hash == h {
			storedKey := idx.readString(s.StrOff, s.StrLen)
			if storedKey == ck {
				s.Hash = hashTombstone
				idx.entryCount--
				idx.updateHeaderCounts()
				idx.mu.Unlock()

				// Update in-memory key list.
				idx.removeBucketKey(bucket, key)
				return true
			}
		}
	}

	idx.mu.Unlock()
	return false
}

// hasBucket returns true if any keys exist for the given bucket.
func (idx *diskIndex) hasBucket(bucket string) bool {
	v, ok := idx.bucketKeys.Load(bucket)
	if !ok {
		return false
	}
	bk := v.(*bucketKeyList)
	bk.mu.RLock()
	n := len(bk.keys)
	bk.mu.RUnlock()
	return n > 0
}

// list returns all entries matching bucket and prefix, sorted by key.
func (idx *diskIndex) list(bucket, prefix string) []listResult {
	v, ok := idx.bucketKeys.Load(bucket)
	if !ok {
		return nil
	}
	bk := v.(*bucketKeyList)

	bk.mu.Lock()
	if bk.dirty {
		bk.sorted = make([]string, 0, len(bk.keys))
		for k := range bk.keys {
			bk.sorted = append(bk.sorted, k)
		}
		sort.Strings(bk.sorted)
		bk.dirty = false
	}
	sorted := bk.sorted
	bk.mu.Unlock()

	start := sort.SearchStrings(sorted, prefix)

	var results []listResult
	for i := start; i < len(sorted); i++ {
		key := sorted[i]
		if !strings.HasPrefix(key, prefix) {
			break
		}

		if r, ok := idx.get(bucket, key); ok {
			results = append(results, listResult{
				key:         key,
				valOff:      r.valOff,
				valSize:     r.valSize,
				contentType: r.contentType,
				created:     r.created,
				updated:     r.updated,
			})
		}
	}

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

// addBucketKey adds a key to the in-memory per-bucket list.
func (idx *diskIndex) addBucketKey(bucket, key string) {
	v, ok := idx.bucketKeys.Load(bucket)
	if !ok {
		bk := &bucketKeyList{
			keys:  make(map[string]struct{}, 64),
			dirty: true,
		}
		actual, _ := idx.bucketKeys.LoadOrStore(bucket, bk)
		v = actual
	}
	bk := v.(*bucketKeyList)
	bk.mu.Lock()
	if _, exists := bk.keys[key]; !exists {
		bk.keys[key] = struct{}{}
		bk.dirty = true
	}
	bk.mu.Unlock()
}

// removeBucketKey removes a key from the in-memory per-bucket list.
func (idx *diskIndex) removeBucketKey(bucket, key string) {
	v, ok := idx.bucketKeys.Load(bucket)
	if !ok {
		return
	}
	bk := v.(*bucketKeyList)
	bk.mu.Lock()
	delete(bk.keys, key)
	bk.dirty = true
	bk.mu.Unlock()
}

// rebuildKeyLists scans the hash table and rebuilds in-memory key lists.
func (idx *diskIndex) rebuildKeyLists() {
	for i := uint64(0); i < idx.slotCount; i++ {
		s := idx.slotAt(i)
		if s.Hash <= hashTombstone {
			continue
		}
		ck := idx.readString(s.StrOff, s.StrLen)
		// Split composite key at null byte.
		nullIdx := strings.IndexByte(ck, 0)
		if nullIdx < 0 {
			continue
		}
		bucket := ck[:nullIdx]
		key := ck[nullIdx+1:]
		idx.addBucketKey(bucket, key)
	}
}

// rehash grows the hash table to newSlotCount and re-inserts all entries.
// Caller must hold the write lock.
func (idx *diskIndex) rehash(newSlotCount uint64) {
	newSlotCount = nextPow2(newSlotCount)

	// Collect all live entries.
	type entry struct {
		hash    uint64
		ck      string
		ctLen   uint16
		ct      string
		valOff  int64
		valSize int64
		created int64
		updated int64
	}

	entries := make([]entry, 0, idx.entryCount)
	for i := uint64(0); i < idx.slotCount; i++ {
		s := idx.slotAt(i)
		if s.Hash <= hashTombstone {
			continue
		}
		ck := idx.readString(s.StrOff, s.StrLen)
		ct := idx.readString(s.StrOff+uint64(s.StrLen), uint32(s.CtLen))
		entries = append(entries, entry{
			hash:    s.Hash,
			ck:      ck,
			ctLen:   s.CtLen,
			ct:      ct,
			valOff:  s.ValOff,
			valSize: s.ValSize,
			created: s.Created,
			updated: s.Updated,
		})
	}

	// munmap old.
	syscall.Munmap(idx.data)

	// Compute new file size.
	stringsStart := int64(idxHdrSize) + int64(newSlotCount)*idxSlotSize
	// Estimate string pool size: double what we have plus some headroom.
	stringPoolSize := int64(idx.stringsPos) - (int64(idxHdrSize) + int64(idx.slotCount)*idxSlotSize)
	if stringPoolSize < defaultStringPoolCap {
		stringPoolSize = defaultStringPoolCap
	}
	stringPoolSize *= 2
	fileSize := stringsStart + stringPoolSize

	idx.fd.Truncate(fileSize)

	data, err := syscall.Mmap(int(idx.fd.Fd()), 0, int(fileSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		panic(fmt.Sprintf("pony: rehash remap failed: %v", err))
	}

	idx.data = data
	idx.fileSize = fileSize
	idx.slotCount = newSlotCount
	idx.entryCount = 0
	idx.stringsPos = uint64(stringsStart)

	// Clear slots.
	slotsEnd := idxHdrSize + newSlotCount*idxSlotSize
	for i := uint64(idxHdrSize); i < slotsEnd; i++ {
		idx.data[i] = 0
	}

	idx.writeHeader()

	// Re-insert all entries.
	mask := newSlotCount - 1
	for _, e := range entries {
		startSlot := e.hash & mask
		for j := uint64(0); j < newSlotCount; j++ {
			si := (startSlot + j) & mask
			s := idx.slotAt(si)
			if s.Hash == hashEmpty {
				strOff := idx.appendString(e.ck + e.ct)
				s.Hash = e.hash
				s.StrOff = strOff
				s.StrLen = uint32(len(e.ck))
				s.CtLen = e.ctLen
				s.ValOff = e.valOff
				s.ValSize = e.valSize
				s.Created = e.created
				s.Updated = e.updated
				idx.entryCount++
				break
			}
		}
	}

	idx.updateHeaderCounts()
}

// reset clears the index for reuse (e.g., after volume recovery).
func (idx *diskIndex) reset() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear all slots.
	slotsEnd := uint64(idxHdrSize) + idx.slotCount*idxSlotSize
	for i := uint64(idxHdrSize); i < slotsEnd; i++ {
		idx.data[i] = 0
	}

	stringsStart := int64(idxHdrSize) + int64(idx.slotCount)*idxSlotSize
	idx.entryCount = 0
	idx.stringsPos = uint64(stringsStart)
	idx.writeHeader()

	// Clear key lists.
	idx.bucketKeys.Range(func(k, _ any) bool {
		idx.bucketKeys.Delete(k)
		return true
	})
}

func (idx *diskIndex) close() error {
	idx.writeHeader()
	if idx.data != nil {
		syscall.Munmap(idx.data)
	}
	if idx.fd != nil {
		return idx.fd.Close()
	}
	return nil
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

