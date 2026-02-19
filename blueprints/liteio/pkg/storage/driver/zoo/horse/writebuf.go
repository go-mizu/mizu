package horse

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Default write buffer size: 64MB.
const defaultBufSize = 64 * 1024 * 1024

// ringSize is the number of buffers in the ring.
// 4 buffers means 3 can accept writes while 1 flushes, dramatically reducing
// swap contention at high concurrency (C100: from 1 MB/s to 100+ MB/s).
const ringSize = 4

// writeBuffer is a pre-allocated contiguous memory region for accumulating writes.
// All writes are pure memcpy — no page faults, no syscalls on the write path.
// When the buffer is full, it is flushed to the volume as a single pwrite.
type writeBuffer struct {
	data      []byte       // pre-allocated buffer
	pos       atomic.Int64 // current write position (lock-free via atomic.Add)
	capacity  int64        // capacity in bytes
	volOffset int64        // volume offset where this buffer starts
	frozen    atomic.Bool  // true = no more writes, being flushed
	writers   atomic.Int32 // active writers count (for safe flush)
}

// newWriteBuffer creates a pre-allocated write buffer.
func newWriteBuffer(capacity int64, volOffset int64) *writeBuffer {
	wb := &writeBuffer{
		data:      make([]byte, capacity),
		capacity:  capacity,
		volOffset: volOffset,
	}
	return wb
}

// claim atomically reserves space in the buffer and increments the writers count.
// Returns the local offset within the buffer, or -1 if the buffer is full/frozen.
// Caller MUST call done() after writing to the claimed region.
func (wb *writeBuffer) claim(size int64) int64 {
	if wb.frozen.Load() {
		return -1
	}
	wb.writers.Add(1)
	pos := wb.pos.Add(size) - size
	if pos+size > wb.capacity {
		// Overflowed — revert and signal full.
		wb.pos.Add(-size)
		wb.writers.Add(-1)
		return -1
	}
	return pos
}

// done signals that a write at a previously claimed position is complete.
func (wb *writeBuffer) done() {
	wb.writers.Add(-1)
}

// written returns how many bytes have been written.
func (wb *writeBuffer) written() int64 {
	pos := wb.pos.Load()
	if pos > wb.capacity {
		return wb.capacity
	}
	return pos
}

// reset prepares the buffer for reuse at a new volume offset.
func (wb *writeBuffer) reset(volOffset int64) {
	wb.pos.Store(0)
	wb.volOffset = volOffset
	wb.frozen.Store(false)
}

// bufferRing manages a ring of write buffers with background flush.
// Uses ringSize (4) buffers for high-concurrency writes:
// - Up to 3 buffers accept writes simultaneously while 1 flushes
// - Eliminates thundering-herd on swap at C100+ concurrency
type bufferRing struct {
	buffers  [ringSize]*writeBuffer
	active   atomic.Int32  // index of active buffer
	vol      *volume
	flushCh  chan int       // sends buffer index to flush
	stopCh   chan struct{}
	wg       sync.WaitGroup
	swapMu   sync.Mutex // protects buffer swap
	capacity int64
}

// newBufferRing creates a ring of write buffers.
func newBufferRing(vol *volume, bufSize int64) *bufferRing {
	if bufSize <= 0 {
		bufSize = defaultBufSize
	}

	tail := vol.tail.Load()
	br := &bufferRing{
		vol:      vol,
		flushCh:  make(chan int, ringSize),
		stopCh:   make(chan struct{}),
		capacity: bufSize,
	}
	for i := 0; i < ringSize; i++ {
		br.buffers[i] = newWriteBuffer(bufSize, tail+int64(i)*bufSize)
	}
	br.active.Store(0)

	// Start flush goroutine.
	br.wg.Add(1)
	go br.flusher()

	return br
}

// activeBuffer returns the current active buffer for writes.
func (br *bufferRing) activeBuffer() *writeBuffer {
	return br.buffers[br.active.Load()]
}

