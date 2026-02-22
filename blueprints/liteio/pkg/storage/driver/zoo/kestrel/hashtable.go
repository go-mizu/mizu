package kestrel

// htable is a Robin Hood open-addressing hash table that replaces
// Go's built-in map[string]*hotRecord for hot path operations.
//
// Benefits over Go map:
//   - Flat array layout (better cache utilization, no bucket chains)
//   - Robin Hood probing (shorter average probe distances at high load)
//   - Backward-shift deletion (no tombstones, no degradation over time)
//   - Direct function calls (no runtime.mapaccess1_faststr indirection)
//   - 64-bit hash with early-exit on hash mismatch (fast reject)

const (
	htMinCap  = 64
	htMaxLoad = 75 // percent — balances probe distance vs cache utilization
)

type htEntry struct {
	hash uint64     // full 64-bit hash; 0 = empty slot
	key  string     // composite key (bucket\x00key)
	rec  *hotRecord // record pointer
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
// without building the composite key string.
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

// get looks up a record by pre-computed hash and composite key.
// The key may be a stack-backed unsafeString — comparison is byte-by-byte.
func (t *htable) get(hash uint64, key string) (*hotRecord, bool) {
	if t.count == 0 {
		return nil, false
	}
	pos := int(hash & t.mask)
	dist := 0
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			return nil, false
		}
		if t.probeDist(pos, e.hash) < dist {
			return nil, false
		}
		if e.hash == hash && e.key == key {
			return e.rec, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

// update replaces the record for an existing key without Robin Hood swapping.
// Returns (oldRecord, true) if found, (nil, false) if key doesn't exist.
// Safe to call with stack-backed key strings.
func (t *htable) update(hash uint64, key string, rec *hotRecord) (*hotRecord, bool) {
	if t.count == 0 {
		return nil, false
	}
	pos := int(hash & t.mask)
	dist := 0
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			return nil, false
		}
		if t.probeDist(pos, e.hash) < dist {
			return nil, false
		}
		if e.hash == hash && e.key == key {
			old := e.rec
			e.rec = rec
			return old, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

// put inserts or updates an entry. The key must be heap-allocated for new entries
// (it will be stored in the table). Returns (oldRecord, true) on replacement.
func (t *htable) put(hash uint64, key string, rec *hotRecord) (*hotRecord, bool) {
	if t.count*100 >= t.capacity*htMaxLoad {
		t.grow()
	}
	pos := int(hash & t.mask)
	dist := 0
	ih, ik, ir := hash, key, rec
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			e.hash = ih
			e.key = ik
			e.rec = ir
			t.count++
			return nil, false
		}
		if e.hash == ih && e.key == ik {
			old := e.rec
			e.rec = ir
			return old, true
		}
		// Robin Hood: steal from entries closer to their ideal position.
		ed := t.probeDist(pos, e.hash)
		if ed < dist {
			ih, e.hash = e.hash, ih
			ik, e.key = e.key, ik
			ir, e.rec = e.rec, ir
			dist = ed
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

// putOrUpdate performs insert-or-update in a single table traversal.
// searchKey is used for matching (may be stack-backed unsafeString).
// insertKey is used for storage on insert (must be heap-allocated).
// For updates: zero allocation (uses searchKey comparison only).
// For inserts: zero allocation inside this function (insertKey pre-allocated by caller).
// Returns (oldRecord, true) for updates, (nil, false) for inserts.
func (t *htable) putOrUpdate(hash uint64, searchKey, insertKey string, rec *hotRecord) (*hotRecord, bool) {
	if t.count*100 >= t.capacity*htMaxLoad {
		t.grow()
	}
	pos := int(hash & t.mask)
	dist := 0
	mask := int(t.mask)
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			// Empty slot — key doesn't exist. Insert here.
			e.hash = hash
			e.key = insertKey
			e.rec = rec
			t.count++
			return nil, false
		}
		if e.hash == hash && e.key == searchKey {
			// Key exists — update in place.
			old := e.rec
			e.rec = rec
			return old, true
		}
		ed := t.probeDist(pos, e.hash)
		if ed < dist {
			// Key doesn't exist (Robin Hood invariant). Insert with displacement.
			ih, ik, ir := hash, insertKey, rec
			ih, e.hash = e.hash, ih
			ik, e.key = e.key, ik
			ir, e.rec = e.rec, ir
			dist = ed
			pos = (pos + 1) & mask
			dist++
			// Displacement loop for the evicted entry.
			for {
				e2 := &t.entries[pos]
				if e2.hash == 0 {
					e2.hash = ih
					e2.key = ik
					e2.rec = ir
					t.count++
					return nil, false
				}
				ed2 := t.probeDist(pos, e2.hash)
				if ed2 < dist {
					ih, e2.hash = e2.hash, ih
					ik, e2.key = e2.key, ik
					ir, e2.rec = e2.rec, ir
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

// remove deletes an entry by hash+key. Uses backward-shift deletion
// (no tombstones, keeps probe distances optimal).
func (t *htable) remove(hash uint64, key string) (*hotRecord, bool) {
	if t.count == 0 {
		return nil, false
	}
	pos := int(hash & t.mask)
	dist := 0
	for {
		e := &t.entries[pos]
		if e.hash == 0 {
			return nil, false
		}
		if t.probeDist(pos, e.hash) < dist {
			return nil, false
		}
		if e.hash == hash && e.key == key {
			old := e.rec
			t.backShift(pos)
			t.count--
			return old, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

// backwardShift moves entries after a deleted slot backward to fill the gap.
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
			t.put(old[i].hash, old[i].key, old[i].rec)
		}
	}
}
