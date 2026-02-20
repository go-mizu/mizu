// Package spider implements a storage driver inspired by the SplinterDB + Maplets
// paper (SIGMOD 2023). It uses an LSM-like tree-of-trees architecture with:
//
//   - Level 0: in-memory sorted memtable
//   - Level 1+: on-disk sorted run files (SST) with per-run Bloom filters
//   - Size-tiered compaction: merge runs when a level has too many
//   - Trunk manifest tracking which runs exist at which level
//
// DSN format:
//
//	spider:///path/to/data?sync=none&levels=4&memtable_size=4194304
package spider

import (
	"bytes"
	"context"
	"crypto/md5"
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
	storage.Register("spider", &driver{})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultMemtableSize = 4 * 1024 * 1024 // 4 MB
	defaultMaxLevels    = 4
	defaultRunsPerLevel = 4

	sstMagic   = "SPDR"
	sstVersion = 1

	tombstoneMarker = ^uint64(0) // 0xFFFFFFFFFFFFFFFF

	dirPerm  = 0o750
	filePerm = 0o600

	bloomBitsPerItem = 10
	bloomNumHash     = 7

	maxPartNumber = 10000

	maxSSTSize         = 256 * 1024 * 1024 // 256 MB
	maxBuckets         = 10000
	maxMultipartAssembly = 1 << 30 // 1 GB
)

// ---------------------------------------------------------------------------
// Driver
// ---------------------------------------------------------------------------

type driver struct{}

func (d *driver) Open(_ context.Context, dsn string) (storage.Storage, error) {
	root, opts, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	syncMode := opts.Get("sync")
	if syncMode == "" {
		syncMode = "none"
	}
	maxLevels := intOpt(opts, "levels", defaultMaxLevels)
	memSize := intOpt(opts, "memtable_size", defaultMemtableSize)
	runsPerLevel := intOpt(opts, "runs_per_level", defaultRunsPerLevel)

	if err := os.MkdirAll(root, dirPerm); err != nil {
		return nil, fmt.Errorf("spider: mkdir %q: %w", root, err)
	}

	s := &store{
		root:         root,
		syncMode:     syncMode,
		maxLevels:    maxLevels,
		memtableSize: int64(memSize),
		runsPerLevel: runsPerLevel,
		mem:          newMemtable(),
		manifest:     &manifest{Levels: make(map[int][]string)},
	}

	if err := s.loadManifest(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("spider: load manifest: %w", err)
	}

	// Load bloom filters for all existing runs.
	s.loadBloomFilters()

	return s, nil
}

// ---------------------------------------------------------------------------
// DSN parsing
// ---------------------------------------------------------------------------

func parseDSN(dsn string) (string, url.Values, error) {
	if dsn == "" {
		return "", nil, errors.New("spider: empty dsn")
	}

	var queryStr string
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		queryStr = dsn[idx+1:]
		dsn = dsn[:idx]
	}
	opts, _ := url.ParseQuery(queryStr)

	if strings.HasPrefix(dsn, "spider:") {
		rest := strings.TrimPrefix(dsn, "spider:")
		if strings.HasPrefix(rest, "//") {
			rest = strings.TrimPrefix(rest, "//")
		}
		if rest == "" {
			return "", nil, errors.New("spider: missing path")
		}
		return filepath.Clean(rest), opts, nil
	}

	if strings.HasPrefix(dsn, "/") {
		return filepath.Clean(dsn), opts, nil
	}

	return "", nil, fmt.Errorf("spider: invalid dsn %q", dsn)
}

