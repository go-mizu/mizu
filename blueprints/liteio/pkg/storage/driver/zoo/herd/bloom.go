package herd

import (
	"math/bits"
	"sync/atomic"
	"unsafe"
)

// bloomFilter is a lock-free concurrent bloom filter for fast negative lookups.
// Uses atomic OR for adds and plain reads for queries (safe because bits are only set, never cleared).
//
// v4 optimization: Uses mmap-backed memory instead of Go heap []atomic.Uint64.
// The mmap region is invisible to the Go GC scanner, eliminating ~3 GB of heap
// that was previously scanned by tryDeferToSpanScan + scanObject.
// unsafe.Slice creates an atomic.Uint64 view over the mmap'd memory for fast
// atomic OR operations (same single-instruction performance as v3).
type bloomFilter struct {
	data    []byte           // mmap-backed storage (ownership for munmap)
	bits    []atomic.Uint64  // view into data via unsafe.Slice — GC sees header only, not data
	mmaped  bool             // true if data was allocated via mmap
	numBits uint64
	numHash int
}

// newBloomFilter creates a bloom filter sized for expectedItems with target FPR ~0.1%.
// Uses 10 bits per item and 7 hash functions.
// v4: backing memory is mmap'd (not Go heap), invisible to GC.
func newBloomFilter(expectedItems int) *bloomFilter {
	if expectedItems < 1024 {
		expectedItems = 1024
	}
	numBits := uint64(expectedItems) * 10
	numBits = (numBits + 63) &^ 63
	numWords := numBits / 64

	size := int(numWords * 8) // bytes needed
	data, err := mmapAlloc(size)
	mmaped := err == nil
	if err != nil {
		data = make([]byte, size)
	}

	// Create atomic.Uint64 view over the mmap'd memory.
	// Safe because: (1) atomic.Uint64 is exactly 8 bytes (same layout as uint64),
	// (2) mmap returns page-aligned memory (always 8-byte aligned),
	// (3) GC sees the slice header but NOT the data (mmap is outside Go heap).
	atomicBits := unsafe.Slice((*atomic.Uint64)(unsafe.Pointer(&data[0])), numWords)

	return &bloomFilter{
		data:    data,
		bits:    atomicBits,
		mmaped:  mmaped,
		numBits: numBits,
		numHash: 7,
	}
}

// add inserts a key into the bloom filter. Lock-free via atomic OR.
func (bf *bloomFilter) add(bucket, key string) {
	h1, h2 := bloomHashFast(bucket, key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		bf.bits[bit/64].Or(1 << (bit % 64))
	}
}

// mayContain returns true if the key might be in the set, false if definitely not.
func (bf *bloomFilter) mayContain(bucket, key string) bool {
	h1, h2 := bloomHashFast(bucket, key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		if bf.bits[bit/64].Load()&(1<<(bit%64)) == 0 {
			return false
		}
	}
	return true
}

// close releases the mmap'd memory back to the OS.
func (bf *bloomFilter) close() {
	if bf.mmaped && bf.data != nil {
		mmapFree(bf.data)
		bf.data = nil
		bf.bits = nil
	}
}

// wymix is a fast mixing function from wyhash.
// Two multiply + XOR operations provide excellent avalanche.
func wymix(a, b uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	return hi ^ lo
}

// bloomHashFast computes two independent hashes using wyhash-style mixing.
// Much faster than double FNV-1a: single data pass + algebraic mixing.
func bloomHashFast(bucket, key string) (uint64, uint64) {
	// Seed values (arbitrary primes)
	const (
		s0 = 0xa0761d6478bd642f
		s1 = 0xe7037ed1a0b428db
		s2 = 0x8ebc6af09c88c6e3
		s3 = 0x589965cc75374cc3
	)

	// Single-pass hash over bucket + separator + key.
	h := uint64(s0)
	for i := 0; i < len(bucket); i++ {
		h = (h ^ uint64(bucket[i])) * s1
	}
	h ^= s2 // separator
	for i := 0; i < len(key); i++ {
		h = (h ^ uint64(key[i])) * s1
	}

	// Generate two independent hashes via mixing.
	h1 := wymix(h, s3)
	h2 := wymix(h, s0) | 1 // ensure odd for better double-hashing distribution

	return h1, h2
}
