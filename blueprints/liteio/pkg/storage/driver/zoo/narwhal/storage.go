// Package narwhal implements a stripeless erasure-coded storage driver inspired
// by the Nos/Nostor paper (OSDI 2025). It simulates N data volumes with XOR
// parity within a single process, using SBIBD-inspired deterministic key-to-volume
// assignment and local parity computation with no cross-volume coordination on writes.
//
// DSN format:
//
//	narwhal:///path/to/data
//	narwhal:///path/to/data?sync=none&stripes=4
package narwhal

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	storage.Register("narwhal", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultStripes = 4
	parityBlock    = 64 * 1024 // 64KB parity block size

	volumeMagic   = "NARWH001"
	volumeVersion = uint32(1)
	headerSize    = 32

	recPut    byte = 1
	recDelete byte = 2

	dirPerm  = 0o750
	filePerm = 0o600

	maxPartNumber = 10000

	maxBuckets          = 10000
	maxParityFileSize   = 10 * 1024 * 1024 * 1024 // 10GB
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, q, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	syncMode := q.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}

	stripes := defaultStripes
	if s := q.Get("stripes"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			stripes = n
		}
	}

	if err := os.MkdirAll(root, dirPerm); err != nil {
		return nil, fmt.Errorf("narwhal: mkdir %q: %w", root, err)
	}

	st := &store{
		root:     root,
		numVols:  stripes,
		syncMode: syncMode,
		buckets:  make(map[string]time.Time),
		mp:       &mpRegistry{uploads: make(map[string]*mpUpload)},
		stopTick: startTimeTicker(),
	}

	// Open data volumes.
	st.vols = make([]*dataVolume, stripes)
	for i := 0; i < stripes; i++ {
		p := filepath.Join(root, fmt.Sprintf("vol_%d.dat", i))
		v, err := openDataVolume(p)
		if err != nil {
			st.closeVolumes()
			return nil, err
		}
		st.vols[i] = v
	}

	// Open parity volume.
	pp := filepath.Join(root, "parity.dat")
	pv, err := openParityVolume(pp)
	if err != nil {
		st.closeVolumes()
		return nil, err
	}
	st.parity = pv

	// Build index.
	st.idx = newIndex()
	if err := st.recover(); err != nil {
		st.Close()
		return nil, fmt.Errorf("narwhal: recovery: %w", err)
	}

	// Load metadata.
	st.loadMeta()

	return st, nil
}

func parseDSN(dsn string) (string, url.Values, error) {
	if dsn == "" {
		return "", nil, errors.New("narwhal: empty dsn")
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("narwhal: parse dsn: %w", err)
	}

	if u.Scheme != "" && u.Scheme != "narwhal" {
		return "", nil, fmt.Errorf("narwhal: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		return "", nil, errors.New("narwhal: missing path in dsn")
	}

	return root, u.Query(), nil
}

// ---------------------------------------------------------------------------
// Store (storage.Storage)
// ---------------------------------------------------------------------------

type store struct {
	root     string
	numVols  int
	syncMode string

	vols   []*dataVolume
	parity *parityVolume
	idx    *shardedIndex

	mu      sync.RWMutex
	buckets map[string]time.Time

	mp *mpRegistry

	stopTick chan struct{} // stops the cached-time ticker goroutine
}

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	name = safeName(name)

	s.mu.Lock()
	if _, ok := s.buckets[name]; !ok {
		if len(s.buckets) < maxBuckets {
			s.buckets[name] = fastNowTime()
		}
	}
	s.mu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	names := make([]string, 0, len(s.buckets))
	for n := range s.buckets {
		names = append(names, n)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	s.mu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, n := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      n,
			CreatedAt: s.buckets[n],
		})
	}
	s.mu.RUnlock()

	infos = paginate(infos, limit, offset)
	return &bucketIter{list: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("narwhal: bucket name required")
	}
	name = safeName(name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	if len(s.buckets) >= maxBuckets {
		return nil, fmt.Errorf("narwhal: too many buckets (max %d)", maxBuckets)
	}

	now := fastNowTime()
	s.buckets[name] = now
	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("narwhal: bucket name required")
	}
	name = safeName(name)

	force := boolOpt(opts, "force")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; !ok {
		return storage.ErrNotExist
	}

	if !force && s.idx.hasBucket(name) {
		return storage.ErrPermission
	}

	delete(s.buckets, name)
	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":             true,
		"server_side_move": true,
		"server_side_copy": true,
		"directories":      true,
		"multipart":        true,
	}
}