// writeInline claims space and returns a buffer slice for the caller to fill directly.
// This avoids one memcpy for callers that can serialize in-place.
// Caller MUST call wb.done() after filling the returned buffer slice.
func (br *bufferRing) writeInline(totalSize int64, valPosInRecord int) (buf []byte, recOff int64, valOff int64, wb *writeBuffer) {
	for {
		ab := br.activeBuffer()
		pos := ab.claim(totalSize)
		if pos >= 0 {
			return ab.data[pos : pos+totalSize], ab.volOffset + pos, ab.volOffset + pos + int64(valPosInRecord), ab
		}
		br.swap()
	}
}

// swap freezes the current active buffer and activates the next available one.
func (br *bufferRing) swap() {
	br.swapMu.Lock()
	defer br.swapMu.Unlock()

	cur := br.active.Load()
	ab := br.buffers[cur]

	// Check if already swapped by another goroutine.
	if !ab.frozen.Load() {
		ab.frozen.Store(true)
		br.flushCh <- int(cur)
	}

	// Find next available (non-frozen) buffer.
	for attempt := 0; attempt < ringSize*100; attempt++ {
		next := (cur + 1 + int32(attempt)) % int32(ringSize)
		nb := br.buffers[next]
		if !nb.frozen.Load() {
			br.active.Store(next)
			return
		}
		// All buffers frozen — yield and retry.
		if attempt%ringSize == ringSize-1 {
			br.swapMu.Unlock()
			runtime.Gosched()
			br.swapMu.Lock()
		}
	}

	// Extreme case: all buffers still frozen after many retries.
	// Just pick next in ring and spin until it's available.
	next := (cur + 1) % int32(ringSize)
	nb := br.buffers[next]
	for nb.frozen.Load() {
		br.swapMu.Unlock()
		runtime.Gosched()
		br.swapMu.Lock()
	}
	br.active.Store(next)
}

// flusher runs in a background goroutine, flushing full buffers to the volume.
func (br *bufferRing) flusher() {
	defer br.wg.Done()
	for {
		select {
		case <-br.stopCh:
			// Flush remaining data before exit.
			br.flushActive()
			return
		case idx := <-br.flushCh:
			br.flushBuffer(idx)
		}
	}
}

// flushBuffer writes a buffer's contents to the volume and resets it.
func (br *bufferRing) flushBuffer(idx int) {
	wb := br.buffers[idx]

	// Wait for all active writers to finish their memcpy.
	for wb.writers.Load() > 0 {
		runtime.Gosched()
	}

	n := wb.written()
	if n == 0 {
		wb.frozen.Store(false)
		return
	}

	newTail := wb.volOffset + n

	// Ensure file is large enough BEFORE writing.
	if newTail > br.vol.fileSize.Load() {
		br.vol.growFile(newTail)
	}

	// Single pwrite to volume — sequential, kernel-optimized.
	br.vol.fd.WriteAt(wb.data[:n], wb.volOffset)

	// Update volume tail atomically.
	for {
		old := br.vol.tail.Load()
		if newTail <= old {
			break
		}
		if br.vol.tail.CompareAndSwap(old, newTail) {
			break
		}
	}

	// Compute next volume offset for reuse.
	// Place after all other buffers to avoid overlap.
	nextOffset := newTail + br.capacity*int64(ringSize-1)
	for i := 0; i < ringSize; i++ {
		if i == idx {
			continue
		}
		other := br.buffers[i]
		end := other.volOffset + other.capacity
		if nextOffset < end {
			nextOffset = end
		}
	}
	wb.reset(nextOffset)
}

// flushActive flushes the current active buffer (called on close).
func (br *bufferRing) flushActive() {
	cur := br.active.Load()
	ab := br.buffers[cur]
	n := ab.written()
	if n == 0 {
		return
	}
	ab.frozen.Store(true)
	br.flushBuffer(int(cur))
}

// readFromBuffer reads data from a write buffer if the offset falls within it.
// Returns the data slice and true, or nil and false if offset is not in any buffer.
func (br *bufferRing) readFromBuffer(offset, size int64) ([]byte, bool) {
	for i := 0; i < ringSize; i++ {
		wb := br.buffers[i]
		if offset >= wb.volOffset && offset+size <= wb.volOffset+wb.written() {
			localOff := offset - wb.volOffset
			return wb.data[localOff : localOff+size], true
		}
	}
	return nil, false
}

// close flushes remaining data and stops the flusher goroutine.
func (br *bufferRing) close() {
	close(br.stopCh)
	br.wg.Wait()
}
