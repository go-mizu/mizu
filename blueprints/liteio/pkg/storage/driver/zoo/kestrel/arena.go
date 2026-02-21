package kestrel

import "sync"

// ---------------------------------------------------------------------------
// Value chunk allocator (sync.Pool, per-P locality, zero lock contention)
//
// Replaces mmap arena. Benefits:
//   - Per-P chunk caches via sync.Pool (zero contention on hot path)
//   - No madvise overhead (eliminated 10% CPU from baseline)
//   - Values ≤ 2MB sub-allocate from 4MB chunks (~64x fewer GC objects)
//   - Values > 2MB get individual allocations
//   - GC sees only chunk metadata, not value contents ([]byte has no pointers)
// ---------------------------------------------------------------------------

const (
	valueChunkSize = 4 << 20 // 4MB per chunk
	valueChunkMax  = 2 << 20 // values > 2MB use individual make()
)

type valueChunk struct {
	buf []byte
	off int
}

var valueChunkPool = sync.Pool{
	New: func() any { return &valueChunk{buf: make([]byte, valueChunkSize)} },
}

// allocValue sub-allocates size bytes from a per-P chunk buffer.
// Lock-free: sync.Pool uses per-P caches, so concurrent goroutines on
// different Ps never contend. Same-P goroutines serialize naturally.
func allocValue(size int) []byte {
	if size <= 0 {
		return nil
	}
	if size > valueChunkMax {
		return make([]byte, size)
	}
	vc := valueChunkPool.Get().(*valueChunk)
	if vc.off+size > len(vc.buf) {
		vc.buf = make([]byte, valueChunkSize)
		vc.off = 0
	}
	s := vc.buf[vc.off : vc.off+size : vc.off+size]
	vc.off += size
	valueChunkPool.Put(vc)
	return s
}
