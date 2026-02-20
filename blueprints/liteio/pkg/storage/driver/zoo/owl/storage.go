// Package owl implements a learned-index storage driver inspired by XStore (OSDI 2020).
//
// Architecture: Sorted data file + piecewise linear learned index model.
// Writes buffer in memory and periodically compact (merge-sort) into the
// sorted data file, retraining the learned model on each compaction.
// Reads predict position via the learned model and binary search within
// a bounded error range.
//
// DSN format:
//
//	owl:///path/to/data
//	owl:///path/to/data?sync=none&segments=128&buffer_size=4194304
package owl

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
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
	storage.Register("owl", &driver{})
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	defaultSegments   = 128
	defaultBufferSize = 4 * 1024 * 1024 // 4MB write buffer
	defaultSyncMode   = "none"
	maxPartNumber     = 10000

	dataFileName  = "data.dat"
	modelFileName = "model.dat"
	metaFileName  = "meta.json"
	tmpSuffix     = ".owl-tmp"

	// Binary entry layout offsets.
	// keyLen(2) | key | ctLen(2) | ct | valLen(8) | value | created(8) | updated(8) | deleted(1)
	entryDeletedByte = 1
	entryAliveByte   = 0
)

// =============================================================================
// DRIVER
// =============================================================================

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("owl: parse dsn: %w", err)
	}
	if u.Scheme != "owl" && u.Scheme != "" {
		return nil, fmt.Errorf("owl: unexpected scheme %q", u.Scheme)
	}

	root := filepath.Clean(u.Path)
	if root == "" || root == "." {
		root = "/tmp/owl-data"
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, fmt.Errorf("owl: create root %q: %w", root, err)
	}

	syncMode := u.Query().Get("sync")
	if syncMode == "" {
		syncMode = defaultSyncMode
	}

	segments := defaultSegments
	if s := u.Query().Get("segments"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			segments = n
		}
	}

	bufferSize := int64(defaultBufferSize)
	if bs := u.Query().Get("buffer_size"); bs != "" {
		if n, err := strconv.ParseInt(bs, 10, 64); err == nil && n > 0 {
			bufferSize = n
		}
	}

	st := &store{
		root:          root,
		syncMode:      syncMode,
		targetSegs:    segments,
		maxBufferSize: bufferSize,
		buckets:       make(map[string]time.Time),
		writeBuf:      make(map[string]*bufEntry),
		mp:            newMultipartRegistry(),
		stopTick:      make(chan struct{}),
	}

	// Load existing data if present.
	if err := st.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("owl: load: %w", err)
	}

	// Start per-store ticker for cached time.
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-st.stopTick:
				return
			case <-ticker.C:
				cachedTimeNano.Store(time.Now().UnixNano())
			}
		}
	}()

	return st, nil
}

// =============================================================================
// STORE
// =============================================================================

const maxBuckets = 10000

// store implements storage.Storage.
type store struct {
	root          string
	syncMode      string
	targetSegs    int
	maxBufferSize int64

	// Bucket registry.
	bmu     sync.RWMutex
	buckets map[string]time.Time

	// Write buffer: compositeKey -> bufEntry.
	wmu        sync.RWMutex
	writeBuf   map[string]*bufEntry
	bufBytes   int64
	compactMu  sync.Mutex     // serialises compaction
	compacting atomic.Bool    // true while a compaction goroutine is running
	compactCh  chan struct{}   // closed when current compaction finishes (for backpressure)

	// Background compaction tracking.
	closed    int32          // atomic: 1 = store is closing/closed
	compactWg sync.WaitGroup // tracks in-flight background compactions

	// Per-store stoppable ticker for cached time.
	stopTick chan struct{}

	// Sorted data (loaded from data.dat or rebuilt on compaction).
	dmu       sync.RWMutex
	sortedArr []sortedEntry // sorted by compositeKey
	keyHashes []uint64      // parallel array of fnv hashes for sortedArr
	model     learnedModel

	// Multipart uploads.
	mp *multipartRegistry
}

var _ storage.Storage = (*store)(nil)

// bufEntry holds an uncommitted write.
type bufEntry struct {
	value       []byte
	contentType string
	created     int64 // unix nano
	updated     int64 // unix nano
	deleted     bool
	size        int64
}

// sortedEntry is one row in the sorted data array.
type sortedEntry struct {
	compositeKey string
	value        []byte
	contentType  string
	created      int64
	updated      int64
}

// =============================================================================
// LEARNED INDEX MODEL
// =============================================================================

type segment struct {
	minKeyHash uint64
	slope      float64
	intercept  float64
	maxError   int
}

type learnedModel struct {
	segments []segment
}

// predict returns (predictedPos, lo, hi) for binary search.
func (m *learnedModel) predict(keyHash uint64, arrayLen int) (int, int, int) {
	if len(m.segments) == 0 || arrayLen == 0 {
		return 0, 0, arrayLen - 1
	}

	// Find segment via binary search on minKeyHash.
	segIdx := sort.Search(len(m.segments), func(i int) bool {
		return m.segments[i].minKeyHash > keyHash
	}) - 1
	if segIdx < 0 {
		segIdx = 0
	}

	seg := m.segments[segIdx]
	predicted := int(seg.slope*float64(keyHash) + seg.intercept)

	// Clamp to valid range.
	lo := predicted - seg.maxError
	hi := predicted + seg.maxError
	if lo < 0 {
		lo = 0
	}
	if hi >= arrayLen {
		hi = arrayLen - 1
	}
	if predicted < 0 {
		predicted = 0
	}
	if predicted >= arrayLen {
		predicted = arrayLen - 1
	}

	return predicted, lo, hi
}

