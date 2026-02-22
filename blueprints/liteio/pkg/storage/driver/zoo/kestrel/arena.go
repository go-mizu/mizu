package kestrel

import (
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Mmap helpers
// ---------------------------------------------------------------------------

func mmapAlloc(size int) ([]byte, error) {
	return syscall.Mmap(-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE)
}

func mmapFree(data []byte) error {
	return syscall.Munmap(data)
}

// ---------------------------------------------------------------------------
// Value arena (mmap'd bump-pointer allocator for value data)
//
// Allocation is a single atomic Add (lock-free fast path).
// When a chunk fills, a new one is prepended via CAS (rare, ~once per 128MB).
// Uses mmap(MAP_ANON|MAP_PRIVATE) instead of make([]byte):
//   - Zero-filled on demand by OS (eliminates runtime.memclrNoHeapPointers)
//   - Not tracked by GC scanner (eliminates tryDeferToSpanScan)
//   - Memory returned to OS on munmap
// ---------------------------------------------------------------------------

const arenaChunkSize = 128 * 1024 * 1024 // 128MB

type arenaChunk struct {
	data []byte
	pos  atomic.Int64
	next unsafe.Pointer // *arenaChunk
}

type valueArena struct {
	head unsafe.Pointer // *arenaChunk, atomic
}

func newValueArena() (*valueArena, error) {
	data, err := mmapAlloc(arenaChunkSize)
	if err != nil {
		return nil, err
	}
	chunk := &arenaChunk{data: data}
	a := &valueArena{}
	atomic.StorePointer(&a.head, unsafe.Pointer(chunk))
	return a, nil
}

// alloc sub-allocates size bytes from the arena. Lock-free fast path.
func (a *valueArena) alloc(size int) []byte {
	if size <= 0 {
		return nil
	}
	sz := int64(size)
	for {
		hp := atomic.LoadPointer(&a.head)
		chunk := (*arenaChunk)(hp)
		pos := chunk.pos.Add(sz) - sz
		if pos >= 0 && pos+sz <= int64(len(chunk.data)) {
			return chunk.data[pos : pos+sz]
		}
		// Chunk full — allocate new one via mmap and CAS it in.
		newSize := arenaChunkSize
		if size > newSize {
			newSize = size + 4096
		}
		data, err := mmapAlloc(newSize)
		if err != nil {
			data = make([]byte, newSize) // fallback to heap
		}
		newChunk := &arenaChunk{data: data}
		newChunk.next = hp
		atomic.CompareAndSwapPointer(&a.head, hp, unsafe.Pointer(newChunk))
		// Retry regardless of CAS result.
	}
}

func (a *valueArena) close() {
	hp := atomic.LoadPointer(&a.head)
	for hp != nil {
		chunk := (*arenaChunk)(hp)
		mmapFree(chunk.data)
		hp = chunk.next
	}
}

// ---------------------------------------------------------------------------
// Value chunk allocator (sync.Pool, per-P locality for large values)
//
// For values > 2MB where arena atomic contention matters,
// this provides per-P locality via sync.Pool.
// For values ≤ 2MB, use the mmap arena (lower overhead).
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

// allocValue sub-allocates size bytes for values > 2MB.
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
