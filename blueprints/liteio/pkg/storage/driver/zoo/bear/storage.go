// Package bear implements a B-tree backed storage driver inspired by the
// "B-Trees Are Back" paper (SIGMOD 2025). It uses a single mmap'd file of
// 4KB pages with indirection-slot node layouts and head optimization for
// fast variable-length key comparisons.
//
// DSN format: bear:///path/to/root?sync=none&page_size=4096
package bear

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("bear", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	pageSize    = 4096
	magicString = "BEAR0001"
	magicLen    = 8

	// Page types
	pageTypeInner byte = 0x01
	pageTypeLeaf  byte = 0x02

	// Header offsets (page 0)
	hdrMagic      = 0
	hdrRootPage   = 8
	hdrPageCount  = 12
	hdrHeight     = 16
	hdrEntryCount = 20
	hdrFreeHead   = 28

	// Page header sizes
	innerHeaderSize = 1 + 2 + 2         // type(1) + count(2) + freeOff(2)
	leafHeaderSize  = 1 + 2 + 2 + 4 + 4 // type(1) + count(2) + freeOff(2) + nextLeaf(4) + prevLeaf(4)

	// Slot sizes
	innerSlotSize  = 4 + 2 // keyHead(4) + keyOffset(2)
	innerChildSize = 4     // childPage(4)
	leafSlotSize   = 4 + 2 // keyHead(4) + entryOffset(2)

	// Entry overhead for inline values:
	// keyLen(2) + ctLen(2) + flags(1) + valLen(4) + created(8) + updated(8) = 25
	leafEntryOverheadInline = 2 + 2 + 1 + 4 + 8 + 8

	// Entry overhead for external values (stored in value log):
	// keyLen(2) + ctLen(2) + flags(1) + valOffset(8) + valLen(8) + created(8) + updated(8) = 37
	leafEntryOverheadExternal = 2 + 2 + 1 + 8 + 8 + 8 + 8

	// Value log threshold: values larger than this go to the value log.
	// We keep non-empty values in the append-only value log to keep B-tree pages
	// dense (keys + metadata only), which dramatically reduces split churn.
	valLogThreshold = 0

	// Buffered append size for the value log. This amortizes syscall cost for
	// small writes (e.g. 1KB benchmark workload).
	valLogBufferSize = 8 * 1024 * 1024

	// Flags byte values for leaf entry encoding.
	flagInline   byte = 0x00
	flagExternal byte = 0x01

	// Minimum number of entries before page is considered for merge
	minLeafEntries = 2

	// Minimum pages required in a valid file (header + root leaf).
	initialPages = 2

	// Initial file allocation: 256 pages (1 MB) to avoid early remaps.
	initialAllocPages = 256

	// Multipart
	maxPartNumber = 10000

	// Max file size for the mmap'd B-tree file (4 GB).
	maxBearFileSize = 4 * 1024 * 1024 * 1024

	// Max number of buckets to prevent unbounded map growth.
	maxBuckets = 10000

	// Permissions
	dirPerms  = 0750
	filePerms = 0600
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, opts, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	syncMode := opts.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	if info, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(root, dirPerms); err != nil {
				return nil, fmt.Errorf("bear: create root %q: %w", root, err)
			}
		} else {
			return nil, fmt.Errorf("bear: stat root %q: %w", root, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("bear: root %q is not a directory", root)
	}

	s, err := newStore(root, syncMode)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func parseDSN(dsn string) (string, url.Values, error) {
	if dsn == "" {
		return "", nil, errors.New("bear: empty dsn")
	}

	var queryStr string
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		queryStr = dsn[idx+1:]
		dsn = dsn[:idx]
	}

	opts, _ := url.ParseQuery(queryStr)

	if strings.HasPrefix(dsn, "/") {
		return filepath.Clean(dsn), opts, nil
	}

	if strings.HasPrefix(dsn, "bear:") {
		rest := strings.TrimPrefix(dsn, "bear:")
		if strings.HasPrefix(rest, "//") {
			rest = strings.TrimPrefix(rest, "//")
		}
		if rest == "" {
			return "", nil, errors.New("bear: missing path")
		}
		return filepath.Clean(rest), opts, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("bear: parse dsn: %w", err)
	}
	if u.Scheme != "bear" && u.Scheme != "" {
		return "", nil, fmt.Errorf("bear: unsupported scheme %q", u.Scheme)
	}
	p := u.Path
	if u.Host == "." {
		p = "./" + p
	}
	if p == "" {
		return "", nil, errors.New("bear: missing path in dsn")
	}
	return filepath.Clean(p), opts, nil
}

// ---------------------------------------------------------------------------
// Store (storage.Storage)
// ---------------------------------------------------------------------------

type store struct {
	root     string
	syncMode string

	mu   sync.RWMutex
	file *os.File
	mmap []byte

	// Cached header fields (protected by mu)
	rootPage   uint32
	pageCount  uint32
	height     uint32
	entryCount uint64
	freeHead   uint32

	// Value log for large values (protected by valMu).
	valMu       sync.Mutex
	valLog      *os.File
	valLogPos   int64 // logical end (includes buffered, not-yet-flushed bytes)
	valFlushed  int64 // durable on-disk end (all reads must be <= this)
	valBufStart int64
	valBuf      []byte
	valTmpPool  sync.Pool

	// Bucket metadata: name -> creation time
	bucketsMu sync.RWMutex
	buckets   map[string]time.Time

	// Multipart state
	mpMu      sync.Mutex
	mpUploads map[string]*multipartState
	mpCounter atomic.Int64

	closed atomic.Bool
}

var _ storage.Storage = (*store)(nil)

func newStore(root, syncMode string) (*store, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("bear: abs path: %w", err)
	}

	s := &store{
		root:      absRoot,
		syncMode:  syncMode,
		buckets:   make(map[string]time.Time),
		mpUploads: make(map[string]*multipartState),
	}
	s.mpCounter.Store(time.Now().UnixNano())

	btreePath := filepath.Join(absRoot, "btree.dat")

	// Check if file exists
	_, statErr := os.Stat(btreePath)
	needsInit := errors.Is(statErr, os.ErrNotExist)

	f, err := os.OpenFile(btreePath, os.O_RDWR|os.O_CREATE, filePerms)
	if err != nil {
		return nil, fmt.Errorf("bear: open btree: %w", err)
	}
	s.file = f

	if needsInit {
		if err := s.initFile(); err != nil {
			f.Close()
			return nil, err
		}
	}

	if err := s.loadMmap(); err != nil {
		f.Close()
		return nil, err
	}

	s.readHeader()

	// Open or create the value log for large values.
	if err := s.openValueLog(); err != nil {
		_ = syscall.Munmap(s.mmap)
		f.Close()
		return nil, err
	}

	s.loadBucketMeta()

	return s, nil
}

// prepareEntry creates a leafEntry, writing the value to the value log if
// it exceeds valLogThreshold. Must be called BEFORE acquiring mu.
func (s *store) prepareEntry(key, contentType, value []byte, created, updated int64) (*leafEntry, error) {
	if shouldStoreExternal(len(value)) {
		offset, err := s.writeToValueLog(value)
		if err != nil {
			return nil, err
		}
		return &leafEntry{
			key:         key,
			contentType: contentType,
			created:     created,
			updated:     updated,
			valOffset:   offset,
			valLen:      int64(len(value)),
		}, nil
	}
	return &leafEntry{
		key:         key,
		contentType: contentType,
		value:       value,
		created:     created,
		updated:     updated,
		valOffset:   -1,
	}, nil
}

// writeToValueLog appends data to the value log and returns the offset
// where the data was written. Caller must NOT hold valMu.
func (s *store) writeToValueLog(data []byte) (int64, error) {
	if len(data) == 0 {
		return -1, nil
	}
	s.valMu.Lock()
	defer s.valMu.Unlock()

	return s.appendValueLogBytesLocked(data)
}

// writeStreamToValueLog streams src into the value log with a small reusable
// buffer to avoid per-write heap allocations for small objects.
func (s *store) writeStreamToValueLog(src io.Reader, expected int64) (int64, int64, error) {
	s.valMu.Lock()
	defer s.valMu.Unlock()

	bufAny := s.valTmpPool.Get()
	var tmp []byte
	if b, ok := bufAny.([]byte); ok && len(b) > 0 {
		tmp = b
	} else {
		tmp = make([]byte, 32*1024)
	}
	defer s.valTmpPool.Put(tmp)

	start := s.valLogPos
	var wrote int64

	if expected >= 0 {
		remaining := expected
		for remaining > 0 {
			nmax := len(tmp)
			if remaining < int64(nmax) {
				nmax = int(remaining)
			}
			n, err := io.ReadFull(src, tmp[:nmax])
			if n > 0 {
				if _, werr := s.appendValueLogBytesLocked(tmp[:n]); werr != nil {
					return 0, wrote, werr
				}
				wrote += int64(n)
				remaining -= int64(n)
			}
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					break
				}
				return 0, wrote, fmt.Errorf("bear: stream value log read: %w", err)
			}
		}
	} else {
		for {
			n, err := src.Read(tmp)
			if n > 0 {
				if _, werr := s.appendValueLogBytesLocked(tmp[:n]); werr != nil {
					return 0, wrote, werr
				}
				wrote += int64(n)
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return 0, wrote, fmt.Errorf("bear: stream value log read: %w", err)
			}
		}
	}

	if wrote == 0 {
		return -1, 0, nil
	}
	return start, wrote, nil
}