func intOpt(q url.Values, key string, def int) int {
	v := q.Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// ---------------------------------------------------------------------------
// Manifest
// ---------------------------------------------------------------------------

type manifest struct {
	Levels    map[int][]string `json:"levels"`
	NextRunID int              `json:"next_run_id"`
	Buckets   map[string]int64 `json:"buckets,omitempty"` // name -> creation unix nano
}

func (s *store) loadManifest() error {
	data, err := os.ReadFile(filepath.Join(s.root, "manifest.json"))
	if err != nil {
		return err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("spider: unmarshal manifest: %w", err)
	}
	if m.Levels == nil {
		m.Levels = make(map[int][]string)
	}
	s.manifest = &m

	// Restore bucket map.
	if m.Buckets != nil {
		for name, nano := range m.Buckets {
			s.bucketMap.Store(name, time.Unix(0, nano))
		}
	}
	return nil
}

func (s *store) saveManifest() error {
	// Snapshot bucket map into manifest.
	buckets := make(map[string]int64)
	s.bucketMap.Range(func(key, value any) bool {
		buckets[key.(string)] = value.(time.Time).UnixNano()
		return true
	})
	s.manifest.Buckets = buckets

	data, err := json.MarshalIndent(s.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("spider: marshal manifest: %w", err)
	}
	tmp := filepath.Join(s.root, ".manifest.tmp")
	if err := os.WriteFile(tmp, data, filePerm); err != nil {
		return fmt.Errorf("spider: write manifest: %w", err)
	}
	return os.Rename(tmp, filepath.Join(s.root, "manifest.json"))
}

// ---------------------------------------------------------------------------
// Bloom filter
// ---------------------------------------------------------------------------

type bloomFilter struct {
	bits    []byte
	numBits uint64
	numHash int
}

func newBloomFilter(expectedItems int) *bloomFilter {
	if expectedItems < 64 {
		expectedItems = 64
	}
	numBits := uint64(expectedItems) * bloomBitsPerItem
	numBits = (numBits + 7) &^ 7 // byte-align
	return &bloomFilter{
		bits:    make([]byte, numBits/8),
		numBits: numBits,
		numHash: bloomNumHash,
	}
}

func (bf *bloomFilter) add(key []byte) {
	h1, h2 := bloomHash(key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		bf.bits[bit/8] |= 1 << (bit % 8)
	}
}

func (bf *bloomFilter) mayContain(key []byte) bool {
	h1, h2 := bloomHash(key)
	for i := 0; i < bf.numHash; i++ {
		bit := (h1 + uint64(i)*h2) % bf.numBits
		if bf.bits[bit/8]&(1<<(bit%8)) == 0 {
			return false
		}
	}
	return true
}

func bloomHash(key []byte) (uint64, uint64) {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h1 := uint64(offset64)
	for _, b := range key {
		h1 ^= uint64(b)
		h1 *= prime64
	}
	h2 := h1 ^ 0xDEADBEEFCAFEBABE
	for i := len(key) - 1; i >= 0; i-- {
		h2 ^= uint64(key[i])
		h2 *= prime64
	}
	h2 |= 1 // ensure odd for better distribution
	return h1, h2
}

// loadBloomFilters loads bloom filters from all existing SST files.
func (s *store) loadBloomFilters() {
	for _, runs := range s.manifest.Levels {
		for _, fname := range runs {
			bf, err := loadBloomFromSST(filepath.Join(s.root, fname))
			if err == nil {
				s.blooms.Store(fname, bf)
			}
		}
	}
}

func loadBloomFromSST(path string) (*bloomFilter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// Read bloom from end of file: last 5 bytes = bloomLen(4) + numHash(1),
	// then bloomLen bytes before that.
	size := info.Size()
	if size < 5 {
		return nil, errors.New("spider: sst too small for bloom")
	}

	// Read the trailing 5-byte footer.
	footer := make([]byte, 5)
	if _, err := f.ReadAt(footer, size-5); err != nil {
		return nil, err
	}
	bloomLen := binary.LittleEndian.Uint32(footer[:4])
	numHash := int(footer[4])
	if numHash == 0 || int64(bloomLen)+5 > size {
		return nil, errors.New("spider: invalid bloom footer")
	}

	bits := make([]byte, bloomLen)
	if _, err := f.ReadAt(bits, size-5-int64(bloomLen)); err != nil {
		return nil, err
	}

	return &bloomFilter{
		bits:    bits,
		numBits: uint64(bloomLen) * 8,
		numHash: numHash,
	}, nil
}

// ---------------------------------------------------------------------------
// Memtable
// ---------------------------------------------------------------------------

type memEntry struct {
	key         []byte // composite: bucket + "\x00" + objectKey
	contentType string
	value       []byte // nil for tombstone
	isTombstone bool
	created     int64 // unix nano
	updated     int64
}

type memtable struct {
	mu      sync.RWMutex
	entries []*memEntry
	size    int64 // approximate byte size
}

func newMemtable() *memtable {
	return &memtable{
		entries: make([]*memEntry, 0, 1024),
	}
}

// put inserts or updates an entry. Returns the new approximate size.
func (m *memtable) put(compositeKey []byte, contentType string, value []byte, isTombstone bool, created, updated int64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := sort.Search(len(m.entries), func(i int) bool {
		return bytes.Compare(m.entries[i].key, compositeKey) >= 0
	})

	e := &memEntry{
		key:         compositeKey,
		contentType: contentType,
		value:       value,
		isTombstone: isTombstone,
		created:     created,
		updated:     updated,
	}

	entrySize := int64(len(compositeKey) + len(contentType) + len(value) + 64)

	if idx < len(m.entries) && bytes.Equal(m.entries[idx].key, compositeKey) {
		old := m.entries[idx]
		oldSize := int64(len(old.key) + len(old.contentType) + len(old.value) + 64)
		m.size -= oldSize
		m.entries[idx] = e
	} else {
		m.entries = append(m.entries, nil)
		copy(m.entries[idx+1:], m.entries[idx:])
		m.entries[idx] = e
	}

	m.size += entrySize
	return m.size
}

// get looks up a key. Returns (entry, found).
func (m *memtable) get(compositeKey []byte) (*memEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx := sort.Search(len(m.entries), func(i int) bool {
		return bytes.Compare(m.entries[i].key, compositeKey) >= 0
	})
	if idx < len(m.entries) && bytes.Equal(m.entries[idx].key, compositeKey) {
		return m.entries[idx], true
	}
	return nil, false
}

// snapshot returns a sorted copy of all entries for flushing.
func (m *memtable) snapshot() []*memEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cp := make([]*memEntry, len(m.entries))
	copy(cp, m.entries)
	return cp
}

