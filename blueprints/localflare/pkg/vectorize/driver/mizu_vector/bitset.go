package mizu_vector

import "sync"

// bitset is a compact bit array for tracking visited nodes.
// Much faster than map[int]bool or []bool for sparse access patterns.
type bitset struct {
	words []uint64
	size  int
}

// newBitset creates a bitset that can hold n bits.
func newBitset(n int) *bitset {
	numWords := (n + 63) / 64
	return &bitset{
		words: make([]uint64, numWords),
		size:  n,
	}
}

// Set sets bit i to 1.
func (b *bitset) Set(i int32) {
	b.words[i/64] |= 1 << (i % 64)
}

// Test returns true if bit i is set.
func (b *bitset) Test(i int32) bool {
	return b.words[i/64]&(1<<(i%64)) != 0
}

// TestAndSet atomically tests and sets bit i. Returns true if was unset.
func (b *bitset) TestAndSet(i int32) bool {
	word := i / 64
	bit := uint64(1) << (i % 64)
	if b.words[word]&bit != 0 {
		return false
	}
	b.words[word] |= bit
	return true
}

// Clear resets all bits to 0.
func (b *bitset) Clear() {
	for i := range b.words {
		b.words[i] = 0
	}
}

// Size returns the capacity in bits.
func (b *bitset) Size() int {
	return b.size
}

// bitsetPool pools bitsets to reduce allocations.
type bitsetPool struct {
	pool sync.Pool
	size int
}

// newBitsetPool creates a pool for bitsets of given size.
func newBitsetPool(size int) *bitsetPool {
	return &bitsetPool{
		size: size,
		pool: sync.Pool{
			New: func() any {
				return newBitset(size)
			},
		},
	}
}

// Get retrieves a cleared bitset from the pool.
func (p *bitsetPool) Get() *bitset {
	b := p.pool.Get().(*bitset)
	// Ensure correct size (pool may have old size)
	if b.size < p.size {
		b = newBitset(p.size)
	} else {
		b.Clear()
	}
	return b
}

// Put returns a bitset to the pool.
func (p *bitsetPool) Put(b *bitset) {
	p.pool.Put(b)
}

// UpdateSize updates the pool size for new allocations.
func (p *bitsetPool) UpdateSize(size int) {
	p.size = size
}