func (s *store) Close() error {
	if s.stopTick != nil {
		close(s.stopTick)
		s.stopTick = nil
	}

	s.saveMeta()

	if s.syncMode != "none" {
		for _, v := range s.vols {
			if v != nil {
				v.sync()
			}
		}
	}

	s.closeVolumes()

	if s.parity != nil {
		s.parity.close()
	}
	return nil
}

func (s *store) closeVolumes() {
	for i, v := range s.vols {
		if v != nil {
			v.close()
			s.vols[i] = nil
		}
	}
}

// volumeFor returns the volume index for a composite key.
func (s *store) volumeFor(ck string) int {
	return int(fnv1a(ck) % uint32(s.numVols))
}

// recover replays all volume files to rebuild the in-memory index.
func (s *store) recover() error {
	for volID, v := range s.vols {
		if err := v.replay(s.idx, volID); err != nil {
			return fmt.Errorf("vol_%d: %w", volID, err)
		}
	}
	return nil
}

// meta.json persistence.

type metaFile struct {
	Buckets map[string]time.Time `json:"buckets"`
}

func (s *store) saveMeta() {
	s.mu.RLock()
	m := metaFile{Buckets: make(map[string]time.Time, len(s.buckets))}
	for k, v := range s.buckets {
		m.Buckets[k] = v
	}
	s.mu.RUnlock()

	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(s.root, "meta.json"), data, filePerm)
}

func (s *store) loadMeta() {
	data, err := os.ReadFile(filepath.Join(s.root, "meta.json"))
	if err != nil {
		return
	}
	var m metaFile
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	s.mu.Lock()
	for k, v := range m.Buckets {
		if _, exists := s.buckets[k]; !exists {
			s.buckets[k] = v
		}
	}
	s.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Data Volume
// ---------------------------------------------------------------------------

type dataVolume struct {
	fd   *os.File
	path string
	tail atomic.Int64
	mu   sync.Mutex // protects concurrent appends
}

func openDataVolume(path string) (*dataVolume, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("narwhal: mkdir %q: %w", dir, err)
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, fmt.Errorf("narwhal: open %q: %w", path, err)
	}

	info, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("narwhal: stat %q: %w", path, err)
	}

	v := &dataVolume{fd: fd, path: path}

	if info.Size() == 0 {
		// Write header.
		hdr := make([]byte, headerSize)
		copy(hdr[0:8], volumeMagic)
		binary.LittleEndian.PutUint32(hdr[8:12], volumeVersion)
		// flags at 12:16 = 0
		binary.LittleEndian.PutUint64(hdr[16:24], uint64(headerSize))
		// reserved at 24:32 = 0
		if _, err := fd.Write(hdr); err != nil {
			fd.Close()
			return nil, fmt.Errorf("narwhal: write header: %w", err)
		}
		v.tail.Store(headerSize)
	} else {
		// Read header.
		hdr := make([]byte, headerSize)
		if _, err := fd.ReadAt(hdr, 0); err != nil {
			fd.Close()
			return nil, fmt.Errorf("narwhal: read header: %w", err)
		}
		if string(hdr[0:8]) != volumeMagic {
			fd.Close()
			return nil, fmt.Errorf("narwhal: invalid magic in %q", path)
		}
		ver := binary.LittleEndian.Uint32(hdr[8:12])
		if ver != volumeVersion {
			fd.Close()
			return nil, fmt.Errorf("narwhal: unsupported version %d in %q", ver, path)
		}
		tail := binary.LittleEndian.Uint64(hdr[16:24])
		if tail < headerSize {
			tail = headerSize
		}
		v.tail.Store(int64(tail))
	}

	return v, nil
}

