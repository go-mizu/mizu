// Package algo provides StreamlineIndexer - ultra-fast single-threaded indexer.
// Key insight: Lock-free single-threaded can be faster than parallel with locks.
// Target: 1M+ docs/sec with <5GB memory via streaming disk writes.
package algo

import (
	"bufio"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"unsafe"
)

// StreamlineConfig configures the streamline indexer.
type StreamlineConfig struct {
	SegmentDocs int // Documents per segment (default: 250000)
	FlushBytes  int // Memory threshold for flushing (default: 500MB)
}

// StreamlineIndexer is an ultra-fast single-threaded indexer.
// Architecture: Single thread → Memory segment → Disk flush → Repeat
// No channels, no locks, no goroutines during indexing.
type StreamlineIndexer struct {
	config  StreamlineConfig
	outDir  string

	// Current segment state
	terms       map[string]*compactPostings
	docLens     []uint16
	totalLen    int64
	segDocStart uint32

	// Global state
	allSegments   []*streamDiskSeg
	totalDocs     int
	globalDocLens []uint16

	// Memory tracking
	estMemBytes int64
}

// compactPostings uses more compact memory layout.
type compactPostings struct {
	data []byte // Packed: [docID:4][freq:2] repeated
}

// streamDiskSeg represents a segment written to disk.
type streamDiskSeg struct {
	path    string
	numDocs int
	numTerms int
}

// NewStreamlineIndexer creates a new streamline indexer.
func NewStreamlineIndexer(outDir string, cfg StreamlineConfig) *StreamlineIndexer {
	if cfg.SegmentDocs <= 0 {
		cfg.SegmentDocs = 250000
	}
	if cfg.FlushBytes <= 0 {
		cfg.FlushBytes = 500 * 1024 * 1024 // 500MB
	}

	os.MkdirAll(outDir, 0755)

	return &StreamlineIndexer{
		config:      cfg,
		outDir:      outDir,
		terms:       make(map[string]*compactPostings, 200000),
		docLens:     make([]uint16, 0, cfg.SegmentDocs),
		allSegments: make([]*streamDiskSeg, 0, 16),
		globalDocLens: make([]uint16, 0, 4000000),
	}
}

// Add indexes a single document. Call sequentially for best performance.
func (si *StreamlineIndexer) Add(docID uint32, text string) {
	// Ultra-fast tokenization directly into postings
	si.tokenizeAndIndex(docID, text)

	// Check if we need to flush segment
	if len(si.docLens) >= si.config.SegmentDocs || si.estMemBytes > int64(si.config.FlushBytes) {
		si.flushSegment()
	}
}