// scan returns entries whose composite key starts with prefix, in sorted order.
func (m *memtable) scan(prefix []byte) []*memEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx := sort.Search(len(m.entries), func(i int) bool {
		return bytes.Compare(m.entries[i].key, prefix) >= 0
	})

	var result []*memEntry
	for i := idx; i < len(m.entries); i++ {
		if !bytes.HasPrefix(m.entries[i].key, prefix) {
			break
		}
		result = append(result, m.entries[i])
	}
	return result
}

// ---------------------------------------------------------------------------
// SST file writing
// ---------------------------------------------------------------------------

// writeSST writes sorted entries to an SST file and returns the bloom filter.
func writeSST(path string, entries []*memEntry) (*bloomFilter, error) {
	if len(entries) == 0 {
		return newBloomFilter(64), nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return nil, fmt.Errorf("spider: create sst: %w", err)
	}
	defer f.Close()

	bf := newBloomFilter(len(entries))

	// Write header.
	minKey := entries[0].key
	maxKey := entries[len(entries)-1].key

	var hdr bytes.Buffer
	hdr.WriteString(sstMagic)
	binary.Write(&hdr, binary.LittleEndian, uint32(sstVersion))
	binary.Write(&hdr, binary.LittleEndian, uint64(len(entries)))
	binary.Write(&hdr, binary.LittleEndian, uint16(len(minKey)))
	hdr.Write(minKey)
	binary.Write(&hdr, binary.LittleEndian, uint16(len(maxKey)))
	hdr.Write(maxKey)

	if _, err := f.Write(hdr.Bytes()); err != nil {
		return nil, fmt.Errorf("spider: write sst header: %w", err)
	}

	// Write entries.
	var buf bytes.Buffer
	for _, e := range entries {
		buf.Reset()
		binary.Write(&buf, binary.LittleEndian, uint16(len(e.key)))
		buf.Write(e.key)
		binary.Write(&buf, binary.LittleEndian, uint16(len(e.contentType)))
		buf.WriteString(e.contentType)
		if e.isTombstone {
			binary.Write(&buf, binary.LittleEndian, uint64(tombstoneMarker))
		} else {
			binary.Write(&buf, binary.LittleEndian, uint64(len(e.value)))
			buf.Write(e.value)
		}
		binary.Write(&buf, binary.LittleEndian, e.created)
		binary.Write(&buf, binary.LittleEndian, e.updated)

		if _, err := f.Write(buf.Bytes()); err != nil {
			return nil, fmt.Errorf("spider: write sst entry: %w", err)
		}

		bf.add(e.key)
	}

	// Write bloom filter at end of file: bits + bloomLen(4) + numHash(1).
	if _, err := f.Write(bf.bits); err != nil {
		return nil, fmt.Errorf("spider: write bloom: %w", err)
	}
	binary.Write(f, binary.LittleEndian, uint32(len(bf.bits)))
	f.Write([]byte{byte(bf.numHash)})

	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("spider: sync sst: %w", err)
	}

	return bf, nil
}

// ---------------------------------------------------------------------------
// SST file reading
// ---------------------------------------------------------------------------