// appendValueLogBytesLocked appends bytes to the buffered value log.
// Caller must hold valMu.
func (s *store) appendValueLogBytesLocked(data []byte) (int64, error) {
	offset := s.valLogPos
	if len(data) == 0 {
		return offset, nil
	}

	s.valLogPos += int64(len(data))
	curOff := offset

	for len(data) > 0 {
		// Direct-write very large payloads when no buffered tail is pending.
		if len(s.valBuf) == 0 && len(data) >= valLogBufferSize {
			if err := s.writeValueLogDirectLocked(curOff, data); err != nil {
				return 0, err
			}
			curOff += int64(len(data))
			data = nil
			break
		}

		if len(s.valBuf) == 0 {
			s.valBufStart = curOff
		}

		space := valLogBufferSize - len(s.valBuf)
		if space == 0 {
			if err := s.flushValueLogLocked(); err != nil {
				return 0, err
			}
			continue
		}
		n := len(data)
		if n > space {
			n = space
		}
		s.valBuf = append(s.valBuf, data[:n]...)
		curOff += int64(n)
		data = data[n:]
		if len(s.valBuf) == valLogBufferSize {
			if err := s.flushValueLogLocked(); err != nil {
				return 0, err
			}
		}
	}

	if s.syncMode == "msync" {
		if err := s.flushValueLogLocked(); err != nil {
			return 0, err
		}
		if err := s.valLog.Sync(); err != nil {
			return 0, fmt.Errorf("bear: sync value log: %w", err)
		}
	}
	return offset, nil
}

// writeValueLogDirectLocked writes data at the given offset without buffering.
// Caller must hold valMu. If data is nil/empty, this is a no-op.
func (s *store) writeValueLogDirectLocked(offset int64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if _, err := s.valLog.WriteAt(data, offset); err != nil {
		return fmt.Errorf("bear: write value log: %w", err)
	}
	end := offset + int64(len(data))
	if end > s.valFlushed {
		s.valFlushed = end
	}
	return nil
}

// flushValueLogLocked flushes buffered value bytes to disk. Caller must hold valMu.
func (s *store) flushValueLogLocked() error {
	if len(s.valBuf) == 0 {
		return nil
	}
	if _, err := s.valLog.WriteAt(s.valBuf, s.valBufStart); err != nil {
		return fmt.Errorf("bear: flush value log: %w", err)
	}
	s.valFlushed = s.valBufStart + int64(len(s.valBuf))
	s.valBuf = s.valBuf[:0]
	return nil
}

// ensureValueLogReadable flushes pending buffered bytes so ReadAt/SectionReader
// sees the requested region.
func (s *store) ensureValueLogReadable(offset, length int64) error {
	if length <= 0 {
		return nil
	}
	end := offset + length
	s.valMu.Lock()
	defer s.valMu.Unlock()
	if end <= s.valFlushed {
		return nil
	}
	return s.flushValueLogLocked()
}

// readFromValueLog reads length bytes from the value log at the given offset.
func (s *store) readFromValueLog(offset, length int64) ([]byte, error) {
	if err := s.ensureValueLogReadable(offset, length); err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, fmt.Errorf("bear: negative value length %d", length)
	}
	if length == 0 {
		return nil, nil
	}
	buf := make([]byte, length)
	_, err := s.valLog.ReadAt(buf, offset)
	if err != nil {
		return nil, fmt.Errorf("bear: read value log at offset %d length %d: %w", offset, length, err)
	}
	return buf, nil
}

// openValueLogReader returns a reader for a value log region without allocating
// a new value slice. Callers can stream directly to the client.
func (s *store) openValueLogReader(offset, length int64) (io.ReadCloser, error) {
	if err := s.ensureValueLogReadable(offset, length); err != nil {
		return nil, err
	}
	return io.NopCloser(io.NewSectionReader(s.valLog, offset, length)), nil
}

// resolveValue resolves the value for a leaf entry. If the value is stored
// externally in the value log, it reads it from there. Otherwise, it
// returns the inline value. The returned entry has the value field populated.
func (s *store) resolveValue(e *leafEntry) (*leafEntry, error) {
	if e == nil {
		return nil, nil
	}
	if e.valOffset < 0 {
		// Inline — value already populated
		return e, nil
	}
	val, err := s.readFromValueLog(e.valOffset, e.valLen)
	if err != nil {
		return nil, err
	}
	// Return a copy with value filled in
	return &leafEntry{
		key:         e.key,
		contentType: e.contentType,
		value:       val,
		created:     e.created,
		updated:     e.updated,
		valOffset:   e.valOffset,
		valLen:      e.valLen,
	}, nil
}

// openValueLog opens (or creates) the append-only value log file and seeks
// to the end so that subsequent writes append correctly.
func (s *store) openValueLog() error {
	vlPath := filepath.Join(s.root, "values.log")
	vf, err := os.OpenFile(vlPath, os.O_RDWR|os.O_CREATE, filePerms)
	if err != nil {
		return fmt.Errorf("bear: open value log: %w", err)
	}
	info, err := vf.Stat()
	if err != nil {
		vf.Close()
		return fmt.Errorf("bear: stat value log: %w", err)
	}
	s.valLog = vf
	s.valLogPos = info.Size()
	s.valFlushed = info.Size()
	s.valBuf = make([]byte, 0, valLogBufferSize)
	s.valBufStart = s.valFlushed
	s.valTmpPool = sync.Pool{
		New: func() any { return make([]byte, 32*1024) },
	}
	return nil
}

// initFile creates the initial file: header + empty root leaf, pre-allocated
// to initialAllocPages to avoid early remaps.
func (s *store) initFile() error {
	buf := make([]byte, initialAllocPages*pageSize)

	// Page 0: header
	copy(buf[hdrMagic:], magicString)
	binary.LittleEndian.PutUint32(buf[hdrRootPage:], 1)  // root is page 1
	binary.LittleEndian.PutUint32(buf[hdrPageCount:], 2) // 2 pages total
	binary.LittleEndian.PutUint32(buf[hdrHeight:], 1)    // height 1 (single leaf)
	binary.LittleEndian.PutUint64(buf[hdrEntryCount:], 0)
	binary.LittleEndian.PutUint32(buf[hdrFreeHead:], 0)

	// Page 1: empty leaf node
	pg := buf[pageSize:]
	pg[0] = pageTypeLeaf
	binary.LittleEndian.PutUint16(pg[1:], 0)                // count = 0
	binary.LittleEndian.PutUint16(pg[3:], uint16(pageSize)) // freeOffset = end of page
	binary.LittleEndian.PutUint32(pg[5:], 0)                // nextLeaf = 0
	binary.LittleEndian.PutUint32(pg[9:], 0)                // prevLeaf = 0

	if _, err := s.file.WriteAt(buf, 0); err != nil {
		return fmt.Errorf("bear: init file: %w", err)
	}
	return s.file.Sync()
}

// loadMmap maps the entire file into memory.
func (s *store) loadMmap() error {
	info, err := s.file.Stat()
	if err != nil {
		return fmt.Errorf("bear: stat file: %w", err)
	}

	size := int(info.Size())
	if size < initialPages*pageSize {
		return fmt.Errorf("bear: file too small (%d bytes)", size)
	}

	data, err := syscall.Mmap(int(s.file.Fd()), 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("bear: mmap: %w", err)
	}

	s.mmap = data
	return nil
}