// appendRecord appends a record and returns (recordOffset, error).
// Record format: recType(1) | keyLen(2) | key | ctLen(2) | contentType | valLen(8) | value | created(8) | updated(8)
func (v *dataVolume) appendRecord(recType byte, compositeKey, contentType string, value []byte, created, updated int64) (int64, error) {
	kl := len(compositeKey)
	cl := len(contentType)
	vl := len(value)
	totalSize := 1 + 2 + kl + 2 + cl + 8 + vl + 8 + 8

	buf := make([]byte, totalSize)
	pos := 0

	buf[pos] = recType
	pos++

	binary.LittleEndian.PutUint16(buf[pos:], uint16(kl))
	pos += 2
	copy(buf[pos:], compositeKey)
	pos += kl

	binary.LittleEndian.PutUint16(buf[pos:], uint16(cl))
	pos += 2
	copy(buf[pos:], contentType)
	pos += cl

	binary.LittleEndian.PutUint64(buf[pos:], uint64(vl))
	pos += 8
	copy(buf[pos:], value)
	pos += vl

	binary.LittleEndian.PutUint64(buf[pos:], uint64(created))
	pos += 8
	binary.LittleEndian.PutUint64(buf[pos:], uint64(updated))

	v.mu.Lock()
	offset := v.tail.Load()
	_, err := v.fd.WriteAt(buf, offset)
	if err != nil {
		v.mu.Unlock()
		return 0, fmt.Errorf("narwhal: write record: %w", err)
	}
	v.tail.Add(int64(totalSize))
	v.mu.Unlock()

	return offset, nil
}

// readValue reads the value bytes for a record at the given entry location.
func (v *dataVolume) readValue(e *indexEntry) ([]byte, error) {
	buf := make([]byte, e.size)
	_, err := v.fd.ReadAt(buf, e.valueOffset)
	if err != nil {
		return nil, fmt.Errorf("narwhal: read value: %w", err)
	}
	return buf, nil
}