// readSST reads all entries from an SST file.
func readSST(path string) ([]*memEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("spider: stat sst: %w", err)
	}
	if info.Size() > maxSSTSize {
		return nil, fmt.Errorf("spider: sst file %s too large (%d bytes, max %d)", filepath.Base(path), info.Size(), maxSSTSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("spider: read sst: %w", err)
	}

	if len(data) < 4 || string(data[:4]) != sstMagic {
		return nil, errors.New("spider: invalid sst magic")
	}

	r := bytes.NewReader(data[4:])

	var version uint32
	binary.Read(r, binary.LittleEndian, &version)

	var count uint64
	binary.Read(r, binary.LittleEndian, &count)

	// Skip min/max keys in header.
	var keyLen uint16
	binary.Read(r, binary.LittleEndian, &keyLen)
	r.Seek(int64(keyLen), io.SeekCurrent)
	binary.Read(r, binary.LittleEndian, &keyLen)
	r.Seek(int64(keyLen), io.SeekCurrent)

	entries := make([]*memEntry, 0, count)
	for i := uint64(0); i < count; i++ {
		e, err := readSSTEntry(r)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func readSSTEntry(r *bytes.Reader) (*memEntry, error) {
	var kl uint16
	if err := binary.Read(r, binary.LittleEndian, &kl); err != nil {
		return nil, err
	}
	key := make([]byte, kl)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}

	var ctLen uint16
	if err := binary.Read(r, binary.LittleEndian, &ctLen); err != nil {
		return nil, err
	}
	ct := make([]byte, ctLen)
	if _, err := io.ReadFull(r, ct); err != nil {
		return nil, err
	}

	var valLen uint64
	if err := binary.Read(r, binary.LittleEndian, &valLen); err != nil {
		return nil, err
	}

	e := &memEntry{
		key:         key,
		contentType: string(ct),
	}

	if valLen == tombstoneMarker {
		e.isTombstone = true
	} else {
		e.value = make([]byte, valLen)
		if _, err := io.ReadFull(r, e.value); err != nil {
			return nil, err
		}
	}

	binary.Read(r, binary.LittleEndian, &e.created)
	binary.Read(r, binary.LittleEndian, &e.updated)

	return e, nil
}

// searchSST searches for a key in an SST file using binary search-like sequential scan.
// For a proper implementation we would build an index; here we scan entries because
// the bloom filter already eliminates most files.
func searchSST(path string, compositeKey []byte) (*memEntry, error) {
	entries, err := readSST(path)
	if err != nil {
		return nil, err
	}

	idx := sort.Search(len(entries), func(i int) bool {
		return bytes.Compare(entries[i].key, compositeKey) >= 0
	})
	if idx < len(entries) && bytes.Equal(entries[idx].key, compositeKey) {
		return entries[idx], nil
	}
	return nil, nil
}

// scanSST returns all entries in an SST file whose key starts with prefix.
func scanSST(path string, prefix []byte) ([]*memEntry, error) {
	entries, err := readSST(path)
	if err != nil {
		return nil, err
	}

	idx := sort.Search(len(entries), func(i int) bool {
		return bytes.Compare(entries[i].key, prefix) >= 0
	})

	var result []*memEntry
	for i := idx; i < len(entries); i++ {
		if !bytes.HasPrefix(entries[i].key, prefix) {
			break
		}
		result = append(result, entries[i])
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Composite key helpers
// ---------------------------------------------------------------------------

func compositeKey(bucket, key string) []byte {
	b := make([]byte, len(bucket)+1+len(key))
	copy(b, bucket)
	b[len(bucket)] = 0
	copy(b[len(bucket)+1:], key)
	return b
}

func splitCompositeKey(ck []byte) (bucket, key string) {
	idx := bytes.IndexByte(ck, 0)
	if idx < 0 {
		return string(ck), ""
	}
	return string(ck[:idx]), string(ck[idx+1:])
}

func bucketPrefix(bucket string) []byte {
	b := make([]byte, len(bucket)+1)
	copy(b, bucket)
	b[len(bucket)] = 0
	return b
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

type store struct {
	root         string
	syncMode     string
	maxLevels    int
	memtableSize int64
	runsPerLevel int

	mu       sync.Mutex // protects flush/compaction and manifest writes
	mem      *memtable
	manifest *manifest

	blooms    sync.Map // filename -> *bloomFilter
	bucketMap sync.Map // name -> time.Time

	mp mpRegistry

	closed atomic.Bool
}

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}
	if _, ok := s.bucketMap.Load(name); !ok {
		// Count existing buckets before adding a new one.
		count := 0
		s.bucketMap.Range(func(_, _ any) bool {
			count++
			return count < maxBuckets
		})
		if count < maxBuckets {
			s.bucketMap.LoadOrStore(name, time.Now())
		}
	}
	return &bucket{st: s, name: name}
}

func (s *store) Buckets(_ context.Context, limit, offset int, _ storage.Options) (storage.BucketIter, error) {
	var infos []*storage.BucketInfo
	s.bucketMap.Range(func(key, value any) bool {
		infos = append(infos, &storage.BucketInfo{
			Name:      key.(string),
			CreatedAt: value.(time.Time),
		})
		return true
	})
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })

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

func (s *store) CreateBucket(_ context.Context, name string, _ storage.Options) (*storage.BucketInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("spider: bucket name required")
	}
	now := time.Now()
	if _, loaded := s.bucketMap.LoadOrStore(name, now); loaded {
		return nil, storage.ErrExist
	}
	return &storage.BucketInfo{Name: name, CreatedAt: now}, nil
}

func (s *store) DeleteBucket(_ context.Context, name string, opts storage.Options) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("spider: bucket name required")
	}
	if _, ok := s.bucketMap.Load(name); !ok {
		return storage.ErrNotExist
	}

	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	if !force {
		// Check if bucket has any entries.
		prefix := bucketPrefix(name)
		entries := s.mem.scan(prefix)
		if len(entries) > 0 {
			return storage.ErrPermission
		}
		// Check on-disk levels.
		for level := 1; level <= s.maxLevels; level++ {
			runs, ok := s.manifest.Levels[level]
			if !ok {
				continue
			}
			for _, fname := range runs {
				results, _ := scanSST(filepath.Join(s.root, fname), prefix)
				for _, e := range results {
					if !e.isTombstone {
						return storage.ErrPermission
					}
				}
			}
		}
	}

	s.bucketMap.Delete(name)
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
	if s.closed.Swap(true) {
		return nil
	}

	// Flush memtable to disk.
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.mem.entries) > 0 {
		if err := s.flushMemtableLocked(); err != nil {
			return fmt.Errorf("spider: close flush: %w", err)
		}
	}

	return s.saveManifest()
}

