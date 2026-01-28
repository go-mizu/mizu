// Package algo provides segment-based search without merging.
// This is the LSM-tree approach used by Lucene/Elasticsearch.
package algo

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

// SegmentedIndex stores multiple segment files and searches across them.
// No merge required - segments are searched in parallel at query time.
type SegmentedIndex struct {
	segments []*SearchSegment
	numDocs  int
	avgDocLen float64
	docLens  []uint16
	mu       sync.RWMutex
}

// SearchSegment is a single searchable segment loaded in memory.
type SearchSegment struct {
	id        int
	terms     map[string]*SegmentPostings
	docLens   map[uint32]uint16
	numDocs   int
}

// SegmentPostings stores posting list for a term in a segment.
type SegmentPostings struct {
	DocIDs []uint32
	Freqs  []uint16
}

// NoMergeIndexer indexes documents into segments without final merge.
// Achieves high throughput by avoiding the expensive merge phase.
type NoMergeIndexer struct {
	// Configuration
	SegmentSize int
	NumWorkers  int
	OutputDir   string
	Tokenizer   TokenizerFunc

	// Pipeline channels
	docCh       chan nmDoc
	tokenizedCh chan nmTokenized
	segmentCh   chan *nmSegment

	// State
	currentSeg  *nmSegmentBuilder
	segmentID   int
	segmentMu   sync.Mutex
	segments    []string // paths to written segments

	// Document lengths (for BM25)
	docLens     []uint16
	docLensMu   sync.Mutex
	totalDocLen int64
	numDocs     int

	// Sync
	wg       sync.WaitGroup
	indexWg  sync.WaitGroup
	writeWg  sync.WaitGroup
}

type nmDoc struct {
	docID uint32
	text  string
}

type nmTokenized struct {
	docID  uint32
	terms  map[string]int
	docLen int
}

type nmSegmentBuilder struct {
	id           int
	termPostings map[string]*SegmentPostings
	docLens      map[uint32]uint16
	numDocs      int
}

type nmSegment struct {
	id           int
	termPostings map[string]*SegmentPostings
	docLens      map[uint32]uint16
	numDocs      int
}

