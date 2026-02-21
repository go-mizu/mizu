package herd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
)

// Volume file constants.
const (
	magic           = "HERD0001"
	version         = 1
	headerSize      = 64
	defaultPrealloc = 1024 * 1024 * 1024 // 1GB per stripe

	// Record types.
	recPut    byte = 1
	recDelete byte = 2

	// Record header size: type(1) + crc(4) + bucketLen(2) + keyLen(2) + ctLen(2) + valueLen(8) + timestamp(8) = 27
	recFixedSize = 27
)

// mmapRegion holds a single mmap'd region and its capacity.
type mmapRegion struct {
	buf      []byte
	capacity int64
}

const pwriteThreshold = 4096

// writeBufPools provides tiered sync.Pools for pwrite buffers.
var writeBufPools = [5]sync.Pool{
	{New: func() any { b := make([]byte, 64*1024+1024); return &b }},
	{New: func() any { b := make([]byte, 1024*1024+1024); return &b }},
	{New: func() any { b := make([]byte, 10*1024*1024+1024); return &b }},
	{New: func() any { b := make([]byte, 100*1024*1024+4096); return &b }},
	{New: func() any { b := make([]byte, 256*1024*1024); return &b }},
}

func getWriteBuf(size int64) ([]byte, *[]byte, int) {
	tiers := [5]int64{
		64*1024 + 1024,
		1024*1024 + 1024,
		10*1024*1024 + 1024,
		100*1024*1024 + 4096,
		256 * 1024 * 1024,
	}
	for i, tier := range tiers {
		if size <= tier {
			bp := writeBufPools[i].Get().(*[]byte)
			return (*bp)[:size], bp, i
		}
	}
	b := make([]byte, size)
	return b, nil, -1
}

func putWriteBuf(bp *[]byte, poolIdx int) {
	if bp != nil && poolIdx >= 0 {
		writeBufPools[poolIdx].Put(bp)
	}
}

// volume manages a single append-only data file with mmap.
type volume struct {
	fd       *os.File
	path     string
	region   atomic.Pointer[mmapRegion]
	tail     atomic.Int64
	fileSize atomic.Int64
	mu       sync.Mutex
	crcTable *crc32.Table
	noCRC    bool
}

func newVolume(path string, prealloc int64) (*volume, error) {
	if prealloc <= 0 {
		prealloc = defaultPrealloc
	}

	dir := path
	if idx := lastSlash(path); idx >= 0 {
		dir = path[:idx]
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("herd: mkdir %q: %w", dir, err)
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("herd: open volume: %w", err)
	}

	info, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("herd: stat volume: %w", err)
	}

	isNew := info.Size() == 0
	allocSize := prealloc
	if info.Size() > allocSize {
		allocSize = info.Size()
	}

	if info.Size() < allocSize {
		if err := fd.Truncate(allocSize); err != nil {
			fd.Close()
			return nil, fmt.Errorf("herd: truncate volume: %w", err)
		}
	}

	data, err := syscall.Mmap(int(fd.Fd()), 0, int(allocSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("herd: mmap: %w", err)
	}

	v := &volume{
		fd:       fd,
		path:     path,
		crcTable: crc32.MakeTable(crc32.IEEE),
	}
	v.region.Store(&mmapRegion{buf: data, capacity: allocSize})
	v.fileSize.Store(allocSize)

	if isNew {
		v.writeHeader()
		v.tail.Store(headerSize)
	} else {
		if err := v.readHeader(); err != nil {
			syscall.Munmap(data)
			fd.Close()
			return nil, err
		}
	}

	return v, nil
}

func (v *volume) writeHeader() {
	r := v.region.Load()
	copy(r.buf[0:8], magic)
	binary.LittleEndian.PutUint32(r.buf[8:12], version)
	binary.LittleEndian.PutUint32(r.buf[12:16], 0)
	binary.LittleEndian.PutUint64(r.buf[16:24], headerSize)
}

func (v *volume) readHeader() error {
	r := v.region.Load()
	if len(r.buf) < headerSize {
		return errors.New("herd: volume too small for header")
	}
	if string(r.buf[0:8]) != magic {
		return errors.New("herd: invalid volume magic")
	}
	ver := binary.LittleEndian.Uint32(r.buf[8:12])
	if ver != version {
		return fmt.Errorf("herd: unsupported version %d", ver)
	}
	tail := binary.LittleEndian.Uint64(r.buf[16:24])
	if tail < headerSize {
		tail = headerSize
	}
	v.tail.Store(int64(tail))
	return nil
}

func (v *volume) flushHeader() {
	r := v.region.Load()
	binary.LittleEndian.PutUint64(r.buf[16:24], uint64(v.tail.Load()))
}