// train builds piecewise linear segments from sorted key hashes.
// Uses a greedy algorithm: extend current segment until error exceeds bound,
// then start a new segment.
func (m *learnedModel) train(hashes []uint64, targetSegments int) {
	n := len(hashes)
	if n == 0 {
		m.segments = nil
		return
	}

	if targetSegments <= 0 {
		targetSegments = defaultSegments
	}

	// Target keys per segment.
	keysPerSeg := n / targetSegments
	if keysPerSeg < 2 {
		keysPerSeg = 2
	}

	var segs []segment
	i := 0
	for i < n {
		// Determine segment range.
		end := i + keysPerSeg
		if end > n {
			end = n
		}
		// Ensure at least 2 points for regression, or use the rest.
		if n-i <= keysPerSeg {
			end = n
		}

		seg := fitSegment(hashes, i, end)
		segs = append(segs, seg)
		i = end
	}

	m.segments = segs
}

// fitSegment fits a linear regression on hashes[start:end] predicting
// position indices [start, end).
func fitSegment(hashes []uint64, start, end int) segment {
	count := end - start
	if count <= 0 {
		return segment{}
	}
	if count == 1 {
		return segment{
			minKeyHash: hashes[start],
			slope:      0,
			intercept:  float64(start),
			maxError:   0,
		}
	}

	// Least-squares linear regression: position = slope * hash + intercept.
	// Use float64 for all intermediate calculations.
	var sumX, sumY, sumXY, sumXX float64
	for j := start; j < end; j++ {
		x := float64(hashes[j])
		y := float64(j)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	n := float64(count)
	denom := n*sumXX - sumX*sumX
	var slope, intercept float64
	if math.Abs(denom) < 1e-12 {
		// All keys hash to the same value; flat model.
		slope = 0
		intercept = float64(start+end-1) / 2.0
	} else {
		slope = (n*sumXY - sumX*sumY) / denom
		intercept = (sumY - slope*sumX) / n
	}

	// Compute max error.
	maxErr := 0
	for j := start; j < end; j++ {
		predicted := int(slope*float64(hashes[j]) + intercept)
		e := predicted - j
		if e < 0 {
			e = -e
		}
		if e > maxErr {
			maxErr = e
		}
	}

	return segment{
		minKeyHash: hashes[start],
		slope:      slope,
		intercept:  intercept,
		maxError:   maxErr + 1, // +1 safety margin
	}
}

// =============================================================================
// PERSISTENCE: LOAD / SAVE
// =============================================================================

// metadata stored in meta.json.
type metadata struct {
	EntryCount   int `json:"entry_count"`
	SegmentCount int `json:"segment_count"`
}

func (s *store) dataPath() string  { return filepath.Join(s.root, dataFileName) }
func (s *store) modelPath() string { return filepath.Join(s.root, modelFileName) }
func (s *store) metaPath() string  { return filepath.Join(s.root, metaFileName) }

// loadFromDisk loads sorted data, model, and metadata from disk.
func (s *store) loadFromDisk() error {
	metaPath := s.metaPath()
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil // fresh store
	}

	// Load metadata.
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read meta: %w", err)
	}
	var meta metadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return fmt.Errorf("parse meta: %w", err)
	}

	// Load data file.
	dataBytes, err := os.ReadFile(s.dataPath())
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}
	entries, err := decodeDataFile(dataBytes)
	if err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	// Build sorted array and key hashes.
	hashes := make([]uint64, len(entries))
	for i, e := range entries {
		hashes[i] = hashKey(e.compositeKey)
	}

	// Load model file.
	model, err := loadModel(s.modelPath())
	if err != nil {
		// Retrain if model file is missing or corrupt.
		model.train(hashes, s.targetSegs)
	}

	// Discover buckets from composite keys.
	for _, e := range entries {
		bucket, _ := splitCompositeKey(e.compositeKey)
		if bucket != "" {
			if _, ok := s.buckets[bucket]; !ok {
				s.buckets[bucket] = time.Unix(0, e.created)
			}
		}
	}

	s.dmu.Lock()
	s.sortedArr = entries
	s.keyHashes = hashes
	s.model = model
	s.dmu.Unlock()

	return nil
}

// saveToDisk writes the sorted array, model, and metadata atomically.
func (s *store) saveToDisk(entries []sortedEntry, hashes []uint64, model *learnedModel) error {
	// Write data file.
	dataBuf := encodeDataFile(entries)
	tmpData := s.dataPath() + tmpSuffix
	if err := writeFileAtomic(tmpData, s.dataPath(), dataBuf, s.syncMode); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// Write model file.
	if err := saveModel(s.modelPath(), model); err != nil {
		return fmt.Errorf("write model: %w", err)
	}

	// Write metadata.
	meta := metadata{
		EntryCount:   len(entries),
		SegmentCount: len(model.segments),
	}
	metaBytes, _ := json.Marshal(meta)
	tmpMeta := s.metaPath() + tmpSuffix
	if err := writeFileAtomic(tmpMeta, s.metaPath(), metaBytes, s.syncMode); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	return nil
}

func writeFileAtomic(tmpPath, finalPath string, data []byte, syncMode string) error {
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	// Always clean up the temp file on failure (including failed rename).
	defer os.Remove(tmpPath)

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if syncMode != "none" {
		if err := f.Sync(); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, finalPath)
}