// NewNoMergeIndexer creates an indexer that skips the merge phase.
func NewNoMergeIndexer(outputDir string, tokenizer TokenizerFunc, segmentSize int) *NoMergeIndexer {
	numWorkers := runtime.NumCPU()
	if numWorkers < 4 {
		numWorkers = 4
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	if segmentSize <= 0 {
		segmentSize = 100000 // 100k docs per segment
	}

	os.MkdirAll(outputDir, 0755)

	nm := &NoMergeIndexer{
		SegmentSize:  segmentSize,
		NumWorkers:   numWorkers,
		OutputDir:    outputDir,
		Tokenizer:    tokenizer,
		docCh:        make(chan nmDoc, numWorkers*500),
		tokenizedCh:  make(chan nmTokenized, numWorkers*250),
		segmentCh:    make(chan *nmSegment, 4),
		segments:     make([]string, 0, 32),
		docLens:      make([]uint16, 0, 3000000),
	}

	nm.startTokenizeStage()
	nm.startIndexStage()
	nm.startWriteStage()

	return nm
}

// Add adds a document to be indexed.
func (nm *NoMergeIndexer) Add(docID uint32, text string) {
	nm.docCh <- nmDoc{docID: docID, text: text}
}

func (nm *NoMergeIndexer) startTokenizeStage() {
	for i := 0; i < nm.NumWorkers; i++ {
		nm.wg.Add(1)
		go func() {
			defer nm.wg.Done()
			for doc := range nm.docCh {
				terms := nm.Tokenizer(doc.text)
				docLen := 0
				for _, freq := range terms {
					docLen += freq
				}
				nm.tokenizedCh <- nmTokenized{
					docID:  doc.docID,
					terms:  terms,
					docLen: docLen,
				}
			}
		}()
	}

	go func() {
		nm.wg.Wait()
		close(nm.tokenizedCh)
	}()
}

func (nm *NoMergeIndexer) startIndexStage() {
	nm.indexWg.Add(1)
	go func() {
		defer nm.indexWg.Done()

		nm.currentSeg = nm.newSegmentBuilder()

		for tdoc := range nm.tokenizedCh {
			// Record document length
			nm.docLensMu.Lock()
			for uint32(len(nm.docLens)) <= tdoc.docID {
				nm.docLens = append(nm.docLens, 0)
			}
			docLen := tdoc.docLen
			if docLen > 65535 {
				docLen = 65535
			}
			nm.docLens[tdoc.docID] = uint16(docLen)
			nm.totalDocLen += int64(docLen)
			nm.numDocs++
			nm.docLensMu.Unlock()

			// Add to current segment
			nm.currentSeg.docLens[tdoc.docID] = uint16(docLen)
			for term, freq := range tdoc.terms {
				pl, exists := nm.currentSeg.termPostings[term]
				if !exists {
					pl = &SegmentPostings{
						DocIDs: make([]uint32, 0, 64),
						Freqs:  make([]uint16, 0, 64),
					}
					nm.currentSeg.termPostings[term] = pl
				}
				pl.DocIDs = append(pl.DocIDs, tdoc.docID)
				pl.Freqs = append(pl.Freqs, uint16(freq))
			}
			nm.currentSeg.numDocs++

			// Flush segment when full
			if nm.currentSeg.numDocs >= nm.SegmentSize {
				nm.flushSegment()
			}
		}

		// Flush final partial segment
		if nm.currentSeg.numDocs > 0 {
			nm.flushSegment()
		}

		close(nm.segmentCh)
	}()
}

func (nm *NoMergeIndexer) newSegmentBuilder() *nmSegmentBuilder {
	id := nm.segmentID
	nm.segmentID++
	return &nmSegmentBuilder{
		id:           id,
		termPostings: make(map[string]*SegmentPostings, 50000),
		docLens:      make(map[uint32]uint16, nm.SegmentSize),
	}
}

func (nm *NoMergeIndexer) flushSegment() {
	seg := &nmSegment{
		id:           nm.currentSeg.id,
		termPostings: nm.currentSeg.termPostings,
		docLens:      nm.currentSeg.docLens,
		numDocs:      nm.currentSeg.numDocs,
	}
	nm.segmentCh <- seg
	nm.currentSeg = nm.newSegmentBuilder()
}

func (nm *NoMergeIndexer) startWriteStage() {
	nm.writeWg.Add(1)
	go func() {
		defer nm.writeWg.Done()

		for seg := range nm.segmentCh {
			path := nm.writeSegment(seg)
			nm.segmentMu.Lock()
			nm.segments = append(nm.segments, path)
			nm.segmentMu.Unlock()

			// Clear segment memory
			seg.termPostings = nil
			seg.docLens = nil
			runtime.GC()
		}
	}()
}

func (nm *NoMergeIndexer) writeSegment(seg *nmSegment) string {
	path := filepath.Join(nm.OutputDir, fmt.Sprintf("seg_%05d.bin", seg.id))

	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 4*1024*1024)

	// Write header
	binary.Write(w, binary.LittleEndian, uint32(seg.numDocs))
	binary.Write(w, binary.LittleEndian, uint32(len(seg.termPostings)))

	// Sort terms
	terms := make([]string, 0, len(seg.termPostings))
	for term := range seg.termPostings {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	// Write term dictionary
	for _, term := range terms {
		pl := seg.termPostings[term]
		binary.Write(w, binary.LittleEndian, uint16(len(term)))
		w.WriteString(term)
		binary.Write(w, binary.LittleEndian, uint32(len(pl.DocIDs)))
	}

	// Write posting lists
	for _, term := range terms {
		pl := seg.termPostings[term]
		// Sort by docID
		sortPostingList(pl)
		for i := range pl.DocIDs {
			binary.Write(w, binary.LittleEndian, pl.DocIDs[i])
			binary.Write(w, binary.LittleEndian, pl.Freqs[i])
		}
	}

	// Write doc lengths
	docIDs := make([]uint32, 0, len(seg.docLens))
	for docID := range seg.docLens {
		docIDs = append(docIDs, docID)
	}
	sort.Slice(docIDs, func(i, j int) bool { return docIDs[i] < docIDs[j] })

	binary.Write(w, binary.LittleEndian, uint32(len(docIDs)))
	for _, docID := range docIDs {
		binary.Write(w, binary.LittleEndian, docID)
		binary.Write(w, binary.LittleEndian, seg.docLens[docID])
	}

	w.Flush()
	return path
}

func sortPostingList(pl *SegmentPostings) {
	n := len(pl.DocIDs)
	if n <= 1 {
		return
	}

	// Use indices for sorting
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return pl.DocIDs[indices[i]] < pl.DocIDs[indices[j]]
	})

	// Reorder
	newDocIDs := make([]uint32, n)
	newFreqs := make([]uint16, n)
	for i, idx := range indices {
		newDocIDs[i] = pl.DocIDs[idx]
		newFreqs[i] = pl.Freqs[idx]
	}
	copy(pl.DocIDs, newDocIDs)
	copy(pl.Freqs, newFreqs)
}

