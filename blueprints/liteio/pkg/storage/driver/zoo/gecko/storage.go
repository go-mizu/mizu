// Package gecko implements a KDSep-inspired (Key-Delta Separation) storage driver.
//
// Architecture (based on KDSep, ICDE 2024):
//   - Base store: sorted key-value file (base.dat) with in-memory index
//   - Delta store: N hash-bucketed append-only delta files (delta_NNN.dat)
//   - Write buffer: in-memory map of bucketID -> pending deltas, flushed in batch
//   - Read: check write buffer -> delta file -> base file -> merge
//   - GC: periodically fold accumulated deltas back into base store
//
// DSN format:
//
//	gecko:///path/to/data
//	gecko:///path/to/data?sync=none&delta_buckets=64&gc_threshold=1000
package gecko

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func init() {
	storage.Register("gecko", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultDeltaBuckets = 64
	defaultGCThreshold  = 1000
	defaultBufferLimit  = 1 << 20 // 1 MB write buffer before flush

	dirPerm  = 0750
	filePerm = 0600

	maxPartNumber = 10000

	// Delta operation types.
	opPut    byte = 0
	opDelete byte = 1

	// Sentinel for "value lives in write buffer" vs. delta file vs. base file.
	srcBuffer int8 = 0
	srcDelta  int8 = 1
	srcBase   int8 = 2
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("gecko: parse dsn: %w", err)
	}
	if u.Scheme != "" && u.Scheme != "gecko" {
		return nil, fmt.Errorf("gecko: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		return nil, fmt.Errorf("gecko: missing path in dsn")
	}

	q := u.Query()

	syncMode := q.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	deltaBuckets := defaultDeltaBuckets
	if v := q.Get("delta_buckets"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			deltaBuckets = n
		}
	}

	gcThreshold := defaultGCThreshold
	if v := q.Get("gc_threshold"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			gcThreshold = n
		}
	}

	if err := os.MkdirAll(root, dirPerm); err != nil {
		return nil, fmt.Errorf("gecko: create root: %w", err)
	}

	st := &store{
		root:         root,
		syncMode:     syncMode,
		deltaBuckets: deltaBuckets,
		gcThreshold:  gcThreshold,
		bucketMap:    make(map[string]time.Time),
		index:        make(map[string]*indexEntry),
		writeBuf:     make(map[int][]deltaEntry),
	}

	// Open or create base store file.
	basePath := filepath.Join(root, "base.dat")
	f, err := os.OpenFile(basePath, os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, fmt.Errorf("gecko: open base: %w", err)
	}
	st.baseFile = f

	// Cleanup pattern: if any subsequent step fails, close all opened files.
	success := false
	defer func() {
		if !success {
			st.closeFiles()
		}
	}()

	// Open or create delta bucket files.
	st.deltaFiles = make([]*os.File, deltaBuckets)
	st.deltaMu = make([]sync.Mutex, deltaBuckets)
	for i := range deltaBuckets {
		name := fmt.Sprintf("delta_%03d.dat", i)
		df, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_RDWR|os.O_APPEND, filePerm)
		if err != nil {
			return nil, fmt.Errorf("gecko: open delta %d: %w", i, err)
		}
		st.deltaFiles[i] = df
	}

	// Recover in-memory index from base + delta files.
	if err := st.recover(); err != nil {
		return nil, fmt.Errorf("gecko: recovery: %w", err)
	}

	success = true
	return st, nil
}

// ---------------------------------------------------------------------------
// Index entry
// ---------------------------------------------------------------------------

// indexEntry tracks where the current value for a key lives.
type indexEntry struct {
	source      int8   // srcBuffer, srcDelta, srcBase
	offset      int64  // file offset (base or delta file)
	deltaFileID int    // which delta file (-1 for base)
	size        int64  // value size in bytes
	contentType string // MIME type
	created     int64  // UnixNano
	updated     int64  // UnixNano

	// For buffered (unflushed) values we keep data in memory.
	bufValue []byte
}

// ---------------------------------------------------------------------------
// Delta entry (in write buffer and on disk)
// ---------------------------------------------------------------------------

type deltaEntry struct {
	compositeKey string
	value        []byte
	contentType  string
	timestamp    int64
	op           byte
}

// ---------------------------------------------------------------------------
// Store (implements storage.Storage)
// ---------------------------------------------------------------------------

const maxBuckets = 10000

