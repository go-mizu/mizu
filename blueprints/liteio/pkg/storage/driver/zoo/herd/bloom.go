package herd

import (
	"sync/atomic"
)

// bloomFilter is a lock-free concurrent bloom filter for fast negative lookups.
// Uses atomic OR for adds and plain reads for queries (safe because bits are only set, never cleared).
type bloomFilter struct {
	bits    []atomic.Uint64
	numBits uint64
	numHash int
}

// newBloomFilter creates a bloom filter sized for expectedItems with target FPR ~0.1%.
// Uses 10 bits per item and 7 hash functions.
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

// add inserts a key into the bloom filter. Lock-free via atomic OR.
func (bf *bloomFilter) add(bucket, key string) {
	h1, h2 := bloomHash(bucket, key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		word := bit / 64
		mask := uint64(1) << (bit % 64)
		bf.bits[word].Or(mask)
	}
}

// mayContain returns true if the key might be in the set, false if definitely not.
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

// bloomHash computes two independent FNV-1a hashes for double hashing.
func bloomHash(bucket, key string) (uint64, uint64) {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211

	// h1: standard FNV-1a
	h1 := uint64(offset64)
	for i := 0; i < len(bucket); i++ {
		h1 ^= uint64(bucket[i])
		h1 *= prime64
	}
	h1 ^= 0 // null separator
	h1 *= prime64
	for i := 0; i < len(key); i++ {
		h1 ^= uint64(key[i])
		h1 *= prime64
	}

	// h2: FNV-1a with different seed (XOR with h1)
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

	// Ensure h2 is odd (for better distribution in double hashing).
	h2 |= 1

	return h1, h2
}