// =============================================================================
// DATA FILE ENCODING / DECODING
// =============================================================================

// encodeDataFile serialises sorted entries into binary:
// For each entry: keyLen(2) | key | ctLen(2) | ct | valLen(8) | value | created(8) | updated(8) | deleted(1)
func encodeDataFile(entries []sortedEntry) []byte {
	// Pre-compute total size.
	total := 0
	for _, e := range entries {
		total += 2 + len(e.compositeKey) + 2 + len(e.contentType) + 8 + len(e.value) + 8 + 8 + 1
	}

	buf := make([]byte, 0, total)
	tmp := make([]byte, 8)

	for _, e := range entries {
		// keyLen + key
		binary.LittleEndian.PutUint16(tmp[:2], uint16(len(e.compositeKey)))
		buf = append(buf, tmp[:2]...)
		buf = append(buf, e.compositeKey...)

		// ctLen + ct
		binary.LittleEndian.PutUint16(tmp[:2], uint16(len(e.contentType)))
		buf = append(buf, tmp[:2]...)
		buf = append(buf, e.contentType...)

		// valLen + value
		binary.LittleEndian.PutUint64(tmp[:8], uint64(len(e.value)))
		buf = append(buf, tmp[:8]...)
		buf = append(buf, e.value...)

		// created
		binary.LittleEndian.PutUint64(tmp[:8], uint64(e.created))
		buf = append(buf, tmp[:8]...)

		// updated
		binary.LittleEndian.PutUint64(tmp[:8], uint64(e.updated))
		buf = append(buf, tmp[:8]...)

		// deleted (alive)
		buf = append(buf, entryAliveByte)
	}

	return buf
}

func decodeDataFile(data []byte) ([]sortedEntry, error) {
	var entries []sortedEntry
	off := 0

	for off < len(data) {
		if off+2 > len(data) {
			return entries, fmt.Errorf("truncated at key len, offset %d", off)
		}
		keyLen := int(binary.LittleEndian.Uint16(data[off:]))
		off += 2

		if off+keyLen > len(data) {
			return entries, fmt.Errorf("truncated at key, offset %d", off)
		}
		key := string(data[off : off+keyLen])
		off += keyLen

		if off+2 > len(data) {
			return entries, fmt.Errorf("truncated at ct len, offset %d", off)
		}
		ctLen := int(binary.LittleEndian.Uint16(data[off:]))
		off += 2

		if off+ctLen > len(data) {
			return entries, fmt.Errorf("truncated at ct, offset %d", off)
		}
		ct := string(data[off : off+ctLen])
		off += ctLen

		if off+8 > len(data) {
			return entries, fmt.Errorf("truncated at val len, offset %d", off)
		}
		valLen := int(binary.LittleEndian.Uint64(data[off:]))
		off += 8

		if off+valLen > len(data) {
			return entries, fmt.Errorf("truncated at val, offset %d", off)
		}
		val := make([]byte, valLen)
		copy(val, data[off:off+valLen])
		off += valLen

		if off+17 > len(data) {
			return entries, fmt.Errorf("truncated at timestamps, offset %d", off)
		}
		created := int64(binary.LittleEndian.Uint64(data[off:]))
		off += 8
		updated := int64(binary.LittleEndian.Uint64(data[off:]))
		off += 8
		deleted := data[off]
		off++

		if deleted == entryDeletedByte {
			continue // skip deleted
		}

		entries = append(entries, sortedEntry{
			compositeKey: key,
			value:        val,
			contentType:  ct,
			created:      created,
			updated:      updated,
		})
	}

	return entries, nil
}

// =============================================================================
// MODEL FILE ENCODING / DECODING
// =============================================================================

func saveModel(path string, m *learnedModel) error {
	// Each segment: minKeyHash(8) + slope(8) + intercept(8) + maxError(4) = 28 bytes.
	buf := make([]byte, len(m.segments)*28)
	for i, seg := range m.segments {
		off := i * 28
		binary.LittleEndian.PutUint64(buf[off:], seg.minKeyHash)
		binary.LittleEndian.PutUint64(buf[off+8:], math.Float64bits(seg.slope))
		binary.LittleEndian.PutUint64(buf[off+16:], math.Float64bits(seg.intercept))
		binary.LittleEndian.PutUint32(buf[off+24:], uint32(seg.maxError))
	}
	tmpPath := path + tmpSuffix
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(buf); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	f.Close()
	return os.Rename(tmpPath, path)
}

func loadModel(path string) (learnedModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return learnedModel{}, err
	}

	if len(data)%28 != 0 {
		return learnedModel{}, fmt.Errorf("model file size %d not multiple of 28", len(data))
	}

	n := len(data) / 28
	segs := make([]segment, n)
	for i := 0; i < n; i++ {
		off := i * 28
		segs[i] = segment{
			minKeyHash: binary.LittleEndian.Uint64(data[off:]),
			slope:      math.Float64frombits(binary.LittleEndian.Uint64(data[off+8:])),
			intercept:  math.Float64frombits(binary.LittleEndian.Uint64(data[off+16:])),
			maxError:   int(binary.LittleEndian.Uint32(data[off+24:])),
		}
	}

	return learnedModel{segments: segs}, nil
}

// =============================================================================
// KEY HELPERS
// =============================================================================

func compositeKey(bucket, key string) string {
	return bucket + "\x00" + key
}