// replay scans all records in the volume and populates the index.
func (v *dataVolume) replay(idx *shardedIndex, volID int) error {
	tail := v.tail.Load()
	if tail <= headerSize {
		return nil
	}

	pos := int64(headerSize)
	readBuf := make([]byte, 4096)

	for pos < tail {
		// Read record type + key length header (at least 3 bytes).
		remaining := tail - pos
		if remaining < 3 {
			break
		}

		// Read a chunk from the file.
		chunkSize := remaining
		if chunkSize > int64(len(readBuf)) {
			chunkSize = int64(len(readBuf))
		}
		n, err := v.fd.ReadAt(readBuf[:chunkSize], pos)
		if err != nil && err != io.EOF {
			break
		}
		if n < 3 {
			break
		}
		chunk := readBuf[:n]

		recType := chunk[0]
		if recType != recPut && recType != recDelete {
			break
		}

		off := 1
		kl := int(binary.LittleEndian.Uint16(chunk[off:]))
		off += 2

		// We need: off + kl + 2(ctLen) + ctLen + 8(valLen) + valLen + 16(timestamps)
		// Calculate the minimum header size we need to read.
		minHeader := off + kl + 2
		if int64(minHeader) > remaining {
			break
		}

		// If our chunk is too small, read a bigger buffer.
		if minHeader > n {
			bigBuf := make([]byte, remaining)
			nn, err := v.fd.ReadAt(bigBuf, pos)
			if err != nil && err != io.EOF {
				break
			}
			chunk = bigBuf[:nn]
			n = nn
		}

		compositeKey := string(chunk[off : off+kl])
		off += kl

		cl := int(binary.LittleEndian.Uint16(chunk[off:]))
		off += 2

		needed := off + cl + 8 // contentType + valLen
		if needed > n {
			bigBuf := make([]byte, remaining)
			nn, _ := v.fd.ReadAt(bigBuf, pos)
			chunk = bigBuf[:nn]
			n = nn
		}

		contentType := string(chunk[off : off+cl])
		off += cl

		vl := int64(binary.LittleEndian.Uint64(chunk[off:]))
		off += 8

		// Value offset in the file.
		valueOffset := pos + int64(off)

		// Skip past value + timestamps.
		totalRecSize := int64(off) + vl + 16
		if pos+totalRecSize > tail {
			break
		}

		// Read timestamps.
		tsBuf := make([]byte, 16)
		if _, err := v.fd.ReadAt(tsBuf, pos+int64(off)+vl); err != nil {
			break
		}
		created := int64(binary.LittleEndian.Uint64(tsBuf[0:8]))
		updated := int64(binary.LittleEndian.Uint64(tsBuf[8:16]))

		switch recType {
		case recPut:
			idx.put(compositeKey, volID, &indexEntry{
				valueOffset: valueOffset,
				size:        vl,
				contentType: contentType,
				created:     created,
				updated:     updated,
			})
		case recDelete:
			idx.remove(compositeKey)
		}

		pos += totalRecSize
		_ = contentType // used above
	}

	// Update tail to the last valid position we scanned.
	v.tail.Store(pos)
	return nil
}

func (v *dataVolume) flushHeader() {
	hdr := make([]byte, 8)
	binary.LittleEndian.PutUint64(hdr, uint64(v.tail.Load()))
	v.fd.WriteAt(hdr, 16)
}

func (v *dataVolume) sync() {
	v.flushHeader()
	v.fd.Sync()
}

func (v *dataVolume) close() {
	if v.fd != nil {
		v.flushHeader()
		v.fd.Close()
		v.fd = nil
	}
}

// ---------------------------------------------------------------------------
// Parity Volume
// ---------------------------------------------------------------------------

type parityVolume struct {
	fd   *os.File
	path string
	mu   sync.Mutex
}

func openParityVolume(path string) (*parityVolume, error) {
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, fmt.Errorf("narwhal: open parity: %w", err)
	}
	return &parityVolume{fd: fd, path: path}, nil
}

// xorInto XORs data into the parity volume at the given file-level offset
// of the source data volume. The parity file is conceptually divided into
// parityBlock-sized blocks; each block stores the XOR of corresponding
// blocks across all data volumes.
func (pv *parityVolume) xorInto(volumeOffset int64, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	pv.mu.Lock()
	defer pv.mu.Unlock()

	// Ensure parity file is large enough.
	endPos := volumeOffset + int64(len(data))
	if endPos > maxParityFileSize {
		return fmt.Errorf("narwhal: parity file too large: %d", endPos)
	}
	info, _ := pv.fd.Stat()
	if info != nil && info.Size() < endPos {
		pv.fd.Truncate(endPos)
	}

	// Read existing parity bytes at this offset.
	existing := make([]byte, len(data))
	pv.fd.ReadAt(existing, volumeOffset)

	// XOR.
	for i := range data {
		existing[i] ^= data[i]
	}

	// Write back.
	pv.fd.WriteAt(existing, volumeOffset)
	return nil
}

func (pv *parityVolume) close() {
	if pv.fd != nil {
		pv.fd.Close()
		pv.fd = nil
	}
}

// ---------------------------------------------------------------------------
// Index
// ---------------------------------------------------------------------------

const indexShards = 256

type indexEntry struct {
	valueOffset int64
	size        int64
	contentType string
	created     int64 // UnixNano
	updated     int64 // UnixNano
	volID       int
}