// ---------------------------------------------------------------------------
// Core: get from tree-of-trees
// ---------------------------------------------------------------------------

func (s *store) get(ck []byte) (*memEntry, bool) {
	// Level 0: memtable.
	if e, ok := s.mem.get(ck); ok {
		if e.isTombstone {
			return nil, false
		}
		return e, true
	}

	// Level 1+: check each level top-down, newest run first.
	s.mu.Lock()
	levels := make(map[int][]string, len(s.manifest.Levels))
	for k, v := range s.manifest.Levels {
		runs := make([]string, len(v))
		copy(runs, v)
		levels[k] = runs
	}
	s.mu.Unlock()

	for level := 1; level <= s.maxLevels; level++ {
		runs, ok := levels[level]
		if !ok {
			continue
		}
		// Search runs in reverse order (newest first).
		for i := len(runs) - 1; i >= 0; i-- {
			fname := runs[i]

			// Check bloom filter first.
			if bfVal, ok := s.blooms.Load(fname); ok {
				bf := bfVal.(*bloomFilter)
				if !bf.mayContain(ck) {
					continue
				}
			}

			e, err := searchSST(filepath.Join(s.root, fname), ck)
			if err != nil {
				continue
			}
			if e != nil {
				if e.isTombstone {
					return nil, false
				}
				return e, true
			}
		}
	}

	return nil, false
}

// scan returns all non-tombstone entries matching a prefix, across all levels.
// Later entries (higher levels, newer runs) override earlier ones.
func (s *store) scan(prefix []byte) []*memEntry {
	// Collect from all levels, dedup by key keeping newest.
	seen := make(map[string]*memEntry)

	// Level 0: memtable.
	for _, e := range s.mem.scan(prefix) {
		seen[string(e.key)] = e
	}

	// Level 1+: each level, newest run last (so it overwrites older).
	s.mu.Lock()
	levels := make(map[int][]string, len(s.manifest.Levels))
	for k, v := range s.manifest.Levels {
		runs := make([]string, len(v))
		copy(runs, v)
		levels[k] = runs
	}
	s.mu.Unlock()

	for level := s.maxLevels; level >= 1; level-- {
		runs, ok := levels[level]
		if !ok {
			continue
		}
		for _, fname := range runs {
			entries, err := scanSST(filepath.Join(s.root, fname), prefix)
			if err != nil {
				continue
			}
			for _, e := range entries {
				seen[string(e.key)] = e
			}
		}
	}

	// Collect non-tombstone entries, sorted.
	var result []*memEntry
	for _, e := range seen {
		if !e.isTombstone {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return bytes.Compare(result[i].key, result[j].key) < 0
	})
	return result
}

// ---------------------------------------------------------------------------
// Core: put into memtable, maybe flush
// ---------------------------------------------------------------------------

func (s *store) put(ck []byte, contentType string, value []byte, isTombstone bool) {
	now := time.Now().UnixNano()
	newSize := s.mem.put(ck, contentType, value, isTombstone, now, now)

	if newSize >= s.memtableSize {
		s.mu.Lock()
		// Double-check after acquiring lock.
		if s.mem.size >= s.memtableSize {
			_ = s.flushMemtableLocked()
		}
		s.mu.Unlock()
	}
}

// flushMemtableLocked flushes the current memtable to a new Level 1 run.
// Caller must hold s.mu.
func (s *store) flushMemtableLocked() error {
	entries := s.mem.snapshot()
	if len(entries) == 0 {
		return nil
	}

	runID := s.manifest.NextRunID
	s.manifest.NextRunID++
	fname := fmt.Sprintf("L1_R%d.sst", runID)
	path := filepath.Join(s.root, fname)

	bf, err := writeSST(path, entries)
	if err != nil {
		return err
	}

	s.blooms.Store(fname, bf)

	if s.manifest.Levels[1] == nil {
		s.manifest.Levels[1] = []string{fname}
	} else {
		s.manifest.Levels[1] = append(s.manifest.Levels[1], fname)
	}

	// Replace memtable.
	s.mem = newMemtable()

	// Save manifest.
	if err := s.saveManifest(); err != nil {
		return err
	}

	// Check if compaction is needed.
	return s.compactIfNeededLocked(1)
}