func splitCompositeKey(ck string) (bucket, key string) {
	idx := strings.IndexByte(ck, 0)
	if idx < 0 {
		return "", ck
	}
	return ck[:idx], ck[idx+1:]
}

func hashKey(key string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	return h.Sum64()
}

// =============================================================================
// COMPACTION
// =============================================================================

// compact merges the write buffer into the sorted array, retrains the
// model, and persists everything to disk.
func (s *store) compact() error {
	s.compactMu.Lock()
	defer s.compactMu.Unlock()

	// Snapshot and clear write buffer.
	s.wmu.Lock()
	buf := s.writeBuf
	s.writeBuf = make(map[string]*bufEntry)
	s.bufBytes = 0
	s.wmu.Unlock()

	if len(buf) == 0 {
		return nil
	}

	// Read current sorted array.
	s.dmu.RLock()
	existing := s.sortedArr
	s.dmu.RUnlock()

	// Build sorted list of buffer entries.
	type kv struct {
		key string
		e   *bufEntry
	}
	bufEntries := make([]kv, 0, len(buf))
	for k, e := range buf {
		bufEntries = append(bufEntries, kv{key: k, e: e})
	}
	sort.Slice(bufEntries, func(i, j int) bool {
		return bufEntries[i].key < bufEntries[j].key
	})

	// Merge-sort existing + buffer.
	merged := make([]sortedEntry, 0, len(existing)+len(bufEntries))
	ei, bi := 0, 0

	for ei < len(existing) && bi < len(bufEntries) {
		ek := existing[ei].compositeKey
		bk := bufEntries[bi].key

		if ek < bk {
			merged = append(merged, existing[ei])
			ei++
		} else if ek > bk {
			be := bufEntries[bi].e
			if !be.deleted {
				merged = append(merged, sortedEntry{
					compositeKey: bk,
					value:        be.value,
					contentType:  be.contentType,
					created:      be.created,
					updated:      be.updated,
				})
			}
			bi++
		} else {
			// Buffer entry overrides existing.
			be := bufEntries[bi].e
			if !be.deleted {
				merged = append(merged, sortedEntry{
					compositeKey: bk,
					value:        be.value,
					contentType:  be.contentType,
					created:      existing[ei].created, // preserve original created
					updated:      be.updated,
				})
			}
			ei++
			bi++
		}
	}
	for ; ei < len(existing); ei++ {
		merged = append(merged, existing[ei])
	}
	for ; bi < len(bufEntries); bi++ {
		be := bufEntries[bi].e
		if !be.deleted {
			merged = append(merged, sortedEntry{
				compositeKey: bufEntries[bi].key,
				value:        be.value,
				contentType:  be.contentType,
				created:      be.created,
				updated:      be.updated,
			})
		}
	}

	// Build hash array and train model.
	hashes := make([]uint64, len(merged))
	for i, e := range merged {
		hashes[i] = hashKey(e.compositeKey)
	}

	var model learnedModel
	model.train(hashes, s.targetSegs)

	// Persist to disk.
	if err := s.saveToDisk(merged, hashes, &model); err != nil {
		return err
	}

	// Update in-memory state.
	s.dmu.Lock()
	s.sortedArr = merged
	s.keyHashes = hashes
	s.model = model
	s.dmu.Unlock()

	return nil
}

// errWriteBufferFull is returned when write buffer exceeds 2x threshold
// and a compaction is already in progress.
var errWriteBufferFull = fmt.Errorf("owl: write buffer full, try again")

// maybeCompact triggers compaction if buffer exceeds threshold.
// Returns an error if write buffer is dangerously large and compaction
// is already in progress (backpressure).
func (s *store) maybeCompact() error {
	s.wmu.RLock()
	currentBytes := s.bufBytes
	s.wmu.RUnlock()

	if currentBytes < s.maxBufferSize {
		return nil
	}

	// Backpressure: if buffer exceeds 2x threshold and compaction is
	// already running, reject writes to prevent OOM.
	if currentBytes >= 2*s.maxBufferSize && s.compacting.Load() {
		return errWriteBufferFull
	}

	// Try to start a new compaction. Use CompareAndSwap so only one
	// goroutine runs compaction at a time.
	if s.compacting.CompareAndSwap(false, true) {
		// Re-check closed inside the lock to avoid TOCTOU race.
		if atomic.LoadInt32(&s.closed) != 0 {
			s.compacting.Store(false)
			return nil
		}

		ch := make(chan struct{})
		s.compactMu.Lock()
		s.compactCh = ch
		s.compactMu.Unlock()

		s.compactWg.Add(1)
		go func() {
			defer s.compactWg.Done()
			defer s.compacting.Store(false)
			defer close(ch)
			s.compact()
		}()
	}

	return nil
}

// =============================================================================
// LOOKUP HELPERS
// =============================================================================

// lookupSorted finds a composite key in the sorted array using the learned model.
// Returns the index and true if found, or -1 and false.
func (s *store) lookupSorted(ck string) (int, bool) {
	s.dmu.RLock()
	defer s.dmu.RUnlock()

	if len(s.sortedArr) == 0 {
		return -1, false
	}

	h := hashKey(ck)
	_, lo, hi := s.model.predict(h, len(s.sortedArr))

	// Bounded binary search.
	idx := sort.Search(hi-lo+1, func(i int) bool {
		return s.sortedArr[lo+i].compositeKey >= ck
	})
	idx += lo

	if idx < len(s.sortedArr) && s.sortedArr[idx].compositeKey == ck {
		return idx, true
	}

	return -1, false
}

