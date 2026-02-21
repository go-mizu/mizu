package herd

import (
	"sync/atomic"
	"unsafe"
)

// slabChunkSize is the size of each slab chunk.
// 64MB gives good amortization: one GC-visible object per 64MB vs millions of tiny slices.
const slabChunkSize = 64 * 1024 * 1024

// slabChunk is a single pre-allocated memory region.
type slabChunk struct {
	data []byte
	pos  atomic.Int64
	next unsafe.Pointer // *slabChunk, atomic
}

// slabArena is a lock-free bump allocator for inline value data.
// Each stripe owns one arena. Allocation is a single atomic Add (fast path).
// When a chunk fills, a new one is prepended via CAS (rare, ~once per 64MB).
type slabArena struct {
	head unsafe.Pointer // *slabChunk, atomic
}

func newSlabArena() *slabArena {
	chunk := &slabChunk{data: make([]byte, slabChunkSize)}
	a := &slabArena{}
	atomic.StorePointer(&a.head, unsafe.Pointer(chunk))
	return a
}

// alloc sub-allocates size bytes from the arena. Lock-free fast path.
func (a *slabArena) alloc(size int) []byte {
	if size <= 0 {
		return nil
	}
	sz := int64(size)
	for {
		hp := atomic.LoadPointer(&a.head)
		chunk := (*slabChunk)(hp)
		pos := chunk.pos.Add(sz) - sz
		if pos >= 0 && pos+sz <= int64(len(chunk.data)) {
			return chunk.data[pos : pos+sz]
		}
		// Chunk full — allocate new one and CAS it in.
		newChunk := &slabChunk{data: make([]byte, slabChunkSize)}
		newChunk.next = hp
		atomic.CompareAndSwapPointer(&a.head, hp, unsafe.Pointer(newChunk))
		// Retry regardless of CAS result (another goroutine may have won).
	}
}