// remapIfNeeded grows the file and remaps if we need more pages.
// Caller must hold mu (write).
func (s *store) remapIfNeeded(requiredPages uint32) error {
	currentSize := len(s.mmap)
	neededSize := int(requiredPages) * pageSize

	if neededSize <= currentSize {
		return nil
	}

	if neededSize > maxBearFileSize {
		return fmt.Errorf("bear: file size would exceed limit (%d bytes)", maxBearFileSize)
	}

	// Grow file — use 2x growth factor so remaps are rare.
	newSize := currentSize
	for newSize < neededSize {
		if newSize < 1024*pageSize {
			newSize *= 2
		} else {
			newSize += 1024 * pageSize
		}
	}
	if newSize > maxBearFileSize {
		newSize = maxBearFileSize
	}

	if err := s.file.Truncate(int64(newSize)); err != nil {
		return fmt.Errorf("bear: truncate: %w", err)
	}

	// Unmap old
	if err := syscall.Munmap(s.mmap); err != nil {
		return fmt.Errorf("bear: munmap: %w", err)
	}

	// Remap
	data, err := syscall.Mmap(int(s.file.Fd()), 0, newSize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("bear: remap: %w", err)
	}
	s.mmap = data
	return nil
}

// ensureSpace pre-grows the mmap so that allocPage inside the subsequent
// write-locked critical section will not trigger a slow Munmap+Mmap cycle.
// It briefly acquires mu (write lock) to swap the mmap pointer, keeping the
// window where readers are blocked as short as possible.
//
// extraPages is a hint for how many pages the caller expects to allocate.
// The growth factor in remapIfNeeded means actual growth is typically much
// larger, so this only needs to be a rough estimate.
func (s *store) ensureSpace(extraPages uint32) {
	// Quick check without the write lock — if we have plenty of room, skip.
	s.mu.RLock()
	required := s.pageCount + extraPages
	currentCap := uint32(len(s.mmap) / pageSize)
	s.mu.RUnlock()

	if required <= currentCap {
		return
	}

	// Need to grow: acquire the write lock briefly for the remap.
	s.mu.Lock()
	// Re-check under write lock (another writer may have grown already).
	required = s.pageCount + extraPages
	err := s.remapIfNeeded(required)
	s.mu.Unlock()

	if err != nil {
		// Non-fatal here — allocPage will retry and return the error.
		_ = err
	}
}

func (s *store) readHeader() {
	s.rootPage = binary.LittleEndian.Uint32(s.mmap[hdrRootPage:])
	s.pageCount = binary.LittleEndian.Uint32(s.mmap[hdrPageCount:])
	s.height = binary.LittleEndian.Uint32(s.mmap[hdrHeight:])
	s.entryCount = binary.LittleEndian.Uint64(s.mmap[hdrEntryCount:])
	s.freeHead = binary.LittleEndian.Uint32(s.mmap[hdrFreeHead:])
}

func (s *store) writeHeader() {
	binary.LittleEndian.PutUint32(s.mmap[hdrRootPage:], s.rootPage)
	binary.LittleEndian.PutUint32(s.mmap[hdrPageCount:], s.pageCount)
	binary.LittleEndian.PutUint32(s.mmap[hdrHeight:], s.height)
	binary.LittleEndian.PutUint64(s.mmap[hdrEntryCount:], s.entryCount)
	binary.LittleEndian.PutUint32(s.mmap[hdrFreeHead:], s.freeHead)
}

func (s *store) syncPages() {
	if s.syncMode == "msync" {
		// Use file.Sync which flushes MAP_SHARED dirty pages to disk
		_ = s.file.Sync()
	}
}

// page returns a slice for the given page ID.
func (s *store) page(id uint32) []byte {
	off := int(id) * pageSize
	return s.mmap[off : off+pageSize]
}

// allocPage returns a new page ID, reusing free pages or extending the file.
// Caller must hold mu (write lock).
func (s *store) allocPage() (uint32, error) {
	if s.freeHead != 0 {
		id := s.freeHead
		pg := s.page(id)
		s.freeHead = binary.LittleEndian.Uint32(pg[0:4])
		// Zero the page
		for i := range pg {
			pg[i] = 0
		}
		return id, nil
	}

	id := s.pageCount
	s.pageCount++

	// Fast path: the mmap already has room (ensureSpace pre-grew it).
	neededSize := int(s.pageCount) * pageSize
	if neededSize <= len(s.mmap) {
		return id, nil
	}

	// Slow path: need remap. We hold mu.Lock() so readers (mu.RLock) are
	// blocked during the remap. ensureSpace() pre-grows to avoid this path.
	if err := s.remapIfNeeded(s.pageCount); err != nil {
		s.pageCount--
		return 0, err
	}

	return id, nil
}

// freePage adds a page to the free list.
func (s *store) freePage(id uint32) {
	pg := s.page(id)
	for i := range pg {
		pg[i] = 0
	}
	binary.LittleEndian.PutUint32(pg[0:4], s.freeHead)
	s.freeHead = id
}

// ---------------------------------------------------------------------------
// Composite key helpers
// ---------------------------------------------------------------------------

func shouldStoreExternal(n int) bool {
	return n > 0 && n > valLogThreshold
}

func compositeKey(bucket, key string) []byte {
	b := make([]byte, len(bucket)+1+len(key))
	copy(b, bucket)
	b[len(bucket)] = 0x00
	copy(b[len(bucket)+1:], key)
	return b
}

func splitCompositeKey(ck []byte) (bucket, key string) {
	idx := bytes.IndexByte(ck, 0x00)
	if idx < 0 {
		return string(ck), ""
	}
	return string(ck[:idx]), string(ck[idx+1:])
}

// bucketMetaKey returns the composite key used to store bucket metadata.
func bucketMetaKey(name string) []byte {
	return compositeKey("\x00bucket", name)
}

// keyHead extracts the first 4 bytes of a key for the head optimization.
func keyHead(k []byte) uint32 {
	var h [4]byte
	copy(h[:], k)
	return binary.BigEndian.Uint32(h[:])
}

// ---------------------------------------------------------------------------
// Leaf page operations
// ---------------------------------------------------------------------------

// leafEntry is an in-memory representation of a leaf entry.
// When valOffset >= 0 the value lives in the external value log at that
// offset (valLen bytes). When valOffset < 0 the value is stored inline
// in the B-tree page and the value field holds the actual data.
type leafEntry struct {
	key         []byte
	contentType []byte
	value       []byte // inline value (only when valOffset < 0)
	created     int64
	updated     int64
	valOffset   int64 // >=0: external offset in value log, <0: inline
	valLen      int64 // length of value in value log (only when valOffset >= 0)
}

// entrySize returns the on-disk size of the entry (excluding slot).
func (e *leafEntry) entrySize() int {
	if e.valOffset >= 0 {
		// External: no inline value data, just offset+length
		return leafEntryOverheadExternal + len(e.key) + len(e.contentType)
	}
	// Inline: value data is stored in the page
	return leafEntryOverheadInline + len(e.key) + len(e.contentType) + len(e.value)
}

// writeEntry writes the entry to dst and returns bytes written.
func (e *leafEntry) writeEntry(dst []byte) int {
	off := 0
	binary.LittleEndian.PutUint16(dst[off:], uint16(len(e.key)))
	off += 2
	copy(dst[off:], e.key)
	off += len(e.key)
	binary.LittleEndian.PutUint16(dst[off:], uint16(len(e.contentType)))
	off += 2
	copy(dst[off:], e.contentType)
	off += len(e.contentType)

	if e.valOffset >= 0 {
		// External value: flags(1) + valOffset(8) + valLen(8)
		dst[off] = flagExternal
		off++
		binary.LittleEndian.PutUint64(dst[off:], uint64(e.valOffset))
		off += 8
		binary.LittleEndian.PutUint64(dst[off:], uint64(e.valLen))
		off += 8
	} else {
		// Inline value: flags(1) + valLen(4) + val
		dst[off] = flagInline
		off++
		binary.LittleEndian.PutUint32(dst[off:], uint32(len(e.value)))
		off += 4
		copy(dst[off:], e.value)
		off += len(e.value)
	}

	binary.LittleEndian.PutUint64(dst[off:], uint64(e.created))
	off += 8
	binary.LittleEndian.PutUint64(dst[off:], uint64(e.updated))
	off += 8
	return off
}

