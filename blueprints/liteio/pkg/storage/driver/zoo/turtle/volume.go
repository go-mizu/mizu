package turtle

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
	magic           = "TURTLE01"
	version         = 1
	headerSize      = 64
	defaultPrealloc = 64 * 1024 * 1024 * 1024 // 64GB sparse

	// Record types.
	recPut    byte = 1
	recDelete byte = 2

	// Record header size (fixed fields): type(1) + crc(4) + bucketLen(2) + keyLen(2) + ctLen(2) + valueLen(8) + timestamp(8) = 27
	recFixedSize = 27
)

// mmapRegion holds a single mmap'd region and its capacity.
// Accessed via atomic.Pointer for lock-free concurrent reads.
type mmapRegion struct {
	buf      []byte
	capacity int64
}

// pwriteThreshold: values >= this size use pwrite instead of mmap memcpy.
// Avoids page fault cascade on sparse file pages (64KB = 16 faults).
const pwriteThreshold = 4096

// writeBufPools provides tiered sync.Pools for pwrite buffers (avoids heap allocation per write).
var writeBufPools = [4]sync.Pool{
	{New: func() any { b := make([]byte, 64*1024); return &b }},       // 64KB
	{New: func() any { b := make([]byte, 1*1024*1024); return &b }},   // 1MB
	{New: func() any { b := make([]byte, 16*1024*1024); return &b }},  // 16MB
	{New: func() any { b := make([]byte, 128*1024*1024); return &b }}, // 128MB
}

// getWriteBuf returns a pooled buffer of at least size bytes.
// Returns the buffer, a pointer for returning to pool, and the pool index.
func getWriteBuf(size int64) ([]byte, *[]byte, int) {
	tiers := [4]int64{64 * 1024, 1024 * 1024, 16 * 1024 * 1024, 128 * 1024 * 1024}
	for i, tier := range tiers {
		if size <= tier {
			bp := writeBufPools[i].Get().(*[]byte)
			return (*bp)[:size], bp, i
		}
	}
	// Larger than any pool tier: allocate directly.
	b := make([]byte, size)
	return b, nil, -1
}

// putWriteBuf returns a buffer to its pool.
func putWriteBuf(bp *[]byte, poolIdx int) {
	if bp != nil && poolIdx >= 0 {
		writeBufPools[poolIdx].Put(bp)
	}
}

// volume manages a single append-only data file with mmap.
type volume struct {
	fd       *os.File
	path     string
	region   atomic.Pointer[mmapRegion] // current active mapping (for reads + small writes)
	tail     atomic.Int64
	fileSize atomic.Int64 // actual file size (may exceed mmap capacity)
	mu       sync.Mutex   // protects grow operations only
	crcTable *crc32.Table
	noCRC bool // skip CRC computation (for sync=none mode)
}

