package kestrel

import "sync"

// ---------------------------------------------------------------------------
// Value chunk allocator (sync.Pool, per-P locality)
//
// Sub-allocates value bytes from 4MB chunks pooled via sync.Pool.
// sync.Pool uses per-P caches, giving zero lock contention between
// goroutines on different Ps. Values > 2MB bypass the pool and use
// individual make() allocations.
// ---------------------------------------------------------------------------

const (
	valueChunkSize = 4 << 20 // 4MB per chunk
	valueChunkMax  = 2 << 20 // threshold: above this use individual alloc
)

type valueChunk struct {
	buf []byte
	off int
}

var valueChunkPool = sync.Pool{
	New: func() any { return &valueChunk{buf: make([]byte, valueChunkSize)} },
}

// allocValue sub-allocates size bytes for value storage.
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
