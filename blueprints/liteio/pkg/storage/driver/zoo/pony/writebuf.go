package pony

import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

// Default write buffer size: 4MB.
const defaultBufSize = 4 * 1024 * 1024

// ringSize is the number of buffers in the ring.
const ringSize = 4

// numFlushers: concurrent flush goroutines per stripe.
const numFlushers = 4

// writeBuffer is a pre-allocated contiguous memory region for accumulating writes.
type writeBuffer struct {
	data      []byte
	pos       atomic.Int64
	capacity  int64
	volOffset int64
	frozen    atomic.Bool
	writers   atomic.Int32
	mmaped    bool
}

func newWriteBuffer(capacity int64, volOffset int64) *writeBuffer {
	// Use mmap for GC-invisible buffer memory.
	data, err := syscall.Mmap(-1, 0, int(capacity),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE)
	mmaped := err == nil
	if !mmaped {
		data = make([]byte, capacity)
	}
	return &writeBuffer{
		data:      data,
		capacity:  capacity,
		volOffset: volOffset,
		mmaped:    mmaped,
	}
}

func (wb *writeBuffer) claim(size int64) int64 {
	if wb.frozen.Load() {
		return -1
	}
	wb.writers.Add(1)
	pos := wb.pos.Add(size) - size
	if pos+size > wb.capacity {
		wb.pos.Add(-size)
		wb.writers.Add(-1)
		return -1
	}
	return pos
}

func (wb *writeBuffer) done() {
	wb.writers.Add(-1)
}

func (wb *writeBuffer) written() int64 {
	pos := wb.pos.Load()
	if pos > wb.capacity {
		return wb.capacity
	}
	return pos
}

func (wb *writeBuffer) reset(volOffset int64) {
	wb.pos.Store(0)
	wb.volOffset = volOffset
	wb.frozen.Store(false)
}

func (wb *writeBuffer) free() {
	if wb.mmaped && wb.data != nil {
		syscall.Munmap(wb.data)
		wb.data = nil
	}
}

// bufferRing manages a ring of write buffers with concurrent background flush.
// When all buffers are frozen, falls back to direct pwrite (non-blocking overflow)
// instead of blocking on sync.Cond.Wait.
type bufferRing struct {
	buffers  [ringSize]*writeBuffer
	active   atomic.Int32
	vol      *volume
	flushCh  chan int
	stopCh   chan struct{}
	wg       sync.WaitGroup
	swapMu   sync.Mutex
	bufReady sync.Cond // signaled when a flusher resets a buffer
	capacity int64
	nextBase atomic.Int64
}

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
	br.bufReady.L = &br.swapMu

	for i := 0; i < ringSize; i++ {
		br.buffers[i] = newWriteBuffer(bufSize, tail+int64(i)*bufSize)
	}
	br.active.Store(0)
	br.nextBase.Store(tail + int64(ringSize)*bufSize)

	for i := 0; i < numFlushers; i++ {
		br.wg.Add(1)
		go br.flusher()
	}

	return br
}

func (br *bufferRing) activeBuffer() *writeBuffer {
	return br.buffers[br.active.Load()]
}

// writeInline writes to a buffer in the ring. Falls back to direct pwrite if
// all buffers are frozen after 2 swap attempts (non-blocking overflow).
// When direct=true, caller must call directFlush after filling the returned buf.
func (br *bufferRing) writeInline(totalSize int64, valPosInRecord int) (buf []byte, recOff int64, valOff int64, wb *writeBuffer, direct bool) {
	for attempts := 0; ; attempts++ {
		ab := br.activeBuffer()
		pos := ab.claim(totalSize)
		if pos >= 0 {
			return ab.data[pos : pos+totalSize], ab.volOffset + pos, ab.volOffset + pos + int64(valPosInRecord), ab, false
		}
		if attempts >= 2 {
			// All buffers frozen — direct pwrite overflow (non-blocking).
			b, off := br.directWrite(totalSize)
			return b, off, off + int64(valPosInRecord), nil, true
		}
		br.swap()
	}
}

