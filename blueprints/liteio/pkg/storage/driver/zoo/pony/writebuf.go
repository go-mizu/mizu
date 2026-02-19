package pony

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Default write buffer size: 4MB (vs Horse's 64MB).
const defaultBufSize = 4 * 1024 * 1024

// ringSize is the number of buffers in the ring.
// 2 buffers keeps memory low: 1 accepts writes while 1 flushes.
const ringSize = 2

// writeBuffer is a pre-allocated contiguous memory region for accumulating writes.
type writeBuffer struct {
	data      []byte
	pos       atomic.Int64
	capacity  int64
	volOffset int64
	frozen    atomic.Bool
	writers   atomic.Int32
}

func newWriteBuffer(capacity int64, volOffset int64) *writeBuffer {
	return &writeBuffer{
		data:      make([]byte, capacity),
		capacity:  capacity,
		volOffset: volOffset,
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

// bufferRing manages a ring of write buffers with background flush.
type bufferRing struct {
	buffers  [ringSize]*writeBuffer
	active   atomic.Int32
	vol      *volume
	flushCh  chan int
	stopCh   chan struct{}
	wg       sync.WaitGroup
	swapMu   sync.Mutex
	capacity int64
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
	for i := 0; i < ringSize; i++ {
		br.buffers[i] = newWriteBuffer(bufSize, tail+int64(i)*bufSize)
	}
	br.active.Store(0)

	br.wg.Add(1)
	go br.flusher()

	return br
}

func (br *bufferRing) activeBuffer() *writeBuffer {
	return br.buffers[br.active.Load()]
}

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

func (br *bufferRing) swap() {
	br.swapMu.Lock()
	defer br.swapMu.Unlock()

	cur := br.active.Load()
	ab := br.buffers[cur]

	if !ab.frozen.Load() {
		ab.frozen.Store(true)
		br.flushCh <- int(cur)
	}

	for attempt := 0; attempt < ringSize*100; attempt++ {
		next := (cur + 1 + int32(attempt)) % int32(ringSize)
		nb := br.buffers[next]
		if !nb.frozen.Load() {
			br.active.Store(next)
			return
		}
		if attempt%ringSize == ringSize-1 {
			br.swapMu.Unlock()
			runtime.Gosched()
			br.swapMu.Lock()
		}
	}

	next := (cur + 1) % int32(ringSize)
	nb := br.buffers[next]
	for nb.frozen.Load() {
		br.swapMu.Unlock()
		runtime.Gosched()
		br.swapMu.Lock()
	}
	br.active.Store(next)
}

func (br *bufferRing) flusher() {
	defer br.wg.Done()
	for {
		select {
		case <-br.stopCh:
			br.flushActive()
			return
		case idx := <-br.flushCh:
			br.flushBuffer(idx)
		}
	}
}

func (br *bufferRing) flushBuffer(idx int) {
	wb := br.buffers[idx]

	for wb.writers.Load() > 0 {
		runtime.Gosched()
	}

	n := wb.written()
	if n == 0 {
		wb.frozen.Store(false)
		return
	}

	newTail := wb.volOffset + n

	if newTail > br.vol.fileSize.Load() {
		br.vol.growFile(newTail)
	}

	br.vol.fd.WriteAt(wb.data[:n], wb.volOffset)

	for {
		old := br.vol.tail.Load()
		if newTail <= old {
			break
		}
		if br.vol.tail.CompareAndSwap(old, newTail) {
			break
		}
	}

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
	close(br.stopCh)
	br.wg.Wait()
}
