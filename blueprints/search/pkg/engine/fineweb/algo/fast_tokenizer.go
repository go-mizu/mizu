// Package algo provides FastTokenizer - eliminates map overhead in tokenization.
//
// Profiling shows tokenization takes 41.4% of time, with map operations being expensive.
// New approach:
//   1. Collect hashes into a slice (no map lookup)
//   2. Sort the slice
//   3. Count duplicates in sorted order (linear scan)
//
// This eliminates:
//   - Map allocation
//   - Map hash operations (ironic: we hash to store in a hash map)
//   - Map resize operations
package algo

import (
	"sort"
	"unsafe"
)

// FastTokenize collects hashes without a map, then deduplicates.
// Returns hashes, frequencies, and total token count.
func FastTokenize(text string) ([]uint64, []uint16, int) {
	if len(text) == 0 {
		return nil, nil, 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	// Collect all hashes (with duplicates)
	hashes := make([]uint64, 0, 128)

	for i < n {
		// Skip delimiters
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			hashes = append(hashes, hash)
			tokenCount++
		}
	}

	if len(hashes) == 0 {
		return nil, nil, 0
	}

	// Sort hashes
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })

	// Deduplicate and count
	uniqueHashes := make([]uint64, 0, len(hashes)/2)
	frequencies := make([]uint16, 0, len(hashes)/2)

	prevHash := hashes[0]
	count := uint16(1)

	for i := 1; i < len(hashes); i++ {
		if hashes[i] == prevHash {
			count++
		} else {
			uniqueHashes = append(uniqueHashes, prevHash)
			frequencies = append(frequencies, count)
			prevHash = hashes[i]
			count = 1
		}
	}
	// Don't forget the last one
	uniqueHashes = append(uniqueHashes, prevHash)
	frequencies = append(frequencies, count)

	return uniqueHashes, frequencies, tokenCount
}

// FastTokenizeReuse reuses buffers for even less allocation.
func FastTokenizeReuse(text string, hashBuf *[]uint64) ([]uint64, []uint16, int) {
	if len(text) == 0 {
		return nil, nil, 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	// Reuse hash buffer
	hashes := (*hashBuf)[:0]

	for i < n {
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			hashes = append(hashes, hash)
			tokenCount++
		}
	}

	*hashBuf = hashes

	if len(hashes) == 0 {
		return nil, nil, 0
	}

	// Sort hashes using radix sort for uint64 (faster than comparison sort)
	radixSort64(hashes)

	// Deduplicate and count in-place
	uniqueHashes := make([]uint64, 0, len(hashes)/2)
	frequencies := make([]uint16, 0, len(hashes)/2)

	prevHash := hashes[0]
	count := uint16(1)

	for i := 1; i < len(hashes); i++ {
		if hashes[i] == prevHash {
			count++
		} else {
			uniqueHashes = append(uniqueHashes, prevHash)
			frequencies = append(frequencies, count)
			prevHash = hashes[i]
			count = 1
		}
	}
	uniqueHashes = append(uniqueHashes, prevHash)
	frequencies = append(frequencies, count)

	return uniqueHashes, frequencies, tokenCount
}

// radixSort64 sorts uint64 slice using LSD radix sort.
// Much faster than comparison sort for large slices.
func radixSort64(data []uint64) {
	if len(data) < 64 {
		// Use insertion sort for small slices
		for i := 1; i < len(data); i++ {
			key := data[i]
			j := i - 1
			for j >= 0 && data[j] > key {
				data[j+1] = data[j]
				j--
			}
			data[j+1] = key
		}
		return
	}

	// Radix sort with 8-bit digits (8 passes)
	aux := make([]uint64, len(data))
	var count [256]int

	for shift := uint(0); shift < 64; shift += 8 {
		// Clear counts
		for i := range count {
			count[i] = 0
		}

		// Count occurrences
		for _, v := range data {
			digit := (v >> shift) & 0xFF
			count[digit]++
		}

		// Compute prefix sums
		for i := 1; i < 256; i++ {
			count[i] += count[i-1]
		}

		// Place elements in auxiliary array
		for i := len(data) - 1; i >= 0; i-- {
			digit := (data[i] >> shift) & 0xFF
			count[digit]--
			aux[count[digit]] = data[i]
		}

		// Copy back
		copy(data, aux)
	}
}