// readLeafEntry reads an entry from page data at the given offset.
func readLeafEntry(pg []byte, offset uint16) *leafEntry {
	off := int(offset)
	if off+2 > len(pg) {
		return nil
	}

	keyLen := int(binary.LittleEndian.Uint16(pg[off:]))
	off += 2
	if off+keyLen+2 > len(pg) {
		return nil
	}
	key := make([]byte, keyLen)
	copy(key, pg[off:off+keyLen])
	off += keyLen

	ctLen := int(binary.LittleEndian.Uint16(pg[off:]))
	off += 2
	if off+ctLen+1 > len(pg) {
		return nil
	}
	ct := make([]byte, ctLen)
	copy(ct, pg[off:off+ctLen])
	off += ctLen

	// Read flags byte
	if off+1 > len(pg) {
		return nil
	}
	flags := pg[off]
	off++

	e := &leafEntry{
		key:         key,
		contentType: ct,
		valOffset:   -1, // default: inline
	}

	if flags == flagExternal {
		// External value: valOffset(8) + valLen(8)
		if off+16+16 > len(pg) {
			return nil
		}
		e.valOffset = int64(binary.LittleEndian.Uint64(pg[off:]))
		off += 8
		e.valLen = int64(binary.LittleEndian.Uint64(pg[off:]))
		off += 8
	} else {
		// Inline value: valLen(4) + val
		if off+4 > len(pg) {
			return nil
		}
		valLen := int(binary.LittleEndian.Uint32(pg[off:]))
		off += 4
		if off+valLen+16 > len(pg) {
			return nil
		}
		val := make([]byte, valLen)
		copy(val, pg[off:off+valLen])
		off += valLen
		e.value = val
	}

	if off+16 > len(pg) {
		return nil
	}
	e.created = int64(binary.LittleEndian.Uint64(pg[off:]))
	off += 8
	e.updated = int64(binary.LittleEndian.Uint64(pg[off:]))

	return e
}

// readAllLeafEntries reads all entries from a leaf page.
func readAllLeafEntries(pg []byte) []*leafEntry {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))
	entries := make([]*leafEntry, 0, count)
	for i := 0; i < count; i++ {
		slotOff := leafHeaderSize + i*leafSlotSize
		// skip keyHead (4B)
		entryOff := binary.LittleEndian.Uint16(pg[slotOff+4:])
		e := readLeafEntry(pg, entryOff)
		if e != nil {
			entries = append(entries, e)
		}
	}
	return entries
}

// writeLeafPage builds a leaf page from entries. Returns false if entries don't fit.
func writeLeafPage(pg []byte, entries []*leafEntry, nextLeaf, prevLeaf uint32) bool {
	// Sort entries by key
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].key, entries[j].key) < 0
	})

	// Check if everything fits
	slotsSize := leafHeaderSize + len(entries)*leafSlotSize
	dataSize := 0
	for _, e := range entries {
		dataSize += e.entrySize()
	}
	if slotsSize+dataSize > pageSize {
		return false
	}

	// Clear page
	for i := range pg {
		pg[i] = 0
	}

	// Write header
	pg[0] = pageTypeLeaf
	binary.LittleEndian.PutUint16(pg[1:], uint16(len(entries)))
	binary.LittleEndian.PutUint32(pg[5:], nextLeaf)
	binary.LittleEndian.PutUint32(pg[9:], prevLeaf)

	// Write entries from end backwards, slots from header forward
	freeOff := pageSize
	for i, e := range entries {
		sz := e.entrySize()
		freeOff -= sz
		e.writeEntry(pg[freeOff:])

		// Write slot
		slotOff := leafHeaderSize + i*leafSlotSize
		binary.BigEndian.PutUint32(pg[slotOff:], keyHead(e.key))
		binary.LittleEndian.PutUint16(pg[slotOff+4:], uint16(freeOff))
	}

	binary.LittleEndian.PutUint16(pg[3:], uint16(freeOff))
	return true
}

// leafSearch returns the index where key would be inserted (binary search with head opt).
func leafSearch(pg []byte, key []byte) (int, bool) {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))
	if count == 0 {
		return 0, false
	}

	head := keyHead(key)
	lo, hi := 0, count-1

	for lo <= hi {
		mid := lo + (hi-lo)/2
		slotOff := leafHeaderSize + mid*leafSlotSize
		midHead := binary.BigEndian.Uint32(pg[slotOff:])

		if midHead < head {
			lo = mid + 1
		} else if midHead > head {
			hi = mid - 1
		} else {
			// Heads match — compare full key
			entryOff := binary.LittleEndian.Uint16(pg[slotOff+4:])
			keyLen := int(binary.LittleEndian.Uint16(pg[entryOff:]))
			entryKey := pg[entryOff+2 : entryOff+2+uint16(keyLen)]
			cmp := bytes.Compare(entryKey, key)
			if cmp == 0 {
				return mid, true
			} else if cmp < 0 {
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
	}
	return lo, false
}

// leafFreeSpace returns available free space in the leaf page.
func leafFreeSpace(pg []byte) int {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))
	freeOff := int(binary.LittleEndian.Uint16(pg[3:5]))
	slotsEnd := leafHeaderSize + count*leafSlotSize
	return freeOff - slotsEnd
}

// ---------------------------------------------------------------------------
// Inner page operations
// ---------------------------------------------------------------------------

// readInnerKey reads the key at slot index i from an inner page.
func readInnerKey(pg []byte, i int) []byte {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))
	// Children come first: (count+1) * 4 bytes
	// Then keys: count * innerSlotSize
	keySlotOff := innerHeaderSize + (count+1)*innerChildSize + i*innerSlotSize
	// keyHead at keySlotOff, keyOffset at keySlotOff+4
	keyOff := binary.LittleEndian.Uint16(pg[keySlotOff+4:])

	// Read key: [keyLen(2B)] [key data]
	keyLen := int(binary.LittleEndian.Uint16(pg[keyOff:]))
	key := make([]byte, keyLen)
	copy(key, pg[keyOff+2:keyOff+2+uint16(keyLen)])
	return key
}

// readInnerChild reads the child page ID at position i.
func readInnerChild(pg []byte, i int) uint32 {
	off := innerHeaderSize + i*innerChildSize
	return binary.LittleEndian.Uint32(pg[off:])
}