// compactIfNeededLocked checks if level has too many runs and compacts.
// Caller must hold s.mu.
func (s *store) compactIfNeededLocked(level int) error {
	for level <= s.maxLevels {
		runs := s.manifest.Levels[level]
		if len(runs) <= s.runsPerLevel {
			return nil
		}

		// Merge all runs at this level into one run at level+1.
		targetLevel := level + 1
		if targetLevel > s.maxLevels {
			// At max level, merge within same level.
			targetLevel = level
		}

		var allEntries []*memEntry
		for _, fname := range runs {
			entries, err := readSST(filepath.Join(s.root, fname))
			if err != nil {
				return fmt.Errorf("spider: compact read %s: %w", fname, err)
			}
			allEntries = append(allEntries, entries...)
		}

		// Also merge with existing runs at target level if promoting.
		if targetLevel != level {
			for _, fname := range s.manifest.Levels[targetLevel] {
				entries, err := readSST(filepath.Join(s.root, fname))
				if err != nil {
					return fmt.Errorf("spider: compact read %s: %w", fname, err)
				}
				allEntries = append(allEntries, entries...)
			}
		}

		// K-way merge: sort all entries, dedup by key keeping newest.
		sort.SliceStable(allEntries, func(i, j int) bool {
			c := bytes.Compare(allEntries[i].key, allEntries[j].key)
			if c != 0 {
				return c < 0
			}
			// Same key: newer (higher updated) first.
			return allEntries[i].updated > allEntries[j].updated
		})

		// Dedup: keep first (newest) per key.
		merged := make([]*memEntry, 0, len(allEntries))
		for i, e := range allEntries {
			if i > 0 && bytes.Equal(allEntries[i-1].key, e.key) {
				continue
			}
			// Drop tombstones at max level.
			if e.isTombstone && targetLevel == s.maxLevels {
				continue
			}
			merged = append(merged, e)
		}

		// Write new run.
		runID := s.manifest.NextRunID
		s.manifest.NextRunID++
		fname := fmt.Sprintf("L%d_R%d.sst", targetLevel, runID)
		path := filepath.Join(s.root, fname)

		bf, err := writeSST(path, merged)
		if err != nil {
			return fmt.Errorf("spider: compact write: %w", err)
		}
		s.blooms.Store(fname, bf)

		// Remove old run files.
		for _, old := range runs {
			os.Remove(filepath.Join(s.root, old))
			s.blooms.Delete(old)
		}
		if targetLevel != level {
			for _, old := range s.manifest.Levels[targetLevel] {
				os.Remove(filepath.Join(s.root, old))
				s.blooms.Delete(old)
			}
			s.manifest.Levels[targetLevel] = []string{fname}
		} else {
			s.manifest.Levels[level] = []string{fname}
		}

		if targetLevel != level {
			delete(s.manifest.Levels, level)
		}

		if err := s.saveManifest(); err != nil {
			return err
		}

		// Check next level.
		level = targetLevel
	}

	return nil
}

// ---------------------------------------------------------------------------
// Bucket
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

func (b *bucket) Info(_ context.Context) (*storage.BucketInfo, error) {
	v, ok := b.st.bucketMap.Load(b.name)
	if !ok {
		return nil, storage.ErrNotExist
	}
	return &storage.BucketInfo{Name: b.name, CreatedAt: v.(time.Time)}, nil
}

func (b *bucket) Write(_ context.Context, key string, src io.Reader, size int64, contentType string, _ storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("spider: empty key")
	}

	var data []byte
	var err error
	if size == 0 {
		data = nil
	} else if size > 0 {
		data = make([]byte, size)
		n, readErr := io.ReadFull(src, data)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("spider: read: %w", readErr)
		}
		data = data[:n]
	} else {
		data, err = io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("spider: read: %w", err)
		}
	}

	ck := compositeKey(b.name, key)
	b.st.put(ck, contentType, data, false)

	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) Open(_ context.Context, key string, offset, length int64, _ storage.Options) (io.ReadCloser, *storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, errors.New("spider: empty key")
	}

	ck := compositeKey(b.name, key)
	e, ok := b.st.get(ck)
	if !ok {
		return nil, nil, storage.ErrNotExist
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(e.value)),
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}

	data := e.value
	if offset > 0 {
		if offset >= int64(len(data)) {
			data = nil
		} else {
			data = data[offset:]
		}
	}
	if length > 0 && length < int64(len(data)) {
		data = data[:length]
	}

	return io.NopCloser(bytes.NewReader(data)), obj, nil
}

func (b *bucket) Stat(_ context.Context, key string, _ storage.Options) (*storage.Object, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("spider: empty key")
	}

	// Directory stat.
	if strings.HasSuffix(key, "/") {
		prefix := compositeKey(b.name, key)
		entries := b.st.scan(prefix)
		if len(entries) > 0 {
			return &storage.Object{
				Bucket:  b.name,
				Key:     strings.TrimSuffix(key, "/"),
				IsDir:   true,
				Created: time.Unix(0, entries[0].created),
				Updated: time.Unix(0, entries[0].updated),
			}, nil
		}
		return nil, storage.ErrNotExist
	}

	ck := compositeKey(b.name, key)
	e, ok := b.st.get(ck)
	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(e.value)),
		ContentType: e.contentType,
		Created:     time.Unix(0, e.created),
		Updated:     time.Unix(0, e.updated),
	}, nil
}

func (b *bucket) Delete(_ context.Context, key string, _ storage.Options) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("spider: empty key")
	}

	ck := compositeKey(b.name, key)
	// Check if it exists before inserting tombstone.
	if _, ok := b.st.get(ck); !ok {
		return storage.ErrNotExist
	}

	b.st.put(ck, "", nil, true)
	return nil
}