type store struct {
	root         string
	syncMode     string
	deltaBuckets int
	gcThreshold  int

	// Base store file.
	baseMu   sync.Mutex
	baseFile *os.File

	// Delta bucket files.
	deltaFiles []*os.File
	deltaMu    []sync.Mutex

	// Bucket registry.
	bktMu     sync.RWMutex
	bucketMap map[string]time.Time

	// In-memory index: compositeKey -> indexEntry.
	idxMu sync.RWMutex
	index map[string]*indexEntry

	// Write buffer: deltaBucketID -> pending delta entries.
	wbMu     sync.Mutex
	writeBuf map[int][]deltaEntry
	wbSize   int64 // approximate total buffered bytes

	// Delta entry counts per bucket file (for GC threshold).
	deltaCount []atomic.Int64

	// Multipart uploads.
	mpMu      sync.Mutex
	mpUploads map[string]*multipartUpload

	closed atomic.Bool
}

var _ storage.Storage = (*store)(nil)

// compositeKey builds the internal key from bucket name and object key.
func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

// splitCompositeKey splits a composite key back into bucket and object key.
func splitCompositeKey(ck string) (bucket, key string) {
	i := strings.IndexByte(ck, 0)
	if i < 0 {
		return "", ck
	}
	return ck[:i], ck[i+1:]
}

// hashBucket computes the delta bucket ID for a composite key.
func hashBucket(ck string, numBuckets int) int {
	h := fnv.New32a()
	h.Write([]byte(ck))
	return int(h.Sum32() % uint32(numBuckets))
}

// ---------------------------------------------------------------------------
// Storage interface
// ---------------------------------------------------------------------------

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	name = sanitizeBucketName(name)

	s.bktMu.Lock()
	if _, ok := s.bucketMap[name]; !ok {
		if len(s.bucketMap) < maxBuckets {
			s.bucketMap[name] = time.Now()
		}
	}
	s.bktMu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.bktMu.RLock()
	names := make([]string, 0, len(s.bucketMap))
	for n := range s.bucketMap {
		names = append(names, n)
	}
	s.bktMu.RUnlock()

	sort.Strings(names)

	s.bktMu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      n,
			CreatedAt: s.bucketMap[n],
		})
	}
	s.bktMu.RUnlock()

	if offset < 0 {
		offset = 0
	}
	if offset > len(infos) {
		offset = len(infos)
	}
	infos = infos[offset:]
	if limit > 0 && limit < len(infos) {
		infos = infos[:limit]
	}

	return &bucketIter{list: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("gecko: bucket name is empty")
	}
	name = sanitizeBucketName(name)

	s.bktMu.Lock()
	defer s.bktMu.Unlock()

	if _, ok := s.bucketMap[name]; ok {
		return nil, storage.ErrExist
	}

	if len(s.bucketMap) >= maxBuckets {
		return nil, fmt.Errorf("gecko: maximum number of buckets (%d) reached", maxBuckets)
	}

	now := time.Now()
	s.bucketMap[name] = now

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
		return fmt.Errorf("gecko: bucket name is empty")
	}
	name = sanitizeBucketName(name)

	force := boolOpt(opts, "force")

	s.bktMu.Lock()
	defer s.bktMu.Unlock()

	if _, ok := s.bucketMap[name]; !ok {
		return storage.ErrNotExist
	}

	if !force {
		// Check if bucket has any objects.
		s.idxMu.RLock()
		prefix := name + "\x00"
		hasObjects := false
		for ck := range s.index {
			if strings.HasPrefix(ck, prefix) {
				hasObjects = true
				break
			}
		}
		s.idxMu.RUnlock()
		if hasObjects {
			return storage.ErrPermission
		}
	}

	// Remove all index entries for this bucket.
	if force {
		s.idxMu.Lock()
		prefix := name + "\x00"
		for ck := range s.index {
			if strings.HasPrefix(ck, prefix) {
				delete(s.index, ck)
			}
		}
		s.idxMu.Unlock()
	}

	delete(s.bucketMap, name)
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_copy": true,
		"server_side_move": true,
		"directories":      true,
		"multipart":        true,
	}
}

func (s *store) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	// Flush write buffer.
	if err := s.flushWriteBuffer(); err != nil {
		return fmt.Errorf("gecko: flush on close: %w", err)
	}

	return s.closeFiles()
}

