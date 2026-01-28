package algo

import (
	"math"
	"math/bits"
)

// EliasFano implements quasi-succinct Elias-Fano encoding for sorted integers.
// This achieves near-optimal compression while supporting O(1) random access.
// Reference: https://www.antoniomallia.it/sorted-integers-compression-with-elias-fano-encoding.html

// EliasFano represents an Elias-Fano encoded sequence of sorted integers.
type EliasFano struct {
	LowBits   []uint64 // Packed low bits (l bits per integer)
	HighBits  []uint64 // Unary-encoded high bits with select support
	LowLen    int      // Number of bits in low part
	N         int      // Number of elements
	Universe  uint64   // Maximum possible value + 1

	// Select acceleration structures
	selectSamples []int // Samples for fast select1
	sampleRate    int   // Sample every N bits
}

// NewEliasFano creates an Elias-Fano encoding of sorted integers.
func NewEliasFano(sorted []uint32) *EliasFano {
	if len(sorted) == 0 {
		return &EliasFano{}
	}

	n := len(sorted)
	u := uint64(sorted[n-1]) + 1 // Universe size

	// Calculate optimal low bits length: l = max(0, floor(log2(u/n)))
	var lowLen int
	if u > uint64(n) {
		lowLen = int(math.Floor(math.Log2(float64(u) / float64(n))))
	}
	if lowLen > 32 {
		lowLen = 32
	}

	// High bits length: number of high bit values + n (for unary encoding)
	highBitsLen := n + int(u>>lowLen) + 1

	ef := &EliasFano{
		LowBits:    make([]uint64, (n*lowLen+63)/64),
		HighBits:   make([]uint64, (highBitsLen+63)/64),
		LowLen:     lowLen,
		N:          n,
		Universe:   u,
		sampleRate: 64, // Sample every 64 ones
	}

	lowMask := uint64(1<<lowLen) - 1
	highPos := 0

	for i, v := range sorted {
		val := uint64(v)

		// Store low bits
		if lowLen > 0 {
			low := val & lowMask
			ef.setLowBits(i, low)
		}

		// Store high bits in unary encoding
		// Position = i + high_value
		high := int(val >> lowLen)
		pos := i + high

		// Handle gaps from previous position
		for highPos < pos {
			highPos++
		}

		// Set the 1 bit
		ef.setBit(ef.HighBits, pos)
		highPos = pos + 1
	}

	// Build select samples for O(1) access
	ef.buildSelectSamples()

	return ef
}

func (ef *EliasFano) setLowBits(i int, value uint64) {
	if ef.LowLen == 0 {
		return
	}

	bitPos := i * ef.LowLen
	wordIdx := bitPos / 64
	bitIdx := bitPos % 64

	ef.LowBits[wordIdx] |= value << bitIdx

	// Handle overflow to next word
	if bitIdx+ef.LowLen > 64 && wordIdx+1 < len(ef.LowBits) {
		ef.LowBits[wordIdx+1] |= value >> (64 - bitIdx)
	}
}

func (ef *EliasFano) getLowBits(i int) uint64 {
	if ef.LowLen == 0 {
		return 0
	}

	bitPos := i * ef.LowLen
	wordIdx := bitPos / 64
	bitIdx := bitPos % 64

	mask := uint64(1<<ef.LowLen) - 1
	result := (ef.LowBits[wordIdx] >> bitIdx) & mask

	// Handle overflow from next word
	if bitIdx+ef.LowLen > 64 && wordIdx+1 < len(ef.LowBits) {
		remaining := bitIdx + ef.LowLen - 64
		result |= (ef.LowBits[wordIdx+1] & ((1 << remaining) - 1)) << (64 - bitIdx)
	}

	return result
}

func (ef *EliasFano) setBit(bits []uint64, pos int) {
	if pos/64 < len(bits) {
		bits[pos/64] |= 1 << (pos % 64)
	}
}

func (ef *EliasFano) getBit(bits []uint64, pos int) bool {
	if pos/64 >= len(bits) {
		return false
	}
	return (bits[pos/64] & (1 << (pos % 64))) != 0
}