func (b *bucket) Copy(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, errors.New("spider: empty key")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	// Read source data first (get acquires and releases memtable RLock).
	srcCK := compositeKey(srcBucket, srcKey)
	e, ok := b.st.get(srcCK)
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Snapshot all fields from the entry into local variables before any
	// further lock acquisition.  This avoids holding a read lock across the
	// subsequent put (which takes a write lock), preventing a lock-upgrade
	// deadlock.
	val := make([]byte, len(e.value))
	copy(val, e.value)
	ct := e.contentType
	size := int64(len(e.value))

	// Now write to destination — no lock is held from the get above.
	dstCK := compositeKey(b.name, dstKey)
	b.st.put(dstCK, ct, val, false)

	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        size,
		ContentType: ct,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) Move(_ context.Context, dstKey string, srcBucket, srcKey string, _ storage.Options) (*storage.Object, error) {
	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, errors.New("spider: empty key")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	// Read source data first (get acquires and releases memtable RLock).
	srcCK := compositeKey(srcBucket, srcKey)
	e, ok := b.st.get(srcCK)
	if !ok {
		return nil, storage.ErrNotExist
	}

	// Snapshot all fields from the entry into local variables before any
	// further lock acquisition.  This avoids holding a read lock across the
	// subsequent puts (which take a write lock), preventing a lock-upgrade
	// deadlock.
	val := make([]byte, len(e.value))
	copy(val, e.value)
	ct := e.contentType
	size := int64(len(e.value))

	// Now write to destination — no lock is held from the get above.
	dstCK := compositeKey(b.name, dstKey)
	b.st.put(dstCK, ct, val, false)

	// Delete source (tombstone).
	b.st.put(srcCK, "", nil, true)

	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        size,
		ContentType: ct,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) List(_ context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	cp := compositeKey(b.name, prefix)
	entries := b.st.scan(cp)

	var objects []*storage.Object
	for _, e := range entries {
		_, key := splitCompositeKey(e.key)
		if !recursive {
			rest := strings.TrimPrefix(key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if strings.Contains(rest, "/") {
				continue
			}
		}
		objects = append(objects, &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(e.value)),
			ContentType: e.contentType,
			Created:     time.Unix(0, e.created),
			Updated:     time.Unix(0, e.updated),
		})
	}

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

func (b *bucket) SignedURL(_ context.Context, _ string, _ string, _ time.Duration, _ storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// ---------------------------------------------------------------------------
// HasDirectories
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

func (d *dir) Info(_ context.Context) (*storage.Object, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	cp := compositeKey(d.b.name, prefix)
	entries := d.b.st.scan(cp)
	if len(entries) > 0 {
		return &storage.Object{
			Bucket:  d.b.name,
			Key:     d.path,
			IsDir:   true,
			Created: time.Unix(0, entries[0].created),
			Updated: time.Unix(0, entries[0].updated),
		}, nil
	}
	return nil, storage.ErrNotExist
}

func (d *dir) List(_ context.Context, limit, offset int, _ storage.Options) (storage.ObjectIter, error) {
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	cp := compositeKey(d.b.name, prefix)
	entries := d.b.st.scan(cp)

	var objs []*storage.Object
	for _, e := range entries {
		_, key := splitCompositeKey(e.key)
		rest := strings.TrimPrefix(key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, &storage.Object{
			Bucket:      d.b.name,
			Key:         key,
			Size:        int64(len(e.value)),
			ContentType: e.contentType,
			Created:     time.Unix(0, e.created),
			Updated:     time.Unix(0, e.updated),
		})
	}

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

func (d *dir) Delete(_ context.Context, opts storage.Options) error {
	recursive := false
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	cp := compositeKey(d.b.name, prefix)
	entries := d.b.st.scan(cp)
	if len(entries) == 0 {
		return storage.ErrNotExist
	}

	for _, e := range entries {
		_, key := splitCompositeKey(e.key)
		if !recursive {
			rest := strings.TrimPrefix(key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		d.b.st.put(e.key, "", nil, true)
	}
	return nil
}

func (d *dir) Move(_ context.Context, dstPath string, _ storage.Options) (storage.Directory, error) {
	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")
	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	cp := compositeKey(d.b.name, srcPrefix)
	entries := d.b.st.scan(cp)
	if len(entries) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, e := range entries {
		_, key := splitCompositeKey(e.key)
		rel := strings.TrimPrefix(key, srcPrefix)
		newKey := dstPrefix + rel

		dstCK := compositeKey(d.b.name, newKey)
		val := make([]byte, len(e.value))
		copy(val, e.value)
		d.b.st.put(dstCK, e.contentType, val, false)
		d.b.st.put(e.key, "", nil, true)
	}

	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// ---------------------------------------------------------------------------
// Multipart upload
// ---------------------------------------------------------------------------

type mpRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*mpUpload
	counter atomic.Int64
}

type mpUpload struct {
	id          string
	bucket      string
	key         string
	contentType string
	parts       map[int]*mpPart
	created     time.Time
	metadata    map[string]string
}

type mpPart struct {
	number int
	data   []byte
	etag   string
}

func (b *bucket) InitMultipart(_ context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("spider: empty key")
	}

	id := strconv.FormatInt(b.st.mp.counter.Add(1), 36)

	var metadata map[string]string
	if opts != nil {
		if m, ok := opts["metadata"].(map[string]string); ok {
			metadata = m
		}
	}

	upload := &mpUpload{
		id:          id,
		bucket:      b.name,
		key:         key,
		contentType: contentType,
		parts:       make(map[int]*mpPart),
		created:     time.Now(),
		metadata:    metadata,
	}

	b.st.mp.mu.Lock()
	if b.st.mp.uploads == nil {
		b.st.mp.uploads = make(map[string]*mpUpload)
	}
	b.st.mp.uploads[id] = upload
	b.st.mp.mu.Unlock()

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: id,
		Metadata: metadata,
	}, nil
}

func (b *bucket) UploadPart(_ context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, _ storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("spider: part number %d out of range [1, %d]", number, maxPartNumber)
	}

	b.st.mp.mu.RLock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	var data []byte
	var err error
	if size > 0 {
		data = make([]byte, size)
		n, readErr := io.ReadFull(src, data)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("spider: read part: %w", readErr)
		}
		data = data[:n]
	} else {
		data, err = io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("spider: read part: %w", err)
		}
	}

	h := md5.Sum(data)
	etag := hex.EncodeToString(h[:])

	b.st.mp.mu.Lock()
	upload.parts[number] = &mpPart{
		number: number,
		data:   data,
		etag:   etag,
	}
	b.st.mp.mu.Unlock()

	return &storage.PartInfo{
		Number: number,
		Size:   int64(len(data)),
		ETag:   etag,
	}, nil
}