// innerSearch finds the child index for the given key.
func innerSearch(pg []byte, key []byte) int {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))
	if count == 0 {
		return 0
	}

	head := keyHead(key)
	lo, hi := 0, count-1

	for lo <= hi {
		mid := lo + (hi-lo)/2
		keySlotOff := innerHeaderSize + (count+1)*innerChildSize + mid*innerSlotSize
		midHead := binary.BigEndian.Uint32(pg[keySlotOff:])

		if midHead < head {
			lo = mid + 1
		} else if midHead > head {
			hi = mid - 1
		} else {
			// Heads match — compare full key
			keyOff := binary.LittleEndian.Uint16(pg[keySlotOff+4:])
			keyLen := int(binary.LittleEndian.Uint16(pg[keyOff:]))
			entryKey := pg[keyOff+2 : keyOff+2+uint16(keyLen)]
			cmp := bytes.Compare(entryKey, key)
			if cmp == 0 {
				return mid + 1
			} else if cmp < 0 {
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
	}
	return lo
}

// innerKeyEntry holds key data for inner node reconstruction.
type innerKeyEntry struct {
	key   []byte
	child uint32 // right child of this key
}

// writeInnerPage builds an inner page. keys[i] separates children[i] and children[i+1].
func writeInnerPage(pg []byte, keys [][]byte, children []uint32) bool {
	count := len(keys)
	if len(children) != count+1 {
		return false
	}

	// Calculate space needed
	childrenSize := (count + 1) * innerChildSize
	slotsSize := count * innerSlotSize
	headerAndSlots := innerHeaderSize + childrenSize + slotsSize
	dataSize := 0
	for _, k := range keys {
		dataSize += 2 + len(k) // keyLen(2B) + key
	}
	if headerAndSlots+dataSize > pageSize {
		return false
	}

	// Clear
	for i := range pg {
		pg[i] = 0
	}

	// Header
	pg[0] = pageTypeInner
	binary.LittleEndian.PutUint16(pg[1:], uint16(count))

	// Write children
	for i, c := range children {
		off := innerHeaderSize + i*innerChildSize
		binary.LittleEndian.PutUint32(pg[off:], c)
	}

	// Write keys from end backwards
	freeOff := pageSize
	for i, k := range keys {
		sz := 2 + len(k)
		freeOff -= sz
		binary.LittleEndian.PutUint16(pg[freeOff:], uint16(len(k)))
		copy(pg[freeOff+2:], k)

		// Write key slot
		slotOff := innerHeaderSize + (count+1)*innerChildSize + i*innerSlotSize
		binary.BigEndian.PutUint32(pg[slotOff:], keyHead(k))
		binary.LittleEndian.PutUint16(pg[slotOff+4:], uint16(freeOff))
	}

	binary.LittleEndian.PutUint16(pg[3:], uint16(freeOff))
	return true
}

// ---------------------------------------------------------------------------
// B-tree operations
// ---------------------------------------------------------------------------

// splitResult holds the result of a page split.
type splitResult struct {
	newPageID uint32
	splitKey  []byte
}

// btreeInsert inserts a key-value pair into the B-tree. Returns a splitResult if
// the root split (caller must create new root).
func (s *store) btreeInsert(entry *leafEntry) (*splitResult, error) {
	split, err := s.insertInto(s.rootPage, entry, int(s.height))
	if err != nil {
		return nil, err
	}

	if split != nil {
		// Root split — create new root
		newRootID, err := s.allocPage()
		if err != nil {
			return nil, err
		}
		pg := s.page(newRootID)
		ok := writeInnerPage(pg, [][]byte{split.splitKey}, []uint32{s.rootPage, split.newPageID})
		if !ok {
			return nil, fmt.Errorf("bear: failed to write new root")
		}
		s.rootPage = newRootID
		s.height++
	}
	return nil, nil
}

// insertInto recursively inserts into the subtree rooted at pageID.
func (s *store) insertInto(pageID uint32, entry *leafEntry, level int) (*splitResult, error) {
	if level == 1 {
		// Leaf level — page read inside insertIntoLeaf
		return s.insertIntoLeaf(pageID, entry)
	}

	// Inner node — find child
	pg := s.page(pageID)
	childIdx := innerSearch(pg, entry.key)
	childID := readInnerChild(pg, childIdx)

	split, err := s.insertInto(childID, entry, level-1)
	if err != nil {
		return nil, err
	}

	if split == nil {
		return nil, nil
	}

	// Child split — re-read page (mmap may have changed during recursive insert)
	pg = s.page(pageID)
	return s.insertIntoInner(pageID, pg, split, childIdx)
}

// insertIntoLeaf inserts an entry into a leaf page, splitting if necessary.
func (s *store) insertIntoLeaf(pageID uint32, entry *leafEntry) (*splitResult, error) {
	pg := s.page(pageID)
	idx, found := leafSearch(pg, entry.key)

	if found {
		// Update existing entry — read all, replace, rewrite
		entries := readAllLeafEntries(pg)
		for _, e := range entries {
			if bytes.Equal(e.key, entry.key) {
				e.contentType = entry.contentType
				e.value = entry.value
				e.updated = entry.updated
				e.valOffset = entry.valOffset
				e.valLen = entry.valLen
				break
			}
		}
		nextLeaf := binary.LittleEndian.Uint32(pg[5:])
		prevLeaf := binary.LittleEndian.Uint32(pg[9:])
		if !writeLeafPage(pg, entries, nextLeaf, prevLeaf) {
			return nil, fmt.Errorf("bear: leaf rewrite failed (page %d)", pageID)
		}
		return nil, nil
	}

	// Check if entry fits
	neededSpace := leafSlotSize + entry.entrySize()
	if leafFreeSpace(pg) >= neededSpace {
		// Insert in-place
		s.leafInsertAt(pg, idx, entry)
		return nil, nil
	}

	// Need to split — read all entries before allocating (which may remap)
	entries := readAllLeafEntries(pg)
	nextLeaf := binary.LittleEndian.Uint32(pg[5:])
	prevLeaf := binary.LittleEndian.Uint32(pg[9:])

	entries = append(entries, entry)
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].key, entries[j].key) < 0
	})

	mid := len(entries) / 2
	left := entries[:mid]
	right := entries[mid:]

	// Allocate new page for right half — this may remap!
	newID, err := s.allocPage()
	if err != nil {
		return nil, err
	}

	// Re-read page references after potential remap
	pg = s.page(pageID)

	// Write left half to current page
	if !writeLeafPage(pg, left, newID, prevLeaf) {
		return nil, fmt.Errorf("bear: left leaf write failed")
	}

	// Write right half to new page
	newPg := s.page(newID)
	if !writeLeafPage(newPg, right, nextLeaf, pageID) {
		return nil, fmt.Errorf("bear: right leaf write failed")
	}

	// Update the old next leaf's prevLeaf pointer
	if nextLeaf != 0 {
		nextPg := s.page(nextLeaf)
		binary.LittleEndian.PutUint32(nextPg[9:], newID) // prevLeaf
	}

	return &splitResult{
		newPageID: newID,
		splitKey:  copyBytes(right[0].key),
	}, nil
}

// leafInsertAt inserts entry at position idx in the leaf page.
func (s *store) leafInsertAt(pg []byte, idx int, entry *leafEntry) {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))
	freeOff := int(binary.LittleEndian.Uint16(pg[3:5]))

	// Write entry data at freeOff - entrySize
	sz := entry.entrySize()
	freeOff -= sz
	entry.writeEntry(pg[freeOff:])

	// Shift slots right to make room at idx
	if idx < count {
		src := leafHeaderSize + idx*leafSlotSize
		dst := leafHeaderSize + (idx+1)*leafSlotSize
		copy(pg[dst:dst+(count-idx)*leafSlotSize], pg[src:src+(count-idx)*leafSlotSize])
	}

	// Write slot at idx
	slotOff := leafHeaderSize + idx*leafSlotSize
	binary.BigEndian.PutUint32(pg[slotOff:], keyHead(entry.key))
	binary.LittleEndian.PutUint16(pg[slotOff+4:], uint16(freeOff))

	// Update header
	binary.LittleEndian.PutUint16(pg[1:], uint16(count+1))
	binary.LittleEndian.PutUint16(pg[3:], uint16(freeOff))
}

// insertIntoInner inserts a split result into an inner node, splitting if necessary.
func (s *store) insertIntoInner(pageID uint32, pg []byte, split *splitResult, childIdx int) (*splitResult, error) {
	count := int(binary.LittleEndian.Uint16(pg[1:3]))

	// Read all keys and children
	keys := make([][]byte, count)
	for i := 0; i < count; i++ {
		keys[i] = readInnerKey(pg, i)
	}
	children := make([]uint32, count+1)
	for i := 0; i <= count; i++ {
		children[i] = readInnerChild(pg, i)
	}

	// Insert split key and new child
	newKeys := make([][]byte, count+1)
	newChildren := make([]uint32, count+2)

	copy(newKeys, keys[:childIdx])
	newKeys[childIdx] = split.splitKey
	copy(newKeys[childIdx+1:], keys[childIdx:])

	copy(newChildren, children[:childIdx+1])
	newChildren[childIdx+1] = split.newPageID
	copy(newChildren[childIdx+2:], children[childIdx+1:])

	// Try to fit in current page
	if writeInnerPage(pg, newKeys, newChildren) {
		return nil, nil
	}

	// Need to split inner node
	mid := len(newKeys) / 2
	leftKeys := newKeys[:mid]
	rightKeys := newKeys[mid+1:]
	splitKey := newKeys[mid]
	leftChildren := newChildren[:mid+1]
	rightChildren := newChildren[mid+1:]

	newID, err := s.allocPage()
	if err != nil {
		return nil, err
	}

	// Re-read page after potential remap from allocPage
	pg = s.page(pageID)
	if !writeInnerPage(pg, leftKeys, leftChildren) {
		return nil, fmt.Errorf("bear: left inner write failed")
	}
	newPg := s.page(newID)
	if !writeInnerPage(newPg, rightKeys, rightChildren) {
		return nil, fmt.Errorf("bear: right inner write failed")
	}

	return &splitResult{
		newPageID: newID,
		splitKey:  splitKey,
	}, nil
}

// btreeGet retrieves the entry for the given key.
func (s *store) btreeGet(key []byte) *leafEntry {
	pageID := s.rootPage
	level := int(s.height)

	for level > 1 {
		pg := s.page(pageID)
		childIdx := innerSearch(pg, key)
		pageID = readInnerChild(pg, childIdx)
		level--
	}

	pg := s.page(pageID)
	idx, found := leafSearch(pg, key)
	if !found {
		_ = idx
		return nil
	}

	slotOff := leafHeaderSize + idx*leafSlotSize
	entryOff := binary.LittleEndian.Uint16(pg[slotOff+4:])
	return readLeafEntry(pg, entryOff)
}

// btreeDelete removes the entry for the given key. Returns true if found.
func (s *store) btreeDelete(key []byte) bool {
	pageID := s.rootPage
	level := int(s.height)

	for level > 1 {
		pg := s.page(pageID)
		childIdx := innerSearch(pg, key)
		pageID = readInnerChild(pg, childIdx)
		level--
	}

	pg := s.page(pageID)
	_, found := leafSearch(pg, key)
	if !found {
		return false
	}

	// Remove entry — rewrite the page without the deleted entry
	entries := readAllLeafEntries(pg)
	filtered := make([]*leafEntry, 0, len(entries)-1)
	for _, e := range entries {
		if !bytes.Equal(e.key, key) {
			filtered = append(filtered, e)
		}
	}

	nextLeaf := binary.LittleEndian.Uint32(pg[5:])
	prevLeaf := binary.LittleEndian.Uint32(pg[9:])
	writeLeafPage(pg, filtered, nextLeaf, prevLeaf)
	return true
}