func (ef *EliasFano) buildSelectSamples() {
	// Count ones and build samples
	sampleCount := (ef.N + ef.sampleRate - 1) / ef.sampleRate
	ef.selectSamples = make([]int, sampleCount)

	oneCount := 0
	sampleIdx := 0

	for wordIdx, word := range ef.HighBits {
		if word == 0 {
			continue
		}

		for bitIdx := 0; bitIdx < 64; bitIdx++ {
			if (word & (1 << bitIdx)) != 0 {
				if oneCount%ef.sampleRate == 0 && sampleIdx < len(ef.selectSamples) {
					ef.selectSamples[sampleIdx] = wordIdx*64 + bitIdx
					sampleIdx++
				}
				oneCount++
			}
		}
	}
}

// select1 returns the position of the i-th 1 bit (0-indexed).
func (ef *EliasFano) select1(i int) int {
	if i < 0 || i >= ef.N {
		return -1
	}

	// Initialize sampleRate if not set (after gob decode)
	if ef.sampleRate == 0 {
		ef.sampleRate = 64
	}

	// Start from sample point
	sampleIdx := i / ef.sampleRate
	var startPos int
	if sampleIdx < len(ef.selectSamples) {
		startPos = ef.selectSamples[sampleIdx]
	}

	// Count ones from sample point
	onesNeeded := i - sampleIdx*ef.sampleRate

	wordIdx := startPos / 64
	bitIdx := startPos % 64

	// Mask out bits before startPos in first word
	word := ef.HighBits[wordIdx] >> bitIdx << bitIdx

	for {
		popCount := bits.OnesCount64(word)

		if popCount > onesNeeded {
			// The answer is in this word
			// Find the (onesNeeded+1)-th bit
			for j := 0; j < 64; j++ {
				if (word & (1 << j)) != 0 {
					if onesNeeded == 0 {
						return wordIdx*64 + j
					}
					onesNeeded--
				}
			}
		}

		onesNeeded -= popCount
		wordIdx++
		if wordIdx >= len(ef.HighBits) {
			return -1
		}
		word = ef.HighBits[wordIdx]
	}
}

// Get returns the i-th element (0-indexed) in O(1) time.
func (ef *EliasFano) Get(i int) uint32 {
	if i < 0 || i >= ef.N {
		return 0
	}

	// High bits: position of i-th 1 minus i
	pos := ef.select1(i)
	high := pos - i

	// Low bits: direct access
	low := ef.getLowBits(i)

	return uint32((uint64(high) << ef.LowLen) | low)
}

// Size returns the number of elements.
func (ef *EliasFano) Size() int {
	return ef.N
}

// Bytes returns the total memory used in bytes.
func (ef *EliasFano) Bytes() int {
	return len(ef.LowBits)*8 + len(ef.HighBits)*8 + len(ef.selectSamples)*8 + 48
}

// Decode returns all values as a slice.
func (ef *EliasFano) Decode() []uint32 {
	result := make([]uint32, ef.N)
	for i := 0; i < ef.N; i++ {
		result[i] = ef.Get(i)
	}
	return result
}

// Iterator provides sequential access to Elias-Fano encoded values.
type EFIterator struct {
	ef      *EliasFano
	pos     int // Current position in sequence
	highPos int // Current position in high bits
}

// NewIterator creates an iterator for sequential access.
func (ef *EliasFano) NewIterator() *EFIterator {
	return &EFIterator{ef: ef, pos: 0, highPos: 0}
}

// Next returns the next value and advances the iterator.
func (it *EFIterator) Next() (uint32, bool) {
	if it.pos >= it.ef.N {
		return 0, false
	}

	// Find next 1 bit in high bits
	for !it.ef.getBit(it.ef.HighBits, it.highPos) {
		it.highPos++
	}

	high := it.highPos - it.pos
	low := it.ef.getLowBits(it.pos)

	it.pos++
	it.highPos++

	return uint32((uint64(high) << it.ef.LowLen) | low), true
}

// Skip advances to the first value >= target.
func (it *EFIterator) Skip(target uint32) (uint32, bool) {
	// Binary search using Get for random access
	left, right := it.pos, it.ef.N-1

	for left < right {
		mid := (left + right) / 2
		if it.ef.Get(mid) < target {
			left = mid + 1
		} else {
			right = mid
		}
	}

	if left >= it.ef.N {
		return 0, false
	}

	it.pos = left
	it.highPos = it.ef.select1(left)

	return it.ef.Get(left), true
}

// Reset resets the iterator to the beginning.
func (it *EFIterator) Reset() {
	it.pos = 0
	it.highPos = 0
}