type indexShard struct {
	mu      sync.RWMutex
	entries map[string]*indexEntry // compositeKey -> entry
}

type shardedIndex struct {
	shards [indexShards]indexShard
}

func newIndex() *shardedIndex {
	idx := &shardedIndex{}
	for i := range idx.shards {
		idx.shards[i].entries = make(map[string]*indexEntry, 64)
	}
	return idx
}

func (idx *shardedIndex) shardFor(ck string) *indexShard {
	return &idx.shards[fnv1a(ck)%indexShards]
}

func (idx *shardedIndex) put(compositeKey string, volID int, e *indexEntry) {
	e.volID = volID
	s := idx.shardFor(compositeKey)
	s.mu.Lock()
	s.entries[compositeKey] = e
	s.mu.Unlock()
}

func (idx *shardedIndex) get(compositeKey string) (*indexEntry, bool) {
	s := idx.shardFor(compositeKey)
	s.mu.RLock()
	e, ok := s.entries[compositeKey]
	s.mu.RUnlock()
	return e, ok
}

func (idx *shardedIndex) remove(compositeKey string) bool {
	s := idx.shardFor(compositeKey)
	s.mu.Lock()
	_, ok := s.entries[compositeKey]
	if ok {
		delete(s.entries, compositeKey)
	}
	s.mu.Unlock()
	return ok
}

// list returns all entries matching a bucket and optional key prefix, sorted by key.
func (idx *shardedIndex) list(bucketName, prefix string) []listResult {
	fullPrefix := bucketName + "\x00" + prefix

	var results []listResult
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		for ck, e := range s.entries {
			if !strings.HasPrefix(ck, fullPrefix) {
				continue
			}
			// Extract key from composite key.
			sepIdx := strings.IndexByte(ck, 0)
			if sepIdx < 0 {
				continue
			}
			b := ck[:sepIdx]
			if b != bucketName {
				continue
			}
			key := ck[sepIdx+1:]
			results = append(results, listResult{key: key, entry: e})
		}
		s.mu.RUnlock()
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].key < results[j].key
	})

	return results
}

// hasBucket returns true if any keys exist for the given bucket.
func (idx *shardedIndex) hasBucket(bucketName string) bool {
	prefix := bucketName + "\x00"
	for i := range idx.shards {
		s := &idx.shards[i]
		s.mu.RLock()
		for ck := range s.entries {
			if strings.HasPrefix(ck, prefix) {
				s.mu.RUnlock()
				return true
			}
		}
		s.mu.RUnlock()
	}
	return false
}

type listResult struct {
	key   string
	entry *indexEntry
}

// ---------------------------------------------------------------------------
// Bucket (storage.Bucket + storage.HasDirectories + storage.HasMultipart)
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

