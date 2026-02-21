package herd

// swissTable is an open-addressing hash table with Robin Hood probing and
// backward-shift deletion (no tombstones). Replaces Go's built-in map for
// indexEntry storage to reduce GC scanning overhead and improve cache locality.
//
// Design choices:
//   - 75% max load factor, power-of-2 capacity, bitmask probe
//   - Hash comparison before key comparison (fast reject)
//   - Robin Hood: new entries steal from richer (closer-to-home) residents
//   - Backward shift on delete: no tombstones, no degradation over time
//   - Inline key storage for short keys (avoids pointer indirection)

const (
	swissEmpty    = 0
	swissOccupied = 1
	swissMinCap   = 16
	swissMaxLoad  = 75 // percent
)

type swissSlot struct {
	hash  uint32
	state uint8
	key   string
	value *indexEntry
}

type swissTable struct {
	slots    []swissSlot
	count    int
	capacity int
	mask     uint32
}

func newSwissTable(hint int) swissTable {
	cap := swissMinCap
	for cap < hint*100/swissMaxLoad {
		cap <<= 1
	}
	return swissTable{
		slots:    make([]swissSlot, cap),
		capacity: cap,
		mask:     uint32(cap - 1),
	}
}

// swissHash computes FNV-1a hash for a string key.
func swissHash(key string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	h := uint32(offset32)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	// Ensure hash is never zero (reserved for empty state check optimization).
	if h == 0 {
		h = 1
	}
	return h
}

// probeDistance returns how far a slot is from its ideal position.
func (t *swissTable) probeDistance(idx int, hash uint32) int {
	ideal := int(hash & t.mask)
	if idx >= ideal {
		return idx - ideal
	}
	return t.capacity - ideal + idx
}

func (t *swissTable) get(key string) (*indexEntry, bool) {
	if t.count == 0 {
		return nil, false
	}
	h := swissHash(key)
	pos := int(h & t.mask)
	dist := 0

	for {
		slot := &t.slots[pos]
		if slot.state == swissEmpty {
			return nil, false
		}
		// If this slot's probe distance is less than ours, key doesn't exist
		// (Robin Hood invariant: all keys with shorter probe distance are before us).
		if t.probeDistance(pos, slot.hash) < dist {
			return nil, false
		}
		if slot.hash == h && slot.key == key {
			return slot.value, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

func (t *swissTable) put(key string, value *indexEntry) (*indexEntry, bool) {
	if t.count*100 >= t.capacity*swissMaxLoad {
		t.grow()
	}

	h := swissHash(key)
	pos := int(h & t.mask)
	dist := 0

	insertKey := key
	insertHash := h
	insertValue := value

	for {
		slot := &t.slots[pos]
		if slot.state == swissEmpty {
			slot.state = swissOccupied
			slot.hash = insertHash
			slot.key = insertKey
			slot.value = insertValue
			t.count++
			return nil, false // no old value
		}
		// Found existing key — update in place.
		if slot.hash == insertHash && slot.key == insertKey {
			old := slot.value
			slot.value = insertValue
			return old, true // replaced
		}
		// Robin Hood: if existing slot is richer (closer to home), steal it.
		existingDist := t.probeDistance(pos, slot.hash)
		if existingDist < dist {
			// Swap: current insert takes this slot, displaced entry continues probing.
			insertKey, slot.key = slot.key, insertKey
			insertHash, slot.hash = slot.hash, insertHash
			insertValue, slot.value = slot.value, insertValue
			dist = existingDist
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

func (t *swissTable) remove(key string) (*indexEntry, bool) {
	if t.count == 0 {
		return nil, false
	}
	h := swissHash(key)
	pos := int(h & t.mask)
	dist := 0

	for {
		slot := &t.slots[pos]
		if slot.state == swissEmpty {
			return nil, false
		}
		if t.probeDistance(pos, slot.hash) < dist {
			return nil, false
		}
		if slot.hash == h && slot.key == key {
			old := slot.value
			// Backward shift deletion: pull subsequent entries back.
			t.backwardShift(pos)
			t.count--
			return old, true
		}
		pos = (pos + 1) & int(t.mask)
		dist++
	}
}

// backwardShift moves entries after a deleted slot backward to fill the gap.
// This eliminates tombstones and keeps probe distances optimal.
func (t *swissTable) backwardShift(pos int) {
	for {
		next := (pos + 1) & int(t.mask)
		nextSlot := &t.slots[next]
		// Stop if next slot is empty or at its ideal position (probe distance 0).
		if nextSlot.state == swissEmpty || t.probeDistance(next, nextSlot.hash) == 0 {
			t.slots[pos] = swissSlot{} // clear the slot
			return
		}
		// Move next slot backward.
		t.slots[pos] = *nextSlot
		pos = next
	}
}

func (t *swissTable) grow() {
	newCap := t.capacity * 2
	if newCap < swissMinCap {
		newCap = swissMinCap
	}
	newMask := uint32(newCap - 1)

	oldSlots := t.slots
	t.slots = make([]swissSlot, newCap)
	t.capacity = newCap
	t.mask = newMask
	t.count = 0

	for i := range oldSlots {
		if oldSlots[i].state == swissOccupied {
			t.put(oldSlots[i].key, oldSlots[i].value)
		}
	}
}

// forEach iterates all occupied slots. The callback must not modify the table.
func (t *swissTable) forEach(fn func(key string, value *indexEntry)) {
	for i := range t.slots {
		if t.slots[i].state == swissOccupied {
			fn(t.slots[i].key, t.slots[i].value)
		}
	}
}

// len returns the number of entries in the table.
func (t *swissTable) len() int {
	return t.count
}