// tokenizeAndIndex is a fused tokenize+index operation for cache efficiency.
func (si *StreamlineIndexer) tokenizeAndIndex(docID uint32, text string) {
	data := *(*[]byte)(unsafe.Pointer(&text))
	n := len(data)
	start := -1
	docLen := uint16(0)

	// Posting data: [docID:4][freq:2]
	posting := make([]byte, 6)
	binary.LittleEndian.PutUint32(posting[0:4], docID)

	// Temporary freq map for this document
	freqs := make(map[string]uint16, 64)

	for i := 0; i < n; i++ {
		c := data[i]
		isDelim := c <= ' ' || (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
			(c >= '[' && c <= '`') || (c >= '{' && c <= '~')

		if isDelim {
			if start >= 0 && i-start < 100 {
				term := toLowerInline(data[start:i])
				freqs[term]++
				docLen++
			}
			start = -1
		} else if start < 0 {
			start = i
		}
	}

	// Last token
	if start >= 0 && n-start < 100 {
		term := toLowerInline(data[start:n])
		freqs[term]++
		docLen++
	}

	// Add all postings
	for term, freq := range freqs {
		binary.LittleEndian.PutUint16(posting[4:6], freq)

		pl, exists := si.terms[term]
		if !exists {
			pl = &compactPostings{data: make([]byte, 0, 64)}
			si.terms[term] = pl
			si.estMemBytes += int64(len(term) + 64)
		}
		pl.data = append(pl.data, posting...)
		si.estMemBytes += 6
	}

	// Store doc length
	si.docLens = append(si.docLens, docLen)
	si.globalDocLens = append(si.globalDocLens, docLen)
	si.totalLen += int64(docLen)
	si.totalDocs++
}

// toLowerInline converts bytes to lowercase string without extra allocation.
func toLowerInline(data []byte) string {
	buf := make([]byte, len(data))
	for i, c := range data {
		if c >= 'A' && c <= 'Z' {
			buf[i] = c + 32
		} else {
			buf[i] = c
		}
	}
	return string(buf)
}

// flushSegment writes current segment to disk and resets state.
func (si *StreamlineIndexer) flushSegment() {
	if len(si.docLens) == 0 {
		return
	}

	segID := len(si.allSegments)
	segPath := filepath.Join(si.outDir, segmentFilename(segID))

	// Write segment
	err := si.writeSegmentFile(segPath)
	if err != nil {
		return // Best effort
	}

	si.allSegments = append(si.allSegments, &streamDiskSeg{
		path:     segPath,
		numDocs:  len(si.docLens),
		numTerms: len(si.terms),
	})

	// Reset segment state
	si.segDocStart += uint32(len(si.docLens))
	si.terms = make(map[string]*compactPostings, 200000)
	si.docLens = si.docLens[:0]
	si.estMemBytes = 0

	// Help GC
	runtime.GC()
}

func segmentFilename(id int) string {
	return "seg_" + itoa(id) + ".bin"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// writeSegmentFile writes the current segment to a binary file.
func (si *StreamlineIndexer) writeSegmentFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 4*1024*1024)

	// Sort terms
	sortedTerms := make([]string, 0, len(si.terms))
	for term := range si.terms {
		sortedTerms = append(sortedTerms, term)
	}
	sort.Strings(sortedTerms)

	// Header: [numDocs:4][numTerms:4][segDocStart:4]
	header := make([]byte, 12)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(si.docLens)))
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(sortedTerms)))
	binary.LittleEndian.PutUint32(header[8:12], si.segDocStart)
	w.Write(header)

	// Terms: [termLen:2][term:N][postingsLen:4][postings...]
	buf := make([]byte, 6)
	for _, term := range sortedTerms {
		pl := si.terms[term]

		// Term string
		binary.LittleEndian.PutUint16(buf[0:2], uint16(len(term)))
		w.Write(buf[:2])
		w.WriteString(term)

		// Postings length and data
		binary.LittleEndian.PutUint32(buf[0:4], uint32(len(pl.data)))
		w.Write(buf[:4])
		w.Write(pl.data)
	}

	// Doc lengths: [len:2] repeated
	for _, dl := range si.docLens {
		binary.LittleEndian.PutUint16(buf[0:2], dl)
		w.Write(buf[:2])
	}

	return w.Flush()
}

// Finish completes indexing and returns a searchable index.
func (si *StreamlineIndexer) Finish() (*SegmentedIndex, error) {
	// Flush any remaining segment
	si.flushSegment()

	if si.totalDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(si.totalLen) / float64(si.totalDocs)

	// Load all segments into memory for search
	segments := make([]*SearchSegment, 0, len(si.allSegments))

	for i, diskSeg := range si.allSegments {
		seg, err := loadSegmentFile(diskSeg.path)
		if err != nil {
			continue
		}
		seg.id = i
		segments = append(segments, seg)
	}

	return &SegmentedIndex{
		segments:  segments,
		numDocs:   si.totalDocs,
		avgDocLen: avgDocLen,
		docLens:   si.globalDocLens,
	}, nil
}

