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

func mmapFree(data []byte) { _ = syscall.Munmap(data) }

// ---------------------------------------------------------------------------
// Data arena — lock-free bump allocator over mmap'd memory
//
// All data written here is GC-invisible: mmap'd pages are not tracked by the
// Go heap, so the garbage collector never scans or marks arena contents.
// This eliminates the #1 CPU bottleneck in v5 (39% GC scanning).
//
// Layout per record: [compositeKey bytes][contentType bytes][value bytes]
// ---------------------------------------------------------------------------

const (
	arenaChunkSize  = 128 << 20         // 128MB per mmap chunk
	arenaChunkShift = 27                // log2(128 << 20)
	arenaChunkMask  = arenaChunkSize - 1 // for local offset extraction
)

type chunkList struct {
	chunks [][]byte
}

type dataArena struct {
	pos atomic.Int64               // global write position
	mu  sync.Mutex                 // protects chunk list growth
	cl  atomic.Pointer[chunkList]  // immutable chunk list, replaced on grow
}

func newDataArena() *dataArena {
	a := &dataArena{}
	data, err := mmapAlloc(arenaChunkSize)
	if err != nil {
		panic("kestrel: mmap failed: " + err.Error())
	}
	a.cl.Store(&chunkList{chunks: [][]byte{data}})
	return a
}

// alloc reserves size bytes in the arena.
// Returns the global offset and a byte slice for writing.
// The returned slice points into mmap'd memory (GC-invisible).
func (a *dataArena) alloc(size int) (int64, []byte) {
	if size <= 0 {
		return 0, nil
	}
	sz := int64(size)
	for {
		off := a.pos.Add(sz) - sz
		ci := int(off >> arenaChunkShift)
		lo := int(off & arenaChunkMask)

		if lo+size > arenaChunkSize {
			// Spans chunk boundary — waste remaining space, retry.
			// Cost: at most `size` bytes per 128MB, negligible.
			continue
		}

		a.ensureChunk(ci)
		cl := a.cl.Load()
		return off, cl.chunks[ci][lo : lo+size : lo+size]
	}
}

// bytes returns a slice at the given global offset and length.
func (a *dataArena) bytes(off int64, length int) []byte {
	ci := int(off >> arenaChunkShift)
	lo := int(off & arenaChunkMask)
	cl := a.cl.Load()
	return cl.chunks[ci][lo : lo+length]
}

// str returns an unsafe string at the given global offset and length.
// The string is valid as long as the arena is alive (until Close).
func (a *dataArena) str(off int64, length int) string {
	b := a.bytes(off, length)
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// readCtAndValue reads content-type string and value bytes from a single
// contiguous region starting at off. Uses one atomic chunk list load instead
// of two separate arena.str + arena.bytes calls.
func (a *dataArena) readCtAndValue(off int64, ctLen, valLen int) (string, []byte) {
	ci := int(off >> arenaChunkShift)
	lo := int(off & arenaChunkMask)
	cl := a.cl.Load()
	chunk := cl.chunks[ci]
	ctSlice := chunk[lo : lo+ctLen]
	valSlice := chunk[lo+ctLen : lo+ctLen+valLen]
	return unsafe.String(unsafe.SliceData(ctSlice), ctLen), valSlice
}

// readCt reads only the content-type string. Single atomic load.
func (a *dataArena) readCt(off int64, ctLen int) string {
	ci := int(off >> arenaChunkShift)
	lo := int(off & arenaChunkMask)
	cl := a.cl.Load()
	ctSlice := cl.chunks[ci][lo : lo+ctLen]
	return unsafe.String(unsafe.SliceData(ctSlice), ctLen)
}

func (a *dataArena) ensureChunk(idx int) {
	cl := a.cl.Load()
	if idx < len(cl.chunks) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	cl = a.cl.Load()
	for idx >= len(cl.chunks) {
		data, err := mmapAlloc(arenaChunkSize)
		if err != nil {
			panic("kestrel: mmap failed: " + err.Error())
		}
		newChunks := make([][]byte, len(cl.chunks)+1)
		copy(newChunks, cl.chunks)
		newChunks[len(cl.chunks)] = data
		a.cl.Store(&chunkList{chunks: newChunks})
		cl = a.cl.Load()
	}
}

func (a *dataArena) close() {
	cl := a.cl.Load()
	for _, c := range cl.chunks {
		mmapFree(c)
	}
}