// btreeScan iterates over all entries with keys >= startKey.
// It calls fn for each entry; fn returns false to stop iteration.
// Caller must hold mu (read or write lock) to prevent concurrent remap
// from invalidating the mmap slice. The only exception is single-threaded
// init paths (e.g. loadBucketMeta) where the store is not yet shared.
func (s *store) btreeScan(startKey []byte, fn func(e *leafEntry) bool) {
	// Find the leaf containing startKey
	pageID := s.rootPage
	level := int(s.height)

	for level > 1 {
		pg := s.page(pageID)
		childIdx := innerSearch(pg, startKey)
		pageID = readInnerChild(pg, childIdx)
		level--
	}

	// Scan from this leaf forward
	for pageID != 0 {
		pg := s.page(pageID)
		count := int(binary.LittleEndian.Uint16(pg[1:3]))

		for i := 0; i < count; i++ {
			slotOff := leafHeaderSize + i*leafSlotSize
			entryOff := binary.LittleEndian.Uint16(pg[slotOff+4:])
			e := readLeafEntry(pg, entryOff)
			if e == nil {
				continue
			}
			if bytes.Compare(e.key, startKey) < 0 {
				continue
			}
			if !fn(e) {
				return
			}
		}

		pageID = binary.LittleEndian.Uint32(pg[5:]) // nextLeaf
	}
}

// ---------------------------------------------------------------------------
// Bucket metadata persistence
// ---------------------------------------------------------------------------

func (s *store) loadBucketMeta() {
	prefix := bucketMetaKey("")
	s.btreeScan(prefix, func(e *leafEntry) bool {
		if !bytes.HasPrefix(e.key, []byte("\x00bucket\x00")) {
			return false
		}
		name := string(e.key[len("\x00bucket\x00"):])
		s.buckets[name] = time.Unix(0, e.created)
		return true
	})
}

func (s *store) saveBucketMeta(name string, created time.Time) error {
	key := bucketMetaKey(name)
	entry := &leafEntry{
		key:       key,
		created:   created.UnixNano(),
		updated:   created.UnixNano(),
		valOffset: -1, // inline (no value data)
	}
	_, err := s.btreeInsert(entry)
	return err
}

func (s *store) deleteBucketMeta(name string) {
	key := bucketMetaKey(name)
	s.btreeDelete(key)
}

// ---------------------------------------------------------------------------
// storage.Storage implementation
// ---------------------------------------------------------------------------

func (s *store) Bucket(name string) storage.Bucket {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	name = safeBucketName(name)
	return &bucket{store: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.bucketsMu.RLock()
	list := make([]*storage.BucketInfo, 0, len(s.buckets))
	for name, created := range s.buckets {
		list = append(list, &storage.BucketInfo{
			Name:      name,
			CreatedAt: created,
		})
	}
	s.bucketsMu.RUnlock()

	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

	if offset < 0 {
		offset = 0
	}
	if offset > len(list) {
		offset = len(list)
	}
	list = list[offset:]
	if limit > 0 && limit < len(list) {
		list = list[:limit]
	}

	return &bucketIter{list: list}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("bear: bucket name required")
	}
	name = safeBucketName(name)

	s.bucketsMu.Lock()
	defer s.bucketsMu.Unlock()

	if _, exists := s.buckets[name]; exists {
		return nil, storage.ErrExist
	}

	if len(s.buckets) >= maxBuckets {
		return nil, fmt.Errorf("bear: too many buckets (max %d)", maxBuckets)
	}

	now := time.Now()

	s.ensureSpace(4)

	s.mu.Lock()
	err := s.saveBucketMeta(name, now)
	s.writeHeader()
	s.syncPages()
	s.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("bear: save bucket meta: %w", err)
	}

	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("bear: bucket name required")
	}
	name = safeBucketName(name)

	force := boolOpt(opts, "force")

	s.bucketsMu.Lock()
	defer s.bucketsMu.Unlock()

	if _, exists := s.buckets[name]; !exists {
		return storage.ErrNotExist
	}

	// Check if bucket has objects (unless force)
	if !force {
		prefix := compositeKey(name, "")
		hasEntries := false
		s.mu.RLock()
		s.btreeScan(prefix, func(e *leafEntry) bool {
			if bytes.HasPrefix(e.key, prefix) {
				hasEntries = true
				return false
			}
			return false
		})
		s.mu.RUnlock()
		if hasEntries {
			return storage.ErrPermission
		}
	}

	s.ensureSpace(4)

	s.mu.Lock()

	// Delete all objects in bucket if force
	if force {
		prefix := compositeKey(name, "")
		var toDelete [][]byte
		s.btreeScan(prefix, func(e *leafEntry) bool {
			if bytes.HasPrefix(e.key, prefix) {
				toDelete = append(toDelete, copyBytes(e.key))
				return true
			}
			return false
		})
		for _, k := range toDelete {
			if s.btreeDelete(k) {
				s.entryCount--
			}
		}
	}

	s.deleteBucketMeta(name)
	s.writeHeader()
	s.syncPages()
	s.mu.Unlock()

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":        true,
		"directories": true,
		"multipart":   true,
	}
}

func (s *store) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	s.mu.Lock()

	s.writeHeader()
	if s.syncMode == "msync" {
		_ = s.file.Sync()
	}
	if s.mmap != nil {
		_ = syscall.Munmap(s.mmap)
		s.mmap = nil
	}

	s.mu.Unlock()

	// Close value log
	s.valMu.Lock()
	if s.valLog != nil {
		_ = s.flushValueLogLocked()
		if s.syncMode == "msync" {
			_ = s.valLog.Sync()
		}
		_ = s.valLog.Close()
		s.valLog = nil
	}
	s.valMu.Unlock()

	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Bucket
// ---------------------------------------------------------------------------

type bucket struct {
	store *store
	name  string
}