func (b *bucket) CopyPart(_ context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if number < 1 || number > maxPartNumber {
		return nil, fmt.Errorf("spider: part number %d out of range", number)
	}

	b.st.mp.mu.RLock()
	_, ok := b.st.mp.uploads[mu.UploadID]
	b.st.mp.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotExist
	}

	srcBucket := mu.Bucket
	if sb, ok := opts["source_bucket"].(string); ok && sb != "" {
		srcBucket = sb
	}
	srcKey, _ := opts["source_key"].(string)
	if srcKey == "" {
		return nil, errors.New("spider: source_key required for CopyPart")
	}
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	srcCK := compositeKey(srcBucket, srcKey)
	e, found := b.st.get(srcCK)
	if !found {
		return nil, storage.ErrNotExist
	}

	data := e.value
	if srcOffset > 0 {
		if srcOffset >= int64(len(data)) {
			data = nil
		} else {
			data = data[srcOffset:]
		}
	}
	if srcLength > 0 && srcLength < int64(len(data)) {
		data = data[:srcLength]
	}

	// Copy the slice.
	cp := make([]byte, len(data))
	copy(cp, data)

	return b.UploadPart(context.Background(), mu, number, bytes.NewReader(cp), int64(len(cp)), nil)
}

func (b *bucket) ListParts(_ context.Context, mu *storage.MultipartUpload, limit, offset int, _ storage.Options) ([]*storage.PartInfo, error) {
	b.st.mp.mu.RLock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.RUnlock()
		return nil, storage.ErrNotExist
	}

	var parts []*storage.PartInfo
	for _, p := range upload.parts {
		parts = append(parts, &storage.PartInfo{
			Number: p.number,
			Size:   int64(len(p.data)),
			ETag:   p.etag,
		})
	}
	b.st.mp.mu.RUnlock()

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })

	if offset > 0 && offset < len(parts) {
		parts = parts[offset:]
	}
	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

func (b *bucket) CompleteMultipart(_ context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, _ storage.Options) (*storage.Object, error) {
	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}
	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Sort parts.
	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })

	// Verify all parts exist.
	for _, p := range parts {
		if _, ok := upload.parts[p.Number]; !ok {
			return nil, fmt.Errorf("spider: part %d not found", p.Number)
		}
	}

	// Check total assembly size.
	var totalSize int64
	for _, p := range parts {
		totalSize += int64(len(upload.parts[p.Number].data))
	}
	if totalSize > maxMultipartAssembly {
		return nil, fmt.Errorf("spider: assembled multipart size %d exceeds limit %d", totalSize, maxMultipartAssembly)
	}

	// Assemble.
	var buf bytes.Buffer
	buf.Grow(int(totalSize))
	for _, p := range parts {
		part := upload.parts[p.Number]
		buf.Write(part.data)
	}

	data := buf.Bytes()
	ck := compositeKey(b.name, upload.key)
	b.st.put(ck, upload.contentType, data, false)

	now := time.Now()
	return &storage.Object{
		Bucket:      b.name,
		Key:         upload.key,
		Size:        int64(len(data)),
		ContentType: upload.contentType,
		Created:     now,
		Updated:     now,
	}, nil
}

func (b *bucket) AbortMultipart(_ context.Context, mu *storage.MultipartUpload, _ storage.Options) error {
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