// listSorted returns all entries with composite keys that have the given prefix.
func (s *store) listSorted(prefix string) []sortedEntry {
	s.dmu.RLock()
	defer s.dmu.RUnlock()

	if len(s.sortedArr) == 0 {
		return nil
	}

	// Binary search for the first key >= prefix.
	start := sort.Search(len(s.sortedArr), func(i int) bool {
		return s.sortedArr[i].compositeKey >= prefix
	})

	var result []sortedEntry
	for i := start; i < len(s.sortedArr); i++ {
		if !strings.HasPrefix(s.sortedArr[i].compositeKey, prefix) {
			break
		}
		result = append(result, s.sortedArr[i])
	}

	return result
}

// =============================================================================
// STORAGE INTERFACE
// =============================================================================

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = "default"
	}

	s.bmu.Lock()
	if _, ok := s.buckets[name]; !ok {
		if len(s.buckets) < maxBuckets {
			s.buckets[name] = fastNowTime()
		}
	}
	s.bmu.Unlock()

	return &bucket{st: s, name: name}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	_ = ctx
	_ = opts

	s.bmu.RLock()
	names := make([]string, 0, len(s.buckets))
	for name := range s.buckets {
		names = append(names, name)
	}
	s.bmu.RUnlock()

	sort.Strings(names)

	s.bmu.RLock()
	infos := make([]*storage.BucketInfo, 0, len(names))
	for _, name := range names {
		infos = append(infos, &storage.BucketInfo{
			Name:      name,
			CreatedAt: s.buckets[name],
		})
	}
	s.bmu.RUnlock()

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

	return &bucketIter{buckets: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	_ = ctx
	_ = opts

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("owl: bucket name is empty")
	}

	s.bmu.Lock()
	defer s.bmu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return nil, storage.ErrExist
	}

	if len(s.buckets) >= maxBuckets {
		return nil, fmt.Errorf("owl: maximum number of buckets (%d) reached", maxBuckets)
	}

	now := fastNowTime()
	s.buckets[name] = now

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: now,
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	_ = ctx

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("owl: bucket name is empty")
	}

	force := false
	if opts != nil {
		if v, ok := opts["force"].(bool); ok {
			force = v
		}
	}

	s.bmu.Lock()
	defer s.bmu.Unlock()

	if _, ok := s.buckets[name]; !ok {
		return storage.ErrNotExist
	}

	if !force {
		// Check if bucket has any entries.
		prefix := name + "\x00"
		s.wmu.RLock()
		hasBufEntries := false
		for k, e := range s.writeBuf {
			if strings.HasPrefix(k, prefix) && !e.deleted {
				hasBufEntries = true
				break
			}
		}
		s.wmu.RUnlock()

		if !hasBufEntries {
			sorted := s.listSorted(prefix)
			if len(sorted) > 0 {
				hasBufEntries = true
			}
		}

		if hasBufEntries {
			return storage.ErrPermission
		}
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
	// Signal that the store is closing so no new background compactions start.
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	// Stop the per-store ticker goroutine.
	close(s.stopTick)

	// Wait for any in-flight background compactions to finish.
	s.compactWg.Wait()

	// Final flush of any remaining write buffer to disk.
	return s.compact()
}