// directWrite allocates a volume offset and a temporary buffer for non-blocking
// overflow when the buffer ring is full. Caller fills the buffer then calls directFlush.
func (br *bufferRing) directWrite(totalSize int64) (buf []byte, recOff int64) {
	offset := br.nextBase.Add(totalSize) - totalSize
	if offset+totalSize > br.vol.fileSize.Load() {
		br.vol.growFile(offset + totalSize)
	}
	b, _ := getWriteBuf(totalSize)
	return b, offset
}

// directFlush writes a direct-overflow buffer to disk and advances the volume tail.
func (br *bufferRing) directFlush(b []byte, offset int64) {
	size := int64(len(b))
	br.vol.fd.WriteAt(b, offset)
	// Return buffer to pool based on capacity.
	tierIdx := -1
	for i := range wbPool.tiers {
		if int64(cap(b)) <= wbPool.tiers[i].cap {
			tierIdx = i
			break
		}
	}
	putWriteBuf(b, tierIdx)
	// Advance tail (CAS loop — never go backwards).
	newTail := offset + size
	for {
		old := br.vol.tail.Load()
		if newTail <= old {
			break
		}
		if br.vol.tail.CompareAndSwap(old, newTail) {
			break
		}
	}
}

// swap freezes the current buffer, sends it for flushing, and switches to the
// next available buffer. If all buffers are frozen, blocks on bufReady Cond
// until a flusher resets one.
func (br *bufferRing) swap() {
	br.swapMu.Lock()
	defer br.swapMu.Unlock()

	cur := br.active.Load()
	ab := br.buffers[cur]

	// Freeze current and submit for flush.
	if !ab.frozen.Load() {
		ab.frozen.Store(true)
		select {
		case br.flushCh <- int(cur):
		default:
		}
	}

	// Find next non-frozen buffer. If all frozen, wait for flusher.
	for {
		for i := int32(1); i < int32(ringSize); i++ {
			next := (cur + i) % int32(ringSize)
			if !br.buffers[next].frozen.Load() {
				br.active.Store(next)
				return
			}
		}
		// All frozen — block until flusher resets one.
		br.bufReady.Wait()
	}
}

func (br *bufferRing) flusher() {
	defer br.wg.Done()
	for {
		select {
		case <-br.stopCh:
			return
		case idx := <-br.flushCh:
			br.flushBuffer(idx)
		}
	}
}

func (br *bufferRing) flushBuffer(idx int) {
	wb := br.buffers[idx]

	// Wait for active writers to finish (bounded spin).
	for spins := 0; wb.writers.Load() > 0; spins++ {
		if spins > 1000 {
			runtime.Gosched()
			spins = 0
		}
	}

	n := wb.written()
	if n == 0 {
		wb.frozen.Store(false)
		br.signalReady()
		return
	}

	newTail := wb.volOffset + n

	if newTail > br.vol.fileSize.Load() {
		br.vol.growFile(newTail)
	}

	br.vol.fd.WriteAt(wb.data[:n], wb.volOffset)

	// Advance tail (CAS loop — never go backwards).
	for {
		old := br.vol.tail.Load()
		if newTail <= old {
			break
		}
		if br.vol.tail.CompareAndSwap(old, newTail) {
			break
		}
	}

	// Claim next offset atomically (race-free across concurrent flushers).
	nextOffset := br.nextBase.Add(br.capacity) - br.capacity
	wb.reset(nextOffset)

	// Wake up blocked writers.
	br.signalReady()
}

func (br *bufferRing) signalReady() {
	br.swapMu.Lock()
	br.bufReady.Broadcast()
	br.swapMu.Unlock()
}

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

func (br *bufferRing) close() {
	br.flushActive()
	close(br.stopCh)
	br.wg.Wait()
	for i := 0; i < ringSize; i++ {
		br.buffers[i].free()
	}
}