func (v *volume) buildRecordBuf(buf []byte, recType byte, bucket, key, contentType string, value []byte, timestamp int64) int {
	buf[0] = recType
	pos := 5

	bl := len(bucket)
	binary.LittleEndian.PutUint16(buf[pos:], uint16(bl))
	pos += 2
	copy(buf[pos:], bucket)
	pos += bl

	kl := len(key)
	binary.LittleEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], key)
	pos += kl

	cl := len(contentType)
	binary.LittleEndian.PutUint16(buf[pos:], uint16(cl))
	pos += 2
	copy(buf[pos:], contentType)
	pos += cl

	binary.LittleEndian.PutUint64(buf[pos:], uint64(len(value)))
	pos += 8

	copy(buf[pos:], value)
	valPos := pos
	pos += len(value)

	binary.LittleEndian.PutUint64(buf[pos:], uint64(timestamp))

	if !v.noCRC {
		checksum := crc32.Checksum(buf[5:], v.crcTable)
		binary.LittleEndian.PutUint32(buf[1:5], checksum)
	}

	return valPos
}

func (v *volume) appendRecord(recType byte, bucket, key, contentType string, value []byte, timestamp int64) (int64, int64, error) {
	totalSize := int64(recFixedSize + len(bucket) + len(key) + len(contentType) + len(value))

	offset := v.tail.Add(totalSize) - totalSize

	r := v.region.Load()
	if offset+totalSize <= r.capacity && len(value) < pwriteThreshold {
		buf := r.buf[offset : offset+totalSize]
		valPos := v.buildRecordBuf(buf, recType, bucket, key, contentType, value, timestamp)
		return offset, offset + int64(valPos), nil
	}

	if offset+totalSize > r.capacity {
		if err := v.growFile(offset + totalSize); err != nil {
			return 0, 0, err
		}
	}
	buf, bp, poolIdx := getWriteBuf(totalSize)
	valPos := v.buildRecordBuf(buf, recType, bucket, key, contentType, value, timestamp)
	_, err := v.fd.WriteAt(buf, offset)
	putWriteBuf(bp, poolIdx)
	if err != nil {
		return 0, 0, fmt.Errorf("herd: pwrite: %w", err)
	}
	return offset, offset + int64(valPos), nil
}

func (v *volume) writeFromReader(recType byte, bucket, key, contentType string, src io.Reader, size int64, timestamp int64) (int64, error) {
	bl := len(bucket)
	kl := len(key)
	cl := len(contentType)
	hdrSize := recFixedSize + bl + kl + cl
	totalSize := int64(hdrSize) + size

	offset := v.tail.Add(totalSize) - totalSize

	r := v.region.Load()
	if offset+totalSize <= r.capacity && size < pwriteThreshold {
		buf := r.buf[offset : offset+totalSize]
		buf[0] = recType
		pos := 5

		binary.LittleEndian.PutUint16(buf[pos:], uint16(bl))
		pos += 2
		copy(buf[pos:], bucket)
		pos += bl

		binary.LittleEndian.PutUint16(buf[pos:], uint16(kl))
		pos += 2
		copy(buf[pos:], key)
		pos += kl

		binary.LittleEndian.PutUint16(buf[pos:], uint16(cl))
		pos += 2
		copy(buf[pos:], contentType)
		pos += cl

		binary.LittleEndian.PutUint64(buf[pos:], uint64(size))
		pos += 8

		valOff := offset + int64(pos)
		if size > 0 {
			if _, err := io.ReadFull(src, r.buf[valOff:valOff+size]); err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					return 0, fmt.Errorf("herd: read value: %w", err)
				}
			}
		}
		pos += int(size)

		binary.LittleEndian.PutUint64(buf[pos:], uint64(timestamp))

		if !v.noCRC {
			checksum := crc32.Checksum(buf[5:], v.crcTable)
			binary.LittleEndian.PutUint32(buf[1:5], checksum)
		}

		return valOff, nil
	}

	if offset+totalSize > r.capacity {
		if err := v.growFile(offset + totalSize); err != nil {
			return 0, err
		}
	}

	buf, bp, poolIdx := getWriteBuf(totalSize)
	buf[0] = recType
	pos := 5

	binary.LittleEndian.PutUint16(buf[pos:], uint16(bl))
	pos += 2
	copy(buf[pos:], bucket)
	pos += bl

	binary.LittleEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], key)
	pos += kl

	binary.LittleEndian.PutUint16(buf[pos:], uint16(cl))
	pos += 2
	copy(buf[pos:], contentType)
	pos += cl

	binary.LittleEndian.PutUint64(buf[pos:], uint64(size))
	pos += 8

	valOff := offset + int64(pos)
	if size > 0 {
		if _, err := io.ReadFull(src, buf[pos:pos+int(size)]); err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				putWriteBuf(bp, poolIdx)
				return 0, fmt.Errorf("herd: read value: %w", err)
			}
		}
	}
	pos += int(size)

	binary.LittleEndian.PutUint64(buf[pos:], uint64(timestamp))

	if !v.noCRC {
		checksum := crc32.Checksum(buf[5:], v.crcTable)
		binary.LittleEndian.PutUint32(buf[1:5], checksum)
	}

	_, err := v.fd.WriteAt(buf, offset)
	putWriteBuf(bp, poolIdx)
	if err != nil {
		return 0, fmt.Errorf("herd: pwrite: %w", err)
	}
	return valOff, nil
}

