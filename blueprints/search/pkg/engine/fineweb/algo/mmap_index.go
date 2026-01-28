// Package algo provides memory-mapped index for <1GB peak memory indexing.
// The index is stored entirely on disk and accessed via mmap during search.
package algo

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"syscall"
)

// MmapIndex provides memory-mapped access to an on-disk search index.
// Memory usage is controlled by OS page cache, not Go heap.
type MmapIndex struct {
	// Memory-mapped data
	data     []byte
	file     *os.File

	// Parsed header
	header   *MmapHeader

	// Index into term dictionary (loaded into small amount of memory)
	termIndex map[string]uint64 // term -> offset in posting section

	// Metadata
	NumDocs   int
	AvgDocLen float64
	DocLens   []uint16 // Loaded into memory (small: 2 bytes per doc)
}

// MmapHeader is the index file header.
type MmapHeader struct {
	Magic       [8]byte  // "MMAPIDX1"
	Version     uint32
	NumDocs     uint32
	NumTerms    uint32
	AvgDocLen   float64

	// Section offsets
	TermDictOffset   uint64
	PostingsOffset   uint64
	DocLensOffset    uint64
	DocMetaOffset    uint64

	// Section sizes
	TermDictSize     uint64
	PostingsSize     uint64
	DocLensSize      uint64
	DocMetaSize      uint64
}

const (
	MmapHeaderSize = 128
	MmapMagic      = "MMAPIDX1"
)

// MmapIndexWriter builds a memory-mapped index file with minimal memory.
// Uses a two-file streaming approach:
// 1. Writes postings directly to temp file (no memory accumulation)
// 2. Assembles final file: header, term dict, postings (from temp), doc lens
type MmapIndexWriter struct {
	path           string
	postingsFile   *os.File      // temp file for postings
	postingsWriter *bufio.Writer // buffered writer for performance
	postingsPath   string

	// Buffered writes (small: just term metadata, not postings)
	termDict     []termEntry
	docLens      []uint16

	// Stats
	numDocs      int
	avgDocLen    float64
	postingOff   uint64
}

type termEntry struct {
	term       string
	offset     uint64 // Offset in postings section
	docFreq    uint32
	idf        float32
}

// NewMmapIndexWriter creates a new index writer with streaming postings.
func NewMmapIndexWriter(path string) (*MmapIndexWriter, error) {
	// Create temp file for postings (streamed directly, not buffered in memory)
	postingsPath := path + ".postings.tmp"
	pf, err := os.Create(postingsPath)
	if err != nil {
		return nil, err
	}

	return &MmapIndexWriter{
		path:           path,
		postingsFile:   pf,
		postingsWriter: bufio.NewWriterSize(pf, 4*1024*1024), // 4MB write buffer
		postingsPath:   postingsPath,
		termDict:       make([]termEntry, 0, 100000),
		docLens:        make([]uint16, 0, 100000),
	}, nil
}

// SetDocCount sets the total document count and average length.
func (w *MmapIndexWriter) SetDocCount(numDocs int, avgDocLen float64) {
	w.numDocs = numDocs
	w.avgDocLen = avgDocLen
}

// AddDocLen adds a document length.
func (w *MmapIndexWriter) AddDocLen(docLen int) {
	if docLen > 65535 {
		docLen = 65535
	}
	w.docLens = append(w.docLens, uint16(docLen))
}

// AddTerm adds a term with its posting list.
// Postings are written directly to buffered temp file, not accumulated in memory.
func (w *MmapIndexWriter) AddTerm(term string, docIDs []uint32, freqs []uint16, idf float32) {
	offset := w.postingOff

	// Write posting list to buffered temp file
	// Format: [count:4][docID:4,freq:2]...
	var countBuf [4]byte
	binary.LittleEndian.PutUint32(countBuf[:], uint32(len(docIDs)))
	w.postingsWriter.Write(countBuf[:])

	var buf [6]byte
	for i := range docIDs {
		binary.LittleEndian.PutUint32(buf[0:4], docIDs[i])
		binary.LittleEndian.PutUint16(buf[4:6], freqs[i])
		w.postingsWriter.Write(buf[:])
	}

	w.postingOff += 4 + uint64(len(docIDs))*6

	// Add term entry (small: just metadata)
	w.termDict = append(w.termDict, termEntry{
		term:    term,
		offset:  offset,
		docFreq: uint32(len(docIDs)),
		idf:     idf,
	})
}