// newVolume opens or creates a volume file at path.
// prealloc is the initial size in bytes (default 4GB).
func newVolume(path string, prealloc int64) (*volume, error) {
	if prealloc <= 0 {
		prealloc = defaultPrealloc
	}

	// Ensure parent directory exists.
	dir := path
	if idx := lastSlash(path); idx >= 0 {
		dir = path[:idx]
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("turtle: mkdir %q: %w", dir, err)
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("turtle: open volume: %w", err)
	}

	info, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("turtle: stat volume: %w", err)
	}

	isNew := info.Size() == 0
	allocSize := prealloc
	if info.Size() > allocSize {
		allocSize = info.Size()
	}

	// Pre-allocate file.
	if info.Size() < allocSize {
		if err := fd.Truncate(allocSize); err != nil {
			fd.Close()
			return nil, fmt.Errorf("turtle: truncate volume: %w", err)
		}
	}

	// mmap with MAP_SHARED for read+write.
	data, err := syscall.Mmap(int(fd.Fd()), 0, int(allocSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("turtle: mmap: %w", err)
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
	binary.LittleEndian.PutUint32(r.buf[12:16], 0) // flags
	binary.LittleEndian.PutUint64(r.buf[16:24], headerSize)
}

func (v *volume) readHeader() error {
	r := v.region.Load()
	if len(r.buf) < headerSize {
		return errors.New("turtle: volume too small for header")
	}
	if string(r.buf[0:8]) != magic {
		return errors.New("turtle: invalid volume magic")
	}
	ver := binary.LittleEndian.Uint32(r.buf[8:12])
	if ver != version {
		return fmt.Errorf("turtle: unsupported version %d", ver)
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

// buildRecordBuf serializes a record into buf and returns the value offset within buf.
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

// appendRecord appends a record to the volume and returns the offset where the value starts.
// Returns (recordOffset, valueOffset, error).
func (v *volume) appendRecord(recType byte, bucket, key, contentType string, value []byte, timestamp int64) (int64, int64, error) {
	totalSize := int64(recFixedSize + len(bucket) + len(key) + len(contentType) + len(value))

	// Atomically claim space.
	offset := v.tail.Add(totalSize) - totalSize

	r := v.region.Load()
	if offset+totalSize <= r.capacity && len(value) < pwriteThreshold {
		// Small values within mmap capacity: direct mmap write (≤1 page fault).
		buf := r.buf[offset : offset+totalSize]
		valPos := v.buildRecordBuf(buf, recType, bucket, key, contentType, value, timestamp)
		return offset, offset + int64(valPos), nil
	}

	// Large values or beyond mmap: use pwrite with pooled buffer.
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
		return 0, 0, fmt.Errorf("turtle: pwrite: %w", err)
	}
	return offset, offset + int64(valPos), nil
}

// writeFromReader claims space and writes a record, reading value from src.
func (v *volume) writeFromReader(recType byte, bucket, key, contentType string, src io.Reader, size int64, timestamp int64) (int64, error) {
	bl := len(bucket)
	kl := len(key)
	cl := len(contentType)
	hdrSize := recFixedSize + bl + kl + cl
	totalSize := int64(hdrSize) + size

	// Atomically claim space.
	offset := v.tail.Add(totalSize) - totalSize

	r := v.region.Load()
	if offset+totalSize <= r.capacity && size < pwriteThreshold {
		// Small values within mmap capacity: write directly into mmap.
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
					return 0, fmt.Errorf("turtle: read value: %w", err)
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

	// Large values or beyond mmap: use pwrite.
	if offset+totalSize > r.capacity {
		if err := v.growFile(offset + totalSize); err != nil {
			return 0, err
		}
	}

	// Pooled single-pwrite path (for values up to pool max 128MB).
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
				return 0, fmt.Errorf("turtle: read value: %w", err)
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
		return 0, fmt.Errorf("turtle: pwrite: %w", err)
	}
	return valOff, nil
}

// readValueSlice returns value data.
// Zero-copy from mmap for data within mmap capacity.
// Uses pread for data beyond mmap capacity (written via pwrite).
func (v *volume) readValueSlice(offset, size int64) []byte {
	r := v.region.Load()
	if offset+size <= r.capacity {
		return r.buf[offset : offset+size]
	}
	// Data beyond mmap: use pread.
	buf := make([]byte, size)
	v.fd.ReadAt(buf, offset)
	return buf
}

// recover scans the volume file and rebuilds the index.
func (v *volume) recover(idx *shardedIndex) error {
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
			break // corrupt or end of data
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

		totalRec := int64(p) + vl + 8 // +8 for timestamp
		if pos+totalRec > tail {
			break
		}

		valueOffset := pos + int64(p)
		timestamp := int64(binary.LittleEndian.Uint64(buf[p+int(vl):]))

		// Verify CRC.
		computedCRC := crc32.Checksum(buf[5:totalRec], v.crcTable)
		if computedCRC != storedCRC {
			break // corrupt record
		}

		// Update index.
		switch recType {
		case recPut:
			idx.put(bucket, key, &indexEntry{
				valueOffset: valueOffset,
				size:        vl,
				contentType: contentType,
				created:     timestamp,
				updated:     timestamp,
			})
		case recDelete:
			idx.remove(bucket, key)
		}

		validTail = pos + totalRec
		pos = validTail
		_ = contentType // used above
	}

	v.tail.Store(validTail)
	return nil
}

// growFile ensures the file is at least `needed` bytes.
// Only extends the file (ftruncate), does NOT grow the mmap.
// Used by pwrite path where mmap is grown lazily on reads.
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
		return fmt.Errorf("turtle: truncate: %w", err)
	}
	v.fileSize.Store(newSize)
	return nil
}

// sync forces all pending writes to disk.
func (v *volume) sync() error {
	v.flushHeader()
	r := v.region.Load()
	// msync the written portion of the current mapping.
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC,
		uintptr(unsafePointer(r.buf)),
		uintptr(v.tail.Load()),
		uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return fmt.Errorf("turtle: msync: %w", errno)
	}
	return nil
}

func (v *volume) close() error {
	v.flushHeader()

	// Unmap current mapping.
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