func (b *bucket) Features() storage.Features { return b.st.Features() }

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mu.RLock()
	created, ok := b.st.buckets[b.name]
	b.st.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{Name: b.name, CreatedAt: created}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Read all data.
	var data []byte
	if size >= 0 && size <= 10*1024*1024 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("narwhal: read: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("narwhal: read: %w", err)
		}
		data = buf.Bytes()
	}

	ck := compositeKey(b.name, key)
	volID := b.st.volumeFor(ck)
	vol := b.st.vols[volID]

	now := fastNow()

	offset, err := vol.appendRecord(recPut, ck, contentType, data, now, now)
	if err != nil {
		return nil, err
	}

	// Compute value offset within the record.
	// Record: type(1) + keyLen(2) + key + ctLen(2) + ct + valLen(8) + value + created(8) + updated(8)
	valueOffset := offset + 1 + 2 + int64(len(ck)) + 2 + int64(len(contentType)) + 8

	// XOR value into parity.
	if err := b.st.parity.xorInto(valueOffset, data); err != nil {
		return nil, err
	}

	// Update index.
	b.st.idx.put(ck, volID, &indexEntry{
		valueOffset: valueOffset,
		size:        int64(len(data)),
		contentType: contentType,
		created:     now,
		updated:     now,
	})

	if b.st.syncMode == "full" {
		vol.sync()
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     time.Unix(0, now),
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
	e, ok := b.st.idx.get(ck)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	vol := b.st.vols[e.volID]
	data, err := vol.readValue(e)
	if err != nil {
		return nil, nil, err
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
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
	data = data[offset:end]

	return &bytesReader{data: data}, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	// Check for directory stat.
	if strings.HasSuffix(key, "/") {
		results := b.st.idx.list(b.name, key)
		if len(results) == 0 {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:  b.name,
			Key:     strings.TrimSuffix(key, "/"),
			IsDir:   true,
			Created: time.Unix(0, results[0].entry.created),
			Updated: time.Unix(0, results[0].entry.updated),
		}, nil
	}

	ck := compositeKey(b.name, key)
	e, ok := b.st.idx.get(ck)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        e.size,
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
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

	ck := compositeKey(b.name, key)

	e, ok := b.st.idx.get(ck)
	if !ok {
		return storage.ErrNotExist
	}

	volID := e.volID
	vol := b.st.vols[volID]

	// Append tombstone.
	now := fastNow()
	vol.appendRecord(recDelete, ck, "", nil, now, now)

	// Remove from index.
	b.st.idx.remove(ck)
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
	srcBucket = safeName(srcBucket)

	srcCK := compositeKey(srcBucket, srcKey)
	srcEntry, ok := b.st.idx.get(srcCK)
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcVol := b.st.vols[srcEntry.volID]
	data, err := srcVol.readValue(srcEntry)
	if err != nil {
		return nil, err
	}

	// Write to destination.
	dstCK := compositeKey(b.name, dstKey)
	dstVolID := b.st.volumeFor(dstCK)
	dstVol := b.st.vols[dstVolID]

	now := fastNow()
	offset, err := dstVol.appendRecord(recPut, dstCK, srcEntry.contentType, data, now, now)
	if err != nil {
		return nil, err
	}

	valueOffset := offset + 1 + 2 + int64(len(dstCK)) + 2 + int64(len(srcEntry.contentType)) + 8

	if err := b.st.parity.xorInto(valueOffset, data); err != nil {
		return nil, err
	}

	b.st.idx.put(dstCK, dstVolID, &indexEntry{
		valueOffset: valueOffset,
		size:        int64(len(data)),
		contentType: srcEntry.contentType,
		created:     now,
		updated:     now,
	})

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        int64(len(data)),
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
	if err := sb.Delete(ctx, srcKey, nil); err != nil && !errors.Is(err, storage.ErrNotExist) {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = cleanPrefix(prefix)

	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	results := b.st.idx.list(b.name, prefix)

	objs := make([]*storage.Object, 0, len(results))
	seen := make(map[string]bool) // for non-recursive directory dedup

	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if idx := strings.IndexByte(rest, '/'); idx >= 0 {
				// This is a nested key; emit a directory entry.
				dirName := prefix + rest[:idx+1]
				if seen[dirName] {
					continue
				}
				seen[dirName] = true
				objs = append(objs, &storage.Object{
					Bucket: b.name,
					Key:    strings.TrimSuffix(dirName, "/"),
					IsDir:  true,
				})
				continue
			}
		}
		objs = append(objs, &storage.Object{
			Bucket:      b.name,
			Key:         r.key,
			Size:        r.entry.size,
			ContentType: r.entry.contentType,
			Created:     time.Unix(0, r.entry.created),
			Updated:     time.Unix(0, r.entry.updated),
		})
	}

	objs = paginateObjects(objs, limit, offset)
	return &objectIter{list: objs}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// Directory (storage.HasDirectories + storage.Directory)
// ---------------------------------------------------------------------------

func (b *bucket) Directory(path string) storage.Directory {
	return &dir{b: b, path: strings.Trim(path, "/")}
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

	results := d.b.st.idx.list(d.b.name, prefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: time.Unix(0, results[0].entry.created),
		Updated: time.Unix(0, results[0].entry.updated),
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

	results := d.b.st.idx.list(d.b.name, prefix)

	var objs []*storage.Object
	for _, r := range results {
		rest := strings.TrimPrefix(r.key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         r.key,
			Size:        r.entry.size,
			ContentType: r.entry.contentType,
			Created:     time.Unix(0, r.entry.created),
			Updated:     time.Unix(0, r.entry.updated),
		})
	}

	objs = paginateObjects(objs, limit, offset)
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

	results := d.b.st.idx.list(d.b.name, prefix)
	if len(results) == 0 {
		return storage.ErrNotExist
	}

	for _, r := range results {
		if !recursive {
			rest := strings.TrimPrefix(r.key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		ck := compositeKey(d.b.name, r.key)
		d.b.st.idx.remove(ck)
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

	results := d.b.st.idx.list(d.b.name, srcPrefix)
	if len(results) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, r := range results {
		rel := strings.TrimPrefix(r.key, srcPrefix)
		newKey := dstPrefix + rel

		oldCK := compositeKey(d.b.name, r.key)
		newCK := compositeKey(d.b.name, newKey)
		newVolID := d.b.st.volumeFor(newCK)

		d.b.st.idx.put(newCK, newVolID, &indexEntry{
			valueOffset: r.entry.valueOffset,
			size:        r.entry.size,
			contentType: r.entry.contentType,
			created:     r.entry.created,
			updated:     r.entry.updated,
			volID:       r.entry.volID, // keep reading from original volume
		})
		d.b.st.idx.remove(oldCK)
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart (storage.HasMultipart)
// ---------------------------------------------------------------------------

type mpRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*mpUpload
}

type mpUpload struct {
	mu          *storage.MultipartUpload
	contentType string
	createdAt   time.Time
	parts       map[int]*mpPart
}

type mpPart struct {
	number       int
	data         []byte
	etag         string
	lastModified time.Time
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := cleanKey(key)
	if err != nil {
		return nil, err
	}

	uploadID := newUploadID()
	mu := &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: uploadID,
	}

	b.st.mp.mu.Lock()
	b.st.mp.uploads[uploadID] = &mpUpload{
		mu:          mu,
		contentType: contentType,
		createdAt:   fastNowTime(),
		parts:       make(map[int]*mpPart),
	}
	b.st.mp.mu.Unlock()

	return mu, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("narwhal: part number %d out of range (1-%d)", number, maxPartNumber)
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("narwhal: read part: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("narwhal: read part: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNowTime()
	sum := md5.Sum(data)
	etag := hex.EncodeToString(sum[:])

	pd := &mpPart{
		number:       number,
		data:         data,
		etag:         etag,
		lastModified: now,
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	upload.parts[number] = pd
	b.st.mp.mu.Unlock()

	return &storage.PartInfo{
		Number:       number,
		Size:         int64(len(data)),
		ETag:         etag,
		LastModified: &now,
	}, nil
}

func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("narwhal: part number %d out of range", number)
	}

	b.st.mp.mu.RLock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Get source parameters.
	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("narwhal: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	srcBucket = safeName(srcBucket)
	srcCK := compositeKey(srcBucket, srcKey)
	srcEntry, ok := b.st.idx.get(srcCK)
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcVol := b.st.vols[srcEntry.volID]
	data, err := srcVol.readValue(srcEntry)
	if err != nil {
		return nil, err
	}

	// Apply range.
	if srcOffset > 0 {
		if srcOffset > int64(len(data)) {
			srcOffset = int64(len(data))
		}
		data = data[srcOffset:]
	}
	if srcLength > 0 && srcLength < int64(len(data)) {
		data = data[:srcLength]
	}

	return b.UploadPart(ctx, mu, number, bytes.NewReader(data), int64(len(data)), opts)
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.st.mp.mu.RLock()
	defer b.st.mp.mu.RUnlock()

	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, pd := range upload.parts {
		lm := pd.lastModified
		parts = append(parts, &storage.PartInfo{
			Number:       pd.number,
			Size:         int64(len(pd.data)),
			ETag:         pd.etag,
			LastModified: &lm,
		})
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	if offset < 0 {
		offset = 0
	}
	if offset > len(parts) {
		offset = len(parts)
	}
	parts = parts[offset:]
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, errors.New("narwhal: no parts to complete")
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}

	sortedParts := make([]*storage.PartInfo, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool {
		return sortedParts[i].Number < sortedParts[j].Number
	})

	// Calculate total size and concatenate.
	totalSize := 0
	for _, part := range sortedParts {
		pd, exists := upload.parts[part.Number]
		if !exists {
			b.st.mp.mu.Unlock()
			return nil, fmt.Errorf("narwhal: part %d not found", part.Number)
		}
		totalSize += len(pd.data)
	}

	data := make([]byte, 0, totalSize)
	for _, part := range sortedParts {
		pd := upload.parts[part.Number]
		data = append(data, pd.data...)
	}

	key := upload.mu.Key
	ct := upload.contentType
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Write assembled data.
	ck := compositeKey(b.name, key)
	volID := b.st.volumeFor(ck)
	vol := b.st.vols[volID]

	now := fastNow()
	recOffset, err := vol.appendRecord(recPut, ck, ct, data, now, now)
	if err != nil {
		return nil, err
	}

	valueOffset := recOffset + 1 + 2 + int64(len(ck)) + 2 + int64(len(ct)) + 8

	if err := b.st.parity.xorInto(valueOffset, data); err != nil {
		return nil, err
	}

	b.st.idx.put(ck, volID, &indexEntry{
		valueOffset: valueOffset,
		size:        int64(totalSize),
		contentType: ct,
		created:     now,
		updated:     now,
	})

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(totalSize),
		ContentType: ct,
		Created:     time.Unix(0, now),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.st.mp.mu.Lock()
	defer b.st.mp.mu.Unlock()

	if _, ok := b.st.mp.uploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
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
// bytesReader
// ---------------------------------------------------------------------------

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *bytesReader) WriteTo(w io.Writer) (int64, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n, err := w.Write(r.data[r.pos:])
	r.pos = len(r.data)
	return int64(n), err
}

func (r *bytesReader) Close() error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func safeName(name string) string {
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

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("narwhal: empty key")
	}
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", errors.New("narwhal: empty key")
	}
	// Reject path traversal.
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", storage.ErrPermission
		}
	}
	return key, nil
}

func cleanPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	prefix = strings.TrimPrefix(prefix, "/")
	return prefix
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

func fnv1a(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// Fast cached time (per-store; no global goroutine).
var cachedTimeNano atomic.Int64

func init() {
	cachedTimeNano.Store(time.Now().UnixNano())
}

func startTimeTicker() chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case t := <-ticker.C:
				cachedTimeNano.Store(t.UnixNano())
			}
		}
	}()
	return stop
}

func fastNow() int64 {
	return cachedTimeNano.Load()
}

func fastNowTime() time.Time {
	return time.Unix(0, fastNow())
}

func newUploadID() string {
	now := time.Now().UTC().UnixNano()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%x-0", now)
	}
	r := binary.LittleEndian.Uint64(b[:])
	return fmt.Sprintf("%x-%x", now, r)
}

func paginate[T any](s []T, limit, offset int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset > len(s) {
		offset = len(s)
	}
	s = s[offset:]
	if limit > 0 && limit < len(s) {
		s = s[:limit]
	}
	return s
}

func paginateObjects(objs []*storage.Object, limit, offset int) []*storage.Object {
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
	return objs
}