func (s *store) closeFiles() error {
	var firstErr error

	if s.baseFile != nil {
		if err := s.baseFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, df := range s.deltaFiles {
		if df != nil {
			if err := df.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// ---------------------------------------------------------------------------
// Recovery: rebuild in-memory index from base + delta files
// ---------------------------------------------------------------------------

func (s *store) recover() error {
	s.deltaCount = make([]atomic.Int64, s.deltaBuckets)

	// 1. Scan base file.
	if err := s.recoverBase(); err != nil {
		return err
	}

	// 2. Scan all delta files (deltas override base).
	for i := range s.deltaBuckets {
		if err := s.recoverDelta(i); err != nil {
			return err
		}
	}

	return nil
}

func (s *store) recoverBase() error {
	info, err := s.baseFile.Stat()
	if err != nil {
		return fmt.Errorf("stat base: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}

	data, err := io.ReadAll(s.baseFile)
	if err != nil {
		return fmt.Errorf("read base: %w", err)
	}

	pos := 0
	for pos < len(data) {
		entryStart := int64(pos)

		if pos+2 > len(data) {
			break
		}
		keyLen := int(binary.LittleEndian.Uint16(data[pos:]))
		pos += 2
		if pos+keyLen > len(data) {
			break
		}
		key := string(data[pos : pos+keyLen])
		pos += keyLen

		if pos+2 > len(data) {
			break
		}
		ctLen := int(binary.LittleEndian.Uint16(data[pos:]))
		pos += 2
		if pos+ctLen > len(data) {
			break
		}
		ct := string(data[pos : pos+ctLen])
		pos += ctLen

		if pos+8 > len(data) {
			break
		}
		valLen := int64(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8

		// valueOffset is where the value bytes start in the base file.
		valueOffset := int64(pos)
		if int64(pos)+valLen > int64(len(data)) {
			break
		}
		pos += int(valLen)

		if pos+16 > len(data) {
			break
		}
		created := int64(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8
		updated := int64(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8

		_ = entryStart

		bkt, objKey := splitCompositeKey(key)
		if bkt != "" {
			s.bktMu.Lock()
			if _, ok := s.bucketMap[bkt]; !ok {
				s.bucketMap[bkt] = time.Unix(0, created)
			}
			s.bktMu.Unlock()
		}
		_ = objKey

		s.index[key] = &indexEntry{
			source:      srcBase,
			offset:      valueOffset,
			deltaFileID: -1,
			size:        valLen,
			contentType: ct,
			created:     created,
			updated:     updated,
		}
	}

	return nil
}

func (s *store) recoverDelta(bucketID int) error {
	f := s.deltaFiles[bucketID]
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat delta %d: %w", bucketID, err)
	}
	if info.Size() == 0 {
		return nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek delta %d: %w", bucketID, err)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read delta %d: %w", bucketID, err)
	}

	var count int64
	pos := 0
	for pos < len(data) {
		if pos+2 > len(data) {
			break
		}
		keyLen := int(binary.LittleEndian.Uint16(data[pos:]))
		pos += 2
		if pos+keyLen > len(data) {
			break
		}
		key := string(data[pos : pos+keyLen])
		pos += keyLen

		if pos+8 > len(data) {
			break
		}
		valLen := int64(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8

		valueOffset := int64(pos)
		if int64(pos)+valLen > int64(len(data)) {
			break
		}
		pos += int(valLen)

		if pos+8 > len(data) {
			break
		}
		ts := int64(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8

		if pos+1 > len(data) {
			break
		}
		op := data[pos]
		pos++

		count++

		if op == opDelete {
			delete(s.index, key)
			continue
		}

		// Put operation: update or create index entry.
		existing := s.index[key]
		createdTs := ts
		if existing != nil {
			createdTs = existing.created
		}

		bkt, _ := splitCompositeKey(key)
		if bkt != "" {
			s.bktMu.Lock()
			if _, ok := s.bucketMap[bkt]; !ok {
				s.bucketMap[bkt] = time.Unix(0, ts)
			}
			s.bktMu.Unlock()
		}

		s.index[key] = &indexEntry{
			source:      srcDelta,
			offset:      valueOffset,
			deltaFileID: bucketID,
			size:        valLen,
			contentType: "", // Delta entries don't store content type separately; recovered below.
			created:     createdTs,
			updated:     ts,
		}
	}

	s.deltaCount[bucketID].Store(count)
	return nil
}

// ---------------------------------------------------------------------------
// Write buffer flush
// ---------------------------------------------------------------------------

func (s *store) flushWriteBuffer() error {
	s.wbMu.Lock()
	if len(s.writeBuf) == 0 {
		s.wbMu.Unlock()
		return nil
	}
	buf := s.writeBuf
	s.writeBuf = make(map[int][]deltaEntry)
	s.wbSize = 0
	s.wbMu.Unlock()

	for bucketID, entries := range buf {
		if err := s.flushDeltaBucket(bucketID, entries); err != nil {
			return err
		}
	}
	return nil
}

func (s *store) flushDeltaBucket(bucketID int, entries []deltaEntry) error {
	s.deltaMu[bucketID].Lock()
	defer s.deltaMu[bucketID].Unlock()

	f := s.deltaFiles[bucketID]

	// Get current end-of-file offset so we can compute value offsets.
	baseOffset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("gecko: seek delta %d: %w", bucketID, err)
	}

	var b bytes.Buffer
	// Track the offset of each entry's value within the buffer.
	type entryPos struct {
		compositeKey string
		valueOffset  int64
		valueSize    int64
	}
	positions := make([]entryPos, 0, len(entries))

	for _, de := range entries {
		// Record position: the value starts after keyLen(2) + key + valLen(8).
		entryStart := int64(b.Len())
		writeDeltaEntry(&b, de)

		if de.op == opPut {
			// Value offset within the file = baseOffset + entryStart + 2 + len(key) + 8.
			valueFileOffset := baseOffset + entryStart + 2 + int64(len(de.compositeKey)) + 8
			positions = append(positions, entryPos{
				compositeKey: de.compositeKey,
				valueOffset:  valueFileOffset,
				valueSize:    int64(len(de.value)),
			})
		}
	}

	if _, err := f.Write(b.Bytes()); err != nil {
		return fmt.Errorf("gecko: write delta %d: %w", bucketID, err)
	}

	if s.syncMode == "full" {
		if err := f.Sync(); err != nil {
			return fmt.Errorf("gecko: sync delta %d: %w", bucketID, err)
		}
	}

	s.deltaCount[bucketID].Add(int64(len(entries)))

	// Update index entries: values now live in delta file, not buffer.
	// Clear bufValue to release memory; point to delta file instead.
	s.idxMu.Lock()
	for _, pos := range positions {
		if entry, ok := s.index[pos.compositeKey]; ok && entry.source == srcBuffer {
			entry.source = srcDelta
			entry.deltaFileID = bucketID
			entry.offset = pos.valueOffset
			entry.size = pos.valueSize
			entry.bufValue = nil
		}
	}
	s.idxMu.Unlock()

	return nil
}

// writeDeltaEntry serializes one delta entry into w.
// Format: keyLen(2B) | key | valLen(8B) | value | ts(8B) | op(1B)
func writeDeltaEntry(w *bytes.Buffer, de deltaEntry) {
	var hdr [2]byte
	binary.LittleEndian.PutUint16(hdr[:], uint16(len(de.compositeKey)))
	w.Write(hdr[:])
	w.WriteString(de.compositeKey)

	var vl [8]byte
	binary.LittleEndian.PutUint64(vl[:], uint64(len(de.value)))
	w.Write(vl[:])
	w.Write(de.value)

	var ts [8]byte
	binary.LittleEndian.PutUint64(ts[:], uint64(de.timestamp))
	w.Write(ts[:])

	w.WriteByte(de.op)
}

// bufferDelta adds a delta entry to the write buffer and flushes if threshold is exceeded.
func (s *store) bufferDelta(de deltaEntry) {
	ck := de.compositeKey
	bucketID := hashBucket(ck, s.deltaBuckets)

	s.wbMu.Lock()
	s.writeBuf[bucketID] = append(s.writeBuf[bucketID], de)
	s.wbSize += int64(len(de.compositeKey) + len(de.value) + 19) // approx overhead
	shouldFlush := s.wbSize >= defaultBufferLimit
	s.wbMu.Unlock()

	if shouldFlush {
		// Flush asynchronously would be better, but for correctness we flush inline.
		_ = s.flushWriteBuffer()
	}
}

// ---------------------------------------------------------------------------
// Base file write helpers
// ---------------------------------------------------------------------------

// appendToBase writes a full key-value entry to the base file and returns the
// file offset where the value bytes start.
func (s *store) appendToBase(ck, contentType string, value []byte, created, updated int64) (valueOffset int64, err error) {
	s.baseMu.Lock()
	defer s.baseMu.Unlock()

	// Seek to end.
	off, err := s.baseFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("gecko: seek base: %w", err)
	}

	var buf bytes.Buffer

	// keyLen(2B) | key
	var kl [2]byte
	binary.LittleEndian.PutUint16(kl[:], uint16(len(ck)))
	buf.Write(kl[:])
	buf.WriteString(ck)

	// ctLen(2B) | contentType
	var cl [2]byte
	binary.LittleEndian.PutUint16(cl[:], uint16(len(contentType)))
	buf.Write(cl[:])
	buf.WriteString(contentType)

	// valLen(8B) | value
	var vl [8]byte
	binary.LittleEndian.PutUint64(vl[:], uint64(len(value)))
	buf.Write(vl[:])

	// valueOffset = file position of value start
	headerSize := 2 + len(ck) + 2 + len(contentType) + 8
	valueOffset = off + int64(headerSize)

	buf.Write(value)

	// created(8B) | updated(8B)
	var ts [16]byte
	binary.LittleEndian.PutUint64(ts[:8], uint64(created))
	binary.LittleEndian.PutUint64(ts[8:], uint64(updated))
	buf.Write(ts[:])

	if _, err := s.baseFile.Write(buf.Bytes()); err != nil {
		return 0, fmt.Errorf("gecko: write base: %w", err)
	}

	if s.syncMode == "full" {
		if err := s.baseFile.Sync(); err != nil {
			return 0, fmt.Errorf("gecko: sync base: %w", err)
		}
	}

	return valueOffset, nil
}

// readValueFromBase reads size bytes from the base file at the given offset.
func (s *store) readValueFromBase(offset, size int64) ([]byte, error) {
	data := make([]byte, size)
	n, err := s.baseFile.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("gecko: read base at %d: %w", offset, err)
	}
	return data[:n], nil
}

// readValueFromDelta reads size bytes from a delta file at the given offset.
func (s *store) readValueFromDelta(fileID int, offset, size int64) ([]byte, error) {
	if fileID < 0 || fileID >= len(s.deltaFiles) {
		return nil, fmt.Errorf("gecko: invalid delta file %d", fileID)
	}
	data := make([]byte, size)
	n, err := s.deltaFiles[fileID].ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("gecko: read delta %d at %d: %w", fileID, offset, err)
	}
	return data[:n], nil
}

// readValue reads the value for an index entry.
func (s *store) readValue(e *indexEntry) ([]byte, error) {
	if e.bufValue != nil {
		return e.bufValue, nil
	}
	switch e.source {
	case srcBase:
		return s.readValueFromBase(e.offset, e.size)
	case srcDelta:
		return s.readValueFromDelta(e.deltaFileID, e.offset, e.size)
	case srcBuffer:
		// Should have bufValue set; fall through to error.
	}
	return nil, fmt.Errorf("gecko: no value for entry (source=%d)", e.source)
}

// ---------------------------------------------------------------------------
// Bucket (implements storage.Bucket, storage.HasDirectories, storage.HasMultipart)
// ---------------------------------------------------------------------------

type bucket struct {
	st   *store
	name string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
	_ storage.HasMultipart   = (*bucket)(nil)
)

func (b *bucket) Name() string { return b.name }

func (b *bucket) Features() storage.Features {
	return b.st.Features()
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.bktMu.RLock()
	created, ok := b.st.bucketMap[b.name]
	b.st.bktMu.RUnlock()

	if !ok {
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

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Read value from source.
	var value []byte
	if size == 0 {
		value = nil
	} else if size > 0 {
		value = make([]byte, size)
		n, readErr := io.ReadFull(src, value)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("gecko: read value: %w", readErr)
		}
		value = value[:n]
	} else {
		// Unknown size: buffer all.
		var tmpBuf bytes.Buffer
		if _, err := io.Copy(&tmpBuf, src); err != nil {
			return nil, fmt.Errorf("gecko: read value: %w", err)
		}
		value = tmpBuf.Bytes()
	}

	now := time.Now().UnixNano()
	ck := compositeKey(b.name, key)

	// Check for existing entry to preserve created timestamp.
	b.st.idxMu.RLock()
	existing := b.st.index[ck]
	b.st.idxMu.RUnlock()

	createdTs := now
	if existing != nil {
		createdTs = existing.created
	}

	// Buffer the delta.
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	de := deltaEntry{
		compositeKey: ck,
		value:        valueCopy,
		contentType:  contentType,
		timestamp:    now,
		op:           opPut,
	}
	b.st.bufferDelta(de)

	// Update in-memory index.
	entry := &indexEntry{
		source:      srcBuffer,
		offset:      0,
		deltaFileID: -1,
		size:        int64(len(value)),
		contentType: contentType,
		created:     createdTs,
		updated:     now,
		bufValue:    valueCopy,
	}

	b.st.idxMu.Lock()
	b.st.index[ck] = entry
	b.st.idxMu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(value)),
		ContentType: contentType,
		Created:     time.Unix(0, createdTs),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, nil, err
	}

	ck := compositeKey(b.name, key)

	b.st.idxMu.RLock()
	entry := b.st.index[ck]
	b.st.idxMu.RUnlock()

	if entry == nil {
		return nil, nil, storage.ErrNotExist
	}

	data, err := b.st.readValue(entry)
	if err != nil {
		return nil, nil, err
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        entry.size,
		ContentType: entry.contentType,
		Created:     time.Unix(0, entry.created),
		Updated:     time.Unix(0, entry.updated),
	}

	// Apply range.
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	end := int64(len(data))
	if length > 0 && offset+length < end {
		end = offset + length
	}

	slice := data[offset:end]

	// Copy to avoid mutation issues.
	result := make([]byte, len(slice))
	copy(result, slice)

	return &memReader{data: result}, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Check for directory stat (key ending with "/").
	if strings.HasSuffix(key, "/") {
		prefix := compositeKey(b.name, key)
		b.st.idxMu.RLock()
		found := false
		for ck := range b.st.index {
			if strings.HasPrefix(ck, prefix) {
				found = true
				break
			}
		}
		b.st.idxMu.RUnlock()

		if !found {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket: b.name,
			Key:    strings.TrimSuffix(key, "/"),
			IsDir:  true,
		}, nil
	}

	ck := compositeKey(b.name, key)

	b.st.idxMu.RLock()
	entry := b.st.index[ck]
	b.st.idxMu.RUnlock()

	if entry == nil {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        entry.size,
		ContentType: entry.contentType,
		Created:     time.Unix(0, entry.created),
		Updated:     time.Unix(0, entry.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key, err := cleanKey(key)
	if err != nil {
		return err
	}

	recursive := boolOpt(opts, "recursive")

	if recursive {
		prefix := compositeKey(b.name, key)
		b.st.idxMu.Lock()
		found := false
		for ck := range b.st.index {
			if strings.HasPrefix(ck, prefix) {
				delete(b.st.index, ck)
				found = true

				// Buffer a delete delta for each key.
				b.st.bufferDelta(deltaEntry{
					compositeKey: ck,
					timestamp:    time.Now().UnixNano(),
					op:           opDelete,
				})
			}
		}
		b.st.idxMu.Unlock()

		if !found {
			return storage.ErrNotExist
		}
		return nil
	}

	ck := compositeKey(b.name, key)

	b.st.idxMu.Lock()
	_, ok := b.st.index[ck]
	if !ok {
		b.st.idxMu.Unlock()
		return storage.ErrNotExist
	}
	delete(b.st.index, ck)
	b.st.idxMu.Unlock()

	// Buffer a delete delta.
	b.st.bufferDelta(deltaEntry{
		compositeKey: ck,
		timestamp:    time.Now().UnixNano(),
		op:           opDelete,
	})

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dstKey, err := cleanKey(dstKey)
	if err != nil {
		return nil, err
	}
	srcKey, err = cleanKey(srcKey)
	if err != nil {
		return nil, err
	}

	if srcBucket == "" {
		srcBucket = b.name
	}
	srcBucket = sanitizeBucketName(srcBucket)

	srcCK := compositeKey(srcBucket, srcKey)

	b.st.idxMu.RLock()
	srcEntry := b.st.index[srcCK]
	b.st.idxMu.RUnlock()

	if srcEntry == nil {
		return nil, storage.ErrNotExist
	}

	// Read source value.
	srcData, err := b.st.readValue(srcEntry)
	if err != nil {
		return nil, fmt.Errorf("gecko: copy read src: %w", err)
	}

	// Write as new entry in destination.
	dataCopy := make([]byte, len(srcData))
	copy(dataCopy, srcData)

	now := time.Now().UnixNano()
	dstCK := compositeKey(b.name, dstKey)

	de := deltaEntry{
		compositeKey: dstCK,
		value:        dataCopy,
		contentType:  srcEntry.contentType,
		timestamp:    now,
		op:           opPut,
	}
	b.st.bufferDelta(de)

	entry := &indexEntry{
		source:      srcBuffer,
		offset:      0,
		deltaFileID: -1,
		size:        int64(len(dataCopy)),
		contentType: srcEntry.contentType,
		created:     now,
		updated:     now,
		bufValue:    dataCopy,
	}

	b.st.idxMu.Lock()
	b.st.index[dstCK] = entry
	b.st.idxMu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        int64(len(dataCopy)),
		ContentType: srcEntry.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}

	if srcBucket == "" {
		srcBucket = b.name
	}
	sb := b.st.Bucket(srcBucket)
	if err := sb.Delete(ctx, srcKey, nil); err != nil && err != storage.ErrNotExist {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = strings.TrimSpace(prefix)

	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	ckPrefix := compositeKey(b.name, prefix)

	b.st.idxMu.RLock()
	var objs []*storage.Object
	seen := make(map[string]bool) // for non-recursive directory dedup
	for ck, entry := range b.st.index {
		if !strings.HasPrefix(ck, ckPrefix) {
			continue
		}
		_, objKey := splitCompositeKey(ck)

		if !recursive {
			rest := strings.TrimPrefix(objKey, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if idx := strings.IndexByte(rest, '/'); idx >= 0 {
				// This is inside a subdirectory; emit the directory entry instead.
				dirName := prefix
				if dirName != "" && !strings.HasSuffix(dirName, "/") {
					dirName += "/"
				}
				dirName += rest[:idx]
				if seen[dirName] {
					continue
				}
				seen[dirName] = true
				objs = append(objs, &storage.Object{
					Bucket: b.name,
					Key:    dirName,
					IsDir:  true,
				})
				continue
			}
		}

		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        entry.size,
			ContentType: entry.contentType,
			Created:     time.Unix(0, entry.created),
			Updated:     time.Unix(0, entry.updated),
		})
	}
	b.st.idxMu.RUnlock()

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]
	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{list: objs}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// Directory support (implements storage.HasDirectories)
// ---------------------------------------------------------------------------

func (b *bucket) Directory(p string) storage.Directory {
	return &dir{b: b, path: strings.Trim(p, "/")}
}

type dir struct {
	b    *bucket
	path string
}

var _ storage.Directory = (*dir)(nil)

func (d *dir) Bucket() storage.Bucket { return d.b }
func (d *dir) Path() string           { return d.path }

func (d *dir) Info(ctx context.Context) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ckPrefix := compositeKey(d.b.name, prefix)

	d.b.st.idxMu.RLock()
	found := false
	var earliest int64
	for ck, entry := range d.b.st.index {
		if strings.HasPrefix(ck, ckPrefix) {
			found = true
			if earliest == 0 || entry.created < earliest {
				earliest = entry.created
			}
			break
		}
	}
	d.b.st.idxMu.RUnlock()

	if !found {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: time.Unix(0, earliest),
		Updated: time.Unix(0, earliest),
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ckPrefix := compositeKey(d.b.name, prefix)

	d.b.st.idxMu.RLock()
	var objs []*storage.Object
	seen := make(map[string]bool)
	for ck, entry := range d.b.st.index {
		if !strings.HasPrefix(ck, ckPrefix) {
			continue
		}
		_, objKey := splitCompositeKey(ck)
		rest := strings.TrimPrefix(objKey, prefix)

		if idx := strings.IndexByte(rest, '/'); idx >= 0 {
			// Subdirectory.
			dirKey := prefix + rest[:idx]
			if seen[dirKey] {
				continue
			}
			seen[dirKey] = true
			objs = append(objs, &storage.Object{
				Bucket: d.b.name,
				Key:    dirKey,
				IsDir:  true,
			})
			continue
		}

		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         objKey,
			Size:        entry.size,
			ContentType: entry.contentType,
			Created:     time.Unix(0, entry.created),
			Updated:     time.Unix(0, entry.updated),
		})
	}
	d.b.st.idxMu.RUnlock()

	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })

	if offset < 0 {
		offset = 0
	}
	if offset > len(objs) {
		offset = len(objs)
	}
	objs = objs[offset:]
	if limit > 0 && limit < len(objs) {
		objs = objs[:limit]
	}

	return &objectIter{list: objs}, nil
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	recursive := boolOpt(opts, "recursive")

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ckPrefix := compositeKey(d.b.name, prefix)

	d.b.st.idxMu.Lock()
	found := false
	for ck := range d.b.st.index {
		if !strings.HasPrefix(ck, ckPrefix) {
			continue
		}
		_, objKey := splitCompositeKey(ck)
		if !recursive {
			rest := strings.TrimPrefix(objKey, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		delete(d.b.st.index, ck)
		found = true

		d.b.st.bufferDelta(deltaEntry{
			compositeKey: ck,
			timestamp:    time.Now().UnixNano(),
			op:           opDelete,
		})
	}
	d.b.st.idxMu.Unlock()

	if !found {
		return storage.ErrNotExist
	}
	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	srcCKPrefix := compositeKey(d.b.name, srcPrefix)

	d.b.st.idxMu.Lock()
	defer d.b.st.idxMu.Unlock()

	type moveItem struct {
		oldCK string
		newCK string
		entry *indexEntry
	}
	var items []moveItem

	for ck, entry := range d.b.st.index {
		if !strings.HasPrefix(ck, srcCKPrefix) {
			continue
		}
		_, objKey := splitCompositeKey(ck)
		rel := strings.TrimPrefix(objKey, srcPrefix)
		newKey := dstPrefix + rel
		newCK := compositeKey(d.b.name, newKey)
		items = append(items, moveItem{oldCK: ck, newCK: newCK, entry: entry})
	}

	if len(items) == 0 {
		return nil, storage.ErrNotExist
	}

	now := time.Now().UnixNano()
	for _, item := range items {
		// Read the value so we can re-buffer it under the new key.
		value, err := d.b.st.readValue(item.entry)
		if err != nil {
			continue
		}
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		d.b.st.index[item.newCK] = &indexEntry{
			source:      srcBuffer,
			size:        item.entry.size,
			contentType: item.entry.contentType,
			created:     item.entry.created,
			updated:     now,
			bufValue:    valueCopy,
		}
		delete(d.b.st.index, item.oldCK)

		// Buffer delta for new key (put) and old key (delete).
		d.b.st.bufferDelta(deltaEntry{
			compositeKey: item.newCK,
			value:        valueCopy,
			contentType:  item.entry.contentType,
			timestamp:    now,
			op:           opPut,
		})
		d.b.st.bufferDelta(deltaEntry{
			compositeKey: item.oldCK,
			timestamp:    now,
			op:           opDelete,
		})
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart support (implements storage.HasMultipart)
// ---------------------------------------------------------------------------

var mpIDCounter atomic.Int64

func init() {
	mpIDCounter.Store(time.Now().UnixNano())
}

type multipartUpload struct {
	id          string
	key         string
	contentType string
	parts       map[int]*partData
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

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	id := strconv.FormatInt(mpIDCounter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &multipartUpload{
		id:          id,
		key:         key,
		contentType: contentType,
		parts:       make(map[int]*partData),
		metadata:    metadata,
	}

	b.st.mpMu.Lock()
	if b.st.mpUploads == nil {
		b.st.mpUploads = make(map[string]*multipartUpload)
	}
	b.st.mpUploads[id] = upload
	b.st.mpMu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("gecko: part number %d out of range [1, %d]", number, maxPartNumber)
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	b.st.mpMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Read part data.
	var data []byte
	if size > 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("gecko: read part: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("gecko: read part: %w", err)
		}
		data = buf.Bytes()
	}

	// Compute ETag (MD5).
	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	b.st.mpMu.Lock()
	upload.parts[number] = &partData{
		number: number,
		data:   data,
		etag:   etag,
	}
	b.st.mpMu.Unlock()

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
		return nil, fmt.Errorf("gecko: part number %d out of range", number)
	}

	b.st.mpMu.Lock()
	_, ok := b.st.mpUploads[mu.UploadID]
	b.st.mpMu.Unlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, fmt.Errorf("gecko: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	// Open source object.
	srcBkt := b.st.Bucket(srcBucket)
	rc, _, err := srcBkt.Open(ctx, srcKey, srcOffset, srcLength, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return b.UploadPart(ctx, mu, number, rc, srcLength, opts)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   int64(len(p.data)),
			ETag:   p.etag,
		})
	}
	b.st.mpMu.Unlock()

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

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

	b.st.mpMu.Lock()
	upload, ok := b.st.mpUploads[mu.UploadID]
	if !ok {
		b.st.mpMu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mpUploads, mu.UploadID)
	b.st.mpMu.Unlock()

	// Sort parts by number.
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	// Verify all parts exist.
	for _, p := range parts {
		if _, exists := upload.parts[p.Number]; !exists {
			return nil, fmt.Errorf("gecko: part %d not found", p.Number)
		}
	}

	// Assemble final value.
	var assembled bytes.Buffer
	for _, p := range parts {
		pd := upload.parts[p.Number]
		assembled.Write(pd.data)
	}

	finalData := assembled.Bytes()
	now := time.Now().UnixNano()
	ck := compositeKey(b.name, upload.key)

	valueCopy := make([]byte, len(finalData))
	copy(valueCopy, finalData)

	de := deltaEntry{
		compositeKey: ck,
		value:        valueCopy,
		contentType:  upload.contentType,
		timestamp:    now,
		op:           opPut,
	}
	b.st.bufferDelta(de)

	entry := &indexEntry{
		source:      srcBuffer,
		size:        int64(len(valueCopy)),
		contentType: upload.contentType,
		created:     now,
		updated:     now,
		bufValue:    valueCopy,
	}

	b.st.idxMu.Lock()
	b.st.index[ck] = entry
	b.st.idxMu.Unlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         upload.key,
		Size:        int64(len(valueCopy)),
		ContentType: upload.contentType,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.st.mpMu.Lock()
	defer b.st.mpMu.Unlock()

	if _, ok := b.st.mpUploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}
	delete(b.st.mpUploads, mu.UploadID)
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

func (it *bucketIter) Close() error {
	it.list = nil
	return nil
}

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

func (it *objectIter) Close() error {
	it.list = nil
	return nil
}

// ---------------------------------------------------------------------------
// memReader: in-memory io.ReadCloser
// ---------------------------------------------------------------------------

type memReader struct {
	data []byte
	pos  int
}

func (r *memReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *memReader) Close() error {
	r.data = nil
	return nil
}

func (r *memReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("gecko: empty key")
	}
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", fmt.Errorf("gecko: empty key")
	}
	// Normalize path.
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(key)))
	if cleaned == "." {
		return "", fmt.Errorf("gecko: empty key")
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return cleaned, nil
}

func sanitizeBucketName(name string) string {
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