// Finish assembles the final index file and cleans up temp files.
func (w *MmapIndexWriter) Finish() error {
	// Flush and close temp postings file
	w.postingsWriter.Flush()
	w.postingsFile.Close()

	// Sort term dictionary
	sort.Slice(w.termDict, func(i, j int) bool {
		return w.termDict[i].term < w.termDict[j].term
	})

	// Calculate section offsets
	header := &MmapHeader{
		Version:   1,
		NumDocs:   uint32(w.numDocs),
		NumTerms:  uint32(len(w.termDict)),
		AvgDocLen: w.avgDocLen,
	}
	copy(header.Magic[:], MmapMagic)

	// Term dict starts after header
	header.TermDictOffset = MmapHeaderSize

	// Calculate term dict size
	termDictSize := uint64(0)
	for _, te := range w.termDict {
		// Format: [termLen:2][term:var][offset:8][docFreq:4][idf:4]
		termDictSize += 2 + uint64(len(te.term)) + 8 + 4 + 4
	}
	header.TermDictSize = termDictSize

	// Postings follow term dict
	header.PostingsOffset = header.TermDictOffset + header.TermDictSize
	header.PostingsSize = w.postingOff

	// Doc lens follow postings
	header.DocLensOffset = header.PostingsOffset + header.PostingsSize
	header.DocLensSize = uint64(len(w.docLens)) * 2

	// Doc meta follows doc lens (not used in this version)
	header.DocMetaOffset = header.DocLensOffset + header.DocLensSize
	header.DocMetaSize = 0

	// Create final output file
	outFile, err := os.Create(w.path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Write header
	if err := writeHeader(outFile, header); err != nil {
		return err
	}

	// Write term dictionary
	for _, te := range w.termDict {
		// Term length
		if err := binary.Write(outFile, binary.LittleEndian, uint16(len(te.term))); err != nil {
			return err
		}
		// Term bytes
		if _, err := outFile.Write([]byte(te.term)); err != nil {
			return err
		}
		// Offset, docFreq, IDF
		if err := binary.Write(outFile, binary.LittleEndian, te.offset); err != nil {
			return err
		}
		if err := binary.Write(outFile, binary.LittleEndian, te.docFreq); err != nil {
			return err
		}
		if err := binary.Write(outFile, binary.LittleEndian, te.idf); err != nil {
			return err
		}
	}

	// Copy postings from temp file (streaming, not loading all into memory)
	postingsIn, err := os.Open(w.postingsPath)
	if err != nil {
		return err
	}
	defer postingsIn.Close()

	// Stream copy using buffer
	buf := make([]byte, 4*1024*1024) // 4MB copy buffer
	if _, err := io.CopyBuffer(outFile, postingsIn, buf); err != nil {
		return err
	}

	// Write doc lens
	for _, dl := range w.docLens {
		if err := binary.Write(outFile, binary.LittleEndian, dl); err != nil {
			return err
		}
	}

	// Clean up temp file
	os.Remove(w.postingsPath)

	return nil
}

func writeHeader(f *os.File, h *MmapHeader) error {
	// Write fixed-size header
	buf := make([]byte, MmapHeaderSize)
	copy(buf[0:8], h.Magic[:])
	binary.LittleEndian.PutUint32(buf[8:12], h.Version)
	binary.LittleEndian.PutUint32(buf[12:16], h.NumDocs)
	binary.LittleEndian.PutUint32(buf[16:20], h.NumTerms)

	// AvgDocLen as float64
	binary.LittleEndian.PutUint64(buf[20:28], math.Float64bits(h.AvgDocLen))

	// Section offsets
	binary.LittleEndian.PutUint64(buf[28:36], h.TermDictOffset)
	binary.LittleEndian.PutUint64(buf[36:44], h.PostingsOffset)
	binary.LittleEndian.PutUint64(buf[44:52], h.DocLensOffset)
	binary.LittleEndian.PutUint64(buf[52:60], h.DocMetaOffset)

	// Section sizes
	binary.LittleEndian.PutUint64(buf[60:68], h.TermDictSize)
	binary.LittleEndian.PutUint64(buf[68:76], h.PostingsSize)
	binary.LittleEndian.PutUint64(buf[76:84], h.DocLensSize)
	binary.LittleEndian.PutUint64(buf[84:92], h.DocMetaSize)

	_, err := f.Write(buf)
	return err
}

// OpenMmapIndex opens an existing memory-mapped index.
func OpenMmapIndex(path string) (*MmapIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Get file size
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	size := fi.Size()

	// Memory-map the file
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	idx := &MmapIndex{
		data: data,
		file: f,
	}

	// Parse header
	if err := idx.parseHeader(); err != nil {
		idx.Close()
		return nil, err
	}

	// Load term index (small: just term -> offset mapping)
	if err := idx.loadTermIndex(); err != nil {
		idx.Close()
		return nil, err
	}

	// Load doc lens (small: 2 bytes per doc)
	if err := idx.loadDocLens(); err != nil {
		idx.Close()
		return nil, err
	}

	return idx, nil
}

func (idx *MmapIndex) parseHeader() error {
	if len(idx.data) < MmapHeaderSize {
		return fmt.Errorf("file too small for header")
	}

	h := &MmapHeader{}
	copy(h.Magic[:], idx.data[0:8])
	if string(h.Magic[:]) != MmapMagic {
		return fmt.Errorf("invalid magic: %s", h.Magic)
	}

	h.Version = binary.LittleEndian.Uint32(idx.data[8:12])
	h.NumDocs = binary.LittleEndian.Uint32(idx.data[12:16])
	h.NumTerms = binary.LittleEndian.Uint32(idx.data[16:20])
	h.AvgDocLen = math.Float64frombits(binary.LittleEndian.Uint64(idx.data[20:28]))

	h.TermDictOffset = binary.LittleEndian.Uint64(idx.data[28:36])
	h.PostingsOffset = binary.LittleEndian.Uint64(idx.data[36:44])
	h.DocLensOffset = binary.LittleEndian.Uint64(idx.data[44:52])
	h.DocMetaOffset = binary.LittleEndian.Uint64(idx.data[52:60])

	h.TermDictSize = binary.LittleEndian.Uint64(idx.data[60:68])
	h.PostingsSize = binary.LittleEndian.Uint64(idx.data[68:76])
	h.DocLensSize = binary.LittleEndian.Uint64(idx.data[76:84])
	h.DocMetaSize = binary.LittleEndian.Uint64(idx.data[84:92])

	idx.header = h
	idx.NumDocs = int(h.NumDocs)
	idx.AvgDocLen = h.AvgDocLen

	return nil
}

func (idx *MmapIndex) loadTermIndex() error {
	idx.termIndex = make(map[string]uint64, idx.header.NumTerms)

	offset := idx.header.TermDictOffset
	for i := uint32(0); i < idx.header.NumTerms; i++ {
		// Read term length
		termLen := binary.LittleEndian.Uint16(idx.data[offset : offset+2])
		offset += 2

		// Read term
		term := string(idx.data[offset : offset+uint64(termLen)])
		offset += uint64(termLen)

		// Read posting offset
		postingOffset := binary.LittleEndian.Uint64(idx.data[offset : offset+8])
		offset += 8

		// Skip docFreq and IDF (we'll read them from postings)
		offset += 8 // 4 + 4

		idx.termIndex[term] = postingOffset
	}

	return nil
}

func (idx *MmapIndex) loadDocLens() error {
	idx.DocLens = make([]uint16, idx.header.NumDocs)

	offset := idx.header.DocLensOffset
	for i := uint32(0); i < idx.header.NumDocs; i++ {
		idx.DocLens[i] = binary.LittleEndian.Uint16(idx.data[offset : offset+2])
		offset += 2
	}

	return nil
}

// GetPostings returns the posting list for a term.
// Returns docIDs, frequencies, and IDF. Data is read directly from mmap.
func (idx *MmapIndex) GetPostings(term string) ([]uint32, []uint16, float32, bool) {
	postingOffset, exists := idx.termIndex[term]
	if !exists {
		return nil, nil, 0, false
	}

	// Read from mmap'd data
	baseOffset := idx.header.PostingsOffset + postingOffset

	// Read count
	count := binary.LittleEndian.Uint32(idx.data[baseOffset : baseOffset+4])
	baseOffset += 4

	// Read postings
	docIDs := make([]uint32, count)
	freqs := make([]uint16, count)

	for i := uint32(0); i < count; i++ {
		docIDs[i] = binary.LittleEndian.Uint32(idx.data[baseOffset : baseOffset+4])
		freqs[i] = binary.LittleEndian.Uint16(idx.data[baseOffset+4 : baseOffset+6])
		baseOffset += 6
	}

	// Compute IDF
	n := float64(idx.NumDocs)
	df := float64(count)
	idf := float32(math.Log((n-df+0.5)/(df+0.5) + 1))

	return docIDs, freqs, idf, true
}

// Close unmaps and closes the index file.
func (idx *MmapIndex) Close() error {
	if idx.data != nil {
		syscall.Munmap(idx.data)
	}
	if idx.file != nil {
		return idx.file.Close()
	}
	return nil
}

// MmapSearch performs BM25 search on the mmap'd index.
func (idx *MmapIndex) Search(queryTerms []string, limit int) []MmapSearchResult {
	// Collect posting lists for query terms
	type termData struct {
		docIDs []uint32
		freqs  []uint16
		idf    float32
	}
	termDatas := make([]termData, 0, len(queryTerms))

	for _, term := range queryTerms {
		docIDs, freqs, idf, exists := idx.GetPostings(term)
		if exists {
			termDatas = append(termDatas, termData{docIDs, freqs, idf})
		}
	}

	if len(termDatas) == 0 {
		return nil
	}

	// BM25 scoring
	k1 := float32(1.2)
	b := float32(0.75)
	avgDL := float32(idx.AvgDocLen)

	// Score documents
	scores := make(map[uint32]float32)

	for _, td := range termDatas {
		for i, docID := range td.docIDs {
			tf := float32(td.freqs[i])
			dl := float32(idx.DocLens[docID])
			tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
			scores[docID] += td.idf * tfNorm
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

// MmapSearchResult is a search result from mmap index.
type MmapSearchResult struct {
	DocID uint32
	Score float32
}

// TrueStreamingMerger performs streaming merge with minimal memory:
// - Only holds term dictionary metadata in memory (small: term strings + offsets)
// - Reads postings sequentially from segment files (no seeking during merge)
// - Writes directly to output without accumulating all postings
//
// Memory usage: ~O(num_unique_terms * avg_term_length) for term index
// NOT: O(total_postings) which was the previous problem
type TrueStreamingMerger struct {
	outputPath   string
	segmentPaths []string
	numDocs      int
	avgDocLen    float64
}

// streamingSegment provides sequential reading of a segment's postings.
// Keeps file open and reads terms in order they appear (sorted alphabetically).
type streamingSegment struct {
	path           string
	file           *os.File
	reader         *bufio.Reader
	terms          []streamingTermEntry // terms in file order (alphabetical)
	termIndex      map[string]int       // term -> index in terms slice
	docLens        map[uint32]uint16    // docID -> length
	currentTermIdx int                  // next term to read
}

type streamingTermEntry struct {
	term  string
	count uint32
}

// NewTrueStreamingMerger creates a merger with minimal memory usage.
func NewTrueStreamingMerger(outputPath string, segmentPaths []string) *TrueStreamingMerger {
	return &TrueStreamingMerger{
		outputPath:   outputPath,
		segmentPaths: segmentPaths,
	}
}

// openStreamingSegment opens a segment for sequential reading.
func openStreamingSegment(path string) (*streamingSegment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReaderSize(f, 1024*1024) // 1MB buffer

	seg := &streamingSegment{
		path:      path,
		file:      f,
		reader:    reader,
		termIndex: make(map[string]int, 30000),
		docLens:   make(map[uint32]uint16, 50000),
	}

	// Read header
	var numDocs, numTerms uint32
	binary.Read(reader, binary.LittleEndian, &numDocs)
	binary.Read(reader, binary.LittleEndian, &numTerms)

	// Read term dictionary (in file order = alphabetical)
	seg.terms = make([]streamingTermEntry, numTerms)
	for i := uint32(0); i < numTerms; i++ {
		var termLen uint16
		binary.Read(reader, binary.LittleEndian, &termLen)
		termBytes := make([]byte, termLen)
		io.ReadFull(reader, termBytes)
		var count uint32
		var offset int64 // skip offset, we'll read sequentially
		binary.Read(reader, binary.LittleEndian, &count)
		binary.Read(reader, binary.LittleEndian, &offset)

		term := string(termBytes)
		seg.terms[i] = streamingTermEntry{term: term, count: count}
		seg.termIndex[term] = int(i)
	}

	// Now positioned at start of postings section
	// Will read postings sequentially as terms are requested

	return seg, nil
}

// readNextTerm reads the next term's postings in sequence.
// Returns term name, docIDs, freqs. Must be called in term order.
// Uses bulk read for better I/O efficiency.
func (s *streamingSegment) readNextTerm() (string, []uint32, []uint16) {
	if s.currentTermIdx >= len(s.terms) {
		return "", nil, nil
	}

	entry := s.terms[s.currentTermIdx]
	s.currentTermIdx++

	// Allocate arrays
	docIDs := make([]uint32, entry.count)
	freqs := make([]uint16, entry.count)

	// Bulk read all postings at once (6 bytes per posting)
	postingSize := int(entry.count) * 6
	buf := make([]byte, postingSize)
	io.ReadFull(s.reader, buf)

	// Parse postings from buffer (much faster than binary.Read)
	for i := uint32(0); i < entry.count; i++ {
		offset := int(i) * 6
		docIDs[i] = uint32(buf[offset]) |
			uint32(buf[offset+1])<<8 |
			uint32(buf[offset+2])<<16 |
			uint32(buf[offset+3])<<24
		freqs[i] = uint16(buf[offset+4]) | uint16(buf[offset+5])<<8
	}

	return entry.term, docIDs, freqs
}

// hasTerm checks if segment has the given term.
func (s *streamingSegment) hasTerm(term string) bool {
	_, exists := s.termIndex[term]
	return exists
}

// currentTerm returns the term at current read position (without advancing).
func (s *streamingSegment) currentTerm() string {
	if s.currentTermIdx >= len(s.terms) {
		return ""
	}
	return s.terms[s.currentTermIdx].term
}

// skipToTerm skips postings until reaching the given term.
// Returns docIDs, freqs for the term, or nil if term not in segment.
func (s *streamingSegment) skipToTerm(term string) ([]uint32, []uint16) {
	targetIdx, exists := s.termIndex[term]
	if !exists {
		return nil, nil
	}

	// Skip any terms before our target
	for s.currentTermIdx < targetIdx {
		entry := s.terms[s.currentTermIdx]
		s.currentTermIdx++
		// Skip this term's postings
		skipBytes := int64(entry.count) * 6 // 4 bytes docID + 2 bytes freq
		s.reader.Discard(int(skipBytes))
	}

	// Read the target term
	if s.currentTermIdx == targetIdx {
		_, docIDs, freqs := s.readNextTerm()
		return docIDs, freqs
	}

	return nil, nil
}

// finishReadingPostings skips remaining postings and reads doc lengths.
func (s *streamingSegment) finishReadingPostings() {
	// Skip any remaining postings
	for s.currentTermIdx < len(s.terms) {
		entry := s.terms[s.currentTermIdx]
		s.currentTermIdx++
		skipBytes := int64(entry.count) * 6
		s.reader.Discard(int(skipBytes))
	}

	// Read doc lengths using bulk read
	var countBuf [4]byte
	io.ReadFull(s.reader, countBuf[:])
	docLenCount := uint32(countBuf[0]) | uint32(countBuf[1])<<8 | uint32(countBuf[2])<<16 | uint32(countBuf[3])<<24

	// Bulk read all doc lengths (6 bytes each: 4 docID + 2 length)
	bulkSize := int(docLenCount) * 6
	buf := make([]byte, bulkSize)
	io.ReadFull(s.reader, buf)

	for i := uint32(0); i < docLenCount; i++ {
		offset := int(i) * 6
		docID := uint32(buf[offset]) | uint32(buf[offset+1])<<8 | uint32(buf[offset+2])<<16 | uint32(buf[offset+3])<<24
		length := uint16(buf[offset+4]) | uint16(buf[offset+5])<<8
		s.docLens[docID] = length
	}
}

func (s *streamingSegment) close() {
	if s.file != nil {
		s.file.Close()
	}
}

// Merge performs true streaming merge with minimal memory.
func (m *TrueStreamingMerger) Merge() error {
	// Phase 1: Open all segments for streaming
	segments := make([]*streamingSegment, len(m.segmentPaths))
	termSet := make(map[string]struct{}, 100000)

	for i, path := range m.segmentPaths {
		seg, err := openStreamingSegment(path)
		if err != nil {
			// Clean up already opened segments
			for j := 0; j < i; j++ {
				segments[j].close()
			}
			return fmt.Errorf("opening segment %s: %w", path, err)
		}
		segments[i] = seg

		// Collect unique terms (just strings from term index)
		for term := range seg.termIndex {
			termSet[term] = struct{}{}
		}
	}

	// Build sorted term list
	terms := make([]string, 0, len(termSet))
	for term := range termSet {
		terms = append(terms, term)
	}
	sort.Strings(terms)
	termSet = nil // Free

	// Phase 2: Stream merge postings term by term
	// Create output writer first (will write term dict after postings known)
	writer, err := NewMmapIndexWriter(m.outputPath)
	if err != nil {
		for _, seg := range segments {
			seg.close()
		}
		return err
	}

	// Reusable buffers to reduce allocations
	postingBuf := make([]IndexPosting, 0, 50000)

	for _, term := range terms {
		postingBuf = postingBuf[:0]

		// Read from each segment (sequential, no seeking)
		for _, seg := range segments {
			docIDs, freqs := seg.skipToTerm(term)
			for i := range docIDs {
				postingBuf = append(postingBuf, IndexPosting{
					DocID: docIDs[i],
					Freq:  freqs[i],
				})
			}
		}

		if len(postingBuf) == 0 {
			continue
		}

		// Sort by docID using stdlib sort (highly optimized)
		sort.Slice(postingBuf, func(i, j int) bool {
			return postingBuf[i].DocID < postingBuf[j].DocID
		})

		// Extract sorted arrays
		docIDs := make([]uint32, len(postingBuf))
		freqs := make([]uint16, len(postingBuf))
		for i, p := range postingBuf {
			docIDs[i] = p.DocID
			freqs[i] = p.Freq
		}

		writer.AddTerm(term, docIDs, freqs, 0)
	}

	// Phase 3: Finish reading segments to get doc lengths
	var maxDocID uint32
	var totalDocLen int64

	for _, seg := range segments {
		seg.finishReadingPostings()
		for docID, length := range seg.docLens {
			if docID > maxDocID {
				maxDocID = docID
			}
			totalDocLen += int64(length)
		}
	}

	m.numDocs = int(maxDocID + 1)
	if m.numDocs > 0 {
		m.avgDocLen = float64(totalDocLen) / float64(m.numDocs)
	}

	// Set doc count and lengths
	writer.SetDocCount(m.numDocs, m.avgDocLen)
	docLens := make([]uint16, m.numDocs)
	for _, seg := range segments {
		for docID, length := range seg.docLens {
			docLens[docID] = length
		}
	}
	for _, dl := range docLens {
		writer.AddDocLen(int(dl))
	}

	// Update IDF values in term entries now that we know numDocs
	n := float64(m.numDocs)
	for i := range writer.termDict {
		df := float64(writer.termDict[i].docFreq)
		writer.termDict[i].idf = float32(math.Log((n-df+0.5)/(df+0.5) + 1))
	}

	// Close segments
	for _, seg := range segments {
		seg.close()
	}

	return writer.Finish()
}

// FinishToMmap is a helper for PipelineIndexer to write directly to mmap format.
func (pi *PipelineIndexer) FinishToMmap(outputPath string) (*MmapIndex, error) {
	// Close input and wait for pipeline
	close(pi.docCh)
	pi.wg.Wait()
	pi.indexWg.Wait()
	pi.writeWg.Wait()

	if len(pi.segments) == 0 {
		return nil, fmt.Errorf("no segments to merge")
	}

	// Collect segment paths
	segmentPaths := make([]string, len(pi.segments))
	for i, meta := range pi.segments {
		segmentPaths[i] = meta.Path
	}

	// Use true streaming merger for minimal memory
	merger := NewTrueStreamingMerger(outputPath, segmentPaths)
	if err := merger.Merge(); err != nil {
		return nil, err
	}

	// Clean up segment files
	for _, path := range segmentPaths {
		os.Remove(path)
	}

	// Open the mmap index
	return OpenMmapIndex(outputPath)
}

// Helper to ensure interface is used
var _ io.Closer = (*MmapIndex)(nil)
