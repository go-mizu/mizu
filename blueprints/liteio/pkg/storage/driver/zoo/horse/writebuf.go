package horse

import (
	"sync"
	"sync/atomic"
)

// Default write buffer size: 64MB.
const defaultBufSize = 64 * 1024 * 1024

// writeBuffer is a pre-allocated contiguous memory region for accumulating writes.
// All writes are pure memcpy — no page faults, no syscalls on the write path.
// When the buffer is full, it is flushed to the volume as a single pwrite.
type writeBuffer struct {
	data      []byte       // pre-allocated buffer
	pos       atomic.Int64 // current write position (lock-free via atomic.Add)
	capacity  int64        // capacity in bytes
	volOffset int64        // volume offset where this buffer starts
	frozen    atomic.Bool  // true = no more writes, being flushed
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

// claim atomically reserves space in the buffer.
// Returns the local offset within the buffer, or -1 if the buffer is full/frozen.
func (wb *writeBuffer) claim(size int64) int64 {
	if wb.frozen.Load() {
		return -1
	}
	pos := wb.pos.Add(size) - size
	if pos+size > wb.capacity {
		// Overflowed — revert and signal full.
		wb.pos.Add(-size)
		return -1
	}
	return pos
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

// bufferRing manages double-buffered writes with background flush.
type bufferRing struct {
	buffers  [2]*writeBuffer
	active   atomic.Int32  // index of active buffer (0 or 1)
	vol      *volume
	flushCh  chan int       // sends buffer index to flush
	stopCh   chan struct{}
	wg       sync.WaitGroup
	swapMu   sync.Mutex // protects buffer swap
	capacity int64
}

// newBufferRing creates a double-buffered write ring.
func newBufferRing(vol *volume, bufSize int64) *bufferRing {
	if bufSize <= 0 {
		bufSize = defaultBufSize
	}

	tail := vol.tail.Load()
	br := &bufferRing{
		vol:      vol,
		flushCh:  make(chan int, 2),
		stopCh:   make(chan struct{}),
		capacity: bufSize,
	}
	br.buffers[0] = newWriteBuffer(bufSize, tail)
	br.buffers[1] = newWriteBuffer(bufSize, tail+bufSize)
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

// write writes a pre-serialized record into the active buffer.
// Returns the volume offset where the record starts and the value offset.
// If the active buffer is full, it swaps and retries.
func (br *bufferRing) write(record []byte, valPosInRecord int) (recOff int64, valOff int64) {
	size := int64(len(record))
	for {
		ab := br.activeBuffer()
		pos := ab.claim(size)
		if pos >= 0 {
			copy(ab.data[pos:], record)
			return ab.volOffset + pos, ab.volOffset + pos + int64(valPosInRecord)
		}
		// Buffer full — swap.
		br.swap()
	}
}

// writeInline claims space and returns a buffer slice for the caller to fill directly.
// This avoids one memcpy for callers that can serialize in-place.
// Returns (slice to fill, volume record offset, volume value offset given valPosInRecord).
func (br *bufferRing) writeInline(totalSize int64, valPosInRecord int) (buf []byte, recOff int64, valOff int64) {
	for {
		ab := br.activeBuffer()
		pos := ab.claim(totalSize)
		if pos >= 0 {
			return ab.data[pos : pos+totalSize], ab.volOffset + pos, ab.volOffset + pos + int64(valPosInRecord)
		}
		br.swap()
	}
}

// swap freezes the current active buffer and activates the other one.
func (br *bufferRing) swap() {
	br.swapMu.Lock()
	defer br.swapMu.Unlock()

	cur := br.active.Load()
	ab := br.buffers[cur]

	// Check if already swapped by another goroutine.
	if !ab.frozen.Load() {
		ab.frozen.Store(true)
		// Send for flushing.
		br.flushCh <- int(cur)
	}

	next := 1 - cur
	nb := br.buffers[next]
	// Wait for the next buffer to be available (not frozen = already flushed).
	for nb.frozen.Load() {
		br.swapMu.Unlock()
		// Spin briefly — flush should be fast.
		for i := 0; i < 1000; i++ {
			if !nb.frozen.Load() {
				break
			}
		}
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
	n := wb.written()
	if n == 0 {
		wb.frozen.Store(false)
		return
	}

	// Single pwrite to volume — sequential, kernel-optimized.
	br.vol.fd.WriteAt(wb.data[:n], wb.volOffset)

	// Update volume tail.
	newTail := wb.volOffset + n
	for {
		old := br.vol.tail.Load()
		if newTail <= old {
			break
		}
		if br.vol.tail.CompareAndSwap(old, newTail) {
			break
		}
	}

	// Ensure file is large enough.
	if newTail > br.vol.fileSize.Load() {
		br.vol.growFile(newTail)
	}

	// Compute next volume offset for reuse.
	nextOffset := newTail + br.capacity
	// Align to other buffer's boundary to avoid overlap.
	other := br.buffers[1-idx]
	if nextOffset < other.volOffset+other.capacity {
		nextOffset = other.volOffset + other.capacity
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
	for i := 0; i < 2; i++ {
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