func (v *volume) readValueSlice(offset, size int64) []byte {
	r := v.region.Load()
	if offset+size <= r.capacity {
		return r.buf[offset : offset+size]
	}
	buf := make([]byte, size)
	v.fd.ReadAt(buf, offset)
	return buf
}

func (v *volume) recover(idx *shardedIndex, bloom *bloomFilter, inlineMax int64) error {
	r := v.region.Load()
	tail := v.tail.Load()
	pos := int64(headerSize)
	validTail := pos

	for pos < tail {
		remaining := tail - pos
		if remaining < recFixedSize {
			break
		}

		buf := r.buf[pos:]
		recType := buf[0]
		if recType != recPut && recType != recDelete {
			break
		}

		storedCRC := binary.LittleEndian.Uint32(buf[1:5])

		p := 5
		bl := int(binary.LittleEndian.Uint16(buf[p:]))
		p += 2
		if int64(p+bl) > remaining {
			break
		}
		bucket := string(buf[p : p+bl])
		p += bl

		kl := int(binary.LittleEndian.Uint16(buf[p:]))
		p += 2
		if int64(p+kl) > remaining {
			break
		}
		key := string(buf[p : p+kl])
		p += kl

		cl := int(binary.LittleEndian.Uint16(buf[p:]))
		p += 2
		if int64(p+cl) > remaining {
			break
		}
		contentType := string(buf[p : p+cl])
		p += cl

		vl := int64(binary.LittleEndian.Uint64(buf[p:]))
		p += 8

		totalRec := int64(p) + vl + 8
		if pos+totalRec > tail {
			break
		}

		valueOffset := pos + int64(p)
		timestamp := int64(binary.LittleEndian.Uint64(buf[p+int(vl):]))

		computedCRC := crc32.Checksum(buf[5:totalRec], v.crcTable)
		if computedCRC != storedCRC {
			break
		}

		switch recType {
		case recPut:
			e := acquireIndexEntry()
			e.valueOffset = valueOffset
			e.size = vl
			e.contentType = contentType
			e.created = timestamp
			e.updated = timestamp
			// Inline small values on recovery.
			if vl <= inlineMax && vl > 0 {
				e.inline = make([]byte, vl)
				copy(e.inline, r.buf[valueOffset:valueOffset+vl])
				e.valueOffset = 0
			}
			idx.put(bucket, key, e)
			bloom.add(bucket, key)
		case recDelete:
			idx.remove(bucket, key)
		}

		validTail = pos + totalRec
		pos = validTail
	}

	v.tail.Store(validTail)
	return nil
}

func (v *volume) growFile(needed int64) error {
	if needed <= v.fileSize.Load() {
		return nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	current := v.fileSize.Load()
	if needed <= current {
		return nil
	}

	newSize := current * 2
	for newSize < needed {
		newSize *= 2
	}

	if err := v.fd.Truncate(newSize); err != nil {
		return fmt.Errorf("herd: truncate: %w", err)
	}

	// v4: remap mmap to cover the grown file.
	// Old mapping is intentionally leaked (readers may still reference it).
	// Leak is bounded by geometric growth: total leaked ≤ current size.
	// This eliminates readValueSlice fallback to make([]byte) + ReadAt.
	newData, err := syscall.Mmap(int(v.fd.Fd()), 0, int(newSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err == nil {
		v.region.Store(&mmapRegion{buf: newData, capacity: newSize})
	}

	v.fileSize.Store(newSize)
	return nil
}

func (v *volume) sync() error {
	v.flushHeader()
	r := v.region.Load()
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC,
		uintptr(unsafePtr(r.buf)),
		uintptr(v.tail.Load()),
		uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return fmt.Errorf("herd: msync: %w", errno)
	}
	return nil
}

func (v *volume) close() error {
	v.flushHeader()
	r := v.region.Load()
	if r != nil && r.buf != nil {
		syscall.Munmap(r.buf)
	}
	if v.fd != nil {
		return v.fd.Close()
	}
	return nil
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
