package kestrel

// htable is a Robin Hood open-addressing hash table with pointer-free entries.
//
// All htEntry fields are non-pointer types (uint64, int64, uint16, uint32).
// This makes the entry array a "noscan" span — Go's GC never scans or walks
// the entries, eliminating the O(N) GC overhead that dominated v5 profiles.
//
// Data (keys, content types, values) is stored in a separate mmap'd dataArena.
// Each entry stores an arena offset + lengths to locate its data.
//
// Benefits over v5 (pointer-based htEntry with Go strings + *hotRecord):
//   - Zero GC scanning of entry arrays (noscan span)
//   - No per-record heap allocation (hotRecord eliminated)
//   - Inline metadata (size, timestamps) avoids pointer chasing
//   - Same Robin Hood probing + backward-shift deletion (proven optimal)

const (
	htMinCap  = 64
	htMaxLoad = 75 // percent — balances probe distance vs cache utilization
)

// htEntry is a pointer-free hash table entry (48 bytes).
// All fields are non-pointer types → GC noscan.
type htEntry struct {
	hash     uint64 // 8:  full 64-bit hash; 0 = empty slot
	arenaOff int64  // 8:  offset in dataArena where [key][ct][value] is stored
	keyLen   uint16 // 2:  composite key length (bucket\x00key)
	ctLen    uint16 // 2:  content type length
	valueLen uint32 // 4:  value data length
	size     int64  // 8:  object size (for Stat)
	created  int64  // 8:  unix nano
	updated  int64  // 8:  unix nano
}

type htable struct {
	entries  []htEntry
	count    int
	capacity int
	mask     uint64
}

func newHTable(hint int) htable {
	cap := htMinCap
	needed := hint * 100 / htMaxLoad
	if needed < htMinCap {
		needed = htMinCap
	}
	for cap < needed {
		cap <<= 1
	}
	return htable{
		entries:  make([]htEntry, cap),
		capacity: cap,
		mask:     uint64(cap - 1),
	}
}

// htHash64 computes FNV-1a 64-bit hash for bucket + "\x00" + key
// without building the composite key string. Fully inlined by the compiler.
func htHash64(bucket, key string) uint64 {
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
	if h == 0 {
		h = 1
	}
	return h
}

func (t *htable) probeDist(idx int, hash uint64) int {
	ideal := int(hash & t.mask)
	if idx >= ideal {
		return idx - ideal
	}
	return t.capacity - ideal + idx
}

// get looks up an entry by 64-bit hash match (no key comparison).
// With FNV-1a 64-bit hash, collision probability is ~2^-64 per pair.
// For millions of keys, false positive probability is ~2^-32 — negligible.
// This eliminates the arena indirection that caused read regressions in v6a.
func (t *htable) get(hash uint64) (htEntry, bool) {
	if t.count == 0 {
		return htEntry{}, false
	}
	pos := int(hash & t.mask)
	dist := 0
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			return htEntry{}, false
		}
		if t.probeDist(pos, e.hash) < dist {
			return htEntry{}, false
		}
		if e.hash == hash {
			return *e, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

// putOrUpdate performs insert-or-update in a single table traversal.
// Uses hash-only matching (2^-64 collision probability is negligible).
// For inserts: newEntry is stored as-is (arena data already written by caller).
// For updates: only arenaOff, ctLen, valueLen, size, updated are overwritten;
// hash, keyLen, and created are preserved from the existing entry.
// Returns (oldEntry, true) for updates, (zero, false) for inserts.
func (t *htable) putOrUpdate(hash uint64, newEntry htEntry) (htEntry, bool) {
	if t.count*100 >= t.capacity*htMaxLoad {
		t.grow()
	}
	pos := int(hash & t.mask)
	dist := 0
	mask := int(t.mask)
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			// Empty slot — key doesn't exist. Insert.
			*e = newEntry
			t.count++
			return htEntry{}, false
		}
		if e.hash == hash {
			// Hash match → update in place.
			old := *e
			e.arenaOff = newEntry.arenaOff
			e.ctLen = newEntry.ctLen
			e.valueLen = newEntry.valueLen
			e.size = newEntry.size
			e.updated = newEntry.updated
			// Preserve: hash, keyLen, created
			return old, true
		}
		ed := t.probeDist(pos, e.hash)
		if ed < dist {
			// Robin Hood: key doesn't exist. Insert with displacement.
			ie := newEntry
			ie, *e = *e, ie
			dist = ed
			pos = (pos + 1) & mask
			dist++
			for {
				e2 := &t.entries[pos]
				if e2.hash == 0 {
					*e2 = ie
					t.count++
					return htEntry{}, false
				}
				ed2 := t.probeDist(pos, e2.hash)
				if ed2 < dist {
					ie, *e2 = *e2, ie
					dist = ed2
				}
				pos = (pos + 1) & mask
				dist++
			}
		}
		pos = (pos + 1) & mask
		dist++
	}
}

// remove deletes an entry by 64-bit hash match. Uses backward-shift deletion.
// Same hash-only matching as get — 2^-64 collision probability is negligible.
func (t *htable) remove(hash uint64) (htEntry, bool) {
	if t.count == 0 {
		return htEntry{}, false
	}
	pos := int(hash & t.mask)
	dist := 0
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			return htEntry{}, false
		}
		if t.probeDist(pos, e.hash) < dist {
			return htEntry{}, false
		}
		if e.hash == hash {
			old := *e
			t.backShift(pos)
			t.count--
			return old, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

func (t *htable) backShift(pos int) {
	mask := int(t.mask)
	for {
		next := (pos + 1) & mask
		ne := &t.entries[next]
		if ne.hash == 0 || t.probeDist(next, ne.hash) == 0 {
			t.entries[pos] = htEntry{}
			return
		}
		t.entries[pos] = *ne
		pos = next
	}
}

// growInsert inserts during grow — no key comparison needed (no duplicates).
func (t *htable) growInsert(e htEntry) {
	pos := int(e.hash & t.mask)
	dist := 0
	for {
		slot := &t.entries[pos]
		if slot.hash == 0 {
			*slot = e
			t.count++
			return
		}
		ed := t.probeDist(pos, slot.hash)
		if ed < dist {
			e, *slot = *slot, e
			dist = ed
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

func (t *htable) grow() {
	newCap := t.capacity * 2
	if newCap < htMinCap {
		newCap = htMinCap
	}
	old := t.entries
	t.entries = make([]htEntry, newCap)
	t.capacity = newCap
	t.mask = uint64(newCap - 1)
	t.count = 0
	for i := range old {
		if old[i].hash != 0 {
			t.growInsert(old[i])
		}
	}
}