// =============================================================================
// BUCKET
// =============================================================================

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
	_ = ctx

	b.st.bmu.RLock()
	created, ok := b.st.buckets[b.name]
	b.st.bmu.RUnlock()

	if !ok {
		return nil, storage.ErrNotExist
	}

	return &storage.BucketInfo{
		Name:      b.name,
		CreatedAt: created,
	}, nil
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("owl: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("owl: key is empty")
		}
	}

	// Read value.
	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		if size > 0 {
			n, err := io.ReadFull(src, data)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("owl: read value: %w", err)
			}
			data = data[:n]
		}
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("owl: read value: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNow()
	ck := compositeKey(b.name, key)

	// Check for existing entry to preserve created timestamp.
	created := now
	b.st.wmu.RLock()
	if existing, ok := b.st.writeBuf[ck]; ok && !existing.deleted {
		created = existing.created
	}
	b.st.wmu.RUnlock()
	if created == now {
		// Check sorted array.
		if idx, ok := b.st.lookupSorted(ck); ok {
			b.st.dmu.RLock()
			created = b.st.sortedArr[idx].created
			b.st.dmu.RUnlock()
		}
	}

	entry := &bufEntry{
		value:       data,
		contentType: contentType,
		created:     created,
		updated:     now,
		deleted:     false,
		size:        int64(len(data)),
	}

	b.st.wmu.Lock()
	old, hadOld := b.st.writeBuf[ck]
	b.st.writeBuf[ck] = entry
	if hadOld {
		b.st.bufBytes -= old.size
	}
	b.st.bufBytes += entry.size
	b.st.wmu.Unlock()

	if err := b.st.maybeCompact(); err != nil {
		return nil, err
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		Created:     time.Unix(0, created),
		Updated:     time.Unix(0, now),
	}, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, nil, fmt.Errorf("owl: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("owl: key is empty")
		}
	}

	ck := compositeKey(b.name, key)

	// Check write buffer first.
	b.st.wmu.RLock()
	if entry, ok := b.st.writeBuf[ck]; ok {
		b.st.wmu.RUnlock()
		if entry.deleted {
			return nil, nil, storage.ErrNotExist
		}

		data := entry.value
		obj := &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(data)),
			ContentType: entry.contentType,
			Created:     time.Unix(0, entry.created),
			Updated:     time.Unix(0, entry.updated),
		}

		slice := applyRange(data, offset, length)
		return io.NopCloser(bytes.NewReader(slice)), obj, nil
	}
	b.st.wmu.RUnlock()

	// Check sorted array via learned model.
	idx, found := b.st.lookupSorted(ck)
	if !found {
		return nil, nil, storage.ErrNotExist
	}

	b.st.dmu.RLock()
	entry := b.st.sortedArr[idx]
	b.st.dmu.RUnlock()

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(entry.value)),
		ContentType: entry.contentType,
		Created:     time.Unix(0, entry.created),
		Updated:     time.Unix(0, entry.updated),
	}

	slice := applyRange(entry.value, offset, length)
	return io.NopCloser(bytes.NewReader(slice)), obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	if key == "" {
		return nil, fmt.Errorf("owl: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("owl: key is empty")
		}
	}

	// Directory stat.
	if strings.HasSuffix(key, "/") {
		objs := b.listAll(key)
		if len(objs) == 0 {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:  b.name,
			Key:     strings.TrimSuffix(key, "/"),
			IsDir:   true,
			Created: objs[0].Created,
			Updated: objs[0].Updated,
		}, nil
	}

	ck := compositeKey(b.name, key)

	// Check write buffer.
	b.st.wmu.RLock()
	if entry, ok := b.st.writeBuf[ck]; ok {
		b.st.wmu.RUnlock()
		if entry.deleted {
			return nil, storage.ErrNotExist
		}
		return &storage.Object{
			Bucket:      b.name,
			Key:         key,
			Size:        int64(len(entry.value)),
			ContentType: entry.contentType,
			Created:     time.Unix(0, entry.created),
			Updated:     time.Unix(0, entry.updated),
		}, nil
	}
	b.st.wmu.RUnlock()

	// Check sorted array.
	idx, found := b.st.lookupSorted(ck)
	if !found {
		return nil, storage.ErrNotExist
	}

	b.st.dmu.RLock()
	entry := b.st.sortedArr[idx]
	b.st.dmu.RUnlock()

	return &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        int64(len(entry.value)),
		ContentType: entry.contentType,
		Created:     time.Unix(0, entry.created),
		Updated:     time.Unix(0, entry.updated),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	_ = ctx
	_ = opts

	if key == "" {
		return fmt.Errorf("owl: key is empty")
	}
	if key[0] == ' ' || key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("owl: key is empty")
		}
	}

	ck := compositeKey(b.name, key)

	// Check existence.
	exists := false
	b.st.wmu.RLock()
	if entry, ok := b.st.writeBuf[ck]; ok && !entry.deleted {
		exists = true
	}
	b.st.wmu.RUnlock()

	if !exists {
		if _, found := b.st.lookupSorted(ck); found {
			exists = true
		}
	}

	if !exists {
		return storage.ErrNotExist
	}

	now := fastNow()
	entry := &bufEntry{
		deleted: true,
		updated: now,
		created: now,
	}

	b.st.wmu.Lock()
	old, hadOld := b.st.writeBuf[ck]
	b.st.writeBuf[ck] = entry
	if hadOld {
		b.st.bufBytes -= old.size
	}
	b.st.wmu.Unlock()

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	_ = ctx
	_ = opts

	dstKey = strings.TrimSpace(dstKey)
	srcKey = strings.TrimSpace(srcKey)
	if dstKey == "" || srcKey == "" {
		return nil, fmt.Errorf("owl: key is empty")
	}
	if srcBucket == "" {
		srcBucket = b.name
	}

	srcCK := compositeKey(srcBucket, srcKey)

	// Find source data.
	var srcData []byte
	var srcCT string
	var srcCreated int64

	b.st.wmu.RLock()
	if entry, ok := b.st.writeBuf[srcCK]; ok && !entry.deleted {
		srcData = make([]byte, len(entry.value))
		copy(srcData, entry.value)
		srcCT = entry.contentType
		srcCreated = entry.created
		b.st.wmu.RUnlock()
	} else {
		b.st.wmu.RUnlock()

		idx, found := b.st.lookupSorted(srcCK)
		if !found {
			return nil, storage.ErrNotExist
		}
		b.st.dmu.RLock()
		e := b.st.sortedArr[idx]
		srcData = make([]byte, len(e.value))
		copy(srcData, e.value)
		srcCT = e.contentType
		srcCreated = e.created
		b.st.dmu.RUnlock()
	}

	now := fastNow()
	dstCK := compositeKey(b.name, dstKey)

	entry := &bufEntry{
		value:       srcData,
		contentType: srcCT,
		created:     now,
		updated:     now,
		size:        int64(len(srcData)),
	}
	_ = srcCreated

	b.st.wmu.Lock()
	old, hadOld := b.st.writeBuf[dstCK]
	b.st.writeBuf[dstCK] = entry
	if hadOld {
		b.st.bufBytes -= old.size
	}
	b.st.bufBytes += entry.size
	b.st.wmu.Unlock()

	if err := b.st.maybeCompact(); err != nil {
		return nil, err
	}

	return &storage.Object{
		Bucket:      b.name,
		Key:         dstKey,
		Size:        int64(len(srcData)),
		ContentType: srcCT,
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
	if err := sb.Delete(ctx, srcKey, nil); err != nil {
		return nil, err
	}
	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx

	prefix = strings.TrimSpace(prefix)
	recursive := true
	if opts != nil {
		if v, ok := opts["recursive"].(bool); ok {
			recursive = v
		}
	}

	objs := b.listAll(prefix)

	if !recursive {
		var filtered []*storage.Object
		for _, o := range objs {
			rest := strings.TrimPrefix(o.Key, prefix)
			rest = strings.TrimPrefix(rest, "/")
			if !strings.Contains(rest, "/") {
				filtered = append(filtered, o)
			}
		}
		objs = filtered
	}

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

	return &objectIter{objects: objs}, nil
}

