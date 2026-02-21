package herd

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

// slabChunkSize is the size of each slab chunk.
// 128MB mmap chunks: zero-filled on demand by kernel (no memclr),
// invisible to GC scanner (no tryDeferToSpanScan), huge-page eligible.
const slabChunkSize = 128 * 1024 * 1024

// slabChunk is a single pre-allocated memory region backed by anonymous mmap.
type slabChunk struct {
	data []byte
	pos  atomic.Int64
	next unsafe.Pointer // *slabChunk, atomic
}

// slabArena is a lock-free bump allocator for inline value data.
// Each stripe owns one arena. Allocation is a single atomic Add (fast path).
// When a chunk fills, a new one is prepended via CAS (rare, ~once per 128MB).
//
// v3 optimization: Uses mmap(MAP_ANON|MAP_PRIVATE) instead of make([]byte).
// Benefits:
//   - Zero-filled on demand by OS (eliminates runtime.memclrNoHeapPointers, was 9.22% CPU)
//   - Not tracked by GC scanner (eliminates tryDeferToSpanScan overhead)
//   - Eligible for transparent huge pages (2MB TLB entries)
//   - Memory returned to OS on munmap (vs Go heap which may never release)
type slabArena struct {
	head unsafe.Pointer // *slabChunk, atomic
}

// mmapAlloc allocates a zero-filled memory region via anonymous mmap.
// The region is invisible to the Go GC and zero-filled on demand by the kernel.
func mmapAlloc(size int) ([]byte, error) {
	data, err := syscall.Mmap(-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// mmapFree releases a previously mmap'd region back to the OS.
func mmapFree(data []byte) error {
	return syscall.Munmap(data)
}

func newSlabArena() *slabArena {
	data, err := mmapAlloc(slabChunkSize)
	if err != nil {
		// Fallback to heap allocation if mmap fails.
		data = make([]byte, slabChunkSize)
	}
	chunk := &slabChunk{data: data}
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
		// Chunk full — allocate new one via mmap and CAS it in.
		data, err := mmapAlloc(slabChunkSize)
		if err != nil {
			data = make([]byte, slabChunkSize)
		}
		newChunk := &slabChunk{data: data}
		newChunk.next = hp
		atomic.CompareAndSwapPointer(&a.head, hp, unsafe.Pointer(newChunk))
		// Retry regardless of CAS result (another goroutine may have won).
	}
}