var (
	_ storage.Bucket       = (*bucket)(nil)
	_ storage.HasMultipart = (*bucket)(nil)
)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features {
	return b.store.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.store.bucketsMu.RLock()
	created, exists := b.store.buckets[b.name]
	b.store.bucketsMu.RUnlock()

	if !exists {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: created,
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	ck := compositeKey(b.name, relKey)
	now := time.Now()
	ctBytes := []byte(contentType)

	var (
		data      []byte
		entry     *leafEntry
		actualLen int64
	)

	// Stream known-size external values directly into the buffered value log to
	// avoid allocating a per-write value slice on the Go heap.
	if size > 0 && shouldStoreExternal(int(size)) {
		offset, wrote, werr := b.store.writeStreamToValueLog(src, size)
		if werr != nil {
			return nil, fmt.Errorf("bear: write stream: %w", werr)
		}
		actualLen = wrote
		if wrote > 0 {
			entry = &leafEntry{
				key:         ck,
				contentType: ctBytes,
				created:     now.UnixNano(),
				updated:     now.UnixNano(),
				valOffset:   offset,
				valLen:      wrote,
			}
		} else {
			entry = &leafEntry{
				key:         ck,
				contentType: ctBytes,
				value:       nil,
				created:     now.UnixNano(),
				updated:     now.UnixNano(),
				valOffset:   -1,
			}
		}
	} else {
		// Read all data (fallback path for tiny/unknown-size writes).
		if size > 0 {
			data = make([]byte, size)
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
				return nil, fmt.Errorf("bear: read: %w", err)
			}
			data = data[:n]
		} else if size < 0 {
			var err error
			data, err = io.ReadAll(src)
			if err != nil {
				return nil, fmt.Errorf("bear: read: %w", err)
			}
		} else {
			// size == 0, read anyway in case there's data
			data, _ = io.ReadAll(src)
		}
		actualLen = int64(len(data))

		// Write large values to the value log before acquiring the main lock.
		var err error
		entry, err = b.store.prepareEntry(ck, ctBytes, data, now.UnixNano(), now.UnixNano())
		if err != nil {
			return nil, fmt.Errorf("bear: prepare entry: %w", err)
		}
	}

	// Pre-grow the mmap so that allocPage inside the write lock won't
	// block readers with a slow Munmap+Mmap cycle.
	b.store.ensureSpace(4)

	b.store.mu.Lock()

	// Check if this is an update (preserve created time)
	existing := b.store.btreeGet(ck)
	if existing != nil {
		entry.created = existing.created
	} else {
		b.store.entryCount++
	}

	_, err = b.store.btreeInsert(entry)
	b.store.writeHeader()
	b.store.syncPages()
	b.store.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("bear: insert: %w", err)
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        actualLen,
		ContentType: contentType,
		Created:     time.Unix(0, entry.created),
		Updated:     now,
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, nil, err
	}

	ck := compositeKey(b.name, relKey)

	b.store.mu.RLock()
	entry := b.store.btreeGet(ck)
	b.store.mu.RUnlock()

	if entry == nil {
		return nil, nil, storage.ErrNotExist
	}

	fullSize := int64(len(entry.value))
	if entry.valOffset >= 0 {
		fullSize = entry.valLen
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        fullSize,
		ContentType: string(entry.contentType),
		Created:     time.Unix(0, entry.created),
		Updated:     time.Unix(0, entry.updated),
	}

	// External values can be streamed directly from the value log without
	// allocating an intermediate byte slice.
	if entry.valOffset >= 0 {
		readOff := entry.valOffset
		readLen := entry.valLen
		if offset > 0 {
			if offset >= readLen {
				readLen = 0
			} else {
				readOff += offset
				readLen -= offset
			}
		}
		if length > 0 && readLen > length {
			readLen = length
		}
		rc, err := b.store.openValueLogReader(readOff, readLen)
		if err != nil {
			return nil, nil, err
		}
		return rc, obj, nil
	}

	data := entry.value

	// Apply range
	if offset > 0 {
		if offset >= int64(len(data)) {
			data = nil
		} else {
			data = data[offset:]
		}
	}
	if length > 0 && int64(len(data)) > length {
		data = data[:length]
	}

	return io.NopCloser(bytes.NewReader(data)), obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	ck := compositeKey(b.name, relKey)

	b.store.mu.RLock()
	entry := b.store.btreeGet(ck)
	b.store.mu.RUnlock()

	if entry == nil {
		// Check if this is a directory prefix
		prefix := compositeKey(b.name, relKey+"/")
		isDir := false
		b.store.mu.RLock()
		b.store.btreeScan(prefix, func(e *leafEntry) bool {
			if bytes.HasPrefix(e.key, prefix) {
				isDir = true
			}
			return false
		})
		b.store.mu.RUnlock()

		if isDir {
			return &storage.Object{
				Bucket: b.name,
				Key:    relToKey(relKey),
				IsDir:  true,
			}, nil
		}
		return nil, storage.ErrNotExist
	}

	// Compute size: for external values use valLen, for inline use len(value).
	sz := int64(len(entry.value))
	if entry.valOffset >= 0 {
		sz = entry.valLen
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(relKey),
		Size:        sz,
		ContentType: string(entry.contentType),
		Created:     time.Unix(0, entry.created),
		Updated:     time.Unix(0, entry.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return err
	}

	recursive := boolOpt(opts, "recursive")

	b.store.mu.Lock()
	defer func() {
		b.store.writeHeader()
		b.store.syncPages()
		b.store.mu.Unlock()
	}()

	if recursive {
		prefix := compositeKey(b.name, relKey)
		var toDelete [][]byte

		// Collect keys to delete (exact match + prefix/)
		exact := b.store.btreeGet(prefix)
		if exact != nil {
			toDelete = append(toDelete, copyBytes(prefix))
		}

		dirPrefix := compositeKey(b.name, relKey+"/")
		b.store.btreeScan(dirPrefix, func(e *leafEntry) bool {
			if bytes.HasPrefix(e.key, dirPrefix) {
				toDelete = append(toDelete, copyBytes(e.key))
				return true
			}
			return false
		})

		if len(toDelete) == 0 {
			return storage.ErrNotExist
		}

		for _, k := range toDelete {
			if b.store.btreeDelete(k) {
				b.store.entryCount--
			}
		}
		return nil
	}

	ck := compositeKey(b.name, relKey)
	if !b.store.btreeDelete(ck) {
		return storage.ErrNotExist
	}
	b.store.entryCount--
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcRel, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	dstRel, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}

	srcBucketName := safeBucketName(strings.TrimSpace(srcBucket))
	srcCK := compositeKey(srcBucketName, srcRel)

	// Read the source entry (under read lock) and resolve its value.
	b.store.mu.RLock()
	srcEntry := b.store.btreeGet(srcCK)
	b.store.mu.RUnlock()

	if srcEntry == nil {
		return nil, storage.ErrNotExist
	}

	dstCK := compositeKey(b.name, dstRel)
	now := time.Now()

	var (
		newEntry *leafEntry
		valSize  int64
	)
	if srcEntry.valOffset >= 0 {
		valSize = srcEntry.valLen
		newEntry = &leafEntry{
			key:         dstCK,
			contentType: copyBytes(srcEntry.contentType),
			created:     now.UnixNano(),
			updated:     now.UnixNano(),
			valOffset:   srcEntry.valOffset,
			valLen:      srcEntry.valLen,
		}
	} else {
		valSize = int64(len(srcEntry.value))
		// Prepare the destination entry (may write to value log).
		newEntry, err = b.store.prepareEntry(dstCK, copyBytes(srcEntry.contentType), copyBytes(srcEntry.value), now.UnixNano(), now.UnixNano())
		if err != nil {
			return nil, fmt.Errorf("bear: copy prepare: %w", err)
		}
	}

	b.store.ensureSpace(4)

	b.store.mu.Lock()
	existing := b.store.btreeGet(dstCK)
	if existing == nil {
		b.store.entryCount++
	}

	_, err = b.store.btreeInsert(newEntry)
	b.store.writeHeader()
	b.store.syncPages()
	b.store.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("bear: copy insert: %w", err)
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(dstRel),
		Size:        valSize,
		ContentType: string(srcEntry.contentType),
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcRel, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	dstRel, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}

	srcBucketName := safeBucketName(strings.TrimSpace(srcBucket))
	srcCK := compositeKey(srcBucketName, srcRel)

	// Read the source entry (under read lock) and resolve its value.
	b.store.mu.RLock()
	srcEntry := b.store.btreeGet(srcCK)
	b.store.mu.RUnlock()

	if srcEntry == nil {
		return nil, storage.ErrNotExist
	}

	dstCK := compositeKey(b.name, dstRel)
	now := time.Now()

	var (
		newEntry *leafEntry
		valSize  int64
	)
	if srcEntry.valOffset >= 0 {
		valSize = srcEntry.valLen
		newEntry = &leafEntry{
			key:         dstCK,
			contentType: copyBytes(srcEntry.contentType),
			created:     srcEntry.created,
			updated:     now.UnixNano(),
			valOffset:   srcEntry.valOffset,
			valLen:      srcEntry.valLen,
		}
	} else {
		valSize = int64(len(srcEntry.value))
		// Prepare the destination entry (may write to value log).
		newEntry, err = b.store.prepareEntry(dstCK, copyBytes(srcEntry.contentType), copyBytes(srcEntry.value), srcEntry.created, now.UnixNano())
		if err != nil {
			return nil, fmt.Errorf("bear: move prepare: %w", err)
		}
	}

	b.store.ensureSpace(4)

	b.store.mu.Lock()
	existing := b.store.btreeGet(dstCK)
	if existing == nil {
		b.store.entryCount++
	}

	_, err = b.store.btreeInsert(newEntry)
	if err != nil {
		b.store.mu.Unlock()
		return nil, fmt.Errorf("bear: move insert: %w", err)
	}

	if b.store.btreeDelete(srcCK) {
		b.store.entryCount--
	}

	b.store.writeHeader()
	b.store.syncPages()
	b.store.mu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(dstRel),
		Size:        valSize,
		ContentType: string(newEntry.contentType),
		Created:     time.Unix(0, newEntry.created),
		Updated:     now,
	}, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	recursive := true
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}

	relPrefix, err := cleanPrefix(prefix)
	if err != nil {
		return nil, err
	}

	scanPrefix := compositeKey(b.name, relPrefix)

	var objects []*storage.Object
	seenDirs := make(map[string]bool)

	b.store.mu.RLock()
	b.store.btreeScan(scanPrefix, func(e *leafEntry) bool {
		if !bytes.HasPrefix(e.key, scanPrefix) {
			return false
		}

		eBucket, eKey := splitCompositeKey(e.key)
		if eBucket != b.name {
			return false
		}

		// Skip bucket meta keys
		if strings.HasPrefix(eBucket, "\x00") {
			return true
		}

		if !recursive {
			// Non-recursive: only show entries directly under prefix
			suffix := eKey
			if relPrefix != "" {
				suffix = strings.TrimPrefix(eKey, relPrefix)
				if suffix == eKey {
					return true // doesn't match prefix
				}
				if len(suffix) > 0 && suffix[0] == '/' {
					suffix = suffix[1:]
				}
			}

			if idx := strings.IndexByte(suffix, '/'); idx >= 0 {
				// This is under a subdirectory
				dirName := suffix[:idx]
				dirKey := relPrefix
				if dirKey != "" {
					dirKey += "/"
				}
				dirKey += dirName

				if !seenDirs[dirKey] {
					seenDirs[dirKey] = true
					objects = append(objects, &storage.Object{
						Bucket: b.name,
						Key:    dirKey,
						IsDir:  true,
					})
				}
				return true
			}
		}

		// Compute size: for external values use valLen, for inline use len(value).
		sz := int64(len(e.value))
		if e.valOffset >= 0 {
			sz = e.valLen
		}

		objects = append(objects, &storage.Object{
			Bucket:      b.name,
			Key:         eKey,
			Size:        sz,
			ContentType: string(e.contentType),
			Created:     time.Unix(0, e.created),
			Updated:     time.Unix(0, e.updated),
		})
		return true
	})
	b.store.mu.RUnlock()

	sort.Slice(objects, func(i, j int) bool { return objects[i].Key < objects[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objects) {
		offset = len(objects)
	}
	objects = objects[offset:]
	if limit > 0 && limit < len(objects) {
		objects = objects[:limit]
	}

	return &objectIter{list: objects}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// HasDirectories
// ---------------------------------------------------------------------------

var _ storage.HasDirectories = (*bucket)(nil)

func (b *bucket) Directory(dirPath string) storage.Directory {
	return &directory{bucket: b, path: cleanDirPath(dirPath)}
}

type directory struct {
	bucket *bucket
	path   string
}

var _ storage.Directory = (*directory)(nil)

func (d *directory) Bucket() storage.Bucket { return d.bucket }
func (d *directory) Path() string           { return d.path }

func (d *directory) Info(ctx context.Context) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &storage.Object{
		Bucket: d.bucket.name,
		Key:    d.path,
		IsDir:  true,
	}, nil
}

func (d *directory) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if opts == nil {
		opts = storage.Options{}
	}
	opts["recursive"] = false

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return d.bucket.List(ctx, prefix, limit, offset, opts)
}