// listAll returns all objects matching a given key prefix, merging
// write buffer with sorted array, sorted by key.
func (b *bucket) listAll(prefix string) []*storage.Object {
	ckPrefix := compositeKey(b.name, prefix)

	// Collect from sorted array.
	sortedEntries := b.st.listSorted(ckPrefix)

	// Collect from write buffer.
	b.st.wmu.RLock()
	bufSnapshot := make(map[string]*bufEntry)
	for k, e := range b.st.writeBuf {
		if strings.HasPrefix(k, ckPrefix) {
			bufSnapshot[k] = e
		}
	}
	b.st.wmu.RUnlock()

	// Merge: buffer overrides sorted.
	seen := make(map[string]struct{})
	var result []*storage.Object

	// Buffer entries first (they override).
	for k, e := range bufSnapshot {
		seen[k] = struct{}{}
		if e.deleted {
			continue
		}
		_, objKey := splitCompositeKey(k)
		result = append(result, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        int64(len(e.value)),
			ContentType: e.contentType,
			Created:     time.Unix(0, e.created),
			Updated:     time.Unix(0, e.updated),
		})
	}

	// Sorted entries (only if not overridden).
	for _, e := range sortedEntries {
		if _, ok := seen[e.compositeKey]; ok {
			continue
		}
		_, objKey := splitCompositeKey(e.compositeKey)
		result = append(result, &storage.Object{
			Bucket:      b.name,
			Key:         objKey,
			Size:        int64(len(e.value)),
			ContentType: e.contentType,
			Created:     time.Unix(0, e.created),
			Updated:     time.Unix(0, e.updated),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})

	return result
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	return "", storage.ErrUnsupported
}

// =============================================================================
// DIRECTORY SUPPORT
// =============================================================================

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
	_ = ctx
	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	objs := d.b.listAll(prefix)
	if len(objs) == 0 {
		return nil, storage.ErrNotExist
	}
	return &storage.Object{
		Bucket:  d.b.name,
		Key:     d.path,
		IsDir:   true,
		Created: objs[0].Created,
		Updated: objs[0].Updated,
	}, nil
}

func (d *dir) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	_ = ctx
	_ = opts

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	all := d.b.listAll(prefix)

	// Filter to direct children only.
	var objs []*storage.Object
	for _, o := range all {
		rest := strings.TrimPrefix(o.Key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		objs = append(objs, o)
	}

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

	return &objectIter{objects: objs}, nil
}

func (d *dir) Delete(ctx context.Context, opts storage.Options) error {
	_ = ctx

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

	objs := d.b.listAll(prefix)
	if len(objs) == 0 {
		return storage.ErrNotExist
	}

	for _, o := range objs {
		if !recursive {
			rest := strings.TrimPrefix(o.Key, prefix)
			if strings.Contains(rest, "/") {
				continue
			}
		}
		ck := compositeKey(d.b.name, o.Key)
		now := fastNow()
		d.b.st.wmu.Lock()
		old, hadOld := d.b.st.writeBuf[ck]
		d.b.st.writeBuf[ck] = &bufEntry{
			deleted: true,
			updated: now,
			created: now,
		}
		if hadOld {
			d.b.st.bufBytes -= old.size
		}
		d.b.st.wmu.Unlock()
	}
	return nil
}

func (d *dir) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	_ = ctx
	_ = opts

	srcPrefix := strings.Trim(d.path, "/")
	dstPrefix := strings.Trim(dstPath, "/")

	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	objs := d.b.listAll(srcPrefix)
	if len(objs) == 0 {
		return nil, storage.ErrNotExist
	}

	for _, o := range objs {
		rel := strings.TrimPrefix(o.Key, srcPrefix)
		newKey := dstPrefix + rel

		// Copy data to new key.
		srcCK := compositeKey(d.b.name, o.Key)
		dstCK := compositeKey(d.b.name, newKey)

		var data []byte
		var ct string
		var created, updated int64

		d.b.st.wmu.RLock()
		if entry, ok := d.b.st.writeBuf[srcCK]; ok && !entry.deleted {
			data = make([]byte, len(entry.value))
			copy(data, entry.value)
			ct = entry.contentType
			created = entry.created
			updated = entry.updated
		}
		d.b.st.wmu.RUnlock()

		if data == nil {
			if idx, found := d.b.st.lookupSorted(srcCK); found {
				d.b.st.dmu.RLock()
				e := d.b.st.sortedArr[idx]
				data = make([]byte, len(e.value))
				copy(data, e.value)
				ct = e.contentType
				created = e.created
				updated = e.updated
				d.b.st.dmu.RUnlock()
			}
		}

		if data != nil {
			now := fastNow()
			d.b.st.wmu.Lock()
			// Write new key.
			oldDst, hadOldDst := d.b.st.writeBuf[dstCK]
			d.b.st.writeBuf[dstCK] = &bufEntry{
				value:       data,
				contentType: ct,
				created:     created,
				updated:     updated,
				size:        int64(len(data)),
			}
			if hadOldDst {
				d.b.st.bufBytes -= oldDst.size
			}
			d.b.st.bufBytes += int64(len(data))

			// Delete old key.
			oldSrc, hadOldSrc := d.b.st.writeBuf[srcCK]
			d.b.st.writeBuf[srcCK] = &bufEntry{
				deleted: true,
				updated: now,
				created: now,
			}
			if hadOldSrc {
				d.b.st.bufBytes -= oldSrc.size
			}
			d.b.st.wmu.Unlock()
		}
	}

	if err := d.b.st.maybeCompact(); err != nil {
		return nil, err
	}
	return &dir{b: d.b, path: strings.Trim(dstPath, "/")}, nil
}

