package fts_zig

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// mmapDriver implements Driver using memory-mapped segments.
// This provides zero-copy reads without CGO.
type mmapDriver struct {
	mu       sync.RWMutex
	basePath string
	profile  Profile
	segments []*mmapSegment
	docs     []string // Buffer for building
	built    bool
	docCount uint32
}

// mmapSegment represents a memory-mapped segment file.
type mmapSegment struct {
	data     []byte
	fd       int
	header   segmentHeader
	termMap  map[uint64]termInfo
	docMetas []docMeta
}

// Segment header (must match Zig's SegmentHeader)
type segmentHeader struct {
	Magic          [4]byte
	Version        uint32
	Profile        uint8
	Reserved       [3]byte
	DocCount       uint32
	TermCount      uint32
	TotalTokens    uint64
	TermsOffset    uint64
	PostingsOffset uint64
	DocsOffset     uint64
	IndexSize      uint64
}

type termInfo struct {
	postingOffset uint64
	docFreq       uint32
}

type docMeta struct {
	length uint32
}

func newMmapDriver(cfg Config) (Driver, error) {
	d := &mmapDriver{
		basePath: cfg.BasePath,
		profile:  cfg.Profile,
		segments: make([]*mmapSegment, 0),
		docs:     make([]string, 0),
	}

	// Try to load existing segments
	if err := d.loadSegments(); err != nil {
		// No segments yet, that's OK
	}

	return d, nil
}

func (d *mmapDriver) loadSegments() error {
	entries, err := os.ReadDir(d.basePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".fts" {
			path := filepath.Join(d.basePath, entry.Name())
			seg, err := openMmapSegment(path)
			if err != nil {
				continue // Skip invalid segments
			}
			d.segments = append(d.segments, seg)
			d.docCount += seg.header.DocCount
		}
	}

	if len(d.segments) > 0 {
		d.built = true
	}

	return nil
}

func openMmapSegment(path string) (*mmapSegment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	size := stat.Size()
	if size < 64 { // Minimum header size
		f.Close()
		return nil, errors.New("segment too small")
	}

	// Memory map the file
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size),
		syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, err
	}

	seg := &mmapSegment{
		data: data,
		fd:   int(f.Fd()),
	}

	// Parse header
	if string(data[0:4]) != "FTSZ" {
		seg.Close()
		return nil, errors.New("invalid segment magic")
	}

	seg.header.Version = binary.LittleEndian.Uint32(data[4:8])
	seg.header.Profile = data[8]
	seg.header.DocCount = binary.LittleEndian.Uint32(data[12:16])
	seg.header.TermCount = binary.LittleEndian.Uint32(data[16:20])
	seg.header.TotalTokens = binary.LittleEndian.Uint64(data[20:28])
	seg.header.TermsOffset = binary.LittleEndian.Uint64(data[28:36])
	seg.header.PostingsOffset = binary.LittleEndian.Uint64(data[36:44])
	seg.header.DocsOffset = binary.LittleEndian.Uint64(data[44:52])
	seg.header.IndexSize = binary.LittleEndian.Uint64(data[52:60])

	// Build term map
	seg.termMap = make(map[uint64]termInfo)
	termEntrySize := uint64(24) // hash(8) + offset(8) + docfreq(4) + padding(4)
	termCount := (seg.header.PostingsOffset - seg.header.TermsOffset) / termEntrySize

	for i := uint64(0); i < termCount; i++ {
		offset := seg.header.TermsOffset + i*termEntrySize
		if offset+termEntrySize > uint64(len(data)) {
			break
		}
		hash := binary.LittleEndian.Uint64(data[offset:])
		postingOffset := binary.LittleEndian.Uint64(data[offset+8:])
		docFreq := binary.LittleEndian.Uint32(data[offset+16:])
		seg.termMap[hash] = termInfo{
			postingOffset: postingOffset,
			docFreq:       docFreq,
		}
	}

	// Load doc metas
	seg.docMetas = make([]docMeta, seg.header.DocCount)
	for i := uint32(0); i < seg.header.DocCount; i++ {
		offset := seg.header.DocsOffset + uint64(i)*4
		if offset+4 > uint64(len(data)) {
			break
		}
		seg.docMetas[i].length = binary.LittleEndian.Uint32(data[offset:])
	}

	return seg, nil
}

func (seg *mmapSegment) Close() {
	if seg.data != nil {
		syscall.Munmap(seg.data)
		seg.data = nil
	}
	if seg.fd != 0 {
		syscall.Close(seg.fd)
		seg.fd = 0
	}
}

func (d *mmapDriver) AddDocument(text string) (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.built && len(d.segments) > 0 {
		return 0, ErrAlreadyBuilt
	}

	d.docs = append(d.docs, text)
	docID := d.docCount
	d.docCount++
	return docID, nil
}

func (d *mmapDriver) AddDocuments(texts []string) error {
	for _, text := range texts {
		if _, err := d.AddDocument(text); err != nil {
			return err
		}
	}
	return nil
}

func (d *mmapDriver) Build() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// For mmap driver, building just marks as ready
	// Actual segment creation would require the Zig library
	d.built = true
	return nil
}

func (d *mmapDriver) Search(query string, limit int) ([]SearchResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.built {
		return nil, ErrNotBuilt
	}

	// Search in-memory docs
	var results []SearchResult
	for i, doc := range d.docs {
		if len(results) >= limit {
			break
		}
		if containsWord(doc, query) {
			results = append(results, SearchResult{
				DocID: uint32(i),
				Score: 1.0,
			})
		}
	}

	// Search segments
	for _, seg := range d.segments {
		if len(results) >= limit {
			break
		}
		segResults := seg.search(query, limit-len(results))
		results = append(results, segResults...)
	}

	return results, nil
}

func (seg *mmapSegment) search(query string, limit int) []SearchResult {
	// Compute hash for query term
	h := hashString(query)

	info, ok := seg.termMap[h]
	if !ok {
		return nil
	}

	// For now, just return doc IDs with the term
	// Full implementation would decode posting list and score
	results := make([]SearchResult, 0, min(int(info.docFreq), limit))
	for i := uint32(0); i < info.docFreq && len(results) < limit; i++ {
		results = append(results, SearchResult{
			DocID: i,
			Score: 1.0,
		})
	}

	return results
}

// hashString computes wyhash (simplified version)
func hashString(s string) uint64 {
	var h uint64 = 0
	for i := 0; i < len(s); i++ {
		h = h*31 + uint64(s[i])
	}
	return h
}

func (d *mmapDriver) Stats() (Stats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var memBytes uint64
	for _, seg := range d.segments {
		memBytes += uint64(len(seg.data))
	}

	return Stats{
		DocCount:    d.docCount,
		MemoryBytes: memBytes,
	}, nil
}

func (d *mmapDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, seg := range d.segments {
		seg.Close()
	}
	d.segments = nil
	d.docs = nil

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