func (d *directory) Delete(ctx context.Context, opts storage.Options) error {
	if opts == nil {
		opts = storage.Options{}
	}
	opts["recursive"] = true
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Delete all objects under this directory
	iter, err := d.bucket.List(ctx, prefix, 0, 0, storage.Options{"recursive": true})
	if err != nil {
		return err
	}
	defer iter.Close()

	var keys []string
	for {
		obj, err := iter.Next()
		if err != nil {
			return err
		}
		if obj == nil {
			break
		}
		if !obj.IsDir {
			keys = append(keys, obj.Key)
		}
	}

	for _, k := range keys {
		if err := d.bucket.Delete(ctx, k, nil); err != nil && !errors.Is(err, storage.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (d *directory) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcPrefix := d.path
	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	dstPrefix := cleanDirPath(dstPath)
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	iter, err := d.bucket.List(ctx, srcPrefix, 0, 0, storage.Options{"recursive": true})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var keys []string
	for {
		obj, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if obj == nil {
			break
		}
		if !obj.IsDir {
			keys = append(keys, obj.Key)
		}
	}

	for _, k := range keys {
		suffix := strings.TrimPrefix(k, srcPrefix)
		newKey := dstPrefix + suffix
		if _, err := d.bucket.Move(ctx, newKey, d.bucket.name, k, nil); err != nil {
			return nil, err
		}
	}

	return &directory{bucket: d.bucket, path: cleanDirPath(dstPath)}, nil
}

// ---------------------------------------------------------------------------
// Multipart
// ---------------------------------------------------------------------------

type multipartState struct {
	id          string
	key         string
	contentType string
	parts       map[int]*partData
	created     time.Time
	metadata    map[string]string
}

type partData struct {
	number int
	data   []byte
	etag   string
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	relKey, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	id := strconv.FormatInt(b.store.mpCounter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	st := &multipartState{
		id:          id,
		key:         relKey,
		contentType: contentType,
		parts:       make(map[int]*partData),
		created:     time.Now(),
		metadata:    metadata,
	}

	b.store.mpMu.Lock()
	b.store.mpUploads[id] = st
	b.store.mpMu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      relToKey(relKey),
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("bear: part number %d out of range [1, %d]", number, maxPartNumber)
	}

	b.store.mpMu.Lock()
	st, ok := b.store.mpUploads[mu.UploadID]
	b.store.mpMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("bear: read part: %w", err)
	}

	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.store.mpMu.Lock()
	st.parts[number] = &partData{
		number: number,
		data:   data,
		etag:   etag,
	}
	b.store.mpMu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   int64(len(data)),
		ETag:   etag,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("bear: part number %d out of range", number)
	}

	b.store.mpMu.Lock()
	_, ok := b.store.mpUploads[mu.UploadID]
	b.store.mpMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("bear: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	// Read source
	srcBucketName := safeBucketName(srcBucket)
	srcRel, err := cleanKey(srcKey)
	if err != nil {
		return nil, err
	}
	srcCK := compositeKey(srcBucketName, srcRel)

	b.store.mu.RLock()
	srcEntry := b.store.btreeGet(srcCK)
	b.store.mu.RUnlock()

	if srcEntry == nil {
		return nil, storage.ErrNotExist
	}

	// Resolve external values from the value log.
	srcEntry, err = b.store.resolveValue(srcEntry)
	if err != nil {
		return nil, fmt.Errorf("bear: copy part resolve: %w", err)
	}

	data := srcEntry.value
	if srcOffset > 0 {
		if srcOffset >= int64(len(data)) {
			data = nil
		} else {
			data = data[srcOffset:]
		}
	}
	if srcLength > 0 && int64(len(data)) > srcLength {
		data = data[:srcLength]
	}

	return b.UploadPart(ctx, mu, number, bytes.NewReader(data), int64(len(data)), opts)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.store.mpMu.Lock()
	st, ok := b.store.mpUploads[mu.UploadID]
	if !ok {
		b.store.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(st.parts))
	for _, p := range st.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   int64(len(p.data)),
			ETag:   p.etag,
		})
	}
	b.store.mpMu.Unlock()

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })

	if offset > 0 && offset < len(parts) {
		parts = parts[offset:]
	}
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.store.mpMu.Lock()
	st, ok := b.store.mpUploads[mu.UploadID]
	if !ok {
		b.store.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.store.mpUploads, mu.UploadID)
	b.store.mpMu.Unlock()

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })

	// Verify all parts exist
	for _, p := range parts {
		if _, ok := st.parts[p.Number]; !ok {
			return nil, fmt.Errorf("bear: part %d not found", p.Number)
		}
	}

	// Assemble final value
	var totalSize int64
	for _, p := range parts {
		totalSize += int64(len(st.parts[p.Number].data))
	}
	assembled := make([]byte, 0, totalSize)
	for _, p := range parts {
		assembled = append(assembled, st.parts[p.Number].data...)
	}

	// Write to B-tree
	ck := compositeKey(b.name, st.key)
	now := time.Now()

	// Write large values to the value log before acquiring the main lock.
	entry, err := b.store.prepareEntry(ck, []byte(st.contentType), assembled, now.UnixNano(), now.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("bear: complete multipart prepare: %w", err)
	}

	b.store.ensureSpace(4)

	b.store.mu.Lock()
	existing := b.store.btreeGet(ck)
	if existing != nil {
		entry.created = existing.created
	} else {
		b.store.entryCount++
	}
	_, err = b.store.btreeInsert(entry)
	b.store.writeHeader()
	b.store.syncPages()
	b.store.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("bear: complete multipart: %w", err)
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         relToKey(st.key),
		Size:        totalSize,
		ContentType: st.contentType,
		Created:     time.Unix(0, entry.created),
		Updated:     now,
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.store.mpMu.Lock()
	_, ok := b.store.mpUploads[mu.UploadID]
	if !ok {
		b.store.mpMu.Unlock()
		return storage.ErrNotExist
	}
	delete(b.store.mpUploads, mu.UploadID)
	b.store.mpMu.Unlock()

	return nil
}

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

type bucketIter struct {
	list []*storage.BucketInfo
	pos  int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	b := it.list[it.pos]
	it.pos++
	return b, nil
}

func (it *bucketIter) Close() error { return nil }

type objectIter struct {
	list []*storage.Object
	pos  int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.pos >= len(it.list) {
		return nil, nil
	}
	o := it.list[it.pos]
	it.pos++
	return o, nil
}

func (it *objectIter) Close() error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func safeBucketName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	if name == "" {
		return "default"
	}
	if name == "." || name == ".." {
		return "_" + name
	}
	return name
}

func boolOpt(opts storage.Options, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("bear: empty key")
	}
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("bear: empty key")
	}
	key = path.Clean(key)
	if key == "." {
		return "", errors.New("bear: empty key")
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return key, nil
}

func cleanPrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", nil
	}
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "" {
		return "", nil
	}
	prefix = path.Clean(prefix)
	if prefix == "." {
		return "", nil
	}
	for _, part := range strings.Split(prefix, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return prefix, nil
}

func cleanDirPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	if p == "." || p == ".." {
		return ""
	}
	return path.Clean(p)
}

func relToKey(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "/")
	return rel
}

func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