// Finish completes indexing and returns a searchable SegmentedIndex.
func (nm *NoMergeIndexer) Finish() (*SegmentedIndex, error) {
	close(nm.docCh)
	nm.wg.Wait()
	nm.indexWg.Wait()
	nm.writeWg.Wait()

	if len(nm.segments) == 0 {
		return nil, fmt.Errorf("no segments created")
	}

	// Load all segments for searching
	segments := make([]*SearchSegment, len(nm.segments))
	for i, path := range nm.segments {
		seg, err := loadSearchSegment(path, i)
		if err != nil {
			return nil, fmt.Errorf("loading segment %s: %w", path, err)
		}
		segments[i] = seg
	}

	avgDocLen := float64(nm.totalDocLen) / float64(nm.numDocs)

	return &SegmentedIndex{
		segments:  segments,
		numDocs:   nm.numDocs,
		avgDocLen: avgDocLen,
		docLens:   nm.docLens,
	}, nil
}

func loadSearchSegment(path string, id int) (*SearchSegment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 4*1024*1024)

	var numDocs, numTerms uint32
	binary.Read(r, binary.LittleEndian, &numDocs)
	binary.Read(r, binary.LittleEndian, &numTerms)

	// Read term dictionary
	type termEntry struct {
		term  string
		count uint32
	}
	termEntries := make([]termEntry, numTerms)
	for i := uint32(0); i < numTerms; i++ {
		var termLen uint16
		binary.Read(r, binary.LittleEndian, &termLen)
		termBytes := make([]byte, termLen)
		io.ReadFull(r, termBytes)
		var count uint32
		binary.Read(r, binary.LittleEndian, &count)
		termEntries[i] = termEntry{term: string(termBytes), count: count}
	}

	// Read posting lists
	terms := make(map[string]*SegmentPostings, numTerms)
	for _, entry := range termEntries {
		pl := &SegmentPostings{
			DocIDs: make([]uint32, entry.count),
			Freqs:  make([]uint16, entry.count),
		}
		for j := uint32(0); j < entry.count; j++ {
			binary.Read(r, binary.LittleEndian, &pl.DocIDs[j])
			binary.Read(r, binary.LittleEndian, &pl.Freqs[j])
		}
		terms[entry.term] = pl
	}

	// Read doc lengths
	var docLenCount uint32
	binary.Read(r, binary.LittleEndian, &docLenCount)
	docLens := make(map[uint32]uint16, docLenCount)
	for i := uint32(0); i < docLenCount; i++ {
		var docID uint32
		var length uint16
		binary.Read(r, binary.LittleEndian, &docID)
		binary.Read(r, binary.LittleEndian, &length)
		docLens[docID] = length
	}

	return &SearchSegment{
		id:      id,
		terms:   terms,
		docLens: docLens,
		numDocs: int(numDocs),
	}, nil
}

// Search performs BM25 search across all segments.
func (si *SegmentedIndex) Search(queryTerms []string, limit int) []MmapSearchResult {
	si.mu.RLock()
	defer si.mu.RUnlock()

	if len(si.segments) == 0 || len(queryTerms) == 0 {
		return nil
	}

	// BM25 parameters
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(si.avgDocLen)
	n := float64(si.numDocs)

	// Score documents from all segments
	scores := make(map[uint32]float32)

	for _, term := range queryTerms {
		// Collect postings from all segments
		totalDF := 0
		for _, seg := range si.segments {
			if pl, exists := seg.terms[term]; exists {
				totalDF += len(pl.DocIDs)
			}
		}

		if totalDF == 0 {
			continue
		}

		// Compute IDF using total document frequency
		df := float64(totalDF)
		idf := float32(math.Log((n-df+0.5)/(df+0.5) + 1))

		// Score documents from each segment
		for _, seg := range si.segments {
			pl, exists := seg.terms[term]
			if !exists {
				continue
			}

			for i, docID := range pl.DocIDs {
				tf := float32(pl.Freqs[i])
				dl := float32(si.docLens[docID])
				tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
				scores[docID] += idf * tfNorm
			}
		}
	}

	// Top-k selection
	results := make([]MmapSearchResult, 0, len(scores))
	for docID, score := range scores {
		results = append(results, MmapSearchResult{DocID: docID, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// NumDocs returns the total number of documents.
func (si *SegmentedIndex) NumDocs() int {
	return si.numDocs
}

// Close releases resources.
func (si *SegmentedIndex) Close() error {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.segments = nil
	si.docLens = nil
	return nil
}