// =============================================================================
// MULTIPART UPLOAD
// =============================================================================

type multipartUpload struct {
	mu          *storage.MultipartUpload
	contentType string
	createdAt   time.Time
	parts       map[int]*partData
}

type partData struct {
	number       int
	data         []byte
	etag         string
	lastModified time.Time
}

type multipartRegistry struct {
	mu      sync.RWMutex
	uploads map[string]*multipartUpload
}

func newMultipartRegistry() *multipartRegistry {
	return &multipartRegistry{
		uploads: make(map[string]*multipartUpload),
	}
}

func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	_ = ctx
	_ = opts

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("owl: key is empty")
	}

	uploadID := newUploadID()

	mu := &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: uploadID,
	}

	b.st.mp.mu.Lock()
	b.st.mp.uploads[uploadID] = &multipartUpload{
		mu:          mu,
		contentType: contentType,
		createdAt:   fastNowTime(),
		parts:       make(map[int]*partData),
	}
	b.st.mp.mu.Unlock()

	return mu, nil
}

func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	if number <= 0 || number > maxPartNumber {
		return nil, fmt.Errorf("owl: part number %d out of range (1-%d)", number, maxPartNumber)
	}

	var data []byte
	if size >= 0 {
		data = make([]byte, size)
		n, err := io.ReadFull(src, data)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("owl: read part: %w", err)
		}
		data = data[:n]
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, src); err != nil {
			return nil, fmt.Errorf("owl: read part: %w", err)
		}
		data = buf.Bytes()
	}

	now := fastNowTime()
	sum := md5.Sum(data)
	etag := hex.EncodeToString(sum[:])

	pd := &partData{
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
	_ = ctx
	_ = mu
	_ = number
	_ = opts
	return nil, storage.ErrUnsupported
}

func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	_ = ctx
	_ = opts

	b.st.mp.mu.RLock()
	defer b.st.mp.mu.RUnlock()

	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		return nil, storage.ErrNotExist
	}

	parts := make([]*storage.PartInfo, 0, len(upload.parts))
	for _, pd := range upload.parts {
		lastMod := pd.lastModified
		parts = append(parts, &storage.PartInfo{
			Number:       pd.number,
			Size:         int64(len(pd.data)),
			ETag:         pd.etag,
			LastModified: &lastMod,
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
	_ = ctx
	_ = opts

	if len(parts) == 0 {
		return nil, fmt.Errorf("owl: no parts to complete")
	}

	b.st.mp.mu.Lock()
	upload, ok := b.st.mp.uploads[mu.UploadID]
	if !ok {
		b.st.mp.mu.Unlock()
		return nil, storage.ErrNotExist
	}

	sorted := make([]*storage.PartInfo, len(parts))
	copy(sorted, parts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Number < sorted[j].Number
	})

	totalSize := 0
	for _, part := range sorted {
		pd, exists := upload.parts[part.Number]
		if !exists {
			b.st.mp.mu.Unlock()
			return nil, fmt.Errorf("owl: part %d not found", part.Number)
		}
		totalSize += len(pd.data)
	}

	data := make([]byte, 0, totalSize)
	for _, part := range sorted {
		pd := upload.parts[part.Number]
		data = append(data, pd.data...)
	}

	delete(b.st.mp.uploads, mu.UploadID)
	b.st.mp.mu.Unlock()

	// Write assembled object via normal Write path.
	return b.Write(ctx, upload.mu.Key, bytes.NewReader(data), int64(totalSize), upload.contentType, nil)
}

func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	_ = ctx
	_ = opts

	b.st.mp.mu.Lock()
	defer b.st.mp.mu.Unlock()

	if _, ok := b.st.mp.uploads[mu.UploadID]; !ok {
		return storage.ErrNotExist
	}

	delete(b.st.mp.uploads, mu.UploadID)
	return nil
}

// =============================================================================
// ITERATORS
// =============================================================================

type bucketIter struct {
	buckets []*storage.BucketInfo
	index   int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.index >= len(it.buckets) {
		return nil, nil
	}
	b := it.buckets[it.index]
	it.index++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.buckets = nil
	return nil
}

type objectIter struct {
	objects []*storage.Object
	index   int
}

func (it *objectIter) Next() (*storage.Object, error) {
	if it.index >= len(it.objects) {
		return nil, nil
	}
	o := it.objects[it.index]
	it.index++
	return o, nil
}

func (it *objectIter) Close() error {
	it.objects = nil
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// Cached time to avoid time.Now() overhead.
var cachedTimeNano atomic.Int64

func init() {
	cachedTimeNano.Store(time.Now().UnixNano())
}

func fastNow() int64 {
	return cachedTimeNano.Load()
}

func fastNowTime() time.Time {
	return time.Unix(0, fastNow())
}

func applyRange(data []byte, offset, length int64) []byte {
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
	return data[offset:end]
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