// loadSegmentFile reads a segment from disk into memory.
func loadSegmentFile(path string) (*SearchSegment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) < 12 {
		return nil, os.ErrInvalid
	}

	numDocs := binary.LittleEndian.Uint32(data[0:4])
	numTerms := binary.LittleEndian.Uint32(data[4:8])
	segDocStart := binary.LittleEndian.Uint32(data[8:12])

	offset := 12
	terms := make(map[string]*SegmentPostings, numTerms)

	for i := uint32(0); i < numTerms; i++ {
		if offset+2 > len(data) {
			break
		}
		termLen := int(binary.LittleEndian.Uint16(data[offset : offset+2]))
		offset += 2

		if offset+termLen > len(data) {
			break
		}
		term := string(data[offset : offset+termLen])
		offset += termLen

		if offset+4 > len(data) {
			break
		}
		postingsLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+postingsLen > len(data) {
			break
		}

		// Parse postings
		numPostings := postingsLen / 6
		docIDs := make([]uint32, numPostings)
		freqs := make([]uint16, numPostings)

		for j := 0; j < numPostings; j++ {
			docIDs[j] = binary.LittleEndian.Uint32(data[offset : offset+4])
			freqs[j] = binary.LittleEndian.Uint16(data[offset+4 : offset+6])
			offset += 6
		}

		terms[term] = &SegmentPostings{
			DocIDs: docIDs,
			Freqs:  freqs,
		}
	}

	// Parse doc lengths
	docLens := make(map[uint32]uint16, numDocs)
	for i := uint32(0); i < numDocs; i++ {
		if offset+2 > len(data) {
			break
		}
		dl := binary.LittleEndian.Uint16(data[offset : offset+2])
		docLens[segDocStart+i] = dl
		offset += 2
	}

	return &SearchSegment{
		terms:   terms,
		docLens: docLens,
		numDocs: int(numDocs),
	}, nil
}

// ParallelStreamlineIndexer uses multiple StreamlineIndexers in parallel.
type ParallelStreamlineIndexer struct {
	outDir    string
	numWorkers int

	workers   []*StreamlineIndexer
	docCh     chan indexDoc
	wg        sync.WaitGroup

	mu        sync.Mutex
	nextDocID uint32
}

type indexDoc struct {
	docID uint32
	text  string
}

// NewParallelStreamlineIndexer creates a parallel indexer.
func NewParallelStreamlineIndexer(outDir string, numWorkers int) *ParallelStreamlineIndexer {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	psi := &ParallelStreamlineIndexer{
		outDir:     outDir,
		numWorkers: numWorkers,
		workers:    make([]*StreamlineIndexer, numWorkers),
		docCh:      make(chan indexDoc, numWorkers*1000),
	}

	// Create per-worker subdirectories
	for i := 0; i < numWorkers; i++ {
		workerDir := filepath.Join(outDir, "worker_"+itoa(i))
		psi.workers[i] = NewStreamlineIndexer(workerDir, StreamlineConfig{
			SegmentDocs: 250000 / numWorkers,
			FlushBytes:  500 * 1024 * 1024 / numWorkers,
		})

		psi.wg.Add(1)
		go psi.runWorker(i)
	}

	return psi
}

func (psi *ParallelStreamlineIndexer) runWorker(workerID int) {
	defer psi.wg.Done()
	worker := psi.workers[workerID]

	for doc := range psi.docCh {
		worker.Add(doc.docID, doc.text)
	}
}

// Add indexes a document.
func (psi *ParallelStreamlineIndexer) Add(docID uint32, text string) {
	psi.docCh <- indexDoc{docID: docID, text: text}
}

// Finish completes indexing and returns a merged index.
func (psi *ParallelStreamlineIndexer) Finish() (*SegmentedIndex, error) {
	close(psi.docCh)
	psi.wg.Wait()

	// Collect all segments from all workers
	var allSegments []*SearchSegment
	var totalDocs int
	var totalLen int64
	var globalDocLens []uint16

	for _, worker := range psi.workers {
		worker.flushSegment()

		for _, diskSeg := range worker.allSegments {
			seg, err := loadSegmentFile(diskSeg.path)
			if err != nil {
				continue
			}
			seg.id = len(allSegments)
			allSegments = append(allSegments, seg)
		}

		totalDocs += worker.totalDocs
		totalLen += worker.totalLen
		globalDocLens = append(globalDocLens, worker.globalDocLens...)
	}

	if totalDocs == 0 {
		return nil, nil
	}

	avgDocLen := float64(totalLen) / float64(totalDocs)

	return &SegmentedIndex{
		segments:  allSegments,
		numDocs:   totalDocs,
		avgDocLen: avgDocLen,
		docLens:   globalDocLens,
	}, nil
}