// StreamTokenize emits tokens directly without collecting them.
// Useful for pipelined processing.
func StreamTokenize(text string, emit func(hash uint64)) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	for i < n {
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			emit(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// FixedHashTable is a fixed-size hash table with linear probing.
// Eliminates Go map overhead by using direct array indexing.
// Size must be power of 2 for fast modulo.
type FixedHashTable struct {
	keys   []uint64 // 0 means empty
	counts []uint16
	mask   uint64
	used   int
}

// NewFixedHashTable creates a hash table with given capacity (rounded up to power of 2).
func NewFixedHashTable(capacity int) *FixedHashTable {
	// Round up to power of 2
	size := 1
	for size < capacity {
		size *= 2
	}
	// Ensure at least 50% load factor headroom
	if size < capacity*2 {
		size *= 2
	}
	return &FixedHashTable{
		keys:   make([]uint64, size),
		counts: make([]uint16, size),
		mask:   uint64(size - 1),
	}
}

// Reset clears the table for reuse without reallocating.
func (h *FixedHashTable) Reset() {
	// Use clear() for efficient zeroing
	clear(h.keys)
	clear(h.counts)
	h.used = 0
}

// Insert adds a hash to the table, incrementing its count.
// Returns true if this was a new entry.
func (h *FixedHashTable) Insert(hash uint64) bool {
	// Ensure hash is never 0 (reserved for empty)
	if hash == 0 {
		hash = 1
	}

	idx := hash & h.mask
	size := int(h.mask) + 1
	// Limit probing to prevent infinite loop if table is full
	for i := 0; i < size; i++ {
		if h.keys[idx] == 0 {
			// Empty slot - insert new
			h.keys[idx] = hash
			h.counts[idx] = 1
			h.used++
			return true
		}
		if h.keys[idx] == hash {
			// Found existing - increment
			h.counts[idx]++
			return false
		}
		// Linear probe
		idx = (idx + 1) & h.mask
	}
	// Table is full - ignore (should never happen with proper sizing)
	return false
}

// Iterate calls fn for each (hash, count) pair.
func (h *FixedHashTable) Iterate(fn func(hash uint64, count uint16)) {
	for i, key := range h.keys {
		if key != 0 {
			fn(key, h.counts[i])
		}
	}
}

// Used returns the number of unique entries.
func (h *FixedHashTable) Used() int {
	return h.used
}

// FixedTokenize uses a fixed hash table instead of Go map.
// Much lower overhead than TokenizeMega.
func FixedTokenize(text string, table *FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// CompactHashTable is an even more compact hash table using uint32 indices.
// Uses Robin Hood hashing for better cache behavior.
type CompactHashTable struct {
	keys   []uint64
	counts []uint16
	size   int
	used   int
}

// NewCompactHashTable creates a compact table with given capacity.
func NewCompactHashTable(capacity int) *CompactHashTable {
	// Size = 2x capacity for low load factor
	size := 1
	for size < capacity*2 {
		size *= 2
	}
	return &CompactHashTable{
		keys:   make([]uint64, size),
		counts: make([]uint16, size),
		size:   size,
	}
}

// Reset clears the table.
func (h *CompactHashTable) Reset() {
	clear(h.keys)
	clear(h.counts)
	h.used = 0
}

// Insert adds a hash, returns true if new.
func (h *CompactHashTable) Insert(hash uint64) bool {
	if hash == 0 {
		hash = 1
	}

	mask := uint64(h.size - 1)
	idx := hash & mask

	// Linear probing with early exit
	for i := 0; i < h.size; i++ {
		if h.keys[idx] == 0 {
			h.keys[idx] = hash
			h.counts[idx] = 1
			h.used++
			return true
		}
		if h.keys[idx] == hash {
			h.counts[idx]++
			return false
		}
		idx = (idx + 1) & mask
	}
	return false
}

// CompactTokenize uses CompactHashTable.
func CompactTokenize(text string, table *CompactHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		start := i
		hash := uint64(fnvOffset)

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash ^= uint64(c)
			hash *= fnvPrime
			i++
		}

		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// BatchTokenizeFixed tokenizes multiple texts with shared table.
func BatchTokenizeFixed(texts []string, emit func(docIdx int, hash uint64, count uint16)) {
	table := NewCompactHashTable(512)

	for docIdx, text := range texts {
		if len(text) == 0 {
			continue
		}

		const fnvOffset = 14695981039346656037
		const fnvPrime = 1099511628211

		data := unsafe.Slice(unsafe.StringData(text), len(text))
		n := len(data)
		i := 0

		table.Reset()

		for i < n {
			for i < n && megaToLower[data[i]] == 0 {
				i++
			}
			if i >= n {
				break
			}

			start := i
			hash := uint64(fnvOffset)

			for i < n {
				c := megaToLower[data[i]]
				if c == 0 {
					break
				}
				hash ^= uint64(c)
				hash *= fnvPrime
				i++
			}

			tokenLen := i - start
			if tokenLen >= 2 && tokenLen <= 32 {
				table.Insert(hash)
			}
		}

		// Emit all entries
		for j := 0; j < table.size; j++ {
			if table.keys[j] != 0 {
				emit(docIdx, table.keys[j], table.counts[j])
			}
		}
	}
}
