package pony

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

const (
	magic           = "PONY0001"
	version         = 1
	headerSize      = 64
	defaultPrealloc = 256 * 1024 * 1024 // 256MB (vs Horse's 64GB)

	recPut    byte = 1
	recDelete byte = 2

	// type(1) + crc(4) + bucketLen(2) + keyLen(2) + ctLen(2) + valueLen(8) + timestamp(8) = 27
	recFixedSize = 27
)

// pwriteThreshold: values >= this use pwrite instead of mmap memcpy.
const pwriteThreshold = 4096

// writeBufPools: capped at 10MB to stay within memory budget.
// Horse has 100MB and 256MB tiers — pony omits those.
var writeBufPools = [3]sync.Pool{
	{New: func() any { b := make([]byte, 64*1024+1024); return &b }},   // 65KB
	{New: func() any { b := make([]byte, 1024*1024+1024); return &b }}, // ~1MB
	{New: func() any { b := make([]byte, 10*1024*1024+1024); return &b }}, // ~10MB
}

func getWriteBuf(size int64) ([]byte, *[]byte, int) {
	tiers := [3]int64{
		64*1024 + 1024,
		1024*1024 + 1024,
		10*1024*1024 + 1024,
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

type mmapRegion struct {
	buf      []byte
	capacity int64
}

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
		return nil, fmt.Errorf("pony: mkdir %q: %w", dir, err)
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("pony: open volume: %w", err)
	}

	info, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("pony: stat volume: %w", err)
	}

	isNew := info.Size() == 0
	allocSize := prealloc
	if info.Size() > allocSize {
		allocSize = info.Size()
	}

	if info.Size() < allocSize {
		if err := fd.Truncate(allocSize); err != nil {
			fd.Close()
			return nil, fmt.Errorf("pony: truncate volume: %w", err)
		}
	}

	data, err := syscall.Mmap(int(fd.Fd()), 0, int(allocSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("pony: mmap: %w", err)
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
		return errors.New("pony: volume too small for header")
	}
	if string(r.buf[0:8]) != magic {
		return errors.New("pony: invalid volume magic")
	}
	ver := binary.LittleEndian.Uint32(r.buf[8:12])
	if ver != version {
		return fmt.Errorf("pony: unsupported version %d", ver)
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
		return 0, 0, fmt.Errorf("pony: pwrite: %w", err)
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
					return 0, fmt.Errorf("pony: read value: %w", err)
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
				return 0, fmt.Errorf("pony: read value: %w", err)
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
		return 0, fmt.Errorf("pony: pwrite: %w", err)
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

// readRecord reads bucket, key, contentType from a record at recOff.
func (v *volume) readRecord(recOff int64) (bucket, key, contentType string, ok bool) {
	r := v.region.Load()
	if recOff+recFixedSize > r.capacity {
		return "", "", "", false
	}

	buf := r.buf[recOff:]
	if buf[0] != recPut {
		return "", "", "", false
	}

	p := 5
	bl := int(binary.LittleEndian.Uint16(buf[p:]))
	p += 2
	bucket = string(buf[p : p+bl])
	p += bl

	kl := int(binary.LittleEndian.Uint16(buf[p:]))
	p += 2
	key = string(buf[p : p+kl])
	p += kl

	cl := int(binary.LittleEndian.Uint16(buf[p:]))
	p += 2
	contentType = string(buf[p : p+cl])

	return bucket, key, contentType, true
}

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
			idx.put(bucket, key, contentType, valueOffset, vl, timestamp, timestamp)
		case recDelete:
			idx.remove(bucket, key)
		}

		validTail = pos + totalRec
		pos = validTail
		_ = contentType
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
		return fmt.Errorf("pony: truncate: %w", err)
	}

	// Remap mmap to cover the new file size.
	oldRegion := v.region.Load()
	if oldRegion != nil && oldRegion.buf != nil {
		syscall.Munmap(oldRegion.buf)
	}

	newBuf, err := syscall.Mmap(int(v.fd.Fd()), 0, int(newSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("pony: remap volume: %w", err)
	}

	v.region.Store(&mmapRegion{buf: newBuf, capacity: newSize})
	v.fileSize.Store(newSize)
	return nil
}

func (v *volume) sync() error {
	v.flushHeader()
	r := v.region.Load()
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC,
		uintptr(unsafePointer(r.buf)),
		uintptr(v.tail.Load()),
		uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return fmt.Errorf("pony: msync: %w", errno)
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
