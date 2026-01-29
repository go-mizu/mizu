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
